from __future__ import annotations

import json
import os
import tempfile
import time
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any

from backend import aliases, artifacts, migration_policy, runtime
from backend.verify import verify_api, verify_google_exchange, verify_session
from common.aws import AWSClient
from common.command import CommandError, deployment_lock, protected_json, require_tools, run
from common.terraform import (
    Terraform,
    require_create_only,
    require_cutover_only,
    require_non_destructive_update,
    require_restore_verification_cleanup,
    require_restore_verification_create,
    require_temporary_access_create,
)
from config import Config
from database import backup as database_backup
from database.setup import setup as setup_database
from frontend.publish import load_snapshot_descriptor
from frontend.publish import publish as publish_frontend
from frontend.publish import restore as restore_frontend
from frontend.verify import verify as verify_frontend
from release import (
    ReleaseStore,
    repository_release_identity,
    retention_deletions,
    stale_candidates,
    utc_text,
)


@dataclass(frozen=True)
class Context:
    repo_root: Path
    serverless_root: Path
    terraform_root: Path
    config: Config
    aws: AWSClient


def make_context(config: Config) -> Context:
    root = Path(__file__).resolve().parents[2]
    serverless = root / "deployment/serverless"
    return Context(root, serverless, serverless / "infrastructure/tf", config, AWSClient(config.aws.region))


def step(name: str, status: str = "start") -> None:
    print(f"[{status}] {name}", flush=True)


def require_node_22() -> None:
    version = run(["node", "--version"]).stdout.strip()
    if version.startswith("v"):
        version = version[1:]
    try:
        major, minor, patch = (int(part) for part in version.split("."))
    except ValueError as exc:
        raise CommandError(f"unable to parse Node version: {version!r}") from exc

    if (major, minor, patch) < (22, 23, 2) or major >= 23:
        raise CommandError(
            f"Node 22.23.2 through 22.x is required; found v{version}"
        )


def require_supported_pnpm() -> None:
    version = run(["pnpm", "--version"]).stdout.strip()
    normalized_version = version.removeprefix("v")
    try:
        major, minor, patch = (int(part) for part in normalized_version.split("."))
    except ValueError as exc:
        raise CommandError(f"unable to parse pnpm version: {version!r}") from exc

    if (major, minor, patch) < (11, 25, 0) or major >= 13:
        raise CommandError(
            f"pnpm 11.25.0 through 12.x is required; found {version!r}"
        )


def preflight(context: Context, *, mutation: bool) -> None:
    require_tools(["aws", "terraform", "go", "node", "pnpm", "ssh", "scp", "ssh-keyscan"])
    require_node_22()
    require_supported_pnpm()
    identity = context.aws.identity()
    account = str(identity.get("Account", ""))
    print(f"AWS account={account} region={context.config.aws.region}")
    if account != context.config.deployment.account_id:
        raise CommandError("AWS account does not match deployment.account_id")
    if mutation:
        context.aws.call("ec2", "describe-vpcs", "--vpc-ids", context.config.aws.vpc_id)
        context.aws.call("ec2", "describe-subnets", "--subnet-ids", context.config.aws.subnet_id)
        context.aws.call("ec2", "describe-key-pairs", "--key-names", context.config.aws.key_pair_name)
        zones = context.aws.json("route53", "list-hosted-zones-by-name", "--dns-name", context.config.aws.hosted_zone_name, "--max-items", "1")
        if not zones.get("HostedZones") or zones["HostedZones"][0]["Name"].rstrip(".") != context.config.aws.hosted_zone_name:
            raise CommandError("public Route 53 hosted zone was not found")
        limit = int(context.aws.json("lambda", "get-account-settings")["AccountLimit"]["ConcurrentExecutions"])
        required_concurrency = 9 if context.config.error_alerting_enabled else 8
        if limit < required_concurrency:
            raise CommandError(
                f"Lambda account concurrency must be at least {required_concurrency}"
            )


def _terraform(
    context: Context,
    temporary: bool,
    *,
    restore_verification: bool = False,
    use_lambda_aliases: bool = True,
):
    return protected_json(
        context.config.terraform_variables(
            temporary_access=temporary,
            restore_verification=restore_verification,
            use_lambda_aliases=use_lambda_aliases,
        ),
        prefix="expense-terraform-vars-",
    )


def _confirm(prompt: str, expected: str) -> None:
    if os.environ.get("SERVERLESS_AUTO_APPROVE") == "true":
        return
    entered = input(f"{prompt}\nType {expected} to continue: ").strip()
    if entered != expected:
        raise CommandError("confirmation did not match")


def _print_plan(actions: dict[str, list[str]]) -> None:
    counts: dict[str, int] = {}
    for action in actions.values():
        label = "/".join(action)
        counts[label] = counts.get(label, 0) + 1
    print("Terraform plan: " + ", ".join(f"{name}={count}" for name, count in sorted(counts.items())))


def _assert_no_conflicts(context: Context) -> None:
    prefix = f"{context.config.deployment.name_prefix}-{context.config.deployment.environment}"
    for function in (f"{prefix}-worker", f"{prefix}-ocr", f"{prefix}-bootstrap", f"{prefix}-sender", f"{prefix}-delivery", f"{prefix}-error-notifier"):
        if context.aws.function_exists(function):
            raise CommandError(f"unexpected existing Lambda conflicts with fresh deployment: {function}")
    for role in (f"{prefix}-worker-role", f"{prefix}-ocr-role", f"{prefix}-bootstrap-role", f"{prefix}-sender-role", f"{prefix}-delivery-role", f"{prefix}-error-notifier-role"):
        if context.aws.resource_exists("iam", "get-role", "--role-name", role, missing=("NoSuchEntity",)):
            raise CommandError(f"unexpected existing IAM role conflicts with fresh deployment: {role}")
    bucket = f"{prefix}-frontend-{context.config.deployment.account_id}"
    if context.aws.resource_exists("s3api", "head-bucket", "--bucket", bucket, missing=("404", "NoSuchBucket", "Not Found")):
        raise CommandError(f"unexpected existing frontend bucket conflicts with fresh deployment: {bucket}")
    apis = context.aws.json("apigatewayv2", "get-apis", "--max-results", "100")
    if any(api.get("Name") == f"{prefix}-http-api" for api in apis.get("Items", [])):
        raise CommandError("unexpected existing HTTP API conflicts with fresh deployment")
    instances = context.aws.json(
        "ec2", "describe-instances",
        "--filters", f"Name=tag:Name,Values={prefix}-postgres",
    )
    live = [
        item["InstanceId"]
        for reservation in instances.get("Reservations", [])
        for item in reservation.get("Instances", [])
        if item.get("State", {}).get("Name") not in {"terminated", "shutting-down"}
    ]
    if live:
        raise CommandError(f"unexpected existing database instance conflicts with fresh deployment: {live}")
    for hostname in (context.config.backend.api_hostname, context.config.frontend.hostname):
        records = context.aws.json(
            "route53", "list-resource-record-sets",
            "--hosted-zone-id", context.aws.json(
                "route53", "list-hosted-zones-by-name",
                "--dns-name", context.config.aws.hosted_zone_name, "--max-items", "1",
            )["HostedZones"][0]["Id"],
            "--start-record-name", hostname, "--max-items", "1",
        )
        if any(record.get("Name") == f"{hostname}." for record in records.get("ResourceRecordSets", [])):
            raise CommandError(f"unexpected existing DNS record conflicts with fresh deployment: {hostname}")


def _state(context: Context, terraform: Terraform) -> tuple[str, dict[str, Any]]:
    if not terraform.has_state():
        return "absent", {}
    try:
        outputs = terraform.output()
    except (CommandError, json.JSONDecodeError):
        return "infra_partial", {}
    required = {"database_host", "worker_function_name", "bootstrap_function_name", "api_id", "raw_api_endpoint", "frontend_bucket_name", "cloudfront_distribution_id"}
    if not required.issubset(outputs):
        return "infra_partial", outputs
    if outputs.get("database_temporary_public_ipv4"):
        return "infra_ready_public", outputs
    try:
        concurrency = context.aws.concurrency(str(outputs["worker_function_name"]))
        api = context.aws.json("apigatewayv2", "get-api", "--api-id", str(outputs["api_id"]))
    except Exception:
        return "infra_ready_private", outputs
    if concurrency in (2, 3, 5) and bool(api.get("DisableExecuteApiEndpoint")):
        return "complete", outputs
    return "infra_ready_private", outputs


