from __future__ import annotations

import json
import os
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from backend import artifacts, runtime
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
from frontend.publish import publish as publish_frontend
from frontend.verify import verify as verify_frontend


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

    if (major, minor, patch) < (22, 13, 1) or major >= 23:
        raise CommandError(
            f"Node 22.13.1 through 22.x is required; found v{version}"
        )


def require_pnpm_10() -> None:
    version = run(["pnpm", "--version"]).stdout.strip()
    if version != "10.23.0":
        raise CommandError(f"pnpm 10.23.0 is required; found {version!r}")


def preflight(context: Context, *, mutation: bool) -> None:
    require_tools(["aws", "terraform", "go", "node", "pnpm", "ssh", "scp", "ssh-keyscan"])
    require_node_22()
    require_pnpm_10()
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
        if limit < 4:
            raise CommandError("Lambda account concurrency must be at least 4")


def _terraform(context: Context, temporary: bool, *, restore_verification: bool = False):
    return protected_json(
        context.config.terraform_variables(
            temporary_access=temporary,
            restore_verification=restore_verification,
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
    for function in (f"{prefix}-worker", f"{prefix}-bootstrap"):
        if context.aws.function_exists(function):
            raise CommandError(f"unexpected existing Lambda conflicts with fresh deployment: {function}")
    for role in (f"{prefix}-worker-role", f"{prefix}-bootstrap-role"):
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
    if concurrency == 3 and bool(api.get("DisableExecuteApiEndpoint")):
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
    return state


def plan(context: Context) -> None:
    preflight(context, mutation=True)
    artifacts.build(context.repo_root, context.serverless_root / "build")
    with tempfile.TemporaryDirectory(prefix="expense-serverless-plan-") as temporary, _terraform(context, True) as variables:
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
    step("artifacts")
    artifacts.build(context.repo_root, context.serverless_root / "build")
    with tempfile.TemporaryDirectory(prefix="expense-serverless-deploy-") as temporary:
        temporary_root = Path(temporary)
        with _terraform(context, True) as variables:
            terraform = Terraform(context.terraform_root, variables)
            terraform.init()
            terraform.validate()
            state, outputs = _state(context, terraform)
            print(f"detected_state={state}")
            if state == "complete":
                update(context, "all")
                return
            if state in {"absent", "infra_partial"}:
                if state == "absent":
                    _assert_no_conflicts(context)
                plan_path = temporary_root / "initial.tfplan"
                terraform.plan(plan_path)
                actions = require_create_only(terraform.show_plan(plan_path))
                _print_plan(actions)
                _confirm("Create the unified serverless infrastructure?", context.config.deployment.name_prefix)
                terraform.apply(plan_path)
                outputs = terraform.output()
                state = "infra_ready_public"
            if state == "infra_ready_public":
                step("database setup")
                setup_database(context.config, outputs)
                with _terraform(context, False) as private_variables:
                    private_tf = Terraform(context.terraform_root, private_variables)
                    cutover = temporary_root / "private-cutover.tfplan"
                    private_tf.plan(cutover)
                    actions = require_cutover_only(private_tf.show_plan(cutover))
                    _print_plan(actions)
                    private_tf.apply(cutover)
                    outputs = private_tf.output()

        step("bootstrap runtime and migrations")
        runtime.configure_bootstrap(context.aws, context.config, outputs, context.terraform_root)
        step("worker runtime and activation")
        runtime.configure_worker(context.aws, context.config, outputs, context.terraform_root)
        step("pre-cutover API verification")
        live_api = context.aws.json("apigatewayv2", "get-api", "--api-id", str(outputs["api_id"]))
        raw_already_disabled = bool(live_api.get("DisableExecuteApiEndpoint"))
        verify_api(
            context.config,
            str(outputs["raw_api_endpoint"]) if not raw_already_disabled else None,
            raw_disabled=False,
        )
        step("frontend publication")
        publish_frontend(context.aws, context.repo_root, context.config, outputs)
        verify_frontend(context.config)
        verify_session(context.config)
        verify_google_exchange(context.config)
        step("raw API cutover")
        context.aws.raw_endpoint(str(outputs["api_id"]), True)
        verify_api(context.config, str(outputs["raw_api_endpoint"]), raw_disabled=True)
        step("deployment", "pass")


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
        verify_frontend(context.config)
    except CommandError as error:
        raise CommandError(f"updates require a healthy complete deployment: {error}") from error
    return outputs


def _infrastructure_targets(scope: str) -> tuple[str, ...]:
    targets: list[str] = []
    if scope in {"backend", "all"}:
        targets.append("aws_apigatewayv2_route.google_register")
        targets.append("aws_apigatewayv2_route.google_link")
        targets.append("aws_apigatewayv2_route.authenticated_mutation")
        targets.append("aws_apigatewayv2_stage.default")
    if scope in {"frontend", "all"}:
        targets.extend((
            "aws_cloudfront_response_headers_policy.frontend_security",
            "aws_cloudfront_distribution.frontend",
        ))
    if scope == "all":
        targets.extend((
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
            "aws_instance.postgres",
        ))
    return tuple(targets)


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
        "aws_instance.postgres",
    )


def _apply_infrastructure_updates(context: Context, scope: str) -> None:
    targets = _infrastructure_targets(scope)
    if not targets:
        return

    step("infrastructure")
    with tempfile.TemporaryDirectory(prefix="expense-serverless-update-") as temporary, _terraform(context, False) as variables:
        terraform = Terraform(context.terraform_root, variables)
        terraform.init()
        terraform.validate()
        plan_path = Path(temporary) / "update.tfplan"
        terraform.plan(plan_path, targets=targets)
        actions = require_non_destructive_update(terraform.show_plan(plan_path))
        _print_plan(actions)
        if actions:
            terraform.apply(plan_path)
    step("infrastructure", "pass")


def update(context: Context, scope: str) -> None:
    preflight(context, mutation=True)
    repaired_state_files = runtime.repair_secret_boundary(context.terraform_root, context.config)
    if repaired_state_files:
        print(f"repaired Terraform secret boundary: files={repaired_state_files}")
    outputs = _require_complete(context)
    # Terraform evaluates Lambda artifact hashes even for targeted infrastructure plans.
    built = artifacts.build(context.repo_root, context.serverless_root / "build")
    if scope in {"migrations", "backend", "all"}:
        step("migrations")
        runtime.update_bootstrap(context.aws, built["bootstrap"], context.config, outputs, context.terraform_root)
        step("migrations", "pass")
    _apply_infrastructure_updates(context, scope)
    if scope in {"backend", "all"}:
        step("backend")
        runtime.update_worker(context.aws, built["worker"], context.config, outputs, context.terraform_root)
        verify_api(context.config)
        step("backend", "pass")
    if scope in {"frontend", "all"}:
        step("frontend")
        publish_frontend(context.aws, context.repo_root, context.config, outputs)
        verify_frontend(context.config)
        step("frontend", "pass")
    print("deployment_state=complete")


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
    _apply_backup_infrastructure(context)
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
        _wait_enis(context, [str(outputs["worker_security_group_id"]), str(outputs["bootstrap_security_group_id"])])
        terraform.destroy()
    for function in (str(outputs["worker_function_name"]), str(outputs["bootstrap_function_name"])):
        if context.aws.function_exists(function):
            raise CommandError(f"owned Lambda still exists after destroy: {function}")
    _audit_absent(context, outputs)
    print("deployment_state=absent")


def execute(context: Context, action: str, scope: str) -> None:
    with deployment_lock(context.repo_root):
        if action == "plan":
            plan(context)
        elif action in {"auto", "deploy"}:
            deploy(context)
        elif action == "update":
            update(context, scope)
        elif action == "status":
            status(context)
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
