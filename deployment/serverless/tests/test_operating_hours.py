from __future__ import annotations

import sys
import unittest
import contextlib
from datetime import UTC, datetime
from pathlib import Path
from unittest import mock
from types import SimpleNamespace

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from common.command import CommandError
from operating_hours import parse, validate_backup, backup_freshness
from database import hours, operations


class OperatingHoursTest(unittest.TestCase):
    def test_absent_feature_backend_upgrade_initializes_disabled_policy_only_once(self):
        client = mock.MagicMock()
        with mock.patch.object(hours, "publish") as publish:
            client.resource_exists.return_value = False
            hours.initialize_policy(client, {"frontend_bucket_name": "fixture-public", "cloudfront_distribution_id": "fixture"})
            self.assertFalse(publish.call_args.args[2]["enabled"])
            client.resource_exists.return_value = True
            hours.initialize_policy(client, {"frontend_bucket_name": "fixture-public", "cloudfront_distribution_id": "fixture"})
            publish.assert_called_once()

    def test_defaults_disabled_and_explicit_window(self):
        self.assertFalse(parse(None).enabled)
        policy = parse({"enabled": True})
        self.assertEqual(policy.review_time, "09:30")
        self.assertFalse(policy.contains(datetime(2026, 1, 1, 13, 59, tzinfo=UTC)))
        self.assertTrue(policy.contains(datetime(2026, 1, 1, 14, tzinfo=UTC)))
        self.assertTrue(policy.contains(datetime(2026, 7, 1, 13, tzinfo=UTC)))
        self.assertFalse(policy.contains(datetime(2026, 7, 2, 2, tzinfo=UTC)))

    def test_dst_gaps_repeated_time_and_overnight_duration(self):
        with self.assertRaisesRegex(ValueError, "daylight-saving gap"):
            parse({"safety_stop": "02:30"})
        # Repeated 01:00 is supported; AWS invokes it once before the fold.
        policy = parse({"enabled": True, "safety_stop": "01:00"})
        for instant in ("2026-11-01T05:00:00+00:00", "2026-11-01T06:00:00+00:00"):
            self.assertFalse(policy.contains(datetime.fromisoformat(instant)))
        for day, expected in (("2026-03-07", 10), ("2026-10-31", 12)):
            from datetime import timedelta
            from zoneinfo import ZoneInfo
            start = datetime.fromisoformat(day).replace(hour=22, tzinfo=ZoneInfo("America/Toronto"))
            end = (start + timedelta(days=1)).replace(hour=9)
            self.assertEqual((end.astimezone(UTC) - start.astimezone(UTC)).total_seconds() / 3600, expected)
        with self.assertRaisesRegex(ValueError, "15 elapsed minutes"):
            parse({"enabled": True, "start": "03:00", "stop": "22:00", "safety_stop": "01:59"})
        compressed = parse({"enabled": True, "start": "20:00", "stop": "03:15", "safety_stop": "06:00"})
        with self.assertRaisesRegex(ValueError, "elapsed minute"):
            validate_backup(compressed, "01:30:00", "America/Toronto")

    def test_support_gate_requires_both_deployed_components_before_mutation(self):
        import json
        context = mock.MagicMock()
        outputs = {"frontend_bucket_name": "public-fixture", "sender_function_name": "sender"}
        context.aws.json.return_value = {"bucket": "public-fixture", "version": "1"}
        for version in (None, 1):
            context.aws.call.return_value = "window.__APP_CONFIG__ = Object.freeze(" + json.dumps({"operatingHoursPolicyVersion": version}) + ");\n"
            if version is None:
                with self.assertRaisesRegex(CommandError, "SCOPE=all"):
                    hours.require_support(context, outputs)
            else:
                hours.require_support(context, outputs)
        context.aws.json.return_value = None
        with self.assertRaisesRegex(CommandError, "SCOPE=all"):
            hours.require_support(context, outputs)

    def test_reject_invalid_policy_and_backup_margins(self):
        for value in ({"timezone": "EST"}, {"timezone": "America/Unknown"}, {"enabled": 1}, {"start": "9:00"}, {"stop": "09:00"}, {"safety_stop": "12:00"}, {"unknown": True}):
            with self.subTest(value=value), self.assertRaises(ValueError): parse(value)
        policy = parse({"enabled": True})
        validate_backup(policy, "21:00:00", "America/Toronto")
        for time, zone in (("03:17:00", "UTC"), ("22:00:00", "America/Toronto"), ("21:00:01", "America/Toronto")):
            with self.assertRaises(ValueError): validate_backup(policy, time, zone)
        # Also support a window across midnight; safety is in the off period.
        night = parse({"start": "20:00", "stop": "05:00", "safety_stop": "08:00", "timezone": "UTC"})
        self.assertTrue(night.contains(datetime(2026, 1, 1, 1, tzinfo=UTC)))

    def test_freshness_uses_local_due_time_across_dst(self):
        latest = datetime.fromisoformat("2026-10-31T21:01:00-04:00")
        now = datetime.fromisoformat("2026-11-01T21:30:00-05:00")
        self.assertEqual(backup_freshness("21:00:00", "America/Toronto", latest, now), "fresh")
        self.assertEqual(backup_freshness("21:00:00", "America/Toronto", latest, now.replace(hour=22)), "stale")

    def test_plan_boundaries_and_disable(self):
        plan = {"resource_changes": [{"address": address, "change": {"actions": ["update"]}} for address in hours.ALLOWED]}
        self.assertEqual(len(hours.validate_plan(plan, enabled=True)), len(hours.ALLOWED))
        for address in hours.STOP_ADDRESSES:
            deletion = {"resource_changes": [{"address": address, "change": {"actions": ["delete"]}}]}
            hours.validate_plan(deletion, enabled=False)
            with self.assertRaises(CommandError): hours.validate_plan(deletion, enabled=True)
        for action in (["delete"], ["delete", "create"]):
            with self.assertRaises(CommandError):
                hours.validate_plan({"resource_changes": [{"address": "aws_instance.postgres", "change": {"actions": action}}]}, enabled=False)

    def test_apply_stages_and_safe_rerun_without_build(self):
        import json
        policy = parse({"enabled": True})
        for aliases in (False, True):
            for already_applied in (False, True):
                with self.subTest(aliases=aliases, already_applied=already_applied):
                    context = mock.MagicMock()
                    context.config.operating_hours = policy
                    context.config.backup = SimpleNamespace(time="21:00:00", timezone=policy.timezone)
                    context.config.terraform_variables.return_value = {"use_lambda_aliases": aliases}
                    context.aws.resource_exists.return_value = already_applied
                    previous = {"policy": policy.public(), "backup": {"time": "21:00:00", "timezone": policy.timezone}}
                    context.aws.call.side_effect = lambda *args: json.dumps(previous) if args[1:2] == ("cp",) and "configuration/database-hours.json" in str(args[2:]) and args[3] == "-" else "status=success\nduration_seconds=3\n"
                    configure = mock.Mock()
                    terraform = mock.MagicMock()
                    terraform.show_plan.return_value = {"resource_changes": []}
                    stages = []
                    original = hours.protected_json
                    def capture(value, **kwargs):
                        stages.append(value.copy())
                        return original(value, **kwargs)
                    with mock.patch.object(hours, "require_support"), mock.patch.object(hours, "require_running"), mock.patch.object(hours, "Terraform", return_value=terraform), mock.patch.object(hours, "protected_json", side_effect=capture), mock.patch.object(hours, "publish") as publish, mock.patch.object(hours.time, "sleep"), mock.patch.object(hours, "datetime") as now:
                        now.now.return_value = datetime(2026, 10, 26, 15, tzinfo=UTC)
                        hours.apply(context, {"postgres_backup_bucket_name": "backup"}, configure)
                    self.assertEqual(stages[0], {"use_lambda_aliases": aliases, "activate_operating_hours": False})
                    self.assertEqual(stages[1], {"use_lambda_aliases": aliases, "activate_operating_hours": True})
                    self.assertEqual(configure.call_count, 0 if already_applied else 1)
                    publish.assert_called_once()
                    terraform.apply.assert_not_called()

    def test_failed_backup_does_not_publish_policy_or_enable(self):
        context = mock.MagicMock()
        context.config.operating_hours = parse({"enabled": True})
        context.config.backup = SimpleNamespace(time="21:00:00", timezone="America/Toronto")
        context.config.terraform_variables.return_value = {}
        context.aws.resource_exists.return_value = False
        terraform = mock.MagicMock()
        terraform.show_plan.return_value = {}
        with mock.patch.object(hours, "require_support"), mock.patch.object(hours, "require_running"), mock.patch.object(hours, "Terraform", return_value=terraform), mock.patch.object(hours, "publish") as publish, mock.patch.object(hours.time, "sleep"), mock.patch.object(hours, "datetime") as now:
            now.now.return_value = datetime(2026, 10, 26, 15, tzinfo=UTC)
            with self.assertRaisesRegex(CommandError, "incomplete"):
                hours.apply(context, {"postgres_backup_bucket_name": "backup"}, mock.Mock(side_effect=CommandError("backup failed")))
            publish.assert_not_called()
            self.assertEqual(terraform.plan.call_count, 1)

    def test_offline_deployment_refuses_and_manual_calls_are_discrete(self):
        context = mock.MagicMock()
        outputs = {"database_instance_id": "i-fixture"}
        def state(value): return {"Reservations": [{"Instances": [{"State": {"Name": value}}]}]}
        context.aws.json.return_value = state("stopped")
        with self.assertRaisesRegex(CommandError, "ACTION=database-start"):
            operations.require_running(context, outputs)
        context.aws.json.return_value = state("running")
        with mock.patch.object(operations, "wait_ready") as ready:
            operations.operate(context, outputs, "start")
            context.aws.call.assert_not_called()
            ready.assert_called_once()
        context.aws.json.side_effect = [state("running"), state("stopped")]
        operations.operate(context, outputs, "stop")
        context.aws.call.assert_called_once_with("ec2", "stop-instances", "--instance-ids", "i-fixture")

    def test_readiness_probe_has_an_absolute_retry_deadline(self):
        context = mock.MagicMock()
        context.aws.invoke_bootstrap.side_effect = CommandError("not ready")
        with mock.patch.object(operations.time, "monotonic", side_effect=[0, 0, 0, 15, 25]), mock.patch.object(operations.time, "sleep") as sleep:
            with self.assertRaisesRegex(CommandError, "25 seconds"):
                operations.wait_ready(context, {"bootstrap_function_name": "bootstrap"}, timeout=25)
        self.assertEqual(context.aws.invoke_bootstrap.call_args.kwargs["timeout_seconds"], 15)
        sleep.assert_called_once_with(10)

    def test_command_timeout_is_an_actionable_error(self):
        import subprocess
        from common.command import run
        with mock.patch("common.command.subprocess.run", side_effect=subprocess.TimeoutExpired("aws", 1)):
            with self.assertRaisesRegex(CommandError, "aws timed out"):
                run(["aws", "lambda"], timeout_seconds=1)

    def test_restore_works_without_production_health_and_targets_temporary_resources(self):
        import workflow
        context = mock.MagicMock()
        terraform = mock.MagicMock()
        with contextlib.ExitStack() as stack:
            for patch in (
                mock.patch.object(workflow, "preflight"),
                mock.patch.object(workflow.runtime, "repair_secret_boundary"),
                mock.patch.object(workflow, "_state", return_value=("complete", {})),
                mock.patch.object(workflow, "_terraform", side_effect=lambda *a, **k: contextlib.nullcontext(Path("/tmp/fixture-vars"))),
                mock.patch.object(workflow, "Terraform", return_value=terraform),
                mock.patch.object(workflow.artifacts, "build"),
                mock.patch.object(workflow, "_apply_backup_infrastructure"),
                mock.patch.object(workflow, "_confirm"),
                mock.patch.object(workflow, "require_restore_verification_create", return_value={}),
                mock.patch.object(workflow, "require_restore_verification_cleanup", return_value={}),
                mock.patch.object(workflow.database_backup, "restore_verify"),
            ): stack.enter_context(patch)
            require_healthy = stack.enter_context(mock.patch.object(workflow, "_require_complete"))
            running = stack.enter_context(mock.patch.object(workflow.database_operations, "require_running"))
            workflow.restore_verify(context)
            require_healthy.assert_not_called()
            running.assert_not_called()
        self.assertEqual(len(terraform.plan.call_args_list), 2)
        for call in terraform.plan.call_args_list:
            self.assertEqual(call.kwargs["targets"], workflow.RESTORE_TARGETS)

    def test_backup_configuration_does_not_plan_active_schedules(self):
        import workflow
        context = mock.MagicMock()
        terraform = mock.MagicMock()
        with mock.patch.object(workflow, "_terraform", side_effect=lambda *a, **k: contextlib.nullcontext(Path("/tmp/fixture-vars"))), mock.patch.object(workflow, "Terraform", return_value=terraform), mock.patch.object(workflow, "require_temporary_access_create", return_value={}), mock.patch.object(workflow, "require_cutover_only", return_value={}):
            workflow._configure_backup_with_temporary_access(context, configure=mock.Mock())
        for call in terraform.plan.call_args_list:
            self.assertEqual(call.kwargs["targets"], workflow.TEMPORARY_ACCESS_TARGETS)


if __name__ == "__main__": unittest.main()