def status(context: Context) -> str:
    preflight(context, mutation=False)
    with _terraform(context, False) as variables:
        terraform = Terraform(context.terraform_root, variables)
        state, outputs = _state(context, terraform)
    print(f"deployment_state={state}")
    if outputs:
        worker = str(outputs.get("worker_function_name", ""))
        if worker:
            try:
                print(f"worker_concurrency={context.aws.concurrency(worker)}")
            except Exception:
                print("worker_concurrency=unavailable")
        print(f"api_origin={context.config.api_origin}")
        print(f"frontend_origin={context.config.frontend_origin}")
    pointer = ReleaseStore(context.aws, context.config).get_current()
    if pointer is None:
        print("current_release=unmanaged-legacy")
    else:
        print(f"current_release={pointer['currentReleaseId']}")
        print(f"previous_release={pointer['previousReleaseId'] or 'none'}")
    return state


def history(context: Context) -> None:
    preflight(context, mutation=False)
    store = ReleaseStore(context.aws, context.config)
    pointer = store.get_current()
    current_id = str(pointer["currentReleaseId"]) if pointer else None
    previous_id = str(pointer["previousReleaseId"]) if pointer and pointer["previousReleaseId"] else None
    releases = store.list_releases()
    if not releases:
        print("release_history=empty")
        return
    for manifest in releases:
        release_id = str(manifest["releaseId"])
        labels = []
        if release_id == current_id:
            labels.append("current")
        if release_id == previous_id:
            labels.append("previous")
        print(
            " ".join(
                (
                    f"release={release_id}",
                    f"created_at={manifest['createdAt']}",
                    f"commit={manifest['commitSha']}",
                    f"operation={manifest['operation']}",
                    f"scopes={','.join(manifest['changedScopes'])}",
                    f"schema={manifest['database']['migrationVersion']}",
                    f"labels={','.join(labels) or 'retained'}",
                )
            )
        )


def show_release(context: Context, release_id: str) -> None:
    preflight(context, mutation=False)
    print(json.dumps(ReleaseStore(context.aws, context.config).get_release(release_id), indent=2, sort_keys=True))


def plan(context: Context) -> None:
    preflight(context, mutation=True)
    _migration_preflight(context, "all")
    artifacts.build(context.repo_root, context.serverless_root / "build")
    with tempfile.TemporaryDirectory(prefix="expense-serverless-plan-") as temporary, _terraform(
        context,
        True,
        use_lambda_aliases=False,
    ) as variables:
        terraform = Terraform(context.terraform_root, variables)
        terraform.init()
        terraform.validate()
        if terraform.has_state():
            raise CommandError("ACTION=plan is limited to a fresh deployment; use status/update for an existing state")
        plan_path = Path(temporary) / "serverless.tfplan"
        terraform.plan(plan_path)
        actions = require_create_only(terraform.show_plan(plan_path))
        _print_plan(actions)


def deploy(context: Context) -> None:
    preflight(context, mutation=True)
    with tempfile.TemporaryDirectory(prefix="expense-serverless-deploy-") as temporary:
        temporary_root = Path(temporary)
        with _terraform(context, True, use_lambda_aliases=False) as variables:
            terraform = Terraform(context.terraform_root, variables)
            terraform.init()
            terraform.validate()
            state, outputs = _state(context, terraform)
            print(f"detected_state={state}")
            if state == "complete":
                update(context, "all")
                return
            step("artifacts")
            built = artifacts.build(context.repo_root, context.serverless_root / "build")
            _migration_preflight(context, "all", outputs)
            release_id, commit, created_at = repository_release_identity(context.repo_root)
            store = ReleaseStore(context.aws, context.config)
            store.put_candidate(_candidate(release_id, created_at, "all"))
            try:
                if state in {"absent", "infra_partial"}:
                    if state == "absent":
                        _assert_no_conflicts(context)
                    plan_path = temporary_root / "initial.tfplan"
                    terraform.plan(plan_path)
                    actions = require_create_only(terraform.show_plan(plan_path))
                    _print_plan(actions)
                    _confirm(
                        "Create the unified serverless infrastructure?",
                        context.config.deployment.name_prefix,
                    )
                    terraform.apply(plan_path)
                    outputs = terraform.output()
                    state = "infra_ready_public"
                if state == "infra_ready_public":
                    step("database setup")
                    setup_database(context.config, outputs)
                    with _terraform(
                        context,
                        False,
                        use_lambda_aliases=False,
                    ) as private_variables:
                        private_tf = Terraform(context.terraform_root, private_variables)
                        cutover = temporary_root / "private-cutover.tfplan"
                        private_tf.plan(cutover)
                        actions = require_cutover_only(private_tf.show_plan(cutover))
                        _print_plan(actions)
                        private_tf.apply(cutover)
                        outputs = private_tf.output()

                step("bootstrap runtime and migrations")
                bootstrap, _response = runtime.publish_bootstrap_release(
                    context.aws,
                    built["bootstrap"],
                    context.config,
                    outputs,
                    context.terraform_root,
                    release_id,
                )
                step("error notifier runtime and activation")
                notifier = runtime.publish_error_notifier_release(
                    context.aws,
                    built["notifier"],
                    context.config,
                    outputs,
                    context.terraform_root,
                    release_id,
                )
                step("worker runtime and activation")
                worker = runtime.publish_worker_release(
                    context.aws,
                    built["worker"],
                    context.config,
                    outputs,
                    context.terraform_root,
                    release_id,
                )
                step("OCR runtime and activation")
                ocr = runtime.publish_ocr_release(
                    context.aws,
                    built["ocr"],
                    context.config,
                    outputs,
                    context.terraform_root,
                    release_id,
                )
                step("push sender runtime and activation")
                sender, delivery = runtime.publish_notification_releases(
                    context.aws,
                    built["sender"],
                    built["delivery"],
                    context.config,
                    outputs,
                    context.terraform_root,
                    release_id,
                    activate=False,
                )
                candidate_backend = {
                    "worker": worker,
                    "ocr": ocr,
                    "bootstrap": bootstrap,
                    "sender": sender,
                    "delivery": delivery,
                    "errorNotifier": notifier,
                }
                aliases.ensure_live_aliases(
                    context.aws,
                    outputs,
                    release_id,
                    preferred=candidate_backend,
                )
                aliases.promote_backend(context.aws, candidate_backend)
                backend = candidate_backend
                alias_outputs = _apply_infrastructure_updates(context, "backend")
                if alias_outputs is not None:
                    outputs = alias_outputs
                context.aws.activate_notification_function(
                    str(outputs["delivery_function_name"])
                )
                context.aws.activate_notification_function(
                    str(outputs["sender_function_name"])
                )
                step("pre-cutover API verification")
                live_api = context.aws.json(
                    "apigatewayv2",
                    "get-api",
                    "--api-id",
                    str(outputs["api_id"]),
                )
                raw_already_disabled = bool(live_api.get("DisableExecuteApiEndpoint"))
                verify_api(
                    context.config,
                    str(outputs["raw_api_endpoint"])
                    if not raw_already_disabled
                    else None,
                    raw_disabled=False,
                )
                step("frontend publication")
                frontend = publish_frontend(
                    context.aws,
                    context.repo_root,
                    context.config,
                    outputs,
                    release_id,
                )
                verify_frontend(context.config)
                verify_session(context.config)
                verify_google_exchange(context.config)
                step("raw API cutover")
                context.aws.raw_endpoint(str(outputs["api_id"]), True)
                verify_api(
                    context.config,
                    str(outputs["raw_api_endpoint"]),
                    raw_disabled=True,
                )
                manifest = _manifest(
                    release_id=release_id,
                    commit=commit,
                    created_at=created_at,
                    operation="deploy",
                    changed_scopes=["all"],
                    source_release_id=None,
                    database=_database_record(context, outputs),
                    backend=backend,
                    frontend=frontend,
                )
                digest = store.put_release(manifest)
                store.set_current(manifest, digest)
                _delete_committed_candidate(store, release_id)
                step("deployment", "pass")
            except BaseException as deployment_error:
                try:
                    store.put_candidate(
                        _candidate(
                            release_id,
                            created_at,
                            "all",
                            status="failed",
                            error_type=type(deployment_error).__name__,
                        ),
                        overwrite=True,
                    )
                except Exception as recovery_error:
                    raise CommandError(
                        "fresh deployment failed and candidate status recovery also failed: "
                        f"{recovery_error}"
                    ) from deployment_error
                raise
            _automatic_release_cleanup(context)


