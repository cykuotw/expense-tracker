from __future__ import annotations

import os
import tempfile
import zipfile
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

from backend.migration_policy import MANIFEST_NAME, validate_repository
from common.command import run


ZIP_TIME = (1980, 1, 1, 0, 0, 0)


def _entry(name: str, data: bytes, mode: int) -> zipfile.ZipInfo:
    info = zipfile.ZipInfo(name, ZIP_TIME)
    info.create_system = 3
    info.external_attr = mode << 16
    info.compress_type = zipfile.ZIP_DEFLATED
    return info


def _go_build(repo_root: Path, package: str, destination: Path, ldflags: str = "-s -w") -> None:
    run(
        [
            "go",
            "build",
            "-buildvcs=false",
            "-trimpath",
            f"-ldflags={ldflags}",
            "-o",
            str(destination),
            package,
        ],
        cwd=repo_root,
        env={"GOOS": "linux", "GOARCH": "arm64", "CGO_ENABLED": "0"},
        capture=False,
    )


COMPONENT_PACKAGES = {
    "worker": (
        "./backend/cmd/tracker-serverless",
        "-s -w -X expense-tracker/backend/config.BuildMode=release",
    ),
    "ocr": ("./backend/cmd/ocr-serverless", "-s -w"),
    "bootstrap": ("./backend/cmd/bootstrap-serverless", "-s -w"),
    "sender": ("./backend/cmd/push-sender-serverless", "-s -w"),
    "delivery": ("./backend/cmd/push-delivery-serverless", "-s -w"),
    "notifier": ("./backend/cmd/error-notifier-serverless", "-s -w"),
}
ALL_COMPONENTS = frozenset(COMPONENT_PACKAGES)
MAX_PARALLEL_GO_BUILDS = 3


def build(
    repo_root: Path,
    output_dir: Path,
    *,
    components: frozenset[str] = ALL_COMPONENTS,
) -> dict[str, Path]:
    unknown = components - ALL_COMPONENTS
    if unknown:
        raise ValueError(f"unknown backend artifact components: {sorted(unknown)}")
    if not components:
        return {}
    validate_repository(repo_root)
    output_dir.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="expense-tracker-artifacts-") as temporary:
        staging = Path(temporary)
        binaries = {
            component: staging / f"{component}-bootstrap"
            for component in components
        }
        with ThreadPoolExecutor(
            max_workers=min(MAX_PARALLEL_GO_BUILDS, len(components)),
        ) as executor:
            futures = [
                executor.submit(
                    _go_build,
                    repo_root,
                    COMPONENT_PACKAGES[component][0],
                    binaries[component],
                    COMPONENT_PACKAGES[component][1],
                )
                for component in sorted(components)
            ]
            for future in futures:
                future.result()

        artifacts = {
            component: output_dir / f"{component}.zip"
            for component in components
        }
        for component in sorted(components - {"bootstrap"}):
            data = binaries[component].read_bytes()
            with zipfile.ZipFile(artifacts[component], "w") as archive:
                archive.writestr(_entry("bootstrap", data, 0o100755), data)

        if "bootstrap" in components:
            migrations = sorted(
                (repo_root / "backend/cmd/migrate/migrations").glob("*.sql")
            )
            manifest = repo_root / "backend/cmd/migrate/migrations" / MANIFEST_NAME
            if not any(path.name.endswith(".up.sql") for path in migrations):
                raise RuntimeError("no up migrations found for bootstrap artifact")
            with zipfile.ZipFile(artifacts["bootstrap"], "w") as archive:
                data = binaries["bootstrap"].read_bytes()
                archive.writestr(_entry("bootstrap", data, 0o100755), data)
                for migration in migrations:
                    data = migration.read_bytes()
                    archive.writestr(
                        _entry(f"migrations/{migration.name}", data, 0o100644),
                        data,
                    )
                manifest_data = manifest.read_bytes()
                archive.writestr(
                    _entry(f"migrations/{MANIFEST_NAME}", manifest_data, 0o100644),
                    manifest_data,
                )
        for artifact in artifacts.values():
            os.chmod(artifact, 0o600)
        return artifacts
