from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from common.aws import AWSClient
from common.command import CommandError, run
from config import Config


IMMUTABLE_CACHE_CONTROL = "public, max-age=31536000, immutable"
NO_CACHE = "no-cache"
MUTABLE_FRONTEND_FILES = {
    "index.html": "text/html; charset=utf-8",
    "runtime-config.js": "application/javascript; charset=utf-8",
    "sw.js": "application/javascript; charset=utf-8",
    "manifest.webmanifest": "application/manifest+json",
}


def runtime_config(config: Config) -> str:
    value = {
        "apiOrigin": config.api_origin,
        "apiPath": "/api/v0",
        "googleOAuthEnabled": True,
        "googleClientId": config.backend.google_client_id,
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
    (dist / "runtime-config.js").write_text(runtime_config(config))
    return dist


def publish(client: AWSClient, repo_root: Path, config: Config, outputs: dict[str, Any]) -> None:
    dist = build(repo_root, config)
    bucket = str(outputs["frontend_bucket_name"])
    distribution = str(outputs["cloudfront_distribution_id"])
    mutable_files = dict(MUTABLE_FRONTEND_FILES)
    mutable_files.update(
        {
            asset.relative_to(dist).as_posix(): "application/javascript; charset=utf-8"
            for asset in dist.glob("workbox-*.js")
        }
    )
    immutable_sync_args = [
        "sync",
        f"{dist}/",
        f"s3://{bucket}/",
        "--delete",
        "--cache-control",
        IMMUTABLE_CACHE_CONTROL,
    ]
    for relative_path in mutable_files:
        immutable_sync_args.extend(["--exclude", relative_path])
    client.call("s3", *immutable_sync_args)

    for relative_path, content_type in mutable_files.items():
        source = dist / relative_path
        if not source.is_file():
            continue
        client.call(
            "s3",
            "cp",
            str(source),
            f"s3://{bucket}/{relative_path}",
            "--cache-control",
            NO_CACHE,
            "--content-type",
            content_type,
        )
    invalidation = client.json("cloudfront", "create-invalidation", "--distribution-id", distribution, "--paths", "/*")
    invalidation_id = invalidation["Invalidation"]["Id"]
    client.call("cloudfront", "wait", "invalidation-completed", "--distribution-id", distribution, "--id", invalidation_id)