def _require_complete(context: Context) -> dict[str, Any]:
    with _terraform(context, False) as variables:
        state, outputs = _state(context, Terraform(context.terraform_root, variables))
    if state != "complete":
        raise CommandError(f"updates require deployment_state=complete; found {state}")
    try:
        # The installed release may predate checks introduced by the update.
        # Enforce the new contract after the backend has been replaced below.
        verify_api(
            context.config,
            require_patch_cors=False,
            require_google_register_authorizer=False,
            require_google_link_authorizer=False,
        )
        verify_frontend(
            context.config,
            require_frontend_version=False,
            require_security_headers=False,
        )
    except CommandError as error:
        raise CommandError(f"updates require a healthy complete deployment: {error}") from error
    return outputs


def _infrastructure_targets(scope: str) -> tuple[str, ...]:
    targets: list[str] = []
    if scope in {"backend", "all"}:
        targets.append("aws_apigatewayv2_integration.worker")
        targets.append("aws_apigatewayv2_integration.ocr")
        targets.append("aws_lambda_permission.api_gateway")
        targets.append("aws_lambda_permission.api_gateway_ocr")
        targets.append("aws_apigatewayv2_route.google_register")
        targets.append("aws_apigatewayv2_route.google_link")
        targets.append("aws_apigatewayv2_route.frontend_render_error")
        targets.append("aws_apigatewayv2_route.ocr_capability")
        targets.append("aws_apigatewayv2_route.ocr_draft")
        targets.append("aws_apigatewayv2_route.invitation_lookup")
        targets.append("aws_apigatewayv2_route.authenticated_mutation")
        targets.append(
            'aws_apigatewayv2_route.authenticated_mutation["expire_invitation"]'
        )
        targets.append("aws_apigatewayv2_stage.default")
        targets.extend((
            "aws_dynamodb_table.ocr_replay",
            "aws_iam_role.ocr",
            "aws_iam_role_policy.ocr",
            "aws_cloudwatch_log_group.ocr",
            "aws_lambda_function.ocr",
            "aws_cloudwatch_metric_alarm.ocr_runtime",
            "aws_lambda_permission.ocr_logs_error_notifier",
            "aws_cloudwatch_log_subscription_filter.ocr_error_notifier",
            "aws_iam_role.error_notifier",
            "aws_iam_role_policy.error_notifier",
            "aws_cloudwatch_log_group.error_notifier",
            "aws_lambda_function.error_notifier",
            "aws_lambda_permission.worker_logs_error_notifier",
            "aws_cloudwatch_log_subscription_filter.worker_error_notifier",
            "aws_security_group.sender",
            "aws_vpc_security_group_ingress_rule.postgres_from_sender",
            "aws_vpc_security_group_egress_rule.sender_to_postgres",
            "aws_vpc_security_group_egress_rule.sender_to_push_providers",
            "aws_vpc_security_group_egress_rule.sender_to_vpc_dns_udp",
            "aws_vpc_security_group_egress_rule.sender_to_vpc_dns_tcp",
            "aws_iam_role.sender",
            "aws_iam_role_policy.sender",
            "aws_cloudwatch_log_group.sender",
            "aws_lambda_function.sender",
            "aws_iam_role.delivery",
            "aws_iam_role_policy.delivery",
            "aws_cloudwatch_log_group.delivery",
            "aws_lambda_function.delivery",
            "aws_cloudwatch_event_rule.sender",
            "aws_cloudwatch_event_target.sender",
            "aws_lambda_permission.sender_eventbridge",
            "aws_cloudwatch_event_rule.monthly_review_publisher",
            "aws_cloudwatch_event_target.monthly_review_publisher",
            "aws_lambda_permission.monthly_review_publisher_eventbridge",
        ))
    if scope in {"frontend", "all"}:
        targets.extend((
            "aws_cloudfront_response_headers_policy.frontend_security",
            "aws_cloudfront_distribution.frontend",
        ))
    if scope == "all":
        targets.extend((
            "aws_security_group.operator_access",
            "aws_vpc_security_group_egress_rule.operator_access_to_postgres",
            "aws_vpc_security_group_ingress_rule.operator_access_to_postgres",
            "aws_ec2_instance_connect_endpoint.operator_access",
            "aws_s3_bucket.postgres_backup",
            "aws_s3_bucket_public_access_block.postgres_backup",
            "aws_s3_bucket_ownership_controls.postgres_backup",
            "aws_s3_bucket_server_side_encryption_configuration.postgres_backup",
            "aws_s3_bucket_lifecycle_configuration.postgres_backup",
            "aws_s3_bucket_policy.postgres_backup",
            "aws_vpc_endpoint.s3",
            "aws_iam_role.postgres_backup_writer",
            "aws_iam_role_policy.postgres_backup_writer",
            "aws_iam_instance_profile.postgres_backup_writer",
            "aws_iam_role.postgres_restore_verifier",
            "aws_iam_role_policy.postgres_restore_verifier",
            "aws_iam_instance_profile.postgres_restore_verifier",
        ))
    return tuple(targets)


RETIRED_INVITATION_ROUTE_ADDRESSES = frozenset(
    {
        "aws_apigatewayv2_route.invitation_lookup",
        'aws_apigatewayv2_route.authenticated_mutation["expire_invitation"]',
    }
)

ERROR_ALERTING_RESOURCE_ADDRESSES = frozenset(
    {
        "aws_iam_role.error_notifier[0]",
        "aws_iam_role_policy.error_notifier[0]",
        "aws_cloudwatch_log_group.error_notifier[0]",
        "aws_lambda_function.error_notifier[0]",
        "aws_lambda_permission.worker_logs_error_notifier[0]",
        "aws_cloudwatch_log_subscription_filter.worker_error_notifier[0]",
    }
)


def _backup_infrastructure_targets() -> tuple[str, ...]:
    return (
        "aws_s3_bucket.postgres_backup",
        "aws_s3_bucket_public_access_block.postgres_backup",
        "aws_s3_bucket_ownership_controls.postgres_backup",
        "aws_s3_bucket_server_side_encryption_configuration.postgres_backup",
        "aws_s3_bucket_lifecycle_configuration.postgres_backup",
        "aws_s3_bucket_policy.postgres_backup",
        "aws_vpc_endpoint.s3",
        "aws_iam_role.postgres_backup_writer",
        "aws_iam_role_policy.postgres_backup_writer",
        "aws_iam_instance_profile.postgres_backup_writer",
        "aws_iam_role.postgres_restore_verifier",
        "aws_iam_role_policy.postgres_restore_verifier",
        "aws_iam_instance_profile.postgres_restore_verifier",
    )


def _ensure_postgres_backup_profile(context: Context, outputs: dict[str, Any]) -> None:
    instance_id = str(outputs.get("database_instance_id", ""))
    profile_name = str(outputs.get("postgres_backup_writer_instance_profile_name", ""))
    if not instance_id or not profile_name:
        raise CommandError("backup infrastructure did not expose the database instance profile association")

    def association() -> dict[str, Any] | None:
        associations = context.aws.json(
            "ec2",
            "describe-iam-instance-profile-associations",
            "--filters",
            f"Name=instance-id,Values={instance_id}",
        ).get("IamInstanceProfileAssociations", [])
        if len(associations) > 1:
            raise CommandError("database instance has multiple IAM instance-profile associations")
        return associations[0] if associations else None

    current = association()
    if current:
        attached_arn = str(current.get("IamInstanceProfile", {}).get("Arn", ""))
        if attached_arn.rsplit("/", 1)[-1] != profile_name:
            raise CommandError("database instance already has an unexpected IAM instance profile")
    else:
        context.aws.call(
            "ec2",
            "associate-iam-instance-profile",
            "--instance-id",
            instance_id,
            "--iam-instance-profile",
            f"Name={profile_name}",
        )

    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        current = association()
        if current and current.get("State") == "associated":
            attached_arn = str(current.get("IamInstanceProfile", {}).get("Arn", ""))
            if attached_arn.rsplit("/", 1)[-1] == profile_name:
                return
        time.sleep(2)
    raise CommandError("database backup IAM instance profile did not become associated")


