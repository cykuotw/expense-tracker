from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from backend import runtime
from common.command import CommandError


def runtime_config() -> SimpleNamespace:
    return SimpleNamespace(
        database=SimpleNamespace(
            admin_password="admin-password",
            migration_password="migration-password",
            runtime_password="runtime-password",
        ),
        backend=SimpleNamespace(
            jwt_secret="jwt-secret",
            refresh_jwt_secret="refresh-secret",
            web_push_vapid_public_key="vapid-public-key",
            web_push_vapid_private_key="vapid-private-key",
            web_push_vapid_subject="mailto:ops@example.com",
        ),
        first_admin=None,
        observability=SimpleNamespace(discord_webhook_url=None),
        error_alerting_enabled=False,
        bootstrap_environment=lambda _host: {"SAFE": "value"},
        worker_environment=lambda _host: {"Variables": {"SAFE": "worker"}},
        delivery_environment=lambda _host: {"Variables": {"SAFE": "delivery"}},
        sender_environment=lambda function: {"Variables": {"DELIVERY": function}},
        notifier_environment=lambda: {"Variables": {"SAFE": "notifier"}},
    )


class BootstrapRuntimeTest(unittest.TestCase):
    def test_release_publication_orders_code_config_then_version(self) -> None:
        client = mock.MagicMock()
        client.publish_version.return_value = {
            "functionName": "worker",
            "version": "7",
            "qualifiedArn": "arn:aws:lambda:ca-central-1:123:function:worker:7",
            "codeSha256": "hash",
        }
        with tempfile.TemporaryDirectory() as temporary:
            record = runtime.publish_worker_release(
                client,
                Path("worker.zip"),
                runtime_config(),
                {"worker_function_name": "worker", "database_host": "db.internal"},
                Path(temporary),
                "20260919T184500Z-a1b2c3d4e5f6",
            )

        calls = [item[0] for item in client.mock_calls]
        self.assertLess(calls.index("update_code"), calls.index("update_environment"))
        self.assertLess(calls.index("update_environment"), calls.index("publish_version"))
        self.assertLess(calls.index("publish_version"), calls.index("activate_worker"))
        self.assertEqual(record["version"], "7")

    def test_bootstrap_release_invokes_exact_candidate_version(self) -> None:
        client = mock.MagicMock()
        client.publish_version.return_value = {
            "functionName": "bootstrap",
            "version": "4",
            "qualifiedArn": "arn:aws:lambda:ca-central-1:123:function:bootstrap:4",
            "codeSha256": "hash",
        }
        client.invoke_bootstrap.side_effect = [
            {
                "first_admin_status": "reconciled",
                "migration_version": 35,
                "migration_dirty": False,
            },
            {"first_admin_status": "already_exists"},
        ]
        with tempfile.TemporaryDirectory() as temporary:
            record, response = runtime.publish_bootstrap_release(
                client,
                Path("bootstrap.zip"),
                runtime_config(),
                {
                    "bootstrap_function_name": "bootstrap",
                    "database_host": "db.internal",
                },
                Path(temporary),
                "20260919T184500Z-a1b2c3d4e5f6",
            )

        self.assertEqual(record["version"], "4")
        self.assertEqual(response["migration_version"], 35)
        self.assertEqual(client.invoke_bootstrap.call_count, 2)
        self.assertTrue(
            all(
                item.args[0].endswith(":4")
                for item in client.invoke_bootstrap.call_args_list
            )
        )

    def test_sender_release_invokes_delivery_live_alias(self) -> None:
        client = mock.MagicMock()
        client.publish_version.side_effect = [
            {
                "functionName": "delivery",
                "version": "3",
                "qualifiedArn": "arn:aws:lambda:ca-central-1:123:function:delivery:3",
                "codeSha256": "delivery-hash",
            },
            {
                "functionName": "sender",
                "version": "5",
                "qualifiedArn": "arn:aws:lambda:ca-central-1:123:function:sender:5",
                "codeSha256": "sender-hash",
            },
        ]
        config = runtime_config()
        with mock.patch.object(
            config,
            "sender_environment",
            wraps=config.sender_environment,
        ) as sender_environment, tempfile.TemporaryDirectory() as temporary:
            runtime.publish_notification_releases(
                client,
                Path("sender.zip"),
                Path("delivery.zip"),
                config,
                {
                    "delivery_function_name": "delivery",
                    "sender_function_name": "sender",
                    "database_host": "db.internal",
                },
                Path(temporary),
                "20260919T184500Z-a1b2c3d4e5f6",
            )

        sender_environment.assert_called_once_with("delivery:live")

    def test_error_notifier_configuration_is_optional_and_fail_closed(self) -> None:
        client = mock.MagicMock()
        config = runtime_config()
        with tempfile.TemporaryDirectory() as temporary:
            runtime.configure_error_notifier(client, config, {}, Path(temporary))
        client.assert_not_called()

        config.error_alerting_enabled = True
        config.observability.discord_webhook_url = "private-webhook"
        config.notifier_environment = lambda: {"Variables": {"DISCORD_WEBHOOK_URL": "private-webhook"}}
        outputs = {"error_notifier_function_name": "notifier"}
        with tempfile.TemporaryDirectory() as temporary:
            runtime.configure_error_notifier(client, config, outputs, Path(temporary))
        self.assertEqual(
            [(call[0], call.args[0]) for call in client.mock_calls],
            [("publish_environment", "notifier"), ("activate_notification_function", "notifier")],
        )

    def test_error_notifier_update_publishes_before_configuration(self) -> None:
        client = mock.MagicMock()
        config = runtime_config()
        config.error_alerting_enabled = True
        outputs = {"error_notifier_function_name": "notifier"}
        with mock.patch.object(runtime, "configure_error_notifier") as configure:
            runtime.update_error_notifier(
                client,
                Path("notifier.zip"),
                config,
                outputs,
                Path("/tmp"),
            )
        client.publish_code.assert_called_once_with("notifier", Path("notifier.zip"))
        configure.assert_called_once()

    def test_delivery_is_configured_before_sender_and_failure_stops_activation(self) -> None:
        config = runtime_config()
        config.delivery_environment = lambda _host: {"Variables": {"DB_PUBLIC_HOST": "private"}}
        original_sender_environment = config.sender_environment
        config.sender_environment = mock.MagicMock(wraps=original_sender_environment)
        outputs = {"database_host": "private", "delivery_function_name": "delivery", "sender_function_name": "sender"}
        client = mock.MagicMock()
        with tempfile.TemporaryDirectory() as temporary:
            runtime.configure_notifications(client, config, outputs, Path(temporary))
        calls = [(call[0], call.args[0]) for call in client.mock_calls]
        self.assertEqual(calls, [
            ("publish_environment", "delivery"), ("activate_notification_function", "delivery"),
            ("publish_environment", "sender"), ("activate_notification_function", "sender"),
        ])
        config.sender_environment.assert_called_once_with("delivery:live")
        client.reset_mock()
        config.sender_environment.reset_mock()
        client.publish_environment.side_effect = CommandError("configuration failed")
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaises(CommandError):
                runtime.configure_notifications(client, config, outputs, Path(temporary))
        client.activate_notification_function.assert_not_called()

    def test_notification_configuration_can_defer_activation_until_aliases_exist(self) -> None:
        client = mock.MagicMock()
        config = runtime_config()
        outputs = {
            "database_host": "private",
            "delivery_function_name": "delivery",
            "sender_function_name": "sender",
        }

        with tempfile.TemporaryDirectory() as temporary:
            runtime.configure_notifications(
                client,
                config,
                outputs,
                Path(temporary),
                activate=False,
            )

        self.assertEqual(
            [call.args[0] for call in client.publish_environment.call_args_list],
            ["delivery", "sender"],
        )
        client.activate_notification_function.assert_not_called()

    def test_update_publishes_both_notification_artifacts_before_configuration(self) -> None:
        client = mock.MagicMock()
        outputs = {"delivery_function_name": "delivery", "sender_function_name": "sender"}
        with mock.patch.object(runtime, "configure_notifications") as configure:
            runtime.update_notifications(client, Path("sender.zip"), Path("delivery.zip"), mock.sentinel.config, outputs, Path("/tmp"))
        self.assertEqual(client.publish_code.call_args_list, [mock.call("delivery", Path("delivery.zip")), mock.call("sender", Path("sender.zip"))])
        configure.assert_called_once()

    def test_reconciled_first_invocation_is_idempotent(self) -> None:
        client = mock.MagicMock()
        client.invoke_bootstrap.side_effect = [
            {"first_admin_status": "reconciled"},
            {"first_admin_status": "already_exists"},
        ]
        with tempfile.TemporaryDirectory() as temporary:
            result = runtime.configure_bootstrap(
                client,
                runtime_config(),
                {"database_host": "db.internal", "bootstrap_function_name": "bootstrap"},
                Path(temporary),
            )

        self.assertEqual(result["first_admin_status"], "reconciled")
        self.assertEqual(client.invoke_bootstrap.call_count, 2)

    def test_invalid_first_status_fails_closed(self) -> None:
        client = mock.MagicMock()
        client.invoke_bootstrap.side_effect = [
            {"first_admin_status": "unexpected"},
            {"first_admin_status": "already_exists"},
        ]
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaisesRegex(CommandError, "invalid first-admin status"):
                runtime.configure_bootstrap(
                    client,
                    runtime_config(),
                    {"database_host": "db.internal", "bootstrap_function_name": "bootstrap"},
                    Path(temporary),
                )

    def test_non_idempotent_second_status_fails_closed(self) -> None:
        client = mock.MagicMock()
        client.invoke_bootstrap.side_effect = [
            {"first_admin_status": "created"},
            {"first_admin_status": "reconciled"},
        ]
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaisesRegex(CommandError, "not idempotent"):
                runtime.configure_bootstrap(
                    client,
                    runtime_config(),
                    {"database_host": "db.internal", "bootstrap_function_name": "bootstrap"},
                    Path(temporary),
                )


if __name__ == "__main__":
    unittest.main()
