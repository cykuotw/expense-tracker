from __future__ import annotations

import hashlib
import json
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any


EXPECTED_SCHEMA_VERSION = 1
EXPECTED_BASELINE_VERSION = 35
MANIFEST_NAME = "manifest.json"
MIGRATION_PATTERN = re.compile(r"^(?P<version>\d{6})_(?P<name>.+)\.(?P<direction>up|down)\.sql$")
ALLOWED_CATEGORIES = frozenset(
    {
        "additive",
        "backfill",
        "constraint_tightening",
        "data_rewrite",
        "rename",
        "drop",
        "type_change",
        "index",
    }
)
ALLOWED_DEPLOYMENTS = frozenset({"online", "maintenance_required"})
ALLOWED_BACKFILL_MODES = frozenset({"none", "bounded"})
ALLOWED_LOCKING_RISKS = frozenset({"low", "medium", "high"})
ALLOWED_APPLICATION_ROLLBACK = frozenset({"compatible", "follow_up_required"})
ALLOWED_SCHEMA_ROLLBACK = frozenset({"not_required", "manual_only", "unsafe"})


class MigrationPolicyError(ValueError):
    pass


@dataclass(frozen=True)
class MigrationEntry:
    version: int
    name: str
    categories: tuple[str, ...]
    deployment: str
    application_rollback: str


@dataclass(frozen=True)
class MigrationFile:
    name: str
    sql: str


@dataclass(frozen=True)
class Manifest:
    baseline_version: int
    migrations: tuple[MigrationEntry, ...]

    def summary(self) -> str:
        online = sum(entry.deployment == "online" for entry in self.migrations)
        maintenance = len(self.migrations) - online
        entries = ";".join(
            f"{entry.version:06d}_{entry.name}[{'+'.join(entry.categories)}:{entry.deployment}]"
            for entry in self.migrations
        ) or "none"
        return (
            f"migration_policy baseline={self.baseline_version:06d} "
            f"entries={len(self.migrations)} online={online} "
            f"maintenance_required={maintenance} release_entries={entries}"
        )

    def has_maintenance_required(self) -> bool:
        return any(entry.deployment != "online" for entry in self.migrations)

    def validate_pending(self, current_version: int, dirty: bool) -> None:
        if dirty:
            raise MigrationPolicyError(
                f"database migration state is dirty: version={current_version:06d} dirty=true"
            )
        blocked = [
            entry
            for entry in self.migrations
            if entry.version > current_version and entry.deployment != "online"
        ]
        if blocked:
            rendered = ", ".join(f"{entry.version:06d}_{entry.name}" for entry in blocked)
            raise MigrationPolicyError(
                f"normal deployment rejects maintenance-required pending migrations: {rendered}"
            )

    def validate_application_rollback(
        self,
        target_version: int,
        current_version: int,
        dirty: bool,
    ) -> None:
        if dirty:
            raise MigrationPolicyError(
                f"database migration state is dirty: version={current_version:06d} dirty=true"
            )
        if target_version > current_version:
            raise MigrationPolicyError(
                "application rollback target requires a newer database schema: "
                f"target={target_version:06d} current={current_version:06d}"
            )
        known_versions = {self.baseline_version, *(entry.version for entry in self.migrations)}
        if target_version not in known_versions:
            raise MigrationPolicyError(
                f"target database version {target_version:06d} is not described by the "
                "repository migration manifest"
            )
        if current_version not in known_versions:
            raise MigrationPolicyError(
                f"current database version {current_version:06d} is not described by the "
                "repository migration manifest"
            )
        blocked = [
            entry
            for entry in self.migrations
            if target_version < entry.version <= current_version
            and entry.application_rollback != "compatible"
        ]
        if blocked:
            rendered = ", ".join(f"{entry.version:06d}_{entry.name}" for entry in blocked)
            raise MigrationPolicyError(
                "application rollback is incompatible with the current schema because of: "
                f"{rendered}"
            )


def validate_repository(repo_root: Path) -> Manifest:
    return validate_directory(repo_root / "backend/cmd/migrate/migrations")


def repository_digest(repo_root: Path) -> str:
    return digest_directory(repo_root / "backend/cmd/migrate/migrations")


def digest_directory(migrations_dir: Path) -> str:
    paths = [migrations_dir / MANIFEST_NAME, *sorted(migrations_dir.glob("*.sql"))]
    missing = [path.name for path in paths if not path.is_file()]
    if missing:
        raise MigrationPolicyError(
            f"migration digest inputs are missing: {', '.join(missing)}"
        )
    digest = hashlib.sha256()
    for path in paths:
        name = path.name.encode()
        content = path.read_bytes()
        digest.update(len(name).to_bytes(4, "big"))
        digest.update(name)
        digest.update(len(content).to_bytes(8, "big"))
        digest.update(content)
    return "sha256:" + digest.hexdigest()