def _apply_infrastructure_updates(
    context: Context,
    scope: str,
    *,
    use_lambda_aliases: bool = True,
) -> dict[str, Any] | None:
    targets = _infrastructure_targets(scope)
    if not targets:
        return None

    step("infrastructure")
    with tempfile.TemporaryDirectory(prefix="expense-serverless-update-") as temporary, _terraform(
        context,
        False,
        use_lambda_aliases=use_lambda_aliases,
    ) as variables:
        terraform = Terraform(context.terraform_root, variables)
        terraform.init()
        terraform.validate()
        plan_path = Path(temporary) / "update.tfplan"
        terraform.plan(plan_path, targets=targets)
        actions = require_non_destructive_update(
            terraform.show_plan(plan_path),
            allowed_deletes=(
                RETIRED_INVITATION_ROUTE_ADDRESSES
                | frozenset({"aws_vpc_security_group_egress_rule.sender_to_push_providers"})
                | (
                    ERROR_ALERTING_RESOURCE_ADDRESSES
                    if not context.config.error_alerting_enabled
                    else frozenset()
                )
            ),
            allowed_replacements=frozenset(
                {
                    "aws_lambda_permission.api_gateway",
                    "aws_lambda_permission.api_gateway_ocr",
                    "aws_lambda_permission.sender_eventbridge",
                    "aws_lambda_permission.monthly_review_publisher_eventbridge",
                    "aws_lambda_permission.worker_logs_error_notifier[0]",
                    "aws_lambda_permission.ocr_logs_error_notifier[0]",
                    "aws_cloudwatch_log_subscription_filter.worker_error_notifier[0]",
                }
            ),
        )
        _print_plan(actions)
        if actions:
            terraform.apply(plan_path)
        outputs = terraform.output()
    step("infrastructure", "pass")
    return outputs


def _migration_preflight(
    context: Context,
    scope: str,
    outputs: dict[str, Any] | None = None,
) -> None:
    if scope not in {"migrations", "backend", "all"}:
        return
    manifest = migration_policy.validate_repository(context.repo_root)
    if manifest.has_maintenance_required():
        current_version = 0
        dirty = False
        bootstrap_name = str((outputs or {}).get("bootstrap_function_name", ""))
        if bootstrap_name and context.aws.function_exists(bootstrap_name):
            with tempfile.NamedTemporaryFile(
                prefix="expense-migration-state-",
                suffix=".json",
                delete=False,
            ) as stream:
                response_path = Path(stream.name)
            try:
                response = context.aws.invoke_bootstrap(
                    bootstrap_name,
                    response_path,
                    operation="migration-state",
                )
            finally:
                response_path.unlink(missing_ok=True)
            current_version = response.get("migration_version")
            dirty = response.get("migration_dirty")
            if (
                not isinstance(current_version, int)
                or isinstance(current_version, bool)
                or not isinstance(dirty, bool)
            ):
                raise CommandError("Bootstrap migration-state response is invalid")
        manifest.validate_pending(current_version, dirty)
    print(manifest.summary(), flush=True)


def _migration_state(
    context: Context,
    outputs: dict[str, Any],
    function_reference: str | None = None,
) -> tuple[int, bool]:
    function_name = function_reference or f"{outputs['bootstrap_function_name']}:live"
    with tempfile.NamedTemporaryFile(
        prefix="expense-migration-state-",
        suffix=".json",
        delete=False,
    ) as stream:
        response_path = Path(stream.name)
    try:
        response = context.aws.invoke_bootstrap(
            function_name,
            response_path,
            operation="migration-state",
        )
    finally:
        response_path.unlink(missing_ok=True)
    version = response.get("migration_version")
    dirty = response.get("migration_dirty")
    if (
        not isinstance(version, int)
        or isinstance(version, bool)
        or not isinstance(dirty, bool)
    ):
        raise CommandError("Bootstrap migration-state response is invalid")
    return version, dirty


def _database_record(context: Context, outputs: dict[str, Any]) -> dict[str, Any]:
    version, dirty = _migration_state(context, outputs)
    if dirty:
        raise CommandError(
            f"database migration state is dirty: version={version:06d} dirty=true"
        )
    return {
        "migrationVersion": version,
        "dirty": False,
        "migrationManifestDigest": migration_policy.repository_digest(context.repo_root),
    }


def _changed_scopes(scope: str) -> list[str]:
    if scope == "all":
        return ["all"]
    if scope == "backend":
        return ["migrations", "backend"]
    return [scope]


def _manifest(
    *,
    release_id: str,
    commit: str,
    created_at: str,
    operation: str,
    changed_scopes: list[str],
    source_release_id: str | None,
    database: dict[str, Any],
    backend: dict[str, Any],
    frontend: dict[str, Any] | None,
) -> dict[str, Any]:
    return {
        "schemaVersion": 1,
        "releaseId": release_id,
        "commitSha": commit,
        "createdAt": created_at,
        "status": "successful",
        "operation": operation,
        "changedScopes": changed_scopes,
        "sourceReleaseId": source_release_id,
        "database": database,
        "backend": backend,
        "frontend": frontend,
    }


def _adopt_existing_release(
    context: Context,
    outputs: dict[str, Any],
    store: ReleaseStore,
    commit: str,
    created_at: str,
) -> dict[str, Any]:
    current = store.get_current()
    if current is not None:
        return store.get_release(str(current["currentReleaseId"]))
    boundary_time = datetime.fromisoformat(created_at.replace("Z", "+00:00")) - timedelta(seconds=1)
    adopted_id, adopted_commit, adopted_at = repository_release_identity(
        context.repo_root,
        now=boundary_time,
    )
    if adopted_commit != commit:
        raise CommandError("repository changed while establishing the release boundary")
    backend, _created = aliases.ensure_live_aliases(context.aws, outputs, adopted_id)
    adopted = _manifest(
        release_id=adopted_id,
        commit=commit,
        created_at=adopted_at,
        operation="adopt",
        changed_scopes=["all"],
        source_release_id=None,
        database=_database_record(context, outputs),
        backend=backend,
        frontend=None,
    )
    digest = store.put_release(adopted)
    store.set_current(adopted, digest)
    return adopted


def _candidate(
    release_id: str,
    created_at: str,
    scope: str,
    *,
    status: str = "deploying",
    error_type: str | None = None,
) -> dict[str, Any]:
    value: dict[str, Any] = {
        "schemaVersion": 1,
        "releaseId": release_id,
        "createdAt": created_at,
        "updatedAt": utc_text(),
        "status": status,
        "operation": "deploy",
        "changedScopes": _changed_scopes(scope),
    }
    if error_type is not None:
        value["errorType"] = error_type
    return value


def _delete_committed_candidate(store: ReleaseStore, release_id: str) -> None:
    try:
        store.delete_candidate(release_id)
    except Exception as error:
        print(
            "warning=release_committed_but_candidate_cleanup_failed "
            f"release={release_id} error_type={type(error).__name__}",
            flush=True,
        )


