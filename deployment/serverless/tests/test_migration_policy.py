from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parents[1]
sys.path.insert(0, str(ROOT))

from backend.migration_policy import (
    MigrationPolicyError,
    digest_directory,
    validate_directory,
    validate_repository,
)


def entry(*, deployment: str = "online", categories: list[str] | None = None) -> dict[str, object]:
    return {
        "version": 36,
        "name": "example_expansion",
        "categories": categories or ["additive"],
        "deployment": deployment,
        "backfill": {
            "mode": "none",
            "resumable": False,
            "notes": "No existing rows are rewritten.",
        },
        "locking": {
            "risk": "low",
            "notes": "Metadata-only addition with bounded lock acquisition.",
        },
        "rollback": {
            "application": "compatible",
            "schema": "not_required",
            "dataLossRisk": False,
            "notes": "The previous application ignores the added column.",
        },
    }


class MigrationPolicyTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.migrations = Path(self.temporary.name)
        self.write_sql(35, "baseline", "SELECT 1;")

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def write_sql(self, version: int, name: str, up: str) -> None:
        prefix = f"{version:06d}_{name}"
        (self.migrations / f"{prefix}.up.sql").write_text(up)
        (self.migrations / f"{prefix}.down.sql").write_text("SELECT 1;")

    def write_manifest(self, migrations: list[dict[str, object]]) -> None:
        (self.migrations / "manifest.json").write_text(
            json.dumps(
                {
                    "schemaVersion": 1,
                    "baselineVersion": 35,
                    "migrations": migrations,
                }
            )
        )

    def test_repository_manifest_matches_current_migrations(self) -> None:
        manifest = validate_repository(REPO)
        self.assertEqual(manifest.baseline_version, 35)
        self.assertEqual(len(manifest.migrations), 2)
        migration = manifest.migrations[0]
        self.assertEqual(migration.version, 36)
        self.assertEqual(migration.name, "add_multi_currency_accounting")
        self.assertEqual(migration.deployment, "online")
        feature_grants = manifest.migrations[1]
        self.assertEqual(feature_grants.version, 37)
        self.assertEqual(feature_grants.name, "add_user_feature_grants")
        self.assertEqual(feature_grants.deployment, "online")

    def test_shared_cross_language_fixtures(self) -> None:
        fixtures = REPO / "backend/internal/migrationpolicy/testdata"
        manifest = validate_directory(fixtures / "valid")
        manifest.validate_pending(35, False)
        with self.assertRaisesRegex(MigrationPolicyError, "requires category 'drop'"):
            validate_directory(fixtures / "invalid_destructive")

    def test_valid_online_expansion(self) -> None:
        self.write_sql(36, "example_expansion", "ALTER TABLE expense ADD COLUMN note TEXT;")
        self.write_manifest([entry()])

        manifest = validate_directory(self.migrations)

        self.assertIn("baseline=000035", manifest.summary())
        self.assertIn("000036_example_expansion[additive:online]", manifest.summary())
        manifest.validate_pending(35, False)

    def test_missing_post_baseline_entry_is_rejected(self) -> None:
        self.write_sql(36, "example_expansion", "SELECT 1;")
        self.write_manifest([])

        with self.assertRaisesRegex(MigrationPolicyError, "missing manifest entries: 000036"):
            validate_directory(self.migrations)

    def test_destructive_sql_requires_matching_category(self) -> None:
        self.write_sql(36, "example_expansion", "ALTER TABLE expense DROP COLUMN split_rule;")
        self.write_manifest([entry()])

        with self.assertRaisesRegex(MigrationPolicyError, "requires category 'drop'"):
            validate_directory(self.migrations)

    def test_destructive_sql_variants_require_matching_category(self) -> None:
        cases = {
            "truncate": ("TRUNCATE TABLE expense;", "data_rewrite"),
            "drop view": ("DROP MATERIALIZED VIEW expense_summary;", "drop"),
            "rename table": ("ALTER TABLE expense RENAME TO expense_legacy;", "rename"),
            "rename column without keyword": (
                "ALTER TABLE expense RENAME note TO description;",
                "rename",
            ),
            "rename index": (
                "ALTER INDEX expense_idx RENAME TO expense_legacy_idx;",
                "rename",
            ),
        }
        for name, (sql, category) in cases.items():
            with self.subTest(name=name):
                self.write_sql(36, "example_expansion", sql)
                self.write_manifest([entry()])

                with self.assertRaisesRegex(
                    MigrationPolicyError,
                    f"requires category '{category}'",
                ):
                    validate_directory(self.migrations)

    def test_bounded_backfill_must_be_resumable(self) -> None:
        value = entry(categories=["backfill"])
        value["backfill"] = {
            "mode": "bounded",
            "resumable": False,
            "notes": "Rows are processed in finite batches.",
        }
        self.write_sql(36, "example_expansion", "UPDATE expense SET note = '';")
        self.write_manifest([value])

        with self.assertRaisesRegex(MigrationPolicyError, "bounded backfill must be resumable"):
            validate_directory(self.migrations)

    def test_data_rewrite_must_declare_bounded_backfill(self) -> None:
        value = entry(categories=["data_rewrite"])
        self.write_sql(36, "example_expansion", "UPDATE expense SET note = '';")
        self.write_manifest([value])

        with self.assertRaisesRegex(MigrationPolicyError, "requires bounded mode"):
            validate_directory(self.migrations)

    def test_normal_deploy_rejects_only_pending_maintenance_entry(self) -> None:
        self.write_sql(36, "example_expansion", "ALTER TABLE expense ADD COLUMN note TEXT;")
        self.write_manifest([entry(deployment="maintenance_required")])

        manifest = validate_directory(self.migrations)

        with self.assertRaisesRegex(MigrationPolicyError, "normal deployment rejects"):
            manifest.validate_pending(35, False)
        manifest.validate_pending(36, False)

    def test_dirty_database_state_is_rejected(self) -> None:
        self.write_sql(36, "example_expansion", "ALTER TABLE expense ADD COLUMN note TEXT;")
        self.write_manifest([entry()])

        manifest = validate_directory(self.migrations)

        with self.assertRaisesRegex(MigrationPolicyError, "version=000035 dirty=true"):
            manifest.validate_pending(35, True)

    def test_application_rollback_requires_compatible_intervening_migrations(self) -> None:
        self.write_sql(36, "example_expansion", "ALTER TABLE expense ADD COLUMN note TEXT;")
        compatible = entry()
        self.write_manifest([compatible])
        manifest = validate_directory(self.migrations)
        manifest.validate_application_rollback(35, 36, False)

        incompatible = entry()
        rollback = incompatible["rollback"]
        assert isinstance(rollback, dict)
        rollback["application"] = "follow_up_required"
        self.write_manifest([incompatible])
        manifest = validate_directory(self.migrations)
        with self.assertRaisesRegex(MigrationPolicyError, "application rollback is incompatible"):
            manifest.validate_application_rollback(35, 36, False)

    def test_application_rollback_rejects_unknown_or_dirty_schema(self) -> None:
        self.write_manifest([])
        manifest = validate_directory(self.migrations)

        with self.assertRaisesRegex(MigrationPolicyError, "newer database schema"):
            manifest.validate_application_rollback(36, 35, False)
        with self.assertRaisesRegex(MigrationPolicyError, "target database version.*not described"):
            manifest.validate_application_rollback(34, 35, False)
        with self.assertRaisesRegex(MigrationPolicyError, "current database version.*not described"):
            manifest.validate_application_rollback(35, 36, False)
        with self.assertRaisesRegex(MigrationPolicyError, "dirty=true"):
            manifest.validate_application_rollback(35, 35, True)

    def test_migration_digest_is_stable_and_covers_sql_content(self) -> None:
        self.write_manifest([])
        first = digest_directory(self.migrations)
        self.assertEqual(first, digest_directory(self.migrations))

        baseline = self.migrations / "000035_baseline.up.sql"
        baseline.write_text("SELECT 2;")
        self.assertNotEqual(first, digest_directory(self.migrations))

    def test_manifest_name_must_match_sql_files(self) -> None:
        self.write_sql(36, "actual_name", "SELECT 1;")
        value = entry()
        value["name"] = "different_name"
        self.write_manifest([value])

        with self.assertRaisesRegex(MigrationPolicyError, "does not match migration file name"):
            validate_directory(self.migrations)

    def test_unknown_classification_is_rejected(self) -> None:
        self.write_sql(36, "example_expansion", "SELECT 1;")
        value = entry(categories=["unknown"])
        self.write_manifest([value])

        with self.assertRaisesRegex(MigrationPolicyError, "unsupported values: unknown"):
            validate_directory(self.migrations)

    def test_incomplete_nested_metadata_is_rejected(self) -> None:
        self.write_sql(36, "example_expansion", "SELECT 1;")
        value = entry()
        rollback = value["rollback"]
        assert isinstance(rollback, dict)
        del rollback["notes"]
        self.write_manifest([value])

        with self.assertRaisesRegex(MigrationPolicyError, "rollback is missing fields: notes"):
            validate_directory(self.migrations)

    def test_manifest_versions_must_be_unique_and_increasing(self) -> None:
        self.write_sql(36, "example_expansion", "SELECT 1;")
        self.write_sql(37, "second_expansion", "SELECT 1;")
        first = entry()
        second = entry()
        second["version"] = 37
        second["name"] = "second_expansion"
        self.write_manifest([second, first])

        with self.assertRaisesRegex(MigrationPolicyError, "strictly increasing"):
            validate_directory(self.migrations)

        self.write_manifest([first, first])
        with self.assertRaisesRegex(MigrationPolicyError, "duplicated"):
            validate_directory(self.migrations)

    def test_orphan_manifest_entry_and_unpaired_sql_are_rejected(self) -> None:
        self.write_manifest([entry()])
        with self.assertRaisesRegex(MigrationPolicyError, "has no matching SQL files"):
            validate_directory(self.migrations)

        self.write_sql(36, "example_expansion", "SELECT 1;")
        (self.migrations / "000036_example_expansion.down.sql").unlink()
        with self.assertRaisesRegex(MigrationPolicyError, "matching up and down SQL files"):
            validate_directory(self.migrations)


if __name__ == "__main__":
    unittest.main()
