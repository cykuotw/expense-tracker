from __future__ import annotations

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace


ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from backend.runtime import repair_secret_boundary
from common.command import CommandError


def config() -> SimpleNamespace:
    return SimpleNamespace(
        database=SimpleNamespace(
            admin_password="admin-secret",
            migration_password="migration-secret",
            runtime_password="runtime-secret",
        ),
        backend=SimpleNamespace(
            jwt_secret="jwt-secret",
            refresh_jwt_secret="refresh-secret",
        ),
        first_admin=None,
    )


def state(environment: dict[str, str], *, unrelated: str = "safe") -> dict[str, object]:
    environment_state = [{"variables": environment}] if environment else []
    bootstrap_environment_state = (
        [{"variables": {"DB_ADMIN_PASSWORD": "admin-secret"}}]
        if environment
        else []
    )
    return {
        "version": 4,
        "serial": 10,
        "lineage": "test-lineage",
        "resources": [
            {
                "mode": "managed",
                "type": "aws_lambda_function",
                "name": "worker",
                "instances": [{"attributes": {"environment": environment_state}}],
            },
            {
                "mode": "managed",
                "type": "aws_lambda_function",
                "name": "bootstrap",
                "instances": [{"attributes": {"environment": bootstrap_environment_state}}],
            },
            {
                "mode": "managed",
                "type": "aws_s3_bucket",
                "name": "frontend",
                "instances": [{"attributes": {"tags": {"value": unrelated}}}],
            },
        ],
    }


class RuntimeStateTest(unittest.TestCase):
    def test_repairs_only_unmanaged_lambda_environment(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            path = root / "terraform.tfstate"
            backup = root / "terraform.tfstate.backup"
            original = json.dumps(state({"DB_PASSWORD": "runtime-secret"}))
            path.write_text(original)
            backup.write_text(original)
            os.chmod(path, 0o600)

            self.assertEqual(repair_secret_boundary(root, config()), 2)

            for repaired_path in (path, backup):
                repaired = json.loads(repaired_path.read_text())
                self.assertEqual(repaired["serial"], 11)
                self.assertEqual(repaired["resources"][0]["instances"][0]["attributes"]["environment"], [])
                self.assertEqual(repaired["resources"][1]["instances"][0]["attributes"]["environment"], [])
                self.assertEqual(repaired["resources"][2]["instances"][0]["attributes"]["tags"], {"value": "safe"})
                self.assertEqual(repaired_path.stat().st_mode & 0o777, 0o600)
                self.assertNotIn(b"runtime-secret", repaired_path.read_bytes())
                self.assertNotIn(b"admin-secret", repaired_path.read_bytes())

    def test_refuses_repair_when_secret_exists_outside_lambda_environment(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            path = root / "terraform.tfstate"
            original = json.dumps(state({"DB_PASSWORD": "runtime-secret"}, unrelated="jwt-secret"))
            path.write_text(original)

            with self.assertRaisesRegex(CommandError, "remains outside"):
                repair_secret_boundary(root, config())

            self.assertEqual(path.read_text(), original)

    def test_safe_state_is_not_rewritten(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            path = root / "terraform.tfstate"
            original = json.dumps(state({}))
            path.write_text(original)

            self.assertEqual(repair_secret_boundary(root, config()), 0)
            self.assertEqual(path.read_text(), original)


if __name__ == "__main__":
    unittest.main()