def update(context: Context, scope: str) -> None:
    preflight(context, mutation=True)
    outputs = _require_complete(context)
    store = ReleaseStore(context.aws, context.config)
    current_pointer = store.get_current()
    if scope != "all":
        cutover_incomplete = current_pointer is None
        if current_pointer is not None:
            active = store.get_release(str(current_pointer["currentReleaseId"]))
            cutover_incomplete = (
                active["operation"] == "adopt" and active["frontend"] is None
            )
        if cutover_incomplete:
            raise CommandError(
                "the first release-history cutover for an existing deployment requires "
                "SCOPE=all"
            )
    _migration_preflight(context, scope, outputs)
    repaired_state_files = runtime.repair_secret_boundary(context.terraform_root, context.config)
    if repaired_state_files:
        print(f"repaired Terraform secret boundary: files={repaired_state_files}")
    release_id, commit, created_at = repository_release_identity(context.repo_root)
    built = artifacts.build(context.repo_root, context.serverless_root / "build")
    store.put_candidate(_candidate(release_id, created_at, scope))
    current_manifest: dict[str, Any] | None = None
    promoted = False
    frontend_attempted = False
    sender_name = (
        str(outputs["sender_function_name"])
        if scope in {"backend", "all"} and outputs.get("sender_function_name")
        else None
    )
    previous_sender_concurrency: int | None = None
    sender_pause_attempted = False
    try:
        current_manifest = _adopt_existing_release(
            context,
            outputs,
            store,
            commit,
            created_at,
        )
        policy = migration_policy.validate_repository(context.repo_root)
        current_schema = int(current_manifest["database"]["migrationVersion"])
        expected_schema = (
            policy.migrations[-1].version if policy.migrations else policy.baseline_version
        )
        if scope in {"migrations", "backend", "all"}:
            policy.validate_application_rollback(current_schema, expected_schema, False)

        adding_ocr_boundary = scope in {"backend", "all"} and "ocr" not in current_manifest[
            "backend"
        ]
        if adding_ocr_boundary:
            infrastructure_outputs = _apply_infrastructure_updates(
                context,
                scope,
                use_lambda_aliases=False,
            )
        else:
            infrastructure_outputs = _apply_infrastructure_updates(context, scope)
        if infrastructure_outputs is not None:
            outputs = infrastructure_outputs
        if scope == "all" and infrastructure_outputs is not None:
            _ensure_postgres_backup_profile(context, infrastructure_outputs)

        target_backend = dict(current_manifest["backend"])
        target_frontend = current_manifest["frontend"]
        if scope in {"migrations", "backend", "all"}:
            step("migrations")
            bootstrap, _response = runtime.publish_bootstrap_release(
                context.aws,
                built["bootstrap"],
                context.config,
                outputs,
                context.terraform_root,
                release_id,
            )
            target_backend["bootstrap"] = bootstrap
            step("migrations", "pass")

        if scope in {"backend", "all"}:
            previous_sender_concurrency = context.aws.pause_sender(sender_name)
            sender_pause_attempted = True
            step("backend")
            target_backend["errorNotifier"] = runtime.publish_error_notifier_release(
                context.aws,
                built["notifier"],
                context.config,
                outputs,
                context.terraform_root,
                release_id,
            )
            target_backend["worker"] = runtime.publish_worker_release(
                context.aws,
                built["worker"],
                context.config,
                outputs,
                context.terraform_root,
                release_id,
            )
            target_backend["ocr"] = runtime.publish_ocr_release(
                context.aws,
                built["ocr"],
                context.config,
                outputs,
                context.terraform_root,
                release_id,
            )
            sender, delivery = runtime.publish_notification_releases(
                context.aws,
                built["sender"],
                built["delivery"],
                context.config,
                outputs,
                context.terraform_root,
                release_id,
                activate=False,
            )
            target_backend["sender"] = sender
            target_backend["delivery"] = delivery

        if scope in {"migrations", "backend", "all"}:
            if adding_ocr_boundary:
                aliases.ensure_live_aliases(
                    context.aws,
                    outputs,
                    release_id,
                    preferred=target_backend,
                )
            aliases.promote_backend(context.aws, target_backend)
            promoted = True
            if adding_ocr_boundary:
                alias_outputs = _apply_infrastructure_updates(context, "backend")
                if alias_outputs is not None:
                    outputs = alias_outputs
        if scope in {"backend", "all"}:
            context.aws.activate_notification_function(
                str(outputs["delivery_function_name"])
            )
            verify_api(context.config)
            step("backend", "pass")
        if scope in {"frontend", "all"}:
            step("frontend")
            frontend_attempted = True
            target_frontend = publish_frontend(
                context.aws,
                context.repo_root,
                context.config,
                outputs,
                release_id,
            )
            verify_frontend(context.config)
            step("frontend", "pass")
        database = _database_record(context, outputs)
        if sender_pause_attempted:
            context.aws.restore_sender(sender_name, previous_sender_concurrency)
        manifest = _manifest(
            release_id=release_id,
            commit=commit,
            created_at=created_at,
            operation="deploy",
            changed_scopes=_changed_scopes(scope),
            source_release_id=None,
            database=database,
            backend=target_backend,
            frontend=target_frontend,
        )
        digest = store.put_release(manifest)
        store.set_current(manifest, digest)
        _delete_committed_candidate(store, release_id)
    except BaseException as deployment_error:
        recovery_errors: list[str] = []
        if frontend_attempted and current_manifest is not None and current_manifest["frontend"] is not None:
            try:
                restore_frontend(
                    context.aws,
                    outputs,
                    current_manifest["frontend"],
                )
            except Exception as recovery_error:
                recovery_errors.append(f"frontend: {recovery_error}")
        if promoted and current_manifest is not None:
            try:
                recovery_backend = dict(current_manifest["backend"])
                if "ocr" not in recovery_backend and "ocr" in target_backend:
                    recovery_backend["ocr"] = target_backend["ocr"]
                aliases.promote_backend(context.aws, recovery_backend)
            except Exception as recovery_error:
                recovery_errors.append(f"backend: {recovery_error}")
        if sender_pause_attempted:
            try:
                context.aws.restore_sender(sender_name, previous_sender_concurrency)
            except Exception as recovery_error:
                recovery_errors.append(f"notification Sender concurrency: {recovery_error}")
        try:
            store.put_candidate(
                _candidate(
                    release_id,
                    created_at,
                    scope,
                    status="failed",
                    error_type=type(deployment_error).__name__,
                ),
                overwrite=True,
            )
        except Exception as recovery_error:
            recovery_errors.append(f"candidate status: {recovery_error}")
        if recovery_errors:
            raise CommandError(
                "deployment failed and compensation was incomplete: "
                + "; ".join(recovery_errors)
            ) from deployment_error
        raise
    _automatic_release_cleanup(context)
    print("deployment_state=complete")


def activate_release(
    context: Context,
    operation: str,
    scope: str,
    target_release_id: str,
) -> None:
    if operation not in {"rollback", "promote"}:
        raise CommandError("release activation operation is invalid")
    if scope not in {"backend", "frontend", "all"}:
        raise CommandError("release activation scope must be backend, frontend, or all")
    preflight(context, mutation=True)
    outputs = _require_complete(context)
    store = ReleaseStore(context.aws, context.config)
    pointer = store.get_current()
    if pointer is None:
        raise CommandError("release activation requires managed release history")
    current = store.get_release(str(pointer["currentReleaseId"]))
    target = store.get_release(target_release_id)
    if target["releaseId"] == current["releaseId"]:
        raise CommandError("target release is already current")
    if scope in {"backend", "all"} and (
        (current["backend"]["errorNotifier"] is None)
        != (target["backend"]["errorNotifier"] is None)
    ):
        raise CommandError(
            "release activation cannot change the Error Notifier infrastructure "
            "topology; use a reviewed deployment"
        )
    if scope in {"backend", "all"} and (
        ("ocr" in current["backend"]) != ("ocr" in target["backend"])
    ):
        raise CommandError(
            "release activation cannot cross the OCR infrastructure boundary; "
            "use a reviewed deployment"
        )

    policy = migration_policy.validate_repository(context.repo_root)
    repository_migration_digest = migration_policy.repository_digest(context.repo_root)
    if current["database"]["migrationManifestDigest"] != repository_migration_digest:
        raise CommandError(
            "current release migration manifest digest does not match this repository"
        )
    actual_version, dirty = _migration_state(context, outputs)
    policy.validate_application_rollback(
        int(target["database"]["migrationVersion"]),
        actual_version,
        dirty,
    )
    if scope in {"frontend", "all"} and target["frontend"] is None:
        raise CommandError("target release has no restorable frontend snapshot")

    print(
        "release_activation_plan "
        f"operation={operation} scope={scope} "
        f"active={current['releaseId']} target={target_release_id} "
        f"database_current={actual_version} "
        f"target_recorded_schema={target['database']['migrationVersion']} "
        "database_action=none"
    )
    if scope in {"backend", "all"}:
        for key, record in target["backend"].items():
            if record is not None:
                print(
                    f"activate_lambda={key}:{record['functionName']}:{record['version']}"
                )
    if scope in {"frontend", "all"}:
        print(f"activate_frontend_snapshot={target['frontend']['snapshotPrefix']}")
    expected = f"{operation}-{target_release_id}"
    _confirm(
        f"Activate {scope} from exact release {target_release_id}? The database schema will not be downgraded.",
        expected,
    )
    release_id, commit, created_at = repository_release_identity(context.repo_root)
    candidate = _candidate(release_id, created_at, scope)
    candidate["operation"] = operation
    candidate["sourceReleaseId"] = target_release_id
    store.put_candidate(candidate)

    desired_backend = target["backend"] if scope in {"backend", "all"} else current["backend"]
    desired_frontend = target["frontend"] if scope in {"frontend", "all"} else current["frontend"]
    sender_name = str(outputs.get("sender_function_name", "")) or None
    sender_concurrency: int | None = None
    backend_promoted = False
    frontend_attempted = False
    try:
        if scope in {"backend", "all"}:
            sender_concurrency = context.aws.pause_sender(sender_name)
            aliases.promote_backend(context.aws, desired_backend)
            backend_promoted = True
            verify_api(context.config)
        if scope in {"frontend", "all"}:
            frontend_attempted = True
            restore_frontend(
                context.aws,
                outputs,
                desired_frontend,
            )
            verify_frontend(context.config)
        context.aws.restore_sender(sender_name, sender_concurrency)
        manifest = _manifest(
            release_id=release_id,
            commit=commit,
            created_at=created_at,
            operation=operation,
            changed_scopes=[scope],
            source_release_id=target_release_id,
            database={
                "migrationVersion": actual_version,
                "dirty": False,
                "migrationManifestDigest": repository_migration_digest,
            },
            backend=desired_backend,
            frontend=desired_frontend,
        )
        digest = store.put_release(manifest)
        store.set_current(manifest, digest)
        _delete_committed_candidate(store, release_id)
    except BaseException as activation_error:
        recovery_errors: list[str] = []
        if frontend_attempted and current["frontend"] is not None:
            try:
                restore_frontend(
                    context.aws,
                    outputs,
                    current["frontend"],
                )
            except Exception as recovery_error:
                recovery_errors.append(f"frontend: {recovery_error}")
        if backend_promoted:
            try:
                aliases.promote_backend(context.aws, current["backend"])
            except Exception as recovery_error:
                recovery_errors.append(f"backend: {recovery_error}")
        try:
            context.aws.restore_sender(sender_name, sender_concurrency)
        except Exception as recovery_error:
            recovery_errors.append(f"notification Sender concurrency: {recovery_error}")
        try:
            failed = _candidate(
                release_id,
                created_at,
                scope,
                status="failed",
                error_type=type(activation_error).__name__,
            )
            failed["operation"] = operation
            failed["sourceReleaseId"] = target_release_id
            store.put_candidate(failed, overwrite=True)
        except Exception as recovery_error:
            recovery_errors.append(f"candidate status: {recovery_error}")
        if recovery_errors:
            raise CommandError(
                "release activation failed and compensation was incomplete: "
                + "; ".join(recovery_errors)
            ) from activation_error
        raise
    _automatic_release_cleanup(context)
    print(f"deployment_state=complete current_release={release_id}")


