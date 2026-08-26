from __future__ import annotations

import hashlib
import tempfile
from pathlib import Path
from typing import Any

from common.aws import AWSClient
from common.command import CommandError, protected_json
from config import Config


def _state_digest(terraform_root: Path) -> str | None:
    path = terraform_root / "terraform.tfstate"
    return hashlib.sha256(path.read_bytes()).hexdigest() if path.is_file() else None


def assert_secret_boundary(terraform_root: Path, config: Config) -> None:
    secrets = [
        config.database.admin_password, config.database.migration_password,
        config.database.runtime_password, config.backend.jwt_secret,
        config.backend.refresh_jwt_secret,
    ]
    if config.first_admin:
        secrets.append(config.first_admin.password)
    for path in terraform_root.iterdir():
        if not path.is_file() or not (
            "tfstate" in path.name
            or "tfplan" in path.name
            or path.suffix == ".json"
        ):
            continue
        data = path.read_bytes()
        for secret in secrets:
            if secret.encode() in data:
                raise CommandError(f"protected value found in Terraform artifact: {path.name}")


def configure_bootstrap(client: AWSClient, config: Config, outputs: dict[str, Any], terraform_root: Path) -> dict[str, Any]:
    before = _state_digest(terraform_root)
    with protected_json(config.bootstrap_environment(str(outputs["database_host"])), prefix="expense-bootstrap-env-") as path:
        client.publish_environment(str(outputs["bootstrap_function_name"]), path)
    assert_secret_boundary(terraform_root, config)
    if _state_digest(terraform_root) != before:
        raise CommandError("Terraform state changed while publishing bootstrap runtime")
    with tempfile.NamedTemporaryFile(prefix="expense-bootstrap-response-", suffix=".json", delete=False) as stream:
        response_path = Path(stream.name)
    try:
        first = client.invoke_bootstrap(str(outputs["bootstrap_function_name"]), response_path)
        second = client.invoke_bootstrap(str(outputs["bootstrap_function_name"]), response_path)
        if second.get("first_admin_status") not in {"not_requested", "already_exists"}:
            raise CommandError("second bootstrap invocation was not idempotent")
        return first
    finally:
        response_path.unlink(missing_ok=True)


def configure_worker(client: AWSClient, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    before = _state_digest(terraform_root)
    with protected_json(config.worker_environment(str(outputs["database_host"])), prefix="expense-worker-env-") as path:
        client.publish_environment(str(outputs["worker_function_name"]), path)
    client.activate_worker(str(outputs["worker_function_name"]))
    assert_secret_boundary(terraform_root, config)
    if _state_digest(terraform_root) != before:
        raise CommandError("Terraform state changed while publishing worker runtime")


def update_bootstrap(client: AWSClient, artifact: Path, config: Config, outputs: dict[str, Any], terraform_root: Path) -> dict[str, Any]:
    client.publish_code(str(outputs["bootstrap_function_name"]), artifact)
    return configure_bootstrap(client, config, outputs, terraform_root)


def update_worker(client: AWSClient, artifact: Path, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    client.publish_code(str(outputs["worker_function_name"]), artifact)
    if client.concurrency(str(outputs["worker_function_name"])) != 3:
        raise CommandError("backend update requires worker reserved concurrency 3")
    assert_secret_boundary(terraform_root, config)
