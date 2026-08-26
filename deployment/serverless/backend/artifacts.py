from __future__ import annotations

import os
import tempfile
import zipfile
from pathlib import Path

from common.command import run


ZIP_TIME = (1980, 1, 1, 0, 0, 0)


def _entry(name: str, data: bytes, mode: int) -> zipfile.ZipInfo:
    info = zipfile.ZipInfo(name, ZIP_TIME)
    info.create_system = 3
    info.external_attr = mode << 16
    info.compress_type = zipfile.ZIP_DEFLATED
    return info


def _go_build(repo_root: Path, package: str, destination: Path) -> None:
    run(
        ["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(destination), package],
        cwd=repo_root,
        env={"GOOS": "linux", "GOARCH": "arm64", "CGO_ENABLED": "0"},
        capture=False,
    )


def build(repo_root: Path, output_dir: Path) -> dict[str, Path]:
    output_dir.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="expense-tracker-artifacts-") as temporary:
        staging = Path(temporary)
        worker_binary = staging / "worker-bootstrap"
        bootstrap_binary = staging / "admin-bootstrap"
        _go_build(repo_root, "./backend/cmd/tracker-serverless", worker_binary)
        _go_build(repo_root, "./backend/cmd/bootstrap-serverless", bootstrap_binary)

        artifacts = {
            "worker": output_dir / "worker.zip",
            "bootstrap": output_dir / "bootstrap.zip",
        }
        worker_data = worker_binary.read_bytes()
        with zipfile.ZipFile(artifacts["worker"], "w") as archive:
            archive.writestr(_entry("bootstrap", worker_data, 0o100755), worker_data)
        migrations = sorted((repo_root / "backend/cmd/migrate/migrations").glob("*.sql"))
        if not any(path.name.endswith(".up.sql") for path in migrations):
            raise RuntimeError("no up migrations found for bootstrap artifact")
        with zipfile.ZipFile(artifacts["bootstrap"], "w") as archive:
            data = bootstrap_binary.read_bytes()
            archive.writestr(_entry("bootstrap", data, 0o100755), data)
            for migration in migrations:
                data = migration.read_bytes()
                archive.writestr(_entry(f"migrations/{migration.name}", data, 0o100644), data)
        for artifact in artifacts.values():
            os.chmod(artifact, 0o600)
        return artifacts