def _automatic_release_cleanup(context: Context) -> None:
    try:
        cleanup_releases(
            context,
            apply=True,
            require_confirmation=False,
        )
    except Exception as error:
        raise CommandError(
            "release is active, but automatic retention cleanup failed; "
            "inspect ACTION=history and rerun ACTION=cleanup"
        ) from error


def cleanup_releases(
    context: Context,
    *,
    apply: bool | None = None,
    require_confirmation: bool = True,
) -> None:
    preflight(context, mutation=True)
    outputs = _require_complete(context)
    store = ReleaseStore(context.aws, context.config)
    pointer = store.get_current()
    if pointer is None:
        raise CommandError("release cleanup requires managed release history")
    releases = store.list_releases()
    delete_releases = retention_deletions(releases, pointer)
    delete_ids = {str(item["releaseId"]) for item in delete_releases}
    retained = [item for item in releases if str(item["releaseId"]) not in delete_ids]
    candidates = stale_candidates(store.list_candidates())
    retained_snapshot_prefixes = {
        str(manifest["frontend"]["snapshotPrefix"])
        for manifest in retained
        if manifest["frontend"] is not None
    }
    snapshot_prefix_deletions = sorted(
        {
            str(manifest["frontend"]["snapshotPrefix"])
            for manifest in delete_releases
            if manifest["frontend"] is not None
        }
        - retained_snapshot_prefixes
    )

    retained_versions: dict[str, set[str]] = {}
    for manifest in retained:
        for record in manifest["backend"].values():
            if record is not None:
                retained_versions.setdefault(str(record["functionName"]), set()).add(
                    str(record["version"])
                )
    live_backend = aliases.current_backend(context.aws, outputs)
    for record in live_backend.values():
        if record is not None:
            retained_versions.setdefault(str(record["functionName"]), set()).add(
                str(record["version"])
            )
    lambda_deletions: list[tuple[str, str]] = []
    for function_name in sorted(retained_versions):
        if not context.aws.function_exists(function_name):
            continue
        protected = retained_versions[function_name]
        for version in context.aws.list_versions(function_name):
            number = str(version.get("Version", ""))
            if number.isdigit() and number != "0" and number not in protected:
                lambda_deletions.append((function_name, number))

    bucket = str(outputs["frontend_bucket_name"])
    retained_assets: set[str] = set()
    current_manifest = next(
        item for item in retained if item["releaseId"] == pointer["currentReleaseId"]
    )
    frontend_history_managed = current_manifest["frontend"] is not None
    for manifest in retained:
        frontend = manifest["frontend"]
        if frontend is None:
            continue
        descriptor = load_snapshot_descriptor(
            context.aws,
            bucket,
            frontend,
        )
        retained_assets.update(str(item["path"]) for item in descriptor["assets"])
    objects = (
        context.aws.json(
            "s3api",
            "list-objects-v2",
            "--bucket",
            bucket,
            "--prefix",
            "assets/",
        ).get("Contents", [])
        if frontend_history_managed
        else []
    )
    asset_deletions = sorted(
        str(item["Key"])
        for item in objects
        if isinstance(item, dict) and str(item.get("Key", "")) not in retained_assets
    )

    print(
        "cleanup_plan "
        f"releases={len(delete_releases)} "
        f"lambda_versions={len(lambda_deletions)} "
        f"frontend_snapshots={len(snapshot_prefix_deletions)} "
        f"frontend_assets={len(asset_deletions)} "
        f"candidates={len(candidates)}"
    )
    for manifest in delete_releases:
        print(f"delete_release={manifest['releaseId']}")
    for function_name, version in lambda_deletions:
        print(f"delete_lambda_version={function_name}:{version}")
    for prefix in snapshot_prefix_deletions:
        print(f"delete_frontend_snapshot={prefix}")
    for key in asset_deletions:
        print(f"delete_frontend_asset={key}")
    for candidate in candidates:
        print(f"delete_candidate={candidate['releaseId']}")

    should_apply = (
        os.environ.get("SERVERLESS_CLEANUP_APPLY") == "true"
        if apply is None
        else apply
    )
    if not should_apply:
        print("cleanup_mode=dry-run; set SERVERLESS_CLEANUP_APPLY=true to apply")
        return
    if require_confirmation:
        _confirm(
            "Apply the exact release cleanup plan shown above?",
            f"cleanup-{context.config.deployment.name_prefix}",
        )
    for function_name, version in lambda_deletions:
        context.aws.delete_version(function_name, version)
    for prefix in snapshot_prefix_deletions:
        context.aws.call(
            "s3",
            "rm",
            f"s3://{bucket}/{prefix}",
            "--recursive",
            "--only-show-errors",
        )
    for key in asset_deletions:
        context.aws.call(
            "s3",
            "rm",
            f"s3://{bucket}/{key}",
            "--only-show-errors",
        )
    for manifest in delete_releases:
        context.aws.delete_parameter(store.release_path(str(manifest["releaseId"])))
    for candidate in candidates:
        store.delete_candidate(str(candidate["releaseId"]))
    print("cleanup_status=complete")


def _apply_backup_infrastructure(context: Context) -> dict[str, Any]:
    step("backup infrastructure")
    with tempfile.TemporaryDirectory(prefix="expense-serverless-backup-") as temporary, _terraform(context, False) as variables:
        terraform = Terraform(context.terraform_root, variables)
        terraform.init()
        terraform.validate()
        plan_path = Path(temporary) / "backup.tfplan"
        terraform.plan(plan_path, targets=_backup_infrastructure_targets())
        actions = require_non_destructive_update(terraform.show_plan(plan_path))
        _print_plan(actions)
        if actions:
            terraform.apply(plan_path)
        outputs = terraform.output()
    if not outputs.get("postgres_backup_bucket_name"):
        raise CommandError("backup infrastructure did not expose a PostgreSQL backup bucket")
    step("backup infrastructure", "pass")
    return outputs


