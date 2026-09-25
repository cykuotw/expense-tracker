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
        getattr(config.backend, "ocr_capability_secret", ""),
        config.backend.web_push_vapid_public_key,
        config.backend.web_push_vapid_private_key,
        config.backend.web_push_vapid_subject,
    ]
    if config.first_admin:
        values.append(config.first_admin.password)
    webhook_url = getattr(
        getattr(config, "observability", None),
        "discord_webhook_url",
        None,
    )
    if webhook_url:
        values.append(webhook_url)
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
                or resource.get("name") not in {"worker", "ocr", "bootstrap", "sender", "delivery", "error_notifier"}
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
        if first.get("first_admin_status") not in {
            "created", "reconciled", "already_exists", "not_requested",
        }:
            raise CommandError("first bootstrap invocation returned an invalid first-admin status")
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


def configure_ocr(client: AWSClient, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    before = _state_digest(terraform_root)
    with protected_json(
        config.ocr_environment(str(outputs["ocr_replay_table_name"])),
        prefix="expense-ocr-env-",
    ) as path:
        client.publish_environment(str(outputs["ocr_function_name"]), path)
    client.activate_ocr(str(outputs["ocr_function_name"]))
    assert_secret_boundary(terraform_root, config)
    if _state_digest(terraform_root) != before:
        raise CommandError("Terraform state changed while publishing OCR runtime")


def configure_notifications(
    client: AWSClient,
    config: Config,
    outputs: dict[str, Any],
    terraform_root: Path,
    *,
    activate: bool = True,
) -> None:
    before = _state_digest(terraform_root)
    with protected_json(config.delivery_environment(str(outputs["database_host"])), prefix="expense-delivery-env-") as path:
        client.publish_environment(str(outputs["delivery_function_name"]), path)
    if activate:
        client.activate_notification_function(str(outputs["delivery_function_name"]))
    delivery_reference = f"{outputs['delivery_function_name']}:live"
    with protected_json(config.sender_environment(delivery_reference), prefix="expense-sender-env-") as path:
        client.publish_environment(str(outputs["sender_function_name"]), path)
    if activate:
        client.activate_notification_function(str(outputs["sender_function_name"]))
    assert_secret_boundary(terraform_root, config)
    if _state_digest(terraform_root) != before:
        raise CommandError("Terraform state changed while publishing notification runtimes")


def configure_error_notifier(client: AWSClient, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    function_name = str(outputs.get("error_notifier_function_name", ""))
    if not config.error_alerting_enabled:
        if function_name:
            raise CommandError("disabled error alerting unexpectedly exposed a notifier function")
        return
    if not function_name:
        raise CommandError("enabled error alerting did not expose a notifier function")

    before = _state_digest(terraform_root)
    with protected_json(config.notifier_environment(), prefix="expense-notifier-env-") as path:
        client.publish_environment(function_name, path)
    client.activate_notification_function(function_name)
    assert_secret_boundary(terraform_root, config)
    if _state_digest(terraform_root) != before:
        raise CommandError("Terraform state changed while publishing error notifier runtime")


def update_bootstrap(client: AWSClient, artifact: Path, config: Config, outputs: dict[str, Any], terraform_root: Path) -> dict[str, Any]:
    client.publish_code(str(outputs["bootstrap_function_name"]), artifact)
    return configure_bootstrap(client, config, outputs, terraform_root)


def update_worker(client: AWSClient, artifact: Path, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    client.publish_code(str(outputs["worker_function_name"]), artifact)
    configure_worker(client, config, outputs, terraform_root)


def update_ocr(client: AWSClient, artifact: Path, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    client.publish_code(str(outputs["ocr_function_name"]), artifact)
    configure_ocr(client, config, outputs, terraform_root)


def update_notifications(client: AWSClient, artifact: Path, delivery_artifact: Path, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    client.publish_code(str(outputs["delivery_function_name"]), delivery_artifact)
    client.publish_code(str(outputs["sender_function_name"]), artifact)
    configure_notifications(client, config, outputs, terraform_root)


def update_error_notifier(client: AWSClient, artifact: Path, config: Config, outputs: dict[str, Any], terraform_root: Path) -> None:
    if not config.error_alerting_enabled:
        configure_error_notifier(client, config, outputs, terraform_root)
        return
    function_name = str(outputs.get("error_notifier_function_name", ""))
    if not function_name:
        raise CommandError("enabled error alerting did not expose a notifier function")
    client.publish_code(function_name, artifact)
    configure_error_notifier(client, config, outputs, terraform_root)


def _publish_configured_function(
    client: AWSClient,
    function_name: str,
    artifact: Path,
    environment: dict[str, dict[str, str]],
    release_id: str,
    config: Config,
    terraform_root: Path,
) -> dict[str, str]:
    before = _state_digest(terraform_root)
    client.update_code(function_name, artifact)
    with protected_json(
        environment,
        prefix=f"expense-{function_name}-release-env-",
    ) as path:
        client.update_environment(
            function_name,
            path,
            description=f"Expense Tracker release {release_id}",
        )
    assert_secret_boundary(terraform_root, config)
    if _state_digest(terraform_root) != before:
        raise CommandError(
            f"Terraform state changed while publishing release for {function_name}"
        )
    return client.publish_version(function_name)


def publish_bootstrap_release(
    client: AWSClient,
    artifact: Path,
    config: Config,
    outputs: dict[str, Any],
    terraform_root: Path,
    release_id: str,
) -> tuple[dict[str, str], dict[str, Any]]:
    function_name = str(outputs["bootstrap_function_name"])
    record = _publish_configured_function(
        client,
        function_name,
        artifact,
        config.bootstrap_environment(str(outputs["database_host"])),
        release_id,
        config,
        terraform_root,
    )
    with tempfile.NamedTemporaryFile(
        prefix="expense-bootstrap-response-",
        suffix=".json",
        delete=False,
    ) as stream:
        response_path = Path(stream.name)
    try:
        first = client.invoke_bootstrap(record["qualifiedArn"], response_path)
        second = client.invoke_bootstrap(record["qualifiedArn"], response_path)
        if first.get("first_admin_status") not in {
            "created",
            "reconciled",
            "already_exists",
            "not_requested",
        }:
            raise CommandError("first bootstrap invocation returned an invalid first-admin status")
        if second.get("first_admin_status") not in {
            "not_requested",
            "already_exists",
        }:
            raise CommandError("second bootstrap invocation was not idempotent")
        return record, first
    finally:
        response_path.unlink(missing_ok=True)


def publish_worker_release(
    client: AWSClient,
    artifact: Path,
    config: Config,
    outputs: dict[str, Any],
    terraform_root: Path,
    release_id: str,
) -> dict[str, str]:
    record = _publish_configured_function(
        client,
        str(outputs["worker_function_name"]),
        artifact,
        config.worker_environment(str(outputs["database_host"])),
        release_id,
        config,
        terraform_root,
    )
    client.activate_worker(record["functionName"])
    return record


def publish_ocr_release(
    client: AWSClient,
    artifact: Path,
    config: Config,
    outputs: dict[str, Any],
    terraform_root: Path,
    release_id: str,
) -> dict[str, str]:
    record = _publish_configured_function(
        client,
        str(outputs["ocr_function_name"]),
        artifact,
        config.ocr_environment(str(outputs["ocr_replay_table_name"])),
        release_id,
        config,
        terraform_root,
    )
    client.activate_ocr(record["functionName"])
    return record


def publish_notification_releases(
    client: AWSClient,
    sender_artifact: Path,
    delivery_artifact: Path,
    config: Config,
    outputs: dict[str, Any],
    terraform_root: Path,
    release_id: str,
    *,
    activate: bool = True,
) -> tuple[dict[str, str], dict[str, str]]:
    delivery_name = str(outputs["delivery_function_name"])
    delivery = _publish_configured_function(
        client,
        delivery_name,
        delivery_artifact,
        config.delivery_environment(str(outputs["database_host"])),
        release_id,
        config,
        terraform_root,
    )
    if activate:
        client.activate_notification_function(delivery_name)
    sender_name = str(outputs["sender_function_name"])
    sender = _publish_configured_function(
        client,
        sender_name,
        sender_artifact,
        config.sender_environment(f"{delivery_name}:live"),
        release_id,
        config,
        terraform_root,
    )
    if activate:
        client.activate_notification_function(sender_name)
    return sender, delivery


def publish_error_notifier_release(
    client: AWSClient,
    artifact: Path,
    config: Config,
    outputs: dict[str, Any],
    terraform_root: Path,
    release_id: str,
) -> dict[str, str] | None:
    if not config.error_alerting_enabled:
        if outputs.get("error_notifier_function_name"):
            raise CommandError("disabled error alerting unexpectedly exposed a notifier function")
        return None
    function_name = str(outputs.get("error_notifier_function_name", ""))
    if not function_name:
        raise CommandError("enabled error alerting did not expose a notifier function")
    record = _publish_configured_function(
        client,
        function_name,
        artifact,
        config.notifier_environment(),
        release_id,
        config,
        terraform_root,
    )
    client.activate_notification_function(function_name)
    return record
