from __future__ import annotations

import hashlib
import json
import os
import tempfile
from pathlib import Path
from typing import Any

from common.aws import AWSClient
from common.command import CommandError, protected_json
from config import Config


def _state_digest(terraform_root: Path) -> str | None:
    path = terraform_root / "terraform.tfstate"
    return hashlib.sha256(path.read_bytes()).hexdigest() if path.is_file() else None


def _protected_values(config: Config) -> tuple[bytes, ...]:
    values = [
        config.database.admin_password, config.database.migration_password,
        config.database.runtime_password, config.backend.jwt_secret,
        config.backend.refresh_jwt_secret,
    ]
    if config.first_admin:
        values.append(config.first_admin.password)
    return tuple(value.encode() for value in values if value)


def assert_secret_boundary(terraform_root: Path, config: Config) -> None:
    protected_values = _protected_values(config)
    for path in terraform_root.iterdir():
        if not path.is_file() or not (
            "tfstate" in path.name
            or "tfplan" in path.name
            or path.suffix == ".json"
        ):
            continue
        data = path.read_bytes()
        for value in protected_values:
            if value in data:
                raise CommandError(f"protected value found in Terraform artifact: {path.name}")


def _write_repaired_state(path: Path, data: bytes) -> None:
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="wb",
            prefix=f".{path.name}.repair-",
            dir=path.parent,
            delete=False,
        ) as stream:
            temporary_path = Path(stream.name)
            os.fchmod(stream.fileno(), 0o600)
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary_path, path)
        temporary_path = None
    finally:
        if temporary_path is not None:
            temporary_path.unlink(missing_ok=True)


def repair_secret_boundary(terraform_root: Path, config: Config) -> int:
    """Remove runtime environments that Terraform does not own from local state."""
    repaired = 0
    protected_values = _protected_values(config)
    state_paths = sorted(
        path
        for path in terraform_root.iterdir()
        if path.is_file()
        and (path.name == "terraform.tfstate" or path.name.startswith("terraform.tfstate."))
    )
    for path in state_paths:
        original = path.read_bytes()
        try:
            state = json.loads(original)
        except json.JSONDecodeError as error:
            raise CommandError(f"cannot safely inspect Terraform state structure: {path.name}") from error

        changed = False
        for resource in state.get("resources", []):
            if (
                resource.get("mode", "managed") != "managed"
                or resource.get("type") != "aws_lambda_function"
                or resource.get("name") not in {"worker", "bootstrap"}
            ):
                continue
            for instance in resource.get("instances", []):
                attributes = instance.get("attributes")
                if isinstance(attributes, dict) and attributes.get("environment"):
                    attributes["environment"] = []
                    changed = True

        if not changed:
            continue
        if isinstance(state.get("serial"), int):
            state["serial"] += 1
        repaired_state = (json.dumps(state, separators=(",", ":")) + "\n").encode()
        if any(value in repaired_state for value in protected_values):
            raise CommandError(
                f"protected value remains outside the repairable Lambda environment state: {path.name}"
            )
        _write_repaired_state(path, repaired_state)
        repaired += 1

    assert_secret_boundary(terraform_root, config)
    return repaired


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