def _configure_backup_with_temporary_access(context: Context) -> None:
    with tempfile.TemporaryDirectory(prefix="expense-serverless-backup-access-") as temporary:
        with _terraform(context, True) as variables:
            terraform = Terraform(context.terraform_root, variables)
            terraform.init()
            terraform.validate()
            plan_path = Path(temporary) / "backup-access.tfplan"
            terraform.plan(plan_path)
            actions = require_temporary_access_create(terraform.show_plan(plan_path))
            _print_plan(actions)
            terraform.apply(plan_path)
            public_outputs = terraform.output()
        try:
            database_backup.configure(context.config, public_outputs)
        except CommandError as error:
            try:
                database_backup.diagnose(context.config, public_outputs)
            except CommandError as diagnostic_error:
                raise CommandError(f"backup configuration failed; diagnostics also failed: {diagnostic_error}") from error
            raise
        finally:
            with _terraform(context, False) as variables:
                terraform = Terraform(context.terraform_root, variables)
                cleanup_path = Path(temporary) / "backup-access-cleanup.tfplan"
                terraform.plan(cleanup_path)
                actions = require_cutover_only(terraform.show_plan(cleanup_path))
                _print_plan(actions)
                terraform.apply(cleanup_path)


def configure_backup(context: Context) -> None:
    preflight(context, mutation=True)
    runtime.repair_secret_boundary(context.terraform_root, context.config)
    _require_complete(context)
    # Terraform evaluates Lambda artifact hashes even for targeted plans.
    artifacts.build(context.repo_root, context.serverless_root / "build")
    backup_outputs = _apply_backup_infrastructure(context)
    _ensure_postgres_backup_profile(context, backup_outputs)
    step("database backup configuration")
    _configure_backup_with_temporary_access(context)
    step("database backup configuration", "pass")
    print("deployment_state=complete")


def restore_verify(context: Context) -> None:
    preflight(context, mutation=True)
    runtime.repair_secret_boundary(context.terraform_root, context.config)
    _require_complete(context)
    # Terraform evaluates Lambda artifact hashes even for targeted plans.
    artifacts.build(context.repo_root, context.serverless_root / "build")
    _apply_backup_infrastructure(context)
    expected = f"restore-verify-{context.config.deployment.name_prefix}"
    _confirm("Create an isolated temporary PostgreSQL restore-verification host?", expected)

    with tempfile.TemporaryDirectory(prefix="expense-serverless-restore-") as temporary:
        with _terraform(context, False, restore_verification=True) as variables:
            terraform = Terraform(context.terraform_root, variables)
            terraform.init()
            terraform.validate()
            plan_path = Path(temporary) / "restore-create.tfplan"
            terraform.plan(plan_path)
            actions = require_restore_verification_create(terraform.show_plan(plan_path))
            _print_plan(actions)
            terraform.apply(plan_path)
            restore_outputs = terraform.output()
        try:
            step("restore verification")
            database_backup.restore_verify(context.config, restore_outputs)
            step("restore verification", "pass")
        finally:
            with _terraform(context, False) as variables:
                terraform = Terraform(context.terraform_root, variables)
                cleanup_path = Path(temporary) / "restore-cleanup.tfplan"
                terraform.plan(cleanup_path)
                actions = require_restore_verification_cleanup(terraform.show_plan(cleanup_path))
                _print_plan(actions)
                terraform.apply(cleanup_path)
    print("deployment_state=complete")


def _backup_bucket_has_objects(context: Context, bucket: str) -> bool:
    response = context.aws.json("s3api", "list-objects-v2", "--bucket", bucket, "--max-keys", "1")
    return bool(response.get("Contents"))


def backup_status(context: Context) -> None:
    preflight(context, mutation=False)
    with _terraform(context, False) as variables:
        terraform = Terraform(context.terraform_root, variables)
        if not terraform.has_state():
            raise CommandError("backup status requires an existing Terraform state")
        outputs = terraform.output()
    bucket = str(outputs.get("postgres_backup_bucket_name", ""))
    if not bucket:
        raise CommandError("backup status requires a managed PostgreSQL backup bucket")
    contents = context.aws.json("s3api", "list-objects-v2", "--bucket", bucket, "--prefix", "daily/").get("Contents", [])
    latest = max(contents, key=lambda item: str(item.get("LastModified", "")), default=None)
    print(f"postgres_backup_bucket={bucket}")
    if latest:
        print(f"latest_postgres_backup={latest.get('Key', 'unavailable')}")
        print(f"latest_postgres_backup_time={latest.get('LastModified', 'unavailable')}")
        print(f"latest_postgres_backup_size_bytes={latest.get('Size', 'unavailable')}")
    else:
        print("latest_postgres_backup=none")
    verification_key = "verification/latest.json"
    if context.aws.resource_exists("s3api", "head-object", "--bucket", bucket, "--key", verification_key, missing=("404", "NoSuchKey", "Not Found")):
        verification = context.aws.json("s3api", "head-object", "--bucket", bucket, "--key", verification_key)
        print(f"latest_restore_verification_time={verification.get('LastModified', 'unavailable')}")
    else:
        print("latest_restore_verification_time=none")
    restore_failure_key = "verification/restore-failure.txt"
    if context.aws.resource_exists("s3api", "head-object", "--bucket", bucket, "--key", restore_failure_key, missing=("404", "NoSuchKey", "Not Found")):
        restore_failure = context.aws.call("s3", "cp", f"s3://{bucket}/{restore_failure_key}", "-", "--only-show-errors")
        print("latest_restore_failure_start")
        print(restore_failure.rstrip())
        print("latest_restore_failure_end")
    diagnostic_key = "status/latest.txt"
    if context.aws.resource_exists("s3api", "head-object", "--bucket", bucket, "--key", diagnostic_key, missing=("404", "NoSuchKey", "Not Found")):
        diagnostic = context.aws.call("s3", "cp", f"s3://{bucket}/{diagnostic_key}", "-", "--only-show-errors")
        print("latest_backup_diagnostic_start")
        print(diagnostic.rstrip())
        print("latest_backup_diagnostic_end")


def cleanup_backups(context: Context) -> None:
    preflight(context, mutation=True)
    with _terraform(context, False) as variables:
        terraform = Terraform(context.terraform_root, variables)
        if not terraform.has_state():
            raise CommandError("backup cleanup requires an existing Terraform state")
        outputs = terraform.output()
    bucket = str(outputs.get("postgres_backup_bucket_name", ""))
    if not bucket:
        raise CommandError("backup cleanup requires a managed PostgreSQL backup bucket")
    tags = {
        item.get("Key"): item.get("Value")
        for item in context.aws.json("s3api", "get-bucket-tagging", "--bucket", bucket).get("TagSet", [])
    }
    expected_tags = {
        "Project": context.config.deployment.name_prefix,
        "Environment": context.config.deployment.environment,
        "Component": "database-backup",
        "Recovery": "postgres-logical-dump",
    }
    if any(tags.get(name) != value for name, value in expected_tags.items()):
        raise CommandError("refusing backup cleanup because bucket ownership tags do not match this deployment")
    expected = f"backup-cleanup-{context.config.deployment.name_prefix}"
    _confirm(
        f"Permanently remove all retained PostgreSQL backups from {bucket}? This cannot be undone.",
        expected,
    )
    context.aws.call("s3", "rm", f"s3://{bucket}", "--recursive", "--only-show-errors")
    if _backup_bucket_has_objects(context, bucket):
        raise CommandError("backup cleanup did not remove every object; refusing destroy")
    print("PostgreSQL backup bucket is empty. You may now run ACTION=destroy if intended.")


