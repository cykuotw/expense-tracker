from __future__ import annotations

import contextlib
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from backend import runtime
from config import ConfigError, load
from database import alerts, connection_monitor as monitor
from tests import test_config
import workflow

HOOK = "https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyz_12345"


class MonitorTest(unittest.TestCase):
    def test_consecutive_alarm_recovery_failures_and_gap(self):
        state = {}
        monitor.transition(state, (16, 20), 1000)
        self.assertNotIn("pending", state)
        monitor.transition(state, (16, 20), 1060)
        self.assertEqual(state.pop("pending")["state"], "ALARM")
        monitor.transition(state, (19, 20), 1120)
        self.assertNotIn("pending", state)
        monitor.transition(state, (15, 20), 1180)
        self.assertNotIn("pending", state)
        monitor.transition(state, (10, 20), 1240)
        self.assertEqual(state.pop("pending")["state"], "OK")
        monitor.transition(state, None, 1300)
        monitor.transition(state, None, 1360)
        failure = state.pop("pending")
        self.assertEqual(failure["state"], "CHECK_FAILED")
        self.assertIsNone(failure["connections"])
        monitor.transition(state, (0, 20), 1420)
        monitor.transition(state, (0, 20), 1480)
        self.assertEqual(state.pop("pending")["state"], "OK")
        monitor.transition(state, (16, 20), 1540)
        monitor.transition(state, (16, 20), 1900)
        self.assertNotIn("pending", state, "stale samples cannot complete an alarm")

    def test_pending_event_survives_delivery_failure_and_reuses_id(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "state.json"
            with mock.patch.object(monitor, "collect", return_value=(16, 20)), mock.patch.object(monitor.time, "time", side_effect=[1000, 1060]), mock.patch.object(monitor, "deliver", side_effect=RuntimeError("offline")):
                monitor.tick(path, "bucket", "ca-central-1")
                with self.assertRaises(RuntimeError):
                    monitor.tick(path, "bucket", "ca-central-1")
            event = json.loads(path.read_text())["pending"]
            with mock.patch.object(monitor, "collect", return_value=(16, 20)), mock.patch.object(monitor.time, "time", return_value=1120), mock.patch.object(monitor, "deliver") as send:
                monitor.tick(path, "bucket", "ca-central-1")
            send.assert_called_once_with("bucket", "ca-central-1", event)
            self.assertNotIn("pending", json.loads(path.read_text()))
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)

    def test_collection_is_instance_wide_excludes_itself_and_is_bounded(self):
        with mock.patch.object(monitor.subprocess, "run", return_value=mock.Mock(stdout='{"connections":16,"max_connections":20}')) as run:
            self.assertEqual(monitor.collect(), (16, 20))
        command = run.call_args.args[0]
        self.assertIn("--dbname=postgres", command)
        self.assertIn("pid <> pg_backend_pid()", command[-1])
        self.assertIn("backend_type = 'client backend'", command[-1])
        self.assertNotIn("datname =", command[-1])
        self.assertEqual(run.call_args.kwargs["timeout"], 8)
        for result in ['{"connections":-1,"max_connections":20}', '{"connections":20,"max_connections":20}']:
            with mock.patch.object(monitor.subprocess, "run", return_value=mock.Mock(stdout=result)):
                with self.assertRaises(ValueError): monitor.collect()

    def test_relay_posts_only_state_json_to_stable_private_key(self):
        event = {"event_id": "fixed", "state": "ALARM"}
        with mock.patch.object(monitor.subprocess, "run", return_value=mock.Mock(returncode=0)) as run:
            monitor.deliver("private-bucket", "ca-central-1", event)
        self.assertIn("s3://private-bucket/database-alerts/fixed.json", run.call_args.args[0])
        self.assertEqual(json.loads(run.call_args.kwargs["input"]), event)
        self.assertTrue(run.call_args.kwargs["capture_output"])

    def test_installer_is_valid_shell_and_does_not_contain_webhooks(self):
        script = alerts.installation_script("private-bucket", "ca-central-1")
        subprocess.run(["bash", "-n"], input=script, text=True, check=True)
        subprocess.run(["bash", "-n"], input=alerts.DISABLE_SCRIPT, text=True, check=True)
        self.assertNotIn("discord", script.lower())
        self.assertIn("OnUnitActiveSec=1min", script)
        with self.assertRaises(Exception): alerts.installation_script("bad;command", "ca-central-1")


