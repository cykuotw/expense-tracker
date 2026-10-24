from __future__ import annotations

import json
import re
import sys
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.command import CommandError
from frontend.verify import _csp, verify


def config() -> SimpleNamespace:
    return SimpleNamespace(
        frontend_origin="https://app.example.com",
        api_origin="https://api.example.com",
        backend=SimpleNamespace(google_client_id="client-id"),
        aws=SimpleNamespace(region="ca-central-1"),
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
    def verify_runtime(
        self,
        value: dict[str, object],
        *,
        document_headers: dict[str, str] | None = None,
        **kwargs: object,
    ) -> None:
        headers = document_headers if document_headers is not None else {
            "referrer-policy": "strict-origin-when-cross-origin",
            "x-content-type-options": "nosniff",
            "x-frame-options": "DENY",
            "content-security-policy": _csp(
                "https://api.example.com"
            ),
        }
        with mock.patch("frontend.verify._get", side_effect=[
            (200, '<script src="/runtime-config.js"></script>', headers),
            (200, runtime(value), {}),
        ]):
            verify(config(), **kwargs)

    def test_accepts_versioned_runtime_and_additive_fields(self) -> None:
        self.verify_runtime({**EXPECTED, "frontendVersion": "v-20260908-deadbeef", "futureField": True})

    def test_accepts_content_addressed_frontend_version(self) -> None:
        self.verify_runtime(
            {**EXPECTED, "frontendVersion": "v-deadbeef01234567"}
        )

    def test_receipt_csp_matches_terraform_and_rejects_missing_or_wrong_bucket(self) -> None:
        source = (ROOT / "infrastructure/tf/frontend.tf").read_text()
        directives = source.split('frontend_csp = ', 1)[1].split('])};"', 1)[0]
        policy = "; ".join(re.findall(r'^\s+"(.*)",?$', directives, re.MULTILINE)) + ";"
        policy = policy.replace('${var.api_hostname}', 'api.example.com').replace(
            '${aws_s3_bucket.receipt.bucket_regional_domain_name}',
            'private-receipts.s3.ca-central-1.amazonaws.com',
        )
        headers = {
            "referrer-policy": "strict-origin-when-cross-origin",
            "x-content-type-options": "nosniff",
            "x-frame-options": "DENY",
            "content-security-policy": policy,
        }
        value = {**EXPECTED, "frontendVersion": "v-deadbeef01234567"}
        self.verify_runtime(value, document_headers=headers, receipt_bucket="private-receipts")
        receipt_origin = ' https://private-receipts.s3.ca-central-1.amazonaws.com'
        for invalid in (
            policy.replace(receipt_origin, ''),
            # The original failure: img-src allowed S3, but worker fetch() did not.
            policy.replace(receipt_origin, '', 1),
            policy.replace(f"img-src 'self' data: blob:{receipt_origin}", "img-src 'self' data: blob:"),
            policy.replace('private-receipts.s3.', 'other-bucket.s3.'),
            policy.replace('private-receipts.s3.', '*.s3.'),
        ):
            with self.subTest(policy=invalid), self.assertRaisesRegex(CommandError, "content-security-policy"):
                self.verify_runtime(value, document_headers={**headers, "content-security-policy": invalid}, receipt_bucket="private-receipts")

    def test_upgrade_check_accepts_legacy_runtime_and_headers(self) -> None:
        self.verify_runtime(
            EXPECTED,
            document_headers={},
            require_frontend_version=False,
            require_security_headers=False,
        )

    def test_post_publish_check_requires_security_headers(self) -> None:
        expected_headers = {
            "referrer-policy": "strict-origin-when-cross-origin",
            "x-content-type-options": "nosniff",
            "x-frame-options": "DENY",
            "content-security-policy": _csp(
                "https://api.example.com"
            ),
        }
        for missing in expected_headers:
            headers = dict(expected_headers)
            del headers[missing]
            with self.subTest(missing=missing), self.assertRaisesRegex(
                CommandError, missing
            ):
                self.verify_runtime(
                    {**EXPECTED, "frontendVersion": "v-20260908-deadbeef"},
                    document_headers=headers,
                )

    def test_post_publish_check_rejects_csp_for_another_api_origin(self) -> None:
        headers = {
            "referrer-policy": "strict-origin-when-cross-origin",
            "x-content-type-options": "nosniff",
            "x-frame-options": "DENY",
            "content-security-policy": _csp(
                "https://wrong.example.com"
            ),
        }
        with self.assertRaisesRegex(
            CommandError, "content-security-policy"
        ):
            self.verify_runtime(
                {**EXPECTED, "frontendVersion": "v-20260908-deadbeef"},
                document_headers=headers,
            )

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
