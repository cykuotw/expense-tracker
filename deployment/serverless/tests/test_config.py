from __future__ import annotations

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.command import protected_json
from config import ConfigError, initialize, load, template


class ConfigTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary.name)
        self.key = self.root / "key.pem"
        self.key.write_text("test key")
        os.chmod(self.key, 0o600)
        self.path = self.root / "deploy.json"
        self.value = template()
        self.value["deployment"]["account_id"] = "123456789012"
        self.value["database"].update({
            "admin_password": "admin-password-long-enough",
            "migration_password": "migration-password-long-enough",
            "runtime_password": "runtime-password-long-enough",
        })
        self.value["backend"].update({
            "google_client_id": "client.apps.googleusercontent.com",
            "jwt_secret": "j" * 32,
            "refresh_jwt_secret": "r" * 32,
            "web_push_vapid_public_key": "p" * 43,
            "web_push_vapid_private_key": "k" * 43,
            "web_push_vapid_subject": "mailto:ops@example.com",
        })
        self.value["local_credentials"]["ssh_private_key_file"] = str(self.key)

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def write(self) -> None:
        self.path.write_text(json.dumps(self.value))
        os.chmod(self.path, 0o600)

    def test_derives_runtime_and_non_secret_terraform_values(self) -> None:
        self.write()
        config = load(self.path, Path("/unrelated/repository"))
        self.assertEqual(config.api_origin, "https://api.example.com")
        variables = config.terraform_variables(temporary_access=True)
        self.assertNotIn("runtime_password", variables)
        self.assertNotIn("web_push_vapid_public_key", variables)
        self.assertNotIn("web_push_vapid_private_key", variables)
        self.assertNotIn("web_push_vapid_subject", variables)
        self.assertTrue(variables["enable_temporary_public_access"])
        self.assertFalse(variables["enable_restore_verification"])
        self.assertEqual(config.backup.time, "03:17:00")
        self.assertEqual(config.backup.timezone, "UTC")
        self.assertIsNone(config.observability.discord_webhook_url)
        self.assertFalse(config.error_alerting_enabled)
        self.assertFalse(variables["enable_error_alerting"])
        worker = config.worker_environment("10.0.0.2")["Variables"]
        self.assertEqual(worker["AUTH_COOKIE_SAME_SITE"], "lax")
        self.assertEqual(worker["WEB_PUSH_VAPID_PUBLIC_KEY"], "p" * 43)
        self.assertEqual(worker["DB_MAX_OPEN_CONNS"], "2")
        self.assertEqual(worker["DB_MAX_IDLE_CONNS"], "1")
        sender = config.sender_environment("delivery-function")["Variables"]
        delivery = config.delivery_environment("10.0.0.2")["Variables"]
        self.assertEqual(sender["WEB_PUSH_VAPID_PRIVATE_KEY"], "k" * 43)
        self.assertEqual(sender["PUSH_DELIVERY_FUNCTION_NAME"], "delivery-function")
        self.assertFalse(any(key.startswith("DB_") for key in sender))
        self.assertEqual(delivery["DB_PUBLIC_HOST"], "10.0.0.2")
        self.assertFalse(any(key.startswith("WEB_PUSH_") for key in delivery))
        self.assertNotIn("WEB_PUSH_VAPID_PRIVATE_KEY", config.worker_environment("10.0.0.2")["Variables"])

    def test_validates_webhook_and_projects_it_only_to_protected_runtime(self) -> None:
        webhook_url = "https://discord.com/api/webhooks/1234567890/" + "token_value_" * 3
        self.value["observability"]["discord_webhook_url"] = webhook_url
        self.write()

        config = load(self.path, Path("/unrelated/repository"))
        self.assertEqual(config.observability.discord_webhook_url, webhook_url)
        self.assertTrue(config.error_alerting_enabled)
        self.assertEqual(config.notifier_environment(), {"Variables": {
            "DISCORD_WEBHOOK_URL": webhook_url,
            "DEPLOYMENT_ENVIRONMENT": "serverless",
        }})
        variables = config.terraform_variables(temporary_access=False)
        self.assertTrue(variables["enable_error_alerting"])
        self.assertNotIn(webhook_url, json.dumps(variables))

    def test_rejects_invalid_discord_webhook_urls(self) -> None:
        for value in (
            "http://discord.com/api/webhooks/123/" + "x" * 24,
            "https://example.com/api/webhooks/123/" + "x" * 24,
            "https://discord.com/channels/123/456",
            "https://discord.com/api/webhooks/not-a-number/" + "x" * 24,
            "https://discord.com:8443/api/webhooks/123/" + "x" * 24,
            "https://discord.com/api/webhooks/123/" + "x" * 24 + "?wait=true",
            "",
        ):
            with self.subTest(value=value):
                self.value["observability"]["discord_webhook_url"] = value
                self.write()
                with self.assertRaisesRegex(ConfigError, "discord_webhook_url"):
                    load(self.path, Path("/unrelated/repository"))

    def test_disabled_discord_alerting_has_no_notifier_environment(self) -> None:
        self.write()
        config = load(self.path, Path("/unrelated/repository"))
        with self.assertRaisesRegex(ConfigError, "required when error alerting is enabled"):
            config.notifier_environment()

    def test_rejects_invalid_vapid_configuration(self) -> None:
        self.value["backend"]["web_push_vapid_private_key"] = "short"
        self.write()
        with self.assertRaisesRegex(ConfigError, "web_push_vapid_private_key"):
            load(self.path, Path("/unrelated/repository"))
        self.value["backend"]["web_push_vapid_private_key"] = "k" * 43
        self.value["backend"]["web_push_vapid_subject"] = "ops@example.com"
        self.write()
        with self.assertRaisesRegex(ConfigError, "web_push_vapid_subject"):
            load(self.path, Path("/unrelated/repository"))

    def test_rejects_insecure_token_and_numeric_boundaries(self) -> None:
        cases = (
            ("jwt_secret", "s" * 32, "jwt_secret and backend.refresh_jwt_secret"),
            ("jwt_exp", 59, "backend.jwt_exp"),
            ("jwt_exp", 86_401, "backend.jwt_exp"),
            ("refresh_jwt_exp", 300, "greater than backend.jwt_exp"),
            ("refresh_jwt_exp", 31_536_001, "backend.refresh_jwt_exp"),
            ("expenses_per_page", 1_001, "backend.expenses_per_page"),
            ("db_conn_max_lifetime_seconds", -1, "backend.db_conn_max_lifetime_seconds"),
            ("db_conn_max_idle_time_seconds", 86_401, "backend.db_conn_max_idle_time_seconds"),
        )
        for key, value, message in cases:
            with self.subTest(key=key, value=value):
                original = self.value["backend"][key]
                if key == "jwt_secret":
                    self.value["backend"]["refresh_jwt_secret"] = value
                self.value["backend"][key] = value
                self.write()
                with self.assertRaisesRegex(ConfigError, message):
                    load(self.path, Path("/unrelated/repository"))
                self.value["backend"][key] = original
                self.value["backend"]["refresh_jwt_secret"] = "r" * 32

    def test_accepts_disabled_database_connection_durations(self) -> None:
        self.value["backend"]["db_conn_max_lifetime_seconds"] = 0
        self.value["backend"]["db_conn_max_idle_time_seconds"] = 0
        self.write()
        config = load(self.path, Path("/unrelated/repository"))
        self.assertEqual(config.backend.db_conn_max_lifetime_seconds, 0)
        self.assertEqual(config.backend.db_conn_max_idle_time_seconds, 0)

    def test_validation_errors_do_not_expose_secret_values(self) -> None:
        supplied = "private-value-that-is-too-short"
        self.value["backend"]["jwt_secret"] = supplied
        self.write()
        with self.assertRaises(ConfigError) as raised:
            load(self.path, Path("/unrelated/repository"))
        self.assertIn("backend.jwt_secret", str(raised.exception))
        self.assertNotIn(supplied, str(raised.exception))

    def test_accepts_configured_backup_time_and_iana_timezone(self) -> None:
        self.value["backup"] = {"time": "01:17:00", "timezone": "America/Toronto"}
        self.write()
        config = load(self.path, Path("/unrelated/repository"))
        self.assertEqual(config.backup.time, "01:17:00")
        self.assertEqual(config.backup.timezone, "America/Toronto")

    def test_rejects_invalid_backup_schedule(self) -> None:
        self.value["backup"] = {"time": "1:17", "timezone": "America/Toronto"}
        self.write()
        with self.assertRaisesRegex(ConfigError, "backup.time"):
            load(self.path, Path("/unrelated/repository"))
        self.value["backup"] = {"time": "01:17:00", "timezone": "Toronto"}
        self.write()
        with self.assertRaisesRegex(ConfigError, "backup.timezone"):
            load(self.path, Path("/unrelated/repository"))

    def test_rejects_unknown_keys(self) -> None:
        self.value["backend"]["surprise"] = True
        self.write()
        with self.assertRaisesRegex(ConfigError, "unknown keys"):
            load(self.path, Path("/unrelated/repository"))

    def test_requires_mode_0600(self) -> None:
        self.write()
        os.chmod(self.path, 0o644)
        with self.assertRaisesRegex(ConfigError, "mode 0600"):
            load(self.path, Path("/unrelated/repository"))

    def test_initialize_and_temporary_projection_permissions(self) -> None:
        initialized = self.root / "created.json"
        initialize(initialized, Path("/unrelated/repository"))
        self.assertEqual(initialized.stat().st_mode & 0o777, 0o600)
        with protected_json({"safe": "value"}, prefix="expense-test-") as projection:
            self.assertTrue(projection.is_file())
            self.assertEqual(projection.stat().st_mode & 0o777, 0o600)
        self.assertFalse(projection.exists())
