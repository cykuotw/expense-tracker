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
from database import backup
from frontend.publish import publish, runtime_config
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

    def test_frontend_publish_uses_safe_cache_headers_for_pwa_entrypoints(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            dist = Path(temporary)
            for filename in (
                "index.html",
                "runtime-config.js",
                "sw.js",
                "manifest.webmanifest",
                "workbox-example.js",
                "assets/index-123.js",
            ):
                path = dist / filename
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("test")

            client = mock.MagicMock()
            client.json.return_value = {"Invalidation": {"Id": "invalidation"}}
            outputs = {
                "frontend_bucket_name": "bucket",
                "cloudfront_distribution_id": "distribution",
            }
            with mock.patch("frontend.publish.build", return_value=dist):
                publish(client, Path("/repo"), mock.MagicMock(), outputs)

        sync_call = client.call.call_args_list[0].args
        self.assertEqual(sync_call[:4], ("s3", "sync", f"{dist}/", "s3://bucket/"))
        self.assertIn("public, max-age=31536000, immutable", sync_call)
        for filename in (
            "index.html",
            "runtime-config.js",
            "sw.js",
            "manifest.webmanifest",
            "workbox-example.js",
        ):
            self.assertIn("--exclude", sync_call)
            self.assertIn(filename, sync_call)

        copy_calls = [
            call.args
            for call in client.call.call_args_list
            if call.args[:2] == ("s3", "cp")
        ]
        copied = {Path(call[2]).name: call for call in copy_calls}
        self.assertEqual(
            set(copied),
            {
                "index.html",
                "runtime-config.js",
                "sw.js",
                "manifest.webmanifest",
                "workbox-example.js",
            },
        )
        for call in copied.values():
            self.assertIn("no-cache", call)

    def test_serverless_implementation_never_references_serverful(self) -> None:
        forbidden = "deployment/" + "serverful"
        for path in ROOT.rglob("*"):
            if path.is_file() and path.suffix in {".py", ".tf"}:
                self.assertNotIn(forbidden, path.read_text(), str(path))

    def test_database_setup_avoids_pipefail_sigpipe(self) -> None:
        source = (ROOT / "database/setup.py").read_text()
        self.assertNotIn("rpm -ql postgresql16-contrib | grep -E", source)
        self.assertIn("pgcrypto_control=", source)

    def test_postgres_backup_is_encrypted_retained_and_least_privilege(self) -> None:
        source = (ROOT / "infrastructure/tf/backup.tf").read_text()

        self.assertIn('sse_algorithm = "AES256"', source)
        self.assertIn('days = 90', source)
        self.assertIn('prefix = "daily/"', source)
        self.assertIn('"aws:SecureTransport" = "false"', source)
        self.assertIn('"${aws_s3_bucket.postgres_backup.arn}/daily/*"', source)
        self.assertNotIn('Resource = "${aws_s3_bucket.postgres_backup.arn}/*"', source)

    def test_postgres_backup_and_restore_scripts_do_not_embed_credentials(self) -> None:
        source = (ROOT / "database/backup.py").read_text()

        self.assertIn("pg_dump --dbname", source)
        self.assertIn("--format=custom", source)
        self.assertIn("install -d -o postgres -g postgres -m 0700 /var/lib/expense-tracker-backups", source)
        self.assertIn("chown postgres:postgres /var/lib/expense-tracker-backups", source)
        self.assertIn('chown postgres:postgres "$archive"', source)
        self.assertIn("dnf install -y awscli2", source)
        self.assertIn("REMOTE_BACKUP_DIAGNOSTIC", source)
        self.assertNotIn("bash -x /usr/local/sbin/expense-tracker-postgres-backup", source)
        self.assertIn("OnCalendar=*-*-* $BACKUP_TIME $BACKUP_TIMEZONE", source)
        self.assertIn("expense-tracker-postgres-backup.timer <<TIMER", source)
        self.assertNotIn("expense-tracker-postgres-backup.timer <<'TIMER'", source)
        self.assertIn("systemd-analyze calendar", source)
        self.assertIn("pg_restore --exit-on-error", source)
        self.assertGreaterEqual(source.count('chown postgres:postgres "$archive"'), 2)
        self.assertIn('restore_listing="$(sudo -u postgres pg_restore --list "$archive")"', source)
        self.assertNotIn('pg_restore --list "$archive" | grep', source)
        self.assertIn('status=success\\ncompleted_at=%s\\nbackup_key=%s\\n', source)
        self.assertIn('aws s3 rm "s3://${BACKUP_BUCKET}/verification/restore-failure.txt"', source)
        self.assertIn("expense_tracker_restore_verification", source)
        self.assertIn("restore-failure.txt", source)
        self.assertNotIn("admin_password", source)
        self.assertNotIn("runtime_password", source)

    def test_backup_helpers_require_the_expected_temporary_outputs(self) -> None:
        config = mock.MagicMock()
        config.database.name = "expense_tracker"
        config.aws.region = "ca-central-1"
        with mock.patch.object(backup, "_run_remote") as run_remote:
            backup.configure(
                config,
                {"database_temporary_public_ipv4": "203.0.113.10", "postgres_backup_bucket_name": "backup-bucket"},
            )
        self.assertEqual(run_remote.call_args.args[1], "203.0.113.10")
        self.assertEqual(run_remote.call_args.args[3]["BACKUP_DATABASE"], "expense_tracker")
        self.assertEqual(run_remote.call_args.args[3]["BACKUP_TIME"], config.backup.time)
        self.assertEqual(run_remote.call_args.args[3]["BACKUP_TIMEZONE"], config.backup.timezone)
        with self.assertRaisesRegex(Exception, "temporary database access"):
            backup.configure(config, {"postgres_backup_bucket_name": "backup-bucket"})

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

    def test_api_route_throttling_covers_anonymous_and_authenticated_mutations(self) -> None:
        source = (ROOT / "infrastructure/tf/api.tf").read_text()

        self.assertIn("throttling_burst_limit = 40", source)
        self.assertIn("throttling_rate_limit  = 20", source)
        self.assertIn("resource \"aws_apigatewayv2_route\" \"authenticated_mutation\"", source)
        self.assertIn("for_each = local.authenticated_mutation_routes", source)
        self.assertIn("dynamic \"route_settings\"", source)
        for route_key in (
            "POST ${local.api_path}/create_expense",
            "PUT ${local.api_path}/expense/{expenseId}",
            "POST ${local.api_path}/create_group",
            "PATCH ${local.api_path}/account",
            "POST ${local.api_path}/invitations",
            "POST ${local.api_path}/userInfo",
        ):
            self.assertIn(route_key, source)

    def test_serverful_reference_applies_the_mutation_limit_without_limiting_reads(self) -> None:
        source = (ROOT.parents[0] / "serverful/edge/nginx/expense-tracker.conf").read_text()

        self.assertIn("zone=expense_tracker_mutations:10m rate=5r/s", source)
        self.assertIn("$request_method:$uri", source)
        self.assertIn("limit_req zone=expense_tracker_mutations burst=10 nodelay;", source)
        self.assertIn("limit_req zone=expense_tracker_reads burst=40 nodelay;", source)
        self.assertIn("Retry-After 60", source)
