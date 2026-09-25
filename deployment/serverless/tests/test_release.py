from __future__ import annotations

import json
import sys
import unittest
from datetime import UTC, datetime
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import release
from common.command import CommandError


def config() -> SimpleNamespace:
    return SimpleNamespace(
        deployment=SimpleNamespace(name_prefix="expense", environment="prod"),
        database=SimpleNamespace(
            admin_password="admin-secret",
            migration_password="migration-secret",
            runtime_password="runtime-secret",
        ),
        backend=SimpleNamespace(
            jwt_secret="jwt-secret",
            refresh_jwt_secret="refresh-secret",
            web_push_vapid_public_key="public-key",
            web_push_vapid_private_key="private-key",
            web_push_vapid_subject="mailto:ops@example.com",
        ),
        observability=SimpleNamespace(discord_webhook_url=None),
        first_admin=None,
    )


def function(name: str, version: int) -> dict[str, str]:
    return {
        "functionName": name,
        "version": str(version),
        "qualifiedArn": f"arn:aws:lambda:ca-central-1:123:function:{name}:{version}",
        "codeSha256": f"code-{name}-{version}",
    }


def manifest(release_id: str = "20260919T184500Z-a1b2c3d4e5f6") -> dict[str, object]:
    return {
        "schemaVersion": 1,
        "releaseId": release_id,
        "commitSha": "a1b2c3d4e5f6" + "0" * 28,
        "createdAt": "2026-09-19T18:45:00Z",
        "status": "successful",
        "operation": "deploy",
        "changedScopes": ["all"],
        "sourceReleaseId": None,
        "database": {
            "migrationVersion": 35,
            "dirty": False,
            "migrationManifestDigest": "sha256:" + "1" * 64,
        },
        "backend": {
            "worker": function("worker", 10),
            "bootstrap": function("bootstrap", 11),
            "sender": function("sender", 12),
            "delivery": function("delivery", 13),
            "errorNotifier": None,
        },
        "frontend": {
            "snapshotPrefix": f"releases/{release_id}/frontend",
            "snapshotManifestKey": f"releases/{release_id}/frontend/snapshot.json",
            "snapshotDigest": "sha256:" + "2" * 64,
        },
    }


