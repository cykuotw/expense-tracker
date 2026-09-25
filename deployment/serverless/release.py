from __future__ import annotations

import hashlib
import json
import re
from datetime import UTC, datetime, timedelta
from typing import Any

from common.aws import AWSClient
from common.command import CommandError, run
from config import Config


SCHEMA_VERSION = 1
MAX_PARAMETER_BYTES = 4096
MAX_SUCCESSFUL_RELEASES = 5
MAX_RELEASE_AGE_DAYS = 30
FAILED_CANDIDATE_MAX_AGE_HOURS = 24
RELEASE_ID_PATTERN = re.compile(r"^\d{8}T\d{6}Z-[0-9a-f]{12}$")
SHA256_PATTERN = re.compile(r"^sha256:[0-9a-f]{64}$")
COMMIT_PATTERN = re.compile(r"^[0-9a-f]{40}$")
VERSION_PATTERN = re.compile(r"^[1-9][0-9]*$")
ALLOWED_SCOPES = frozenset({"migrations", "backend", "frontend", "all"})
LEGACY_FUNCTION_KEYS = ("worker", "bootstrap", "sender", "delivery", "errorNotifier")
FUNCTION_KEYS = ("worker", "ocr", "bootstrap", "sender", "delivery", "errorNotifier")


def canonical_json(value: Any) -> str:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True)


def value_digest(value: Any) -> str:
    return "sha256:" + hashlib.sha256(canonical_json(value).encode()).hexdigest()


def utc_text(value: datetime | None = None) -> str:
    current = value or datetime.now(UTC)
    if current.tzinfo is None:
        current = current.replace(tzinfo=UTC)
    return current.astimezone(UTC).isoformat(timespec="seconds").replace("+00:00", "Z")


def repository_release_identity(
    repo_root: Any,
    *,
    now: datetime | None = None,
) -> tuple[str, str, str]:
    dirty = run(
        ["git", "status", "--porcelain", "--untracked-files=normal"],
        cwd=repo_root,
    ).stdout.strip()
    if dirty:
        raise CommandError(
            "release deployment requires a clean worktree; ignored files are allowed"
        )
    commit = run(["git", "rev-parse", "HEAD"], cwd=repo_root).stdout.strip().lower()
    if not COMMIT_PATTERN.fullmatch(commit):
        raise CommandError("unable to resolve a full lowercase Git commit SHA")
    created_at = utc_text(now)
    compact = created_at.replace("-", "").replace(":", "")
    release_id = f"{compact}-{commit[:12]}"
    if not RELEASE_ID_PATTERN.fullmatch(release_id):
        raise CommandError("generated release ID is invalid")
    return release_id, commit, created_at


def _object(value: Any, label: str) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise CommandError(f"{label} must be an object")
    return value


def _exact_fields(value: dict[str, Any], expected: set[str], label: str) -> None:
    if set(value) != expected:
        raise CommandError(f"{label} fields are invalid")