def validate_directory(migrations_dir: Path) -> Manifest:
    manifest_path = migrations_dir / MANIFEST_NAME
    try:
        raw = json.loads(manifest_path.read_text())
    except FileNotFoundError as error:
        raise MigrationPolicyError(f"migration manifest is missing: {manifest_path}") from error
    except json.JSONDecodeError as error:
        raise MigrationPolicyError(
            f"migration manifest is invalid JSON at line {error.lineno}, column {error.colno}"
        ) from error

    root = _object(raw, "manifest")
    _exact_fields(root, {"schemaVersion", "baselineVersion", "migrations"}, "manifest")
    if root.get("schemaVersion") != EXPECTED_SCHEMA_VERSION:
        raise MigrationPolicyError(
            f"migration manifest schemaVersion must be {EXPECTED_SCHEMA_VERSION}"
        )
    if root.get("baselineVersion") != EXPECTED_BASELINE_VERSION:
        raise MigrationPolicyError(
            f"migration manifest baselineVersion must be {EXPECTED_BASELINE_VERSION}"
        )
    raw_entries = root.get("migrations")
    if not isinstance(raw_entries, list):
        raise MigrationPolicyError("migration manifest migrations must be an array")

    files = _migration_files(migrations_dir)
    _validate_pairs(files)
    if EXPECTED_BASELINE_VERSION not in files:
        raise MigrationPolicyError(
            f"migration baseline {EXPECTED_BASELINE_VERSION:06d} SQL files are missing"
        )
    post_baseline_files = {
        version: directions
        for version, directions in files.items()
        if version > EXPECTED_BASELINE_VERSION
    }

    entries: list[MigrationEntry] = []
    seen_versions: set[int] = set()
    previous_version = EXPECTED_BASELINE_VERSION
    for index, raw_entry in enumerate(raw_entries):
        entry = _validate_entry(raw_entry, index)
        if entry.version in seen_versions:
            raise MigrationPolicyError(f"migration version {entry.version:06d} is duplicated")
        if entry.version <= previous_version:
            raise MigrationPolicyError("migration manifest versions must be strictly increasing")
        directions = post_baseline_files.get(entry.version)
        if directions is None:
            raise MigrationPolicyError(
                f"manifest entry {entry.version:06d}_{entry.name} has no matching SQL files"
            )
        file_name = directions["up"].name
        if entry.name != file_name:
            raise MigrationPolicyError(
                f"manifest name {entry.name!r} does not match migration file name {file_name!r} "
                f"for version {entry.version:06d}"
            )
        _validate_sql_consistency(entry, directions["up"].sql)
        entries.append(entry)
        seen_versions.add(entry.version)
        previous_version = entry.version

    missing = sorted(set(post_baseline_files) - seen_versions)
    if missing:
        rendered = ", ".join(f"{version:06d}" for version in missing)
        raise MigrationPolicyError(f"post-baseline migrations are missing manifest entries: {rendered}")

    return Manifest(EXPECTED_BASELINE_VERSION, tuple(entries))


def _migration_files(migrations_dir: Path) -> dict[int, dict[str, MigrationFile]]:
    files: dict[int, dict[str, MigrationFile]] = {}
    for path in sorted(migrations_dir.glob("*.sql")):
        match = MIGRATION_PATTERN.fullmatch(path.name)
        if match is None:
            raise MigrationPolicyError(f"invalid migration filename: {path.name}")
        version = int(match.group("version"))
        direction = match.group("direction")
        if direction in files.setdefault(version, {}):
            raise MigrationPolicyError(
                f"migration version {version:06d} has duplicate {direction} files"
            )
        files[version][direction] = MigrationFile(match.group("name"), path.read_text())
    return files


def _validate_pairs(files: dict[int, dict[str, MigrationFile]]) -> None:
    for version, directions in sorted(files.items()):
        if set(directions) != {"up", "down"}:
            raise MigrationPolicyError(
                f"migration version {version:06d} must have matching up and down SQL files"
            )
        if directions["up"].name != directions["down"].name:
            raise MigrationPolicyError(
                f"migration version {version:06d} has mismatched up and down names"
            )


