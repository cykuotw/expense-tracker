"""Configuration-only application of Terraform-owned operating schedules."""
from __future__ import annotations

import json
import re
import tempfile
import time
from datetime import UTC, datetime
from pathlib import Path
from zoneinfo import ZoneInfo

from common.command import CommandError, protected_json
from common.terraform import Terraform, plan_actions
from database.operations import require_running
from operating_hours import OperatingHours


TARGETS = (
    "aws_scheduler_schedule_group.database_hours", "aws_sqs_queue.database_hours_failures",
    "aws_iam_role.database_hours", "aws_iam_role_policy.database_hours",
    "aws_scheduler_schedule.database_hours", "aws_scheduler_schedule.monthly_review",
    "aws_iam_role_policy.sender_operating_hours", "aws_cloudwatch_event_rule.monthly_review_publisher",
)
ALLOWED = frozenset({
    "aws_scheduler_schedule_group.database_hours[0]", "aws_sqs_queue.database_hours_failures[0]",
    "aws_iam_role.database_hours[0]", "aws_iam_role_policy.database_hours[0]",
    'aws_scheduler_schedule.database_hours["start"]', 'aws_scheduler_schedule.database_hours["stop"]',
    'aws_scheduler_schedule.database_hours["safety-stop"]', "aws_scheduler_schedule.monthly_review[0]",
    "aws_iam_role_policy.sender_operating_hours", "aws_cloudwatch_event_rule.monthly_review_publisher",
})
STOP_ADDRESSES = frozenset(f'aws_scheduler_schedule.database_hours["{key}"]' for key in ("start", "stop", "safety-stop"))


def require_support(context, outputs: dict) -> None:
    message = "operating hours requires deployed frontend and Sender support; run ACTION=update SCOPE=all first"
    bucket = str(outputs["frontend_bucket_name"])
    # Project only public policy fields; never retrieve or print credentials.
    sender = context.aws.json(
        "lambda", "get-function-configuration", "--function-name", f"{outputs['sender_function_name']}:live",
        "--query", "Environment.Variables.{bucket:OPERATING_HOURS_BUCKET,version:OPERATING_HOURS_POLICY_VERSION}",
    )
    if not sender or sender.get("bucket") != bucket or sender.get("version") != "1":
        raise CommandError(message)
    public = context.aws.call("s3", "cp", f"s3://{bucket}/runtime-config.js", "-", "--only-show-errors")
    match = re.fullmatch(r"\s*window\.__APP_CONFIG__ = Object\.freeze\((.*)\);\s*", public, re.DOTALL)
    try:
        valid = match and json.loads(match[1]).get("operatingHoursPolicyVersion") == 1
    except (json.JSONDecodeError, AttributeError):
        valid = False
    if not valid:
        raise CommandError(message)


def validate_plan(plan: dict, *, enabled: bool) -> dict:
    actions = plan_actions(plan)
    unsafe = {address: action for address, action in actions.items()
              if address not in ALLOWED or (action not in (["create"], ["update"])
                                             and not (not enabled and address in STOP_ADDRESSES and action == ["delete"]))}
    if unsafe:
        raise CommandError(f"hours plan contains unrelated, destructive or replacement actions: {unsafe}")
    return actions


def publish(client, outputs: dict, policy: dict) -> None:
    with tempfile.TemporaryDirectory(prefix="expense-public-hours-") as temporary:
        path = Path(temporary) / "operating-hours.json"
        path.write_text(json.dumps(policy, sort_keys=True))
        client.call("s3", "cp", str(path), f"s3://{outputs['frontend_bucket_name']}/operating-hours.json",
                    "--content-type", "application/json", "--cache-control", "no-store", "--only-show-errors")
    result = client.json("cloudfront", "create-invalidation", "--distribution-id", str(outputs["cloudfront_distribution_id"]),
                        "--paths", "/operating-hours.json")
    client.call("cloudfront", "wait", "invalidation-completed", "--distribution-id", str(outputs["cloudfront_distribution_id"]),
                "--id", result["Invalidation"]["Id"])


