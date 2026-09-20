from __future__ import annotations

import hashlib
import json
import mimetypes
import tempfile
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from urllib.parse import quote

from common.aws import AWSClient
from common.command import CommandError, run
from config import Config
from release import RELEASE_ID_PATTERN, canonical_json, value_digest


IMMUTABLE_CACHE_CONTROL = "public, max-age=31536000, immutable"
NO_CACHE = "no-cache"
SERVICE_WORKER_FILE = "service-worker.js"
SNAPSHOT_SCHEMA_VERSION = 1


def frontend_version(dist: Path, *, now: datetime | None = None) -> str:
    digest = hashlib.sha256()
    for path in sorted(dist.rglob("*")):
        if not path.is_file() or path.relative_to(dist).as_posix() == "runtime-config.js":
            continue
        digest.update(path.relative_to(dist).as_posix().encode())
        digest.update(b"\0")
        digest.update(path.read_bytes())
    deployed_at = (now or datetime.now(UTC)).strftime("%Y%m%d")
    return f"v-{deployed_at}-{digest.hexdigest()[:8]}"


def runtime_config(config: Config, version: str) -> str:
    value = {
        "apiOrigin": config.api_origin,
        "apiPath": "/api/v0",
        "googleOAuthEnabled": True,
        "googleClientId": config.backend.google_client_id,
        "frontendVersion": version,
    }
    return f"window.__APP_CONFIG__ = Object.freeze({json.dumps(value, indent=4)});\n"


def build(repo_root: Path, config: Config) -> Path:
    frontend_root = repo_root / "frontend"
    run(["pnpm", "install", "--frozen-lockfile"], cwd=frontend_root, capture=False)
    run(["pnpm", "run", "test:run"], cwd=frontend_root, capture=False)
    run(["pnpm", "run", "build"], cwd=frontend_root, capture=False)
    dist = frontend_root / "dist"
    if not (dist / "index.html").is_file():
        raise CommandError("frontend build did not create dist/index.html")
    if not (dist / SERVICE_WORKER_FILE).is_file():
        raise CommandError(f"frontend build did not create dist/{SERVICE_WORKER_FILE}")
    (dist / "runtime-config.js").write_text(runtime_config(config, frontend_version(dist)))
    return dist


def _sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _content_type(path: Path) -> str:
    overrides = {
        ".html": "text/html; charset=utf-8",
        ".js": "application/javascript; charset=utf-8",
        ".css": "text/css; charset=utf-8",
        ".json": "application/json",
        ".webmanifest": "application/manifest+json",
        ".svg": "image/svg+xml",
    }
    return overrides.get(path.suffix.lower()) or mimetypes.guess_type(path.name)[0] or "application/octet-stream"


def snapshot_descriptor(dist: Path, release_id: str) -> dict[str, Any]:
    if not RELEASE_ID_PATTERN.fullmatch(release_id):
        raise CommandError("frontend snapshot requires an exact valid release ID")
    mutable: list[dict[str, str]] = []
    assets: list[dict[str, str]] = []
    for path in sorted(item for item in dist.rglob("*") if item.is_file()):
        relative = path.relative_to(dist).as_posix()
        record = {
            "path": relative,
            "sha256": _sha256(path),
            "contentType": _content_type(path),
        }
        (assets if relative.startswith("assets/") else mutable).append(record)
    if not any(item["path"] == "index.html" for item in mutable):
        raise CommandError("frontend snapshot does not contain index.html")
    return {
        "schemaVersion": SNAPSHOT_SCHEMA_VERSION,
        "releaseId": release_id,
        "mutableFiles": mutable,
        "assets": assets,
    }


def _validate_descriptor(value: Any, release_id: str) -> dict[str, Any]:
    expected = {"schemaVersion", "releaseId", "mutableFiles", "assets"}
    if not isinstance(value, dict) or set(value) != expected:
        raise CommandError("frontend snapshot descriptor fields are invalid")
    if value["schemaVersion"] != SNAPSHOT_SCHEMA_VERSION or value["releaseId"] != release_id:
        raise CommandError("frontend snapshot descriptor identity is invalid")
    all_paths: set[str] = set()
    for label in ("mutableFiles", "assets"):
        records = value[label]
        if not isinstance(records, list):
            raise CommandError(f"frontend snapshot {label} must be an array")
        for record in records:
            if not isinstance(record, dict) or set(record) != {"path", "sha256", "contentType"}:
                raise CommandError("frontend snapshot file record is invalid")
            path = record["path"]
            digest = record["sha256"]
            content_type = record["contentType"]
            if (
                not isinstance(path, str)
                or not path
                or path.startswith("/")
                or ".." in Path(path).parts
                or path in all_paths
                or not isinstance(digest, str)
                or len(digest) != 64
                or any(character not in "0123456789abcdef" for character in digest)
                or not isinstance(content_type, str)
                or not content_type
            ):
                raise CommandError("frontend snapshot file record is invalid")
            if (label == "assets") != path.startswith("assets/"):
                raise CommandError("frontend snapshot file classification is invalid")
            all_paths.add(path)
    if "index.html" not in all_paths:
        raise CommandError("frontend snapshot does not contain index.html")
    return value


