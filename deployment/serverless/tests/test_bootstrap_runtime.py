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
    )


class BootstrapRuntimeTest(unittest.TestCase):
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
        config.sender_environment = lambda function: {"Variables": {"PUSH_DELIVERY_FUNCTION_NAME": function}}
        outputs = {"database_host": "private", "delivery_function_name": "delivery", "sender_function_name": "sender"}
        client = mock.MagicMock()
        with tempfile.TemporaryDirectory() as temporary:
            runtime.configure_notifications(client, config, outputs, Path(temporary))
        calls = [(call[0], call.args[0]) for call in client.mock_calls]
        self.assertEqual(calls, [
            ("publish_environment", "delivery"), ("activate_notification_function", "delivery"),
            ("publish_environment", "sender"), ("activate_notification_function", "sender"),
        ])
        client.reset_mock()
        client.publish_environment.side_effect = CommandError("configuration failed")
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaises(CommandError):
                runtime.configure_notifications(client, config, outputs, Path(temporary))
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