def _wait_enis(context: Context, security_group_ids: list[str]) -> None:
    normal, extra, poll = 1200, 300, 30
    deadline = normal + extra
    elapsed = 0
    latest: list[dict[str, Any]] = []
    while elapsed <= deadline:
        response = context.aws.json("ec2", "describe-network-interfaces", "--filters", f"Name=group-id,Values={','.join(security_group_ids)}")
        latest = response.get("NetworkInterfaces", [])
        if not latest:
            print(f"Lambda ENIs absent after {elapsed}s")
            return
        remaining_normal = max(0, normal - elapsed)
        remaining_extra = extra if elapsed < normal else max(0, deadline - elapsed)
        status_text = ",".join(f"{eni['NetworkInterfaceId']}:{eni['Status']}" for eni in latest)
        print(f"Waiting for Lambda ENIs: normal={remaining_normal}s additional={remaining_extra}s status={status_text}")
        if elapsed == deadline:
            break
        time.sleep(poll)
        elapsed += poll
    if os.environ.get("FORCE_DETACH_LAMBDA_ENI") != "true":
        raise CommandError("Lambda ENIs remain after 25 minutes; inspect them or rerun with FORCE_DETACH_LAMBDA_ENI=true")
    expected_groups = set(security_group_ids)
    for eni in latest:
        groups = {group["GroupId"] for group in eni.get("Groups", [])}
        description = eni.get("Description", "")
        if groups - expected_groups or len(groups) != 1 or not description.startswith("AWS Lambda VPC ENI"):
            raise CommandError(f"refusing force cleanup of unowned ENI {eni['NetworkInterfaceId']}")
        eni_id = eni["NetworkInterfaceId"]
        attachment = eni.get("Attachment", {}).get("AttachmentId")
        if attachment:
            context.aws.call("ec2", "detach-network-interface", "--attachment-id", attachment, "--force")
            context.aws.call("ec2", "wait", "network-interface-available", "--network-interface-ids", eni_id)
        context.aws.call("ec2", "delete-network-interface", "--network-interface-id", eni_id)


def _audit_absent(context: Context, outputs: dict[str, Any]) -> None:
    checks = [
        ("apigatewayv2", ("get-api", "--api-id", str(outputs["api_id"])), ("NotFoundException",), "API"),
        ("apigatewayv2", ("get-domain-name", "--domain-name", str(outputs["api_hostname"])), ("NotFoundException",), "API domain"),
        ("cloudfront", ("get-distribution", "--id", str(outputs["cloudfront_distribution_id"])), ("NoSuchDistribution",), "CloudFront distribution"),
        ("s3api", ("head-bucket", "--bucket", str(outputs["frontend_bucket_name"])), ("404", "NoSuchBucket", "Not Found"), "frontend bucket"),
        ("iam", ("get-role", "--role-name", str(outputs["worker_role_name"])), ("NoSuchEntity",), "Worker role"),
        ("iam", ("get-role", "--role-name", str(outputs["bootstrap_role_name"])), ("NoSuchEntity",), "Bootstrap role"),
        ("ec2", ("describe-launch-templates", "--launch-template-ids", str(outputs["database_launch_template_id"])), ("InvalidLaunchTemplateId.NotFound",), "database launch template"),
        ("ec2", ("describe-volumes", "--volume-ids", str(outputs["database_volume_id"])), ("InvalidVolume.NotFound",), "database volume"),
    ]
    for service, arguments, missing, label in checks:
        if context.aws.resource_exists(service, *arguments, missing=missing):
            raise CommandError(f"owned {label} still exists after destroy")
    east = AWSClient("us-east-1")
    if context.aws.resource_exists("acm", "describe-certificate", "--certificate-arn", str(outputs["api_certificate_arn"]), missing=("ResourceNotFoundException",)):
        raise CommandError("owned API certificate still exists after destroy")
    if east.resource_exists("acm", "describe-certificate", "--certificate-arn", str(outputs["frontend_certificate_arn"]), missing=("ResourceNotFoundException",)):
        raise CommandError("owned frontend certificate still exists after destroy")
    instance_id = str(outputs["database_instance_id"])
    if context.aws.resource_exists("ec2", "describe-instances", "--instance-ids", instance_id, missing=("InvalidInstanceID.NotFound",)):
        instance = context.aws.json("ec2", "describe-instances", "--instance-ids", instance_id)
        reservations = instance.get("Reservations", [])
        states = [item["State"]["Name"] for reservation in reservations for item in reservation.get("Instances", [])]
        if any(state != "terminated" for state in states):
            raise CommandError(f"owned database instance still exists after destroy: {states}")
    for security_group in (
        outputs["database_security_group_id"],
        outputs["worker_security_group_id"],
        outputs["bootstrap_security_group_id"],
    ):
        if context.aws.resource_exists("ec2", "describe-security-groups", "--group-ids", str(security_group), missing=("InvalidGroup.NotFound",)):
            raise CommandError(f"owned security group still exists after destroy: {security_group}")
    exact_logs = {str(outputs["worker_log_group_name"]), str(outputs["bootstrap_log_group_name"])}
    for name in exact_logs:
        response = context.aws.json("logs", "describe-log-groups", "--log-group-name-prefix", name)
        if any(group.get("logGroupName") == name for group in response.get("logGroups", [])):
            raise CommandError(f"owned log group still exists after destroy: {name}")
    for hostname in (str(outputs["api_hostname"]), str(outputs["frontend_hostname"])):
        records = context.aws.json(
            "route53", "list-resource-record-sets",
            "--hosted-zone-id", str(outputs["hosted_zone_id"]),
            "--start-record-name", hostname, "--max-items", "1",
        )
        if any(record.get("Name") == f"{hostname}." and record.get("Type") in {"A", "AAAA"} for record in records.get("ResourceRecordSets", [])):
            raise CommandError(f"owned Route 53 alias still exists after destroy: {hostname}")


def destroy(context: Context) -> None:
    preflight(context, mutation=True)
    artifacts.build(context.repo_root, context.serverless_root / "build")
    with _terraform(context, False) as variables:
        terraform = Terraform(context.terraform_root, variables)
        state, outputs = _state(context, terraform)
        if state == "absent":
            store = ReleaseStore(context.aws, context.config)
            if store.metadata_paths():
                expected = f"destroy-{context.config.deployment.name_prefix}"
                _confirm(
                    "Delete stale release-history metadata for the absent deployment?",
                    expected,
                )
                store.delete_all_metadata()
            print("deployment_state=absent")
            return
        if not outputs:
            raise CommandError("cannot safely destroy state without required outputs")
        backup_bucket = str(outputs.get("postgres_backup_bucket_name", ""))
        if backup_bucket and _backup_bucket_has_objects(context, backup_bucket):
            raise CommandError(
                "refusing destroy while PostgreSQL backups are retained; use ACTION=backup-cleanup for an explicit guarded deletion"
            )
        expected = f"destroy-{context.config.deployment.name_prefix}"
        _confirm("Delete all resources owned by the unified serverless deployment?", expected)
        context.aws.delete_function(str(outputs["worker_function_name"]))
        context.aws.delete_function(str(outputs["bootstrap_function_name"]))
        for key in ("sender_function_name", "delivery_function_name"):
            if outputs.get(key):
                context.aws.delete_function(str(outputs[key]))
        _wait_enis(context, [str(outputs[key]) for key in ("worker_security_group_id", "bootstrap_security_group_id", "delivery_security_group_id") if outputs.get(key)])
        terraform.destroy()
    for function in (str(outputs[key]) for key in ("worker_function_name", "bootstrap_function_name", "sender_function_name", "delivery_function_name") if outputs.get(key)):
        if context.aws.function_exists(function):
            raise CommandError(f"owned Lambda still exists after destroy: {function}")
    _audit_absent(context, outputs)
    ReleaseStore(context.aws, context.config).delete_all_metadata()
    print("deployment_state=absent")


def execute(
    context: Context,
    action: str,
    scope: str,
    release_id: str | None = None,
) -> None:
    with deployment_lock(context.repo_root):
        if action == "plan":
            plan(context)
        elif action in {"auto", "deploy"}:
            deploy(context)
        elif action == "update":
            update(context, scope)
        elif action == "status":
            status(context)
        elif action == "history":
            history(context)
        elif action == "show":
            if release_id is None:
                raise CommandError("ACTION=show requires RELEASE=<exact-release-id>")
            show_release(context, release_id)
        elif action in {"rollback", "promote"}:
            if release_id is None:
                raise CommandError(
                    f"ACTION={action} requires RELEASE=<exact-release-id>"
                )
            activate_release(context, action, scope, release_id)
        elif action == "cleanup":
            cleanup_releases(context)
        elif action == "backup-configure":
            configure_backup(context)
        elif action == "restore-verify":
            restore_verify(context)
        elif action == "backup-status":
            backup_status(context)
        elif action == "backup-cleanup":
            cleanup_backups(context)
        elif action == "destroy":
            destroy(context)
        else:
            raise CommandError(f"unsupported action: {action}")