class ReleaseTest(unittest.TestCase):
    def test_repository_identity_requires_clean_worktree_except_ignored_files(self) -> None:
        with mock.patch.object(
            release,
            "run",
            side_effect=[
                SimpleNamespace(stdout=""),
                SimpleNamespace(stdout="a1b2c3d4e5f6" + "0" * 28 + "\n"),
            ],
        ) as command:
            release_id, commit, created = release.repository_release_identity(
                Path("/repo"),
                now=datetime(2026, 9, 19, 18, 45, tzinfo=UTC),
            )

        self.assertEqual(release_id, "20260919T184500Z-a1b2c3d4e5f6")
        self.assertEqual(commit, "a1b2c3d4e5f6" + "0" * 28)
        self.assertEqual(created, "2026-09-19T18:45:00Z")
        self.assertEqual(
            command.call_args_list[0].args[0],
            ["git", "status", "--porcelain", "--untracked-files=normal"],
        )

        with mock.patch.object(
            release,
            "run",
            return_value=SimpleNamespace(stdout="?? new-source.py\n"),
        ):
            with self.assertRaisesRegex(CommandError, "clean worktree"):
                release.repository_release_identity(Path("/repo"))

    def test_manifest_is_strict_and_below_standard_parameter_limit(self) -> None:
        value = manifest()
        self.assertIs(release.validate_manifest(value), value)
        self.assertLessEqual(len(release.canonical_json(value).encode()), 4096)

        value["database"]["dirty"] = True  # type: ignore[index]
        with self.assertRaisesRegex(CommandError, "must not be dirty"):
            release.validate_manifest(value)

    def test_manifest_accepts_versioned_ocr_and_legacy_backend_sets(self) -> None:
        legacy = manifest()
        self.assertIs(release.validate_manifest(legacy), legacy)

        current = manifest()
        backend = current["backend"]
        assert isinstance(backend, dict)
        backend["ocr"] = function("ocr", 14)
        self.assertIs(release.validate_manifest(current), current)

    def test_manifest_accepts_carried_snapshot_and_rejects_mismatched_path(self) -> None:
        value = manifest()
        frontend = value["frontend"]
        assert isinstance(frontend, dict)
        frontend["snapshotPrefix"] = (
            "releases/20260918T184500Z-b1b2c3d4e5f6/frontend"
        )
        frontend["snapshotManifestKey"] = (
            "releases/20260918T184500Z-b1b2c3d4e5f6/frontend/snapshot.json"
        )

        self.assertIs(release.validate_manifest(value), value)

        frontend["snapshotManifestKey"] = "releases/wrong/frontend/snapshot.json"
        with self.assertRaisesRegex(CommandError, "snapshot paths are invalid"):
            release.validate_manifest(value)

    def test_store_explicitly_uses_standard_tier_and_rejects_secrets(self) -> None:
        client = mock.MagicMock()
        store = release.ReleaseStore(client, config())
        value = manifest()
        digest = store.put_release(value)

        self.assertEqual(digest, release.value_digest(value))
        client.put_parameter.assert_called_once_with(
            "/expense/prod/deploy/releases/20260919T184500Z-a1b2c3d4e5f6",
            release.canonical_json(value),
            overwrite=False,
        )

        value["backend"]["worker"]["codeSha256"] = "jwt-secret"  # type: ignore[index]
        with self.assertRaisesRegex(CommandError, "protected configuration"):
            store.put_release(value)

    def test_current_pointer_preserves_previous_known_good(self) -> None:
        client = mock.MagicMock()
        client.get_parameter.return_value = None
        store = release.ReleaseStore(client, config())
        first = manifest()
        first_digest = release.value_digest(first)
        pointer = store.set_current(first, first_digest)
        self.assertIsNone(pointer["previousReleaseId"])

        second = manifest("20260920T184500Z-b1b2c3d4e5f6")
        second["commitSha"] = "b1b2c3d4e5f6" + "0" * 28
        client.get_parameter.return_value = json.dumps(pointer)
        with mock.patch.object(store, "get_release", return_value=first):
            second_pointer = store.set_current(second, release.value_digest(second))

        self.assertEqual(
            second_pointer["previousReleaseId"],
            "20260919T184500Z-a1b2c3d4e5f6",
        )

    def test_history_fails_closed_on_malformed_manifest(self) -> None:
        client = mock.MagicMock()
        client.parameters_by_path.return_value = [{"Value": "{}"}]
        store = release.ReleaseStore(client, config())
        with self.assertRaisesRegex(CommandError, "fields are invalid"):
            store.list_releases()

    def test_destroy_metadata_cleanup_is_scoped_and_deletes_pointer_last(self) -> None:
        client = mock.MagicMock()
        store = release.ReleaseStore(client, config())
        release_path = (
            "/expense/prod/deploy/releases/20260919T184500Z-a1b2c3d4e5f6"
        )
        candidate_path = (
            "/expense/prod/deploy/candidates/20260920T184500Z-b1b2c3d4e5f6"
        )
        client.parameters_by_path.return_value = [
            {"Name": store.current_path},
            {"Name": release_path},
            {"Name": candidate_path},
        ]

        store.delete_all_metadata()

        self.assertEqual(
            client.delete_parameter.call_args_list,
            [
                mock.call(candidate_path),
                mock.call(release_path),
                mock.call(store.current_path),
            ],
        )

        client.parameters_by_path.return_value = [
            {"Name": "/expense/prod/deploy/unrelated/value"}
        ]
        with self.assertRaisesRegex(CommandError, "unexpected parameter"):
            store.delete_all_metadata()

    def test_retention_caps_history_but_always_protects_current_and_previous(self) -> None:
        releases = []
        for day in range(1, 7):
            release_id = f"202609{day:02d}T120000Z-{day:012x}"
            value = manifest(release_id)
            value["createdAt"] = f"2026-09-{day:02d}T12:00:00Z"
            releases.append(value)
        pointer = {
            "currentReleaseId": releases[0]["releaseId"],
            "previousReleaseId": releases[1]["releaseId"],
        }

        deletions = release.retention_deletions(
            releases,
            pointer,
            now=datetime(2026, 9, 7, tzinfo=UTC),
        )

        deleted_ids = {item["releaseId"] for item in deletions}
        self.assertNotIn(releases[0]["releaseId"], deleted_ids)
        self.assertNotIn(releases[1]["releaseId"], deleted_ids)
        self.assertEqual(len(releases) - len(deletions), 5)

    def test_retention_expires_old_unprotected_releases_and_candidates(self) -> None:
        current = manifest("20260901T120000Z-000000000001")
        current["createdAt"] = "2026-09-01T12:00:00Z"
        old = manifest("20260902T120000Z-000000000002")
        old["createdAt"] = "2026-09-02T12:00:00Z"
        pointer = {"currentReleaseId": current["releaseId"], "previousReleaseId": None}

        deletions = release.retention_deletions(
            [current, old],
            pointer,
            now=datetime(2026, 11, 1, tzinfo=UTC),
        )
        stale = release.stale_candidates(
            [
                {"releaseId": old["releaseId"], "createdAt": "2026-10-30T00:00:00Z"},
                {"releaseId": current["releaseId"], "createdAt": "2026-10-31T18:00:00Z"},
            ],
            now=datetime(2026, 11, 1, tzinfo=UTC),
        )

        self.assertEqual([item["releaseId"] for item in deletions], [old["releaseId"]])
        self.assertEqual([item["releaseId"] for item in stale], [old["releaseId"]])


if __name__ == "__main__":
    unittest.main()
