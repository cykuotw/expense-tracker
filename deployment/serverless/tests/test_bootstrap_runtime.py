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
        ),
        first_admin=None,
        bootstrap_environment=lambda _host: {"SAFE": "value"},
    )


class BootstrapRuntimeTest(unittest.TestCase):
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