class DatabaseAlertConfigTest(unittest.TestCase):
    setUp = test_config.ConfigTest.setUp
    tearDown = test_config.ConfigTest.tearDown
    write = test_config.ConfigTest.write

    def test_separate_channel_is_protected_and_requires_existing_notifier(self):
        self.value["observability"]["discord_webhook_url"] = HOOK
        self.value["observability"]["database_discord_webhook_url"] = HOOK.replace("123456789012345678", "223456789012345678")
        self.write()
        config = load(self.path, Path("/unrelated/repository"))
        self.assertTrue(config.database_alerting_enabled)
        projected = config.notifier_environment("backup-bucket")["Variables"]
        self.assertEqual(projected["DATABASE_ALERT_BUCKET"], "backup-bucket")
        self.assertNotEqual(projected["DATABASE_DISCORD_WEBHOOK_URL"], projected["DISCORD_WEBHOOK_URL"])
        self.assertNotIn("discord.com", json.dumps(config.terraform_variables(temporary_access=False)))
        self.assertTrue(config.terraform_variables(temporary_access=False)["enable_database_connection_alerting"])
        self.assertIn(projected["DATABASE_DISCORD_WEBHOOK_URL"].encode(), runtime._protected_values(config))
        with self.assertRaises(ConfigError): config.notifier_environment()
        self.value["observability"]["database_discord_webhook_url"] = HOOK
        self.write()
        with self.assertRaisesRegex(ConfigError, "separate Discord webhook"):
            load(self.path, Path("/unrelated/repository"))
        self.value["observability"]["discord_webhook_url"] = None
        self.write()
        with self.assertRaisesRegex(ConfigError, "existing application error notifier"):
            load(self.path, Path("/unrelated/repository"))


class DatabaseAlertInfrastructureTest(unittest.TestCase):
    def test_both_alias_paths_and_exact_disablement_deletes(self):
        for alias in (False, True):
            for enabled in (False, True):
                with self.subTest(alias=alias, enabled=enabled):
                    context = mock.MagicMock()
                    context.config.database_alerting_enabled = enabled
                    context.config.error_alerting_enabled = True
                    context.terraform_root = ROOT / "infrastructure/tf"
                    tf = mock.MagicMock()
                    tf.show_plan.return_value = {"resource_changes": [
                        {"address": address, "change": {"actions": ["delete"]}}
                        for address in workflow.DATABASE_ALERT_RESOURCE_ADDRESSES
                    ]}
                    with mock.patch.object(workflow, "_terraform", return_value=contextlib.nullcontext(Path("/tmp/test-vars"))) as variables, mock.patch.object(workflow, "Terraform", return_value=tf), mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0):
                        if enabled:
                            with self.assertRaisesRegex(Exception, "destructive"):
                                workflow._apply_infrastructure_updates(context, "backend", use_lambda_aliases=alias)
                            tf.apply.assert_not_called()
                        else:
                            workflow._apply_infrastructure_updates(context, "backend", use_lambda_aliases=alias)
                            tf.apply.assert_called_once()
                            self.assertEqual(variables.call_args.kwargs["use_lambda_aliases"], alias)
                            self.assertIn("aws_s3_bucket_notification.database_alerts", tf.plan.call_args.kwargs["targets"])
        for alias in (False, True):
            for unrelated in (False, True):
                context = mock.MagicMock()
                context.config.database_alerting_enabled = True
                context.config.error_alerting_enabled = True
                context.terraform_root = ROOT / "infrastructure/tf"
                tf = mock.MagicMock()
                changes = [{"address": "aws_lambda_permission.database_alerts[0]", "change": {"actions": ["delete", "create"]}}]
                if unrelated:
                    changes.append({"address": "aws_s3_bucket.postgres_backup", "change": {"actions": ["delete"]}})
                tf.show_plan.return_value = {"resource_changes": changes}
                with mock.patch.object(workflow, "_terraform", return_value=contextlib.nullcontext(Path("/tmp/test-vars"))), mock.patch.object(workflow, "Terraform", return_value=tf), mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0):
                    if unrelated:
                        with self.assertRaisesRegex(Exception, "destructive"):
                            workflow._apply_infrastructure_updates(context, "backend", use_lambda_aliases=alias)
                        tf.apply.assert_not_called()
                    else:
                        workflow._apply_infrastructure_updates(context, "backend", use_lambda_aliases=alias)
                        tf.apply.assert_called_once()
        source = (ROOT / "infrastructure/tf/database_alerts.tf").read_text()
        self.assertIn('var.use_lambda_aliases ? local.error_notifier_live_arn', source)
        self.assertIn('qualifier      = var.use_lambda_aliases ? "live" : null', source)
        self.assertIn('source_account = var.expected_account_id', source)
        self.assertNotIn("webhook", source.lower())
        self.assertNotIn("aws_cloudwatch_metric", source)

    def test_monitor_install_failure_still_removes_temporary_access(self):
        from common.command import CommandError
        context = mock.MagicMock()
        tf = mock.MagicMock()
        temporary = ["aws_eip.temporary_postgres[0]", "aws_eip_association.temporary_postgres[0]", "aws_vpc_security_group_ingress_rule.ssh[0]"]
        tf.show_plan.side_effect = [
            {"resource_changes": [{"address": address, "change": {"actions": [action]}} for address in temporary]}
            for action in ("create", "delete")
        ]
        configure = mock.Mock(side_effect=CommandError("installation failed"))
        with mock.patch.object(workflow, "_terraform", side_effect=lambda *args, **kwargs: contextlib.nullcontext(Path("/tmp/test-vars"))), mock.patch.object(workflow, "Terraform", return_value=tf), mock.patch.object(workflow.database_backup, "diagnose") as diagnose:
            with self.assertRaisesRegex(CommandError, "installation failed"):
                workflow._configure_backup_with_temporary_access(context, configure=configure)
        self.assertEqual(tf.apply.call_count, 2)
        configure.assert_called_once()
        diagnose.assert_not_called()