def initialize_policy(client, outputs: dict) -> None:
    """Install a disabled default before any Sender runtime starts reading it.

    This also covers backend-only upgrades with no operating_hours configured.
    Later enablement remains configuration-only after capability installation.
    Never overwrite an existing applied policy during application deployment.
    """
    bucket = str(outputs["frontend_bucket_name"])
    if not client.resource_exists("s3api", "head-object", "--bucket", bucket, "--key", "operating-hours.json", missing=("404", "NoSuchKey", "Not Found")):
        publish(client, {**outputs, "frontend_bucket_name": bucket}, OperatingHours().public())


def apply(context, outputs: dict, configure_backup) -> None:
    hours = context.config.operating_hours
    if not hours.configured:
        raise CommandError("hours-apply requires operating_hours; use enabled=false to disable and retain aligned daily jobs")
    require_support(context, outputs)
    require_running(context, outputs)
    # Validate before mutating any resource. All inputs were validated at load.
    policy = hours.public()
    key = "configuration/database-hours.json"
    bucket = str(outputs["postgres_backup_bucket_name"])
    previous = None
    if context.aws.resource_exists("s3api", "head-object", "--bucket", bucket, "--key", key, missing=("404", "NoSuchKey", "Not Found")):
        previous = json.loads(context.aws.call("s3", "cp", f"s3://{bucket}/{key}", "-", "--only-show-errors"))
    desired = {"policy": policy, "backup": {"time": context.config.backup.time, "timezone": context.config.backup.timezone}}
    if hours.enabled and not hours.contains(datetime.now(UTC)):
        raise CommandError("apply enabled schedules during configured operating hours; manual start remains available outside hours")
    local = datetime.now(UTC).astimezone(ZoneInfo(hours.timezone))
    stop_hour, stop_minute = map(int, hours.stop.split(":"))
    remaining = (stop_hour * 60 + stop_minute - local.hour * 60 - local.minute) % 1440
    if hours.enabled and remaining < (60 if previous != desired else 5):
        raise CommandError("insufficient time before shutdown to safely apply hours; retry earlier in the next operating window")

    def infrastructure(active: bool) -> None:
        variables = context.config.terraform_variables(temporary_access=False)
        variables["activate_operating_hours"] = active
        with protected_json(variables, prefix="expense-hours-vars-") as variable_file, tempfile.TemporaryDirectory(prefix="expense-hours-plan-") as temporary:
            terraform = Terraform(context.terraform_root, variable_file)
            terraform.init()
            terraform.validate()
            plan = Path(temporary) / "hours.tfplan"
            terraform.plan(plan, targets=TARGETS)
            actions = validate_plan(terraform.show_plan(plan), enabled=hours.enabled)
            if actions:
                terraform.apply(plan)

    try:
        infrastructure(False)
        # Retry age is 60s, no retries. Also allow one minute for Scheduler delivery.
        for _ in range(4):
            print("hours_apply=draining_old_schedule_events wait_seconds=30", flush=True)
            time.sleep(30)
        if previous != desired:
            configure_backup()
            status = context.aws.call("s3", "cp", f"s3://{bucket}/status/latest.txt", "-", "--only-show-errors")
            values = dict(line.split("=", 1) for line in status.splitlines() if "=" in line)
            if values.get("status") != "success" or not values.get("duration_seconds", "").isdigit() or int(values["duration_seconds"]) > 2700:
                raise CommandError("backup duration is unavailable or exceeds 45 minutes; schedules remain disabled; inspect backup-status and increase the shutdown margin")
            print(f"backup_duration_seconds={values['duration_seconds']}", flush=True)
        publish(context.aws, outputs, policy)
        if hours.enabled and not hours.contains(datetime.now(UTC)):
            raise CommandError("operating window ended during apply; schedules remain disabled; rerun during operating hours")
        infrastructure(True)
        with protected_json(desired, prefix="expense-hours-applied-") as path:
            context.aws.call("s3", "cp", str(path), f"s3://{bucket}/{key}", "--sse", "AES256", "--only-show-errors")
    except BaseException as error:
        raise CommandError(f"hours-apply incomplete; inspect schedule states and rerun the same action: {error}") from error
    print("hours_apply=complete", flush=True)
