from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.command import CommandError
from frontend.verify import verify


def config() -> SimpleNamespace:
    return SimpleNamespace(
        frontend_origin="https://app.example.com",
        api_origin="https://api.example.com",
        backend=SimpleNamespace(google_client_id="client-id"),
    )


def runtime(value: dict[str, object]) -> str:
    return f"window.__APP_CONFIG__ = Object.freeze({json.dumps(value)});"


EXPECTED = {
    "apiOrigin": "https://api.example.com",
    "apiPath": "/api/v0",
    "googleOAuthEnabled": True,
    "googleClientId": "client-id",
}


class FrontendVerifyTest(unittest.TestCase):
    def verify_runtime(self, value: dict[str, object], **kwargs: object) -> None:
        with mock.patch("frontend.verify._get", side_effect=[
            (200, '<script src="/runtime-config.js"></script>'),
            (200, runtime(value)),
        ]):
            verify(config(), **kwargs)

    def test_accepts_versioned_runtime_and_additive_fields(self) -> None:
        self.verify_runtime({**EXPECTED, "frontendVersion": "v-20260908-deadbeef", "futureField": True})

    def test_upgrade_check_accepts_legacy_runtime_without_version(self) -> None:
        self.verify_runtime(EXPECTED, require_frontend_version=False)

    def test_post_publish_check_requires_valid_version(self) -> None:
        for version in (None, "v-development", "v-20260908-nothex"):
            value = dict(EXPECTED)
            if version is not None:
                value["frontendVersion"] = version
            with self.subTest(version=version), self.assertRaisesRegex(CommandError, "valid frontend version"):
                self.verify_runtime(value)

    def test_still_rejects_changed_deployment_fields(self) -> None:
        with self.assertRaisesRegex(CommandError, "does not match"):
            self.verify_runtime({**EXPECTED, "apiOrigin": "https://wrong.example.com", "frontendVersion": "v-20260908-deadbeef"})


if __name__ == "__main__":
    unittest.main()