def _validate_entry(raw: Any, index: int) -> MigrationEntry:
    value = _object(raw, f"migrations[{index}]")
    allowed = {"version", "name", "categories", "deployment", "backfill", "locking", "rollback"}
    unknown = sorted(set(value) - allowed)
    if unknown:
        raise MigrationPolicyError(
            f"migrations[{index}] contains unknown fields: {', '.join(unknown)}"
        )

    version = value.get("version")
    if not isinstance(version, int) or isinstance(version, bool) or version <= EXPECTED_BASELINE_VERSION:
        raise MigrationPolicyError(
            f"migrations[{index}].version must be an integer greater than {EXPECTED_BASELINE_VERSION}"
        )
    name = _nonempty_string(value.get("name"), f"migrations[{index}].name")
    categories_value = value.get("categories")
    if not isinstance(categories_value, list) or not categories_value:
        raise MigrationPolicyError(f"migrations[{index}].categories must be a non-empty array")
    if not all(isinstance(category, str) for category in categories_value):
        raise MigrationPolicyError(f"migrations[{index}].categories must contain strings")
    categories = tuple(categories_value)
    if len(set(categories)) != len(categories):
        raise MigrationPolicyError(f"migrations[{index}].categories must not contain duplicates")
    invalid_categories = sorted(set(categories) - ALLOWED_CATEGORIES)
    if invalid_categories:
        raise MigrationPolicyError(
            f"migrations[{index}].categories contains unsupported values: "
            f"{', '.join(invalid_categories)}"
        )
    deployment = _enum(
        value.get("deployment"), ALLOWED_DEPLOYMENTS, f"migrations[{index}].deployment"
    )

    backfill = _object(value.get("backfill"), f"migrations[{index}].backfill")
    _exact_fields(backfill, {"mode", "resumable", "notes"}, f"migrations[{index}].backfill")
    mode = _enum(backfill.get("mode"), ALLOWED_BACKFILL_MODES, f"migrations[{index}].backfill.mode")
    resumable = backfill.get("resumable")
    if not isinstance(resumable, bool):
        raise MigrationPolicyError(f"migrations[{index}].backfill.resumable must be a boolean")
    _nonempty_string(backfill.get("notes"), f"migrations[{index}].backfill.notes")
    if mode == "bounded" and not resumable:
        raise MigrationPolicyError(f"migrations[{index}] bounded backfill must be resumable")
    if mode == "none" and resumable:
        raise MigrationPolicyError(f"migrations[{index}] backfill mode none cannot be resumable")
    if set(categories).intersection({"backfill", "data_rewrite"}) and mode != "bounded":
        raise MigrationPolicyError(
            f"migrations[{index}] backfill or data_rewrite category requires bounded mode"
        )

    locking = _object(value.get("locking"), f"migrations[{index}].locking")
    _exact_fields(locking, {"risk", "notes"}, f"migrations[{index}].locking")
    _enum(locking.get("risk"), ALLOWED_LOCKING_RISKS, f"migrations[{index}].locking.risk")
    _nonempty_string(locking.get("notes"), f"migrations[{index}].locking.notes")

    rollback = _object(value.get("rollback"), f"migrations[{index}].rollback")
    _exact_fields(
        rollback,
        {"application", "schema", "dataLossRisk", "notes"},
        f"migrations[{index}].rollback",
    )
    application_rollback = _enum(
        rollback.get("application"),
        ALLOWED_APPLICATION_ROLLBACK,
        f"migrations[{index}].rollback.application",
    )
    _enum(
        rollback.get("schema"),
        ALLOWED_SCHEMA_ROLLBACK,
        f"migrations[{index}].rollback.schema",
    )
    if not isinstance(rollback.get("dataLossRisk"), bool):
        raise MigrationPolicyError(f"migrations[{index}].rollback.dataLossRisk must be a boolean")
    _nonempty_string(rollback.get("notes"), f"migrations[{index}].rollback.notes")

    return MigrationEntry(version, name, categories, deployment, application_rollback)


def _validate_sql_consistency(entry: MigrationEntry, sql: str) -> None:
    normalized = re.sub(r"--[^\n]*|/\*.*?\*/", " ", sql, flags=re.DOTALL).upper()
    required_categories = (
        (r"\bDROP\s+", frozenset({"drop"})),
        (r"\bRENAME\b[\s\S]*?\bTO\b", frozenset({"rename"})),
        (r"\bALTER\s+COLUMN\b[\s\S]*?\bTYPE\b", frozenset({"type_change"})),
        (
            r"\b(?:ADD\s+CONSTRAINT|SET\s+NOT\s+NULL|VALIDATE\s+CONSTRAINT)\b",
            frozenset({"constraint_tightening"}),
        ),
        (r"\bCREATE\s+(?:UNIQUE\s+)?INDEX\b", frozenset({"index"})),
        (r"\bTRUNCATE(?:\s+TABLE)?\b", frozenset({"data_rewrite"})),
        (
            r"\b(?:UPDATE|DELETE\s+FROM|INSERT\s+INTO[\s\S]*?SELECT)\b",
            frozenset({"backfill", "data_rewrite"}),
        ),
    )
    category_set = set(entry.categories)
    for pattern, alternatives in required_categories:
        if re.search(pattern, normalized) and category_set.isdisjoint(alternatives):
            rendered = " or ".join(repr(category) for category in sorted(alternatives))
            raise MigrationPolicyError(
                f"migration {entry.version:06d}_{entry.name} SQL requires category {rendered}"
            )


def _object(value: Any, label: str) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise MigrationPolicyError(f"{label} must be an object")
    return value


def _exact_fields(value: dict[str, Any], expected: set[str], label: str) -> None:
    missing = sorted(expected - set(value))
    unknown = sorted(set(value) - expected)
    if missing:
        raise MigrationPolicyError(f"{label} is missing fields: {', '.join(missing)}")
    if unknown:
        raise MigrationPolicyError(f"{label} contains unknown fields: {', '.join(unknown)}")


def _nonempty_string(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise MigrationPolicyError(f"{label} must be a non-empty string")
    return value


def _enum(value: Any, allowed: frozenset[str], label: str) -> str:
    if not isinstance(value, str) or value not in allowed:
        raise MigrationPolicyError(f"{label} must be one of: {', '.join(sorted(allowed))}")
    return value
