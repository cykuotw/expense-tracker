from __future__ import annotations

import hashlib
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parents[1]
sys.path.insert(0, str(ROOT))

from backend.artifacts import build
from frontend.publish import runtime_config
from tests.test_config import ConfigTest


class ComponentTest(unittest.TestCase):
    def test_artifacts_are_deterministic_and_component_owned(self) -> None:
        def fake_build(_repo: Path, package: str, destination: Path) -> None:
            destination.write_bytes((package + " binary").encode())

        with tempfile.TemporaryDirectory() as temporary, mock.patch("backend.artifacts._go_build", side_effect=fake_build):
            output = Path(temporary)
            first = build(REPO, output)
            hashes = {name: hashlib.sha256(path.read_bytes()).hexdigest() for name, path in first.items()}
            second = build(REPO, output)
            self.assertEqual(hashes, {name: hashlib.sha256(path.read_bytes()).hexdigest() for name, path in second.items()})

    def test_frontend_runtime_contract(self) -> None:
        case = ConfigTest()
        case.setUp()
        try:
            case.write()
            from config import load
            config = load(case.path, Path("/unrelated/repository"))
            rendered = runtime_config(config)
            self.assertIn('"apiOrigin": "https://api.example.com"', rendered)
            self.assertIn('"apiPath": "/api/v0"', rendered)
            self.assertIn('"googleOAuthEnabled": true', rendered)
        finally:
            case.tearDown()

    def test_serverless_implementation_never_references_serverful(self) -> None:
        forbidden = "deployment/" + "serverful"
        for path in ROOT.rglob("*"):
            if path.is_file() and path.suffix in {".py", ".tf"}:
                self.assertNotIn(forbidden, path.read_text(), str(path))

    def test_database_setup_avoids_pipefail_sigpipe(self) -> None:
        source = (ROOT / "database/setup.py").read_text()
        self.assertNotIn("rpm -ql postgresql16-contrib | grep -E", source)
        self.assertIn("pgcrypto_control=", source)

    def test_frontend_referrer_policy_uses_cloudfront_security_header(self) -> None:
        source = (ROOT / "infrastructure/tf/frontend.tf").read_text()
        policy = source.split(
            'resource "aws_cloudfront_response_headers_policy" "frontend_security" {',
            maxsplit=1,
        )[1].split("\n}", maxsplit=1)[0]

        self.assertIn("security_headers_config", policy)
        self.assertIn("referrer_policy = \"strict-origin-when-cross-origin\"", policy)
        self.assertNotIn("custom_headers_config", policy)
        self.assertNotIn('header = "Referrer-Policy"', policy)

    def test_terraform_does_not_manage_lambda_runtime_updates(self) -> None:
        source = (ROOT / "infrastructure/tf/backend.tf").read_text()

        self.assertEqual(source.count("source_code_hash,"), 2)
        self.assertEqual(source.count("filename,"), 2)