def _put_file(client: AWSClient, source: Path, bucket: str, key: str, record: dict[str, str], cache_control: str) -> None:
    client.call(
        "s3",
        "cp",
        str(source),
        f"s3://{bucket}/{key}",
        "--cache-control",
        cache_control,
        "--content-type",
        record["contentType"],
        "--metadata",
        f"sha256={record['sha256']}",
        "--only-show-errors",
    )


def _invalidate(client: AWSClient, distribution: str) -> None:
    invalidation = client.json("cloudfront", "create-invalidation", "--distribution-id", distribution, "--paths", "/*")
    invalidation_id = str(invalidation["Invalidation"]["Id"])
    client.call("cloudfront", "wait", "invalidation-completed", "--distribution-id", distribution, "--id", invalidation_id)


def publish(
    client: AWSClient,
    repo_root: Path,
    config: Config,
    outputs: dict[str, Any],
    release_id: str,
) -> dict[str, str]:
    dist = build(repo_root, config)
    bucket = str(outputs["frontend_bucket_name"])
    distribution = str(outputs["cloudfront_distribution_id"])
    descriptor = snapshot_descriptor(dist, release_id)
    prefix = f"releases/{release_id}/frontend"

    for record in descriptor["assets"]:
        _put_file(client, dist / record["path"], bucket, record["path"], record, IMMUTABLE_CACHE_CONTROL)
    for record in descriptor["mutableFiles"]:
        source = dist / record["path"]
        _put_file(client, source, bucket, f"{prefix}/root/{record['path']}", record, NO_CACHE)

    manifest_key = f"{prefix}/snapshot.json"
    serialized = canonical_json(descriptor)
    with tempfile.NamedTemporaryFile(prefix="expense-frontend-snapshot-", suffix=".json") as stream:
        Path(stream.name).write_text(serialized)
        client.call(
            "s3", "cp", stream.name, f"s3://{bucket}/{manifest_key}",
            "--cache-control", NO_CACHE,
            "--content-type", "application/json",
            "--metadata", f"sha256={hashlib.sha256(serialized.encode()).hexdigest()}",
            "--only-show-errors",
        )

    record = {
        "snapshotPrefix": prefix,
        "snapshotManifestKey": manifest_key,
        "snapshotDigest": value_digest(descriptor),
    }
    restore(client, outputs, record, invalidate=False)
    _invalidate(client, distribution)
    return record


def snapshot_release_id(frontend: dict[str, str]) -> str:
    prefix = frontend.get("snapshotPrefix", "")
    parts = prefix.split("/")
    if (
        len(parts) != 3
        or parts[0] != "releases"
        or not RELEASE_ID_PATTERN.fullmatch(parts[1])
        or parts[2] != "frontend"
    ):
        raise CommandError("frontend snapshot prefix is invalid")
    release_id = parts[1]
    expected_key = f"{prefix}/snapshot.json"
    if frontend.get("snapshotManifestKey") != expected_key:
        raise CommandError("frontend snapshot manifest key does not match release")
    return release_id


def load_snapshot_descriptor(client: AWSClient, bucket: str, frontend: dict[str, str]) -> dict[str, Any]:
    release_id = snapshot_release_id(frontend)
    expected_prefix = f"releases/{release_id}/frontend"
    expected_key = f"{expected_prefix}/snapshot.json"
    raw = client.call("s3", "cp", f"s3://{bucket}/{expected_key}", "-", "--only-show-errors")
    try:
        descriptor = _validate_descriptor(json.loads(raw), release_id)
    except json.JSONDecodeError as error:
        raise CommandError("frontend snapshot descriptor is invalid JSON") from error
    if frontend.get("snapshotDigest") != value_digest(descriptor):
        raise CommandError("frontend snapshot descriptor digest does not match release manifest")
    return descriptor


def _head_matches(client: AWSClient, bucket: str, key: str, digest: str) -> None:
    head = client.json("s3api", "head-object", "--bucket", bucket, "--key", key)
    metadata = head.get("Metadata", {})
    if not isinstance(metadata, dict) or metadata.get("sha256") != digest:
        raise CommandError(f"frontend object digest metadata does not match: {key}")


def restore(
    client: AWSClient,
    outputs: dict[str, Any],
    frontend: dict[str, str],
    *,
    invalidate: bool = True,
) -> None:
    bucket = str(outputs["frontend_bucket_name"])
    descriptor = load_snapshot_descriptor(client, bucket, frontend)
    prefix = str(frontend["snapshotPrefix"])
    for record in descriptor["assets"]:
        _head_matches(client, bucket, record["path"], record["sha256"])
    for record in descriptor["mutableFiles"]:
        source_key = f"{prefix}/root/{record['path']}"
        _head_matches(client, bucket, source_key, record["sha256"])
    for record in descriptor["mutableFiles"]:
        source_key = f"{prefix}/root/{record['path']}"
        client.json(
            "s3api", "copy-object",
            "--bucket", bucket,
            "--key", record["path"],
            "--copy-source", quote(f"{bucket}/{source_key}", safe="/"),
            "--metadata-directive", "REPLACE",
            "--cache-control", NO_CACHE,
            "--content-type", record["contentType"],
            "--metadata", f"sha256={record['sha256']}",
        )
        _head_matches(client, bucket, record["path"], record["sha256"])
    if invalidate:
        _invalidate(client, str(outputs["cloudfront_distribution_id"]))
