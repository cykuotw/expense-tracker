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
        self.assertTrue(variables["enable_temporary_public_access"])
        self.assertFalse(variables["enable_restore_verification"])
        self.assertEqual(config.backup.time, "03:17:00")
        self.assertEqual(config.backup.timezone, "UTC")
        self.assertEqual(config.worker_environment("10.0.0.2")["Variables"]["AUTH_COOKIE_SAME_SITE"], "lax")

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
