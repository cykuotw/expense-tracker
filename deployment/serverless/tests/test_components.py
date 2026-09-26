from __future__ import annotations

import hashlib
import json
import sys
import tempfile
import unittest
import zipfile
from datetime import UTC, datetime
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parents[1]
sys.path.insert(0, str(ROOT))

from backend.artifacts import build
from database import backup
from common.command import CommandError
from frontend.publish import frontend_version, publish, restore, runtime_config, snapshot_descriptor
from release import value_digest
from tests.test_config import ConfigTest


class ComponentTest(unittest.TestCase):
    def test_artifacts_are_deterministic_and_component_owned(self) -> None:
        flags = {}

        def fake_build(_repo: Path, package: str, destination: Path, ldflags: str = "-s -w") -> None:
            flags[package] = ldflags
            destination.write_bytes((package + " binary").encode())

        with tempfile.TemporaryDirectory() as temporary, mock.patch("backend.artifacts._go_build", side_effect=fake_build):
            output = Path(temporary)
            first = build(REPO, output)
            self.assertEqual(set(first), {"worker", "ocr", "bootstrap", "sender", "delivery", "notifier"})
            hashes = {name: hashlib.sha256(path.read_bytes()).hexdigest() for name, path in first.items()}
            second = build(REPO, output)
            self.assertEqual(hashes, {name: hashlib.sha256(path.read_bytes()).hexdigest() for name, path in second.items()})
            with zipfile.ZipFile(first["bootstrap"]) as archive:
                self.assertIn("migrations/manifest.json", archive.namelist())
                self.assertEqual(
                    archive.read("migrations/manifest.json"),
                    (REPO / "backend/cmd/migrate/migrations/manifest.json").read_bytes(),
                )
            self.assertIn("config.BuildMode=release", flags["./backend/cmd/tracker-serverless"])
            for package in (
                "./backend/cmd/bootstrap-serverless",
                "./backend/cmd/ocr-serverless",
                "./backend/cmd/push-sender-serverless",
                "./backend/cmd/push-delivery-serverless",
                "./backend/cmd/error-notifier-serverless",
            ):
                self.assertEqual(flags[package], "-s -w")

    def test_invalid_migration_set_is_rejected_before_building_binaries(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            repo = Path(temporary)
            migrations = repo / "backend/cmd/migrate/migrations"
            migrations.mkdir(parents=True)
            with mock.patch("backend.artifacts._go_build") as go_build:
                with self.assertRaisesRegex(ValueError, "manifest is missing"):
                    build(repo, repo / "build")

        go_build.assert_not_called()

    def test_frontend_runtime_contract(self) -> None:
        case = ConfigTest()
        case.setUp()
        try:
            case.write()
            from config import load
            config = load(case.path, Path("/unrelated/repository"))
            rendered = runtime_config(config, "v-20260907-deadbeef")
            self.assertIn('"apiOrigin": "https://api.example.com"', rendered)
            self.assertIn('"apiPath": "/api/v0"', rendered)
            self.assertIn('"googleOAuthEnabled": true', rendered)
            self.assertIn('"frontendVersion": "v-20260907-deadbeef"', rendered)
        finally:
            case.tearDown()

    def test_frontend_version_is_content_based_and_excludes_runtime_config(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            dist = Path(temporary)
            (dist / "assets").mkdir()
            (dist / "index.html").write_text("index")
            (dist / "assets/app.js").write_text("app")
            (dist / "runtime-config.js").write_text("first runtime config")
            now = datetime(2026, 9, 7, tzinfo=UTC)
            first = frontend_version(dist, now=now)
            (dist / "runtime-config.js").write_text("second runtime config")
            self.assertEqual(first, frontend_version(dist, now=now))
            (dist / "assets/app.js").write_text("changed app")
            self.assertNotEqual(first, frontend_version(dist, now=now))

    def test_frontend_publish_snapshots_mutable_files_and_shares_hashed_assets(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            dist = Path(temporary)
            for filename in (
                "index.html",
                "runtime-config.js",
                "service-worker.js",
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
            release_id = "20260919T120000Z-0123456789ab"
            with (
                mock.patch("frontend.publish.build", return_value=dist),
                mock.patch("frontend.publish.restore") as restore,
                mock.patch("frontend.publish._invalidate") as invalidate,
            ):
                record = publish(client, Path("/repo"), mock.MagicMock(), outputs, release_id)

        self.assertEqual(record["snapshotPrefix"], f"releases/{release_id}/frontend")
        restore.assert_called_once_with(client, outputs, record, invalidate=False)
        invalidate.assert_called_once_with(client, "distribution")

        copy_calls = [
            call.args
            for call in client.call.call_args_list
            if call.args[:2] == ("s3", "cp")
        ]
        destinations = {call[3]: call for call in copy_calls}
        self.assertIn("s3://bucket/assets/index-123.js", destinations)
        self.assertIn("public, max-age=31536000, immutable", destinations["s3://bucket/assets/index-123.js"])
        for filename in ("index.html", "runtime-config.js", "service-worker.js", "manifest.webmanifest", "workbox-example.js"):
            destination = f"s3://bucket/releases/{release_id}/frontend/root/{filename}"
            self.assertIn(destination, destinations)
            self.assertIn("no-cache", destinations[destination])
        self.assertFalse(any("--delete" in call for call in copy_calls))

    def test_frontend_snapshot_descriptor_classifies_only_assets_as_immutable(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            dist = Path(temporary)
            (dist / "assets").mkdir()
            (dist / "index.html").write_text("index")
            (dist / "icon.svg").write_text("icon")
            (dist / "assets/app.js").write_text("app")

            descriptor = snapshot_descriptor(dist, "20260919T120000Z-0123456789ab")

        self.assertEqual(
            {record["path"] for record in descriptor["mutableFiles"]},
            {"icon.svg", "index.html"},
        )
        self.assertEqual(
            {record["path"] for record in descriptor["assets"]},
            {"assets/app.js"},
        )

    def test_frontend_restore_rejects_descriptor_digest_before_copying(self) -> None:
        release_id = "20260919T120000Z-0123456789ab"
        descriptor = {
            "schemaVersion": 1,
            "releaseId": release_id,
            "mutableFiles": [
                {
                    "path": "index.html",
                    "sha256": "1" * 64,
                    "contentType": "text/html; charset=utf-8",
                }
            ],
            "assets": [],
        }
        client = mock.MagicMock()
        client.call.return_value = json.dumps(descriptor)
        frontend = {
            "snapshotPrefix": f"releases/{release_id}/frontend",
            "snapshotManifestKey": f"releases/{release_id}/frontend/snapshot.json",
            "snapshotDigest": "sha256:" + "0" * 64,
        }

        with self.assertRaisesRegex(CommandError, "digest does not match"):
            restore(
                client,
                {"frontend_bucket_name": "bucket", "cloudfront_distribution_id": "distribution"},
                frontend,
            )

        client.json.assert_not_called()

    def test_frontend_restore_verifies_snapshot_and_live_copy(self) -> None:
        release_id = "20260919T120000Z-0123456789ab"
        descriptor = {
            "schemaVersion": 1,
            "releaseId": release_id,
            "mutableFiles": [
                {
                    "path": "index.html",
                    "sha256": "1" * 64,
                    "contentType": "text/html; charset=utf-8",
                }
            ],
            "assets": [
                {
                    "path": "assets/app.js",
                    "sha256": "2" * 64,
                    "contentType": "application/javascript; charset=utf-8",
                }
            ],
        }
        client = mock.MagicMock()
        client.call.return_value = json.dumps(descriptor)
        client.json.side_effect = [
            {"Metadata": {"sha256": "2" * 64}},
            {"Metadata": {"sha256": "1" * 64}},
            {"CopyObjectResult": {}},
            {"Metadata": {"sha256": "1" * 64}},
        ]
        frontend = {
            "snapshotPrefix": f"releases/{release_id}/frontend",
            "snapshotManifestKey": f"releases/{release_id}/frontend/snapshot.json",
            "snapshotDigest": value_digest(descriptor),
        }

        with mock.patch("frontend.publish._invalidate") as invalidate:
            restore(
                client,
                {"frontend_bucket_name": "bucket", "cloudfront_distribution_id": "distribution"},
                frontend,
            )

        copy = client.json.call_args_list[2].args
        self.assertEqual(copy[:2], ("s3api", "copy-object"))
        self.assertIn(
            "releases/20260919T120000Z-0123456789ab/frontend/root/index.html",
            " ".join(copy),
        )
        invalidate.assert_called_once_with(client, "distribution")

    def test_frontend_restore_verifies_all_sources_before_first_live_copy(self) -> None:
        release_id = "20260919T120000Z-0123456789ab"
        descriptor = {
            "schemaVersion": 1,
            "releaseId": release_id,
            "mutableFiles": [
                {
                    "path": "index.html",
                    "sha256": "1" * 64,
                    "contentType": "text/html; charset=utf-8",
                },
                {
                    "path": "runtime-config.js",
                    "sha256": "2" * 64,
                    "contentType": "application/javascript; charset=utf-8",
                },
            ],
            "assets": [],
        }
        client = mock.MagicMock()
        client.call.return_value = json.dumps(descriptor)
        client.json.side_effect = [
            {"Metadata": {"sha256": "1" * 64}},
            {"Metadata": {"sha256": "wrong"}},
        ]
        frontend = {
            "snapshotPrefix": f"releases/{release_id}/frontend",
            "snapshotManifestKey": f"releases/{release_id}/frontend/snapshot.json",
            "snapshotDigest": value_digest(descriptor),
        }

        with self.assertRaisesRegex(CommandError, "digest metadata does not match"):
            restore(
                client,
                {
                    "frontend_bucket_name": "bucket",
                    "cloudfront_distribution_id": "distribution",
                },
                frontend,
            )

        self.assertFalse(
            any(call.args[:2] == ("s3api", "copy-object") for call in client.json.call_args_list)
        )

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

    def test_frontend_uses_one_cloudfront_security_headers_policy(self) -> None:
        source = (ROOT / "infrastructure/tf/frontend.tf").read_text()
        policy = source.split(
            'resource "aws_cloudfront_response_headers_policy" "frontend_security" {',
            maxsplit=1,
        )[1].split("\n}", maxsplit=1)[0]

        self.assertIn("security_headers_config", policy)
        self.assertIn("referrer_policy = \"strict-origin-when-cross-origin\"", policy)
        self.assertIn("content_type_options", policy)
        self.assertIn('frame_option = "DENY"', policy)
        self.assertIn("content_security_policy {", policy)
        self.assertIn("content_security_policy = local.frontend_csp", policy)
        self.assertNotIn("custom_headers_config", policy)
        self.assertNotIn("Content-Security-Policy-Report-Only", policy)
        self.assertNotIn('header = "Referrer-Policy"', policy)
        self.assertEqual(
            source.count(
                'resource "aws_cloudfront_response_headers_policy" "frontend_security" {'
            ),
            1,
        )
        csp = source.split("frontend_csp", maxsplit=1)[1].split(
            "}\nresource", maxsplit=1
        )[0]
        for expected in (
            "default-src 'self'",
            "object-src 'none'",
            "frame-ancestors 'none'",
            "script-src 'self' https://accounts.google.com/gsi/client",
            "connect-src 'self' https://${var.api_hostname} https://api.frankfurter.dev https://accounts.google.com/gsi/",
            "frame-src https://accounts.google.com/gsi/",
            "style-src 'self' 'unsafe-inline' https://accounts.google.com/gsi/style",
            "font-src 'self' https://fonts.gstatic.com",
            "worker-src 'self'",
        ):
            self.assertIn(expected, csp)
        self.assertEqual(csp.count("'unsafe-inline'"), 1)
        self.assertNotIn(
            "script-src 'self' 'unsafe-inline'",
            csp,
        )
        self.assertNotIn("unsafe-eval", csp)

    def test_terraform_does_not_manage_lambda_runtime_updates(self) -> None:
        source = (ROOT / "infrastructure/tf/backend.tf").read_text()
        source += (ROOT / "infrastructure/tf/notifications.tf").read_text()

        self.assertEqual(source.count("source_code_hash,"), 4)
        self.assertEqual(source.count("filename,"), 4)

    def test_notification_network_and_iam_boundaries(self) -> None:
        source = (ROOT / "infrastructure/tf/notifications.tf").read_text()
        sender = source.split('resource "aws_lambda_function" "sender" {')[1].split("\n}", 1)[0]
        delivery = source.split('resource "aws_lambda_function" "delivery" {')[1].split("\n}", 1)[0]
        policy = source.split('resource "aws_iam_role_policy" "sender" {')[1].split("\n}", 1)[0]
        self.assertNotIn("vpc_config", sender)
        self.assertIn("vpc_config", delivery)
        self.assertIn(
            'Action = ["lambda:InvokeFunction"], Resource = var.use_lambda_aliases ? local.delivery_live_arn : aws_lambda_function.delivery.arn',
            policy,
        )
        self.assertNotIn("lambda_network_actions", policy)
        self.assertNotIn("aws_nat_gateway", source)
        self.assertNotIn("aws_vpc_endpoint", source)
        self.assertNotIn("sender_to_push_providers", source)

    def test_monthly_review_reuses_delivery_lambda_with_one_daily_schedule(self) -> None:
        source = (ROOT / "infrastructure/tf/notifications.tf").read_text()
        self.assertEqual(
            source.count('resource "aws_cloudwatch_event_rule" "monthly_review_publisher"'),
            1,
        )
        self.assertIn('schedule_expression = "cron(15 5 * * ? *)"', source)
        self.assertIn(
            "arn   = var.use_lambda_aliases ? local.delivery_live_arn : aws_lambda_function.delivery.arn",
            source,
        )
        self.assertIn('qualifier     = var.use_lambda_aliases ? "live" : null', source)
        self.assertIn('input = jsonencode({ action = "publish_monthly_reviews" })', source)
        self.assertIn('source_arn    = aws_cloudwatch_event_rule.monthly_review_publisher.arn', source)
        self.assertEqual(source.count('resource "aws_lambda_function"'), 2)
        self.assertNotIn("aws_nat_gateway", source)
        self.assertNotIn("aws_vpc_endpoint", source)

    def test_api_route_throttling_covers_anonymous_and_authenticated_mutations(self) -> None:
        source = (ROOT / "infrastructure/tf/api.tf").read_text()

        self.assertIn("throttling_burst_limit = 40", source)
        self.assertIn("throttling_rate_limit  = 20", source)
        self.assertIn('route_key = "POST ${local.api_path}/observability/frontend-render-error"', source)
        self.assertIn("throttling_burst_limit = 2", source)
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

    def test_ocr_runtime_is_isolated_throttled_and_least_privilege(self) -> None:
        api = (ROOT / "infrastructure/tf/api.tf").read_text()
        ocr = (ROOT / "infrastructure/tf/ocr.tf").read_text()

        self.assertIn('route_key = "POST ${local.api_path}/ocr/capabilities"', api)
        self.assertIn('route_key          = "POST ${local.api_path}/ocr/drafts"', api)
        self.assertEqual(api.count("aws_apigatewayv2_route.ocr_"), 2)
        self.assertIn("throttling_burst_limit = 1", api)
        self.assertIn("throttling_rate_limit  = 1", api)
        self.assertIn("memory_size                    = 512", ocr)
        self.assertIn("timeout                        = 12", ocr)
        self.assertIn("reserved_concurrent_executions = 0", ocr)
        self.assertIn('billing_mode = "PAY_PER_REQUEST"', ocr)
        self.assertIn('Action = ["dynamodb:PutItem"]', ocr)
        self.assertIn('Action = ["textract:AnalyzeExpense"]', ocr)
        self.assertNotIn("textract:*", ocr.lower())
        self.assertNotIn("vpc_config", ocr)
        self.assertNotIn("DB_", ocr)
        self.assertNotIn("aws_nat_gateway", ocr)
        self.assertIn('namespace   = "AWS/Textract"', ocr)
        self.assertIn('Operation = "AnalyzeExpense"', ocr)