def _text(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value:
        raise CommandError(f"{label} must be a non-empty string")
    return value


def _timestamp(value: Any, label: str) -> str:
    text = _text(value, label)
    try:
        parsed = datetime.fromisoformat(text.replace("Z", "+00:00"))
    except ValueError as error:
        raise CommandError(f"{label} must be an ISO-8601 timestamp") from error
    if parsed.tzinfo is None:
        raise CommandError(f"{label} must include a timezone")
    return text


def _function_record(value: Any, label: str) -> dict[str, str]:
    record = _object(value, label)
    _exact_fields(
        record,
        {"functionName", "version", "qualifiedArn", "codeSha256"},
        label,
    )
    function_name = _text(record["functionName"], f"{label}.functionName")
    version = _text(record["version"], f"{label}.version")
    qualified_arn = _text(record["qualifiedArn"], f"{label}.qualifiedArn")
    code_hash = _text(record["codeSha256"], f"{label}.codeSha256")
    if not VERSION_PATTERN.fullmatch(version):
        raise CommandError(f"{label}.version must be a numbered Lambda version")
    if not qualified_arn.endswith(f":{version}") or f":function:{function_name}:" not in qualified_arn:
        raise CommandError(f"{label}.qualifiedArn does not match the function and version")
    return {
        "functionName": function_name,
        "version": version,
        "qualifiedArn": qualified_arn,
        "codeSha256": code_hash,
    }


def validate_manifest(value: Any) -> dict[str, Any]:
    manifest = _object(value, "release manifest")
    _exact_fields(
        manifest,
        {
            "schemaVersion",
            "releaseId",
            "commitSha",
            "createdAt",
            "status",
            "operation",
            "changedScopes",
            "sourceReleaseId",
            "database",
            "backend",
            "frontend",
        },
        "release manifest",
    )
    if manifest["schemaVersion"] != SCHEMA_VERSION:
        raise CommandError("release manifest schemaVersion is unsupported")
    release_id = _text(manifest["releaseId"], "releaseId")
    if not RELEASE_ID_PATTERN.fullmatch(release_id):
        raise CommandError("releaseId is invalid")
    commit = _text(manifest["commitSha"], "commitSha")
    if not COMMIT_PATTERN.fullmatch(commit):
        raise CommandError("commitSha must be a full lowercase Git SHA")
    _timestamp(manifest["createdAt"], "createdAt")
    if manifest["status"] != "successful":
        raise CommandError("only successful releases may be stored in release history")
    if manifest["operation"] not in {"deploy", "rollback", "promote", "adopt"}:
        raise CommandError("release manifest operation is invalid")
    scopes = manifest["changedScopes"]
    if (
        not isinstance(scopes, list)
        or not scopes
        or any(scope not in ALLOWED_SCOPES for scope in scopes)
        or len(scopes) != len(set(scopes))
    ):
        raise CommandError("changedScopes is invalid")
    source_release = manifest["sourceReleaseId"]
    if source_release is not None and not RELEASE_ID_PATTERN.fullmatch(
        _text(source_release, "sourceReleaseId")
    ):
        raise CommandError("sourceReleaseId is invalid")

    database = _object(manifest["database"], "database")
    _exact_fields(
        database,
        {"migrationVersion", "dirty", "migrationManifestDigest"},
        "database",
    )
    version = database["migrationVersion"]
    if not isinstance(version, int) or isinstance(version, bool) or version < 0:
        raise CommandError("database.migrationVersion must be a non-negative integer")
    if database["dirty"] is not False:
        raise CommandError("successful release database state must not be dirty")
    migration_digest = _text(
        database["migrationManifestDigest"],
        "database.migrationManifestDigest",
    )
    if not SHA256_PATTERN.fullmatch(migration_digest):
        raise CommandError("database.migrationManifestDigest is invalid")

    backend = _object(manifest["backend"], "backend")
    backend_keys = set(backend)
    if backend_keys != set(LEGACY_FUNCTION_KEYS) and backend_keys != set(FUNCTION_KEYS):
        raise CommandError("backend fields are invalid")
    for key in backend:
        if key == "errorNotifier" and backend[key] is None:
            continue
        _function_record(backend[key], f"backend.{key}")

    frontend = manifest["frontend"]
    if frontend is not None:
        frontend_record = _object(frontend, "frontend")
        _exact_fields(
            frontend_record,
            {"snapshotPrefix", "snapshotManifestKey", "snapshotDigest"},
            "frontend",
        )
        prefix = _text(frontend_record["snapshotPrefix"], "frontend.snapshotPrefix")
        manifest_key = _text(
            frontend_record["snapshotManifestKey"],
            "frontend.snapshotManifestKey",
        )
        prefix_parts = prefix.split("/")
        if (
            len(prefix_parts) != 3
            or prefix_parts[0] != "releases"
            or not RELEASE_ID_PATTERN.fullmatch(prefix_parts[1])
            or prefix_parts[2] != "frontend"
            or manifest_key != f"{prefix}/snapshot.json"
        ):
            raise CommandError("frontend snapshot paths are invalid")
        if not SHA256_PATTERN.fullmatch(
            _text(frontend_record["snapshotDigest"], "frontend.snapshotDigest")
        ):
            raise CommandError("frontend.snapshotDigest is invalid")

    serialized = canonical_json(manifest).encode()
    if len(serialized) > MAX_PARAMETER_BYTES:
        raise CommandError("release manifest exceeds the 4 KB standard parameter limit")
    return manifest


def validate_pointer(value: Any) -> dict[str, Any]:
    pointer = _object(value, "current release pointer")
    _exact_fields(
        pointer,
        {
            "schemaVersion",
            "currentReleaseId",
            "currentManifestDigest",
            "previousReleaseId",
            "previousManifestDigest",
            "updatedAt",
        },
        "current release pointer",
    )
    if pointer["schemaVersion"] != SCHEMA_VERSION:
        raise CommandError("current release pointer schemaVersion is unsupported")
    for name in ("currentReleaseId", "previousReleaseId"):
        item = pointer[name]
        if item is not None and not RELEASE_ID_PATTERN.fullmatch(_text(item, name)):
            raise CommandError(f"{name} is invalid")
    for name in ("currentManifestDigest", "previousManifestDigest"):
        item = pointer[name]
        if item is not None and not SHA256_PATTERN.fullmatch(_text(item, name)):
            raise CommandError(f"{name} is invalid")
    if (pointer["previousReleaseId"] is None) != (
        pointer["previousManifestDigest"] is None
    ):
        raise CommandError("previous release pointer fields must both be null or both be set")
    _timestamp(pointer["updatedAt"], "updatedAt")
    return pointer


def _protected_values(config: Config) -> tuple[str, ...]:
    values = [
        config.database.admin_password,
        config.database.migration_password,
        config.database.runtime_password,
        config.backend.jwt_secret,
        config.backend.refresh_jwt_secret,
        getattr(config.backend, "ocr_capability_secret", ""),
        config.backend.web_push_vapid_public_key,
        config.backend.web_push_vapid_private_key,
        config.backend.web_push_vapid_subject,
    ]
    if config.first_admin:
        values.append(config.first_admin.password)
    if config.observability.discord_webhook_url:
        values.append(config.observability.discord_webhook_url)
    return tuple(value for value in values if value)


class ReleaseStore:
    def __init__(self, client: AWSClient, config: Config):
        self.client = client
        self.config = config
        self.prefix = (
            f"/{config.deployment.name_prefix}/"
            f"{config.deployment.environment}/deploy"
        )

    def _encode(self, value: Any) -> str:
        encoded = canonical_json(value)
        if len(encoded.encode()) > MAX_PARAMETER_BYTES:
            raise CommandError("release metadata exceeds the 4 KB standard parameter limit")
        for protected in _protected_values(self.config):
            if protected in encoded:
                raise CommandError("protected configuration found in release metadata")
        return encoded

    def release_path(self, release_id: str) -> str:
        if not RELEASE_ID_PATTERN.fullmatch(release_id):
            raise CommandError("exact valid release ID is required")
        return f"{self.prefix}/releases/{release_id}"

    def candidate_path(self, release_id: str) -> str:
        if not RELEASE_ID_PATTERN.fullmatch(release_id):
            raise CommandError("exact valid release ID is required")
        return f"{self.prefix}/candidates/{release_id}"

    @property
    def current_path(self) -> str:
        return f"{self.prefix}/current"

    def metadata_paths(self) -> list[str]:
        parameters = self.client.parameters_by_path(f"{self.prefix}/")
        allowed_prefixes = (
            f"{self.prefix}/releases/",
            f"{self.prefix}/candidates/",
        )
        paths: list[str] = []
        for parameter in parameters:
            name = parameter.get("Name") if isinstance(parameter, dict) else None
            if not isinstance(name, str):
                raise CommandError("release metadata listing contains an invalid path")
            if name == self.current_path:
                paths.append(name)
                continue
            matching_prefix = next(
                (prefix for prefix in allowed_prefixes if name.startswith(prefix)),
                None,
            )
            if matching_prefix is None or not RELEASE_ID_PATTERN.fullmatch(
                name.removeprefix(matching_prefix)
            ):
                raise CommandError(
                    f"unexpected parameter below the release metadata path: {name}"
                )
            paths.append(name)
        return sorted(
            set(paths),
            key=lambda name: (name == self.current_path, name),
        )

    def delete_all_metadata(self) -> None:
        for path in self.metadata_paths():
            self.client.delete_parameter(path)

    def put_candidate(self, value: dict[str, Any], *, overwrite: bool = False) -> None:
        release_id = _text(value.get("releaseId"), "candidate.releaseId")
        self.client.put_parameter(
            self.candidate_path(release_id),
            self._encode(value),
            overwrite=overwrite,
        )

    def delete_candidate(self, release_id: str) -> None:
        self.client.delete_parameter(self.candidate_path(release_id))

    def put_release(self, value: dict[str, Any]) -> str:
        manifest = validate_manifest(value)
        digest = value_digest(manifest)
        self.client.put_parameter(
            self.release_path(manifest["releaseId"]),
            self._encode(manifest),
            overwrite=False,
        )
        return digest

    def get_release(self, release_id: str) -> dict[str, Any]:
        raw = self.client.get_parameter(self.release_path(release_id))
        if raw is None:
            raise CommandError(f"release does not exist: {release_id}")
        try:
            return validate_manifest(json.loads(raw))
        except json.JSONDecodeError as error:
            raise CommandError(f"release manifest is invalid JSON: {release_id}") from error

    def list_releases(self) -> list[dict[str, Any]]:
        values = self.client.parameters_by_path(f"{self.prefix}/releases/")
        releases: list[dict[str, Any]] = []
        for parameter in values:
            try:
                releases.append(validate_manifest(json.loads(str(parameter["Value"]))))
            except (KeyError, json.JSONDecodeError) as error:
                raise CommandError("release history contains malformed metadata") from error
        return sorted(releases, key=lambda item: item["createdAt"], reverse=True)

    def list_candidates(self) -> list[dict[str, Any]]:
        values = self.client.parameters_by_path(f"{self.prefix}/candidates/")
        candidates: list[dict[str, Any]] = []
        for parameter in values:
            try:
                value = json.loads(str(parameter["Value"]))
            except (KeyError, json.JSONDecodeError) as error:
                raise CommandError("release candidates contain malformed metadata") from error
            if not isinstance(value, dict):
                raise CommandError("release candidate must be an object")
            release_id = value.get("releaseId")
            created_at = value.get("createdAt")
            if (
                not isinstance(release_id, str)
                or not RELEASE_ID_PATTERN.fullmatch(release_id)
                or not isinstance(created_at, str)
            ):
                raise CommandError("release candidate identity is invalid")
            _timestamp(created_at, "candidate.createdAt")
            candidates.append(value)
        return candidates

    def get_current(self) -> dict[str, Any] | None:
        raw = self.client.get_parameter(self.current_path)
        if raw is None:
            return None
        try:
            pointer = validate_pointer(json.loads(raw))
        except json.JSONDecodeError as error:
            raise CommandError("current release pointer is invalid JSON") from error
        current = self.get_release(pointer["currentReleaseId"])
        if value_digest(current) != pointer["currentManifestDigest"]:
            raise CommandError("current release manifest digest does not match its pointer")
        if pointer["previousReleaseId"] is not None:
            previous = self.get_release(pointer["previousReleaseId"])
            if value_digest(previous) != pointer["previousManifestDigest"]:
                raise CommandError("previous release manifest digest does not match its pointer")
        return pointer

    def set_current(
        self,
        manifest: dict[str, Any],
        digest: str,
        *,
        now: datetime | None = None,
    ) -> dict[str, Any]:
        validate_manifest(manifest)
        if digest != value_digest(manifest):
            raise CommandError("refusing current pointer with a mismatched manifest digest")
        old = self.get_current()
        pointer = {
            "schemaVersion": SCHEMA_VERSION,
            "currentReleaseId": manifest["releaseId"],
            "currentManifestDigest": digest,
            "previousReleaseId": old["currentReleaseId"] if old else None,
            "previousManifestDigest": old["currentManifestDigest"] if old else None,
            "updatedAt": utc_text(now),
        }
        validate_pointer(pointer)
        self.client.put_parameter(
            self.current_path,
            self._encode(pointer),
            overwrite=old is not None,
        )
        return pointer


def retention_deletions(
    releases: list[dict[str, Any]],
    pointer: dict[str, Any] | None,
    *,
    now: datetime | None = None,
) -> list[dict[str, Any]]:
    current_time = (now or datetime.now(UTC)).astimezone(UTC)
    protected = set()
    if pointer is not None:
        protected.add(str(pointer["currentReleaseId"]))
        if pointer["previousReleaseId"] is not None:
            protected.add(str(pointer["previousReleaseId"]))
    ordered = sorted(releases, key=lambda item: item["createdAt"], reverse=True)
    kept = set(protected)
    for manifest in ordered:
        if len(kept) >= MAX_SUCCESSFUL_RELEASES:
            break
        kept.add(str(manifest["releaseId"]))
    cutoff = current_time - timedelta(days=MAX_RELEASE_AGE_DAYS)
    deletions: list[dict[str, Any]] = []
    for manifest in ordered:
        release_id = str(manifest["releaseId"])
        created_at = datetime.fromisoformat(str(manifest["createdAt"]).replace("Z", "+00:00"))
        if release_id not in kept or (release_id not in protected and created_at < cutoff):
            deletions.append(manifest)
    return deletions


def stale_candidates(
    candidates: list[dict[str, Any]],
    *,
    now: datetime | None = None,
) -> list[dict[str, Any]]:
    cutoff = (now or datetime.now(UTC)).astimezone(UTC) - timedelta(
        hours=FAILED_CANDIDATE_MAX_AGE_HOURS
    )
    return [
        candidate
        for candidate in candidates
        if datetime.fromisoformat(str(candidate["createdAt"]).replace("Z", "+00:00")) < cutoff
    ]
