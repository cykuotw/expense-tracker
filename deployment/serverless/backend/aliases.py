from __future__ import annotations

from typing import Any

from common.aws import AWSClient
from common.command import CommandError


OUTPUT_KEYS = {
    "worker": "worker_function_name",
    "bootstrap": "bootstrap_function_name",
    "sender": "sender_function_name",
    "delivery": "delivery_function_name",
    "errorNotifier": "error_notifier_function_name",
}
PROMOTION_ORDER = ("bootstrap", "errorNotifier", "worker", "delivery", "sender")


def function_names(outputs: dict[str, Any]) -> dict[str, str | None]:
    result: dict[str, str | None] = {}
    for key, output_key in OUTPUT_KEYS.items():
        value = str(outputs.get(output_key, ""))
        result[key] = value or None
    return result


def _validate_alias(alias: dict[str, Any], function_name: str) -> tuple[str, str]:
    version = str(alias.get("FunctionVersion", ""))
    revision = str(alias.get("RevisionId", ""))
    weights = alias.get("RoutingConfig", {}).get("AdditionalVersionWeights", {})
    if not version.isdigit() or version == "0" or not revision:
        raise CommandError(f"live alias is invalid for {function_name}")
    if weights:
        raise CommandError(
            f"weighted live alias is unsupported for release promotion: {function_name}"
        )
    return version, revision


def current_backend(
    client: AWSClient,
    outputs: dict[str, Any],
) -> dict[str, dict[str, str] | None]:
    backend: dict[str, dict[str, str] | None] = {}
    for key, function_name in function_names(outputs).items():
        if function_name is None:
            backend[key] = None
            continue
        alias = client.alias(function_name)
        if alias is None:
            raise CommandError(f"live alias is missing for {function_name}")
        version, _revision = _validate_alias(alias, function_name)
        backend[key] = client.function_version(function_name, version)
    return backend


def ensure_live_aliases(
    client: AWSClient,
    outputs: dict[str, Any],
    release_id: str,
    preferred: dict[str, Any] | None = None,
) -> tuple[dict[str, dict[str, str] | None], bool]:
    if preferred is not None and set(preferred) != set(OUTPUT_KEYS):
        raise CommandError("preferred backend function set is invalid")
    backend: dict[str, dict[str, str] | None] = {}
    created = False
    for key, function_name in function_names(outputs).items():
        if function_name is None:
            backend[key] = None
            continue
        alias = client.alias(function_name)
        if alias is None:
            record = preferred.get(key) if preferred is not None else None
            if record is not None:
                if not isinstance(record, dict) or record.get("functionName") != function_name:
                    raise CommandError(f"preferred backend record is invalid: {key}")
                actual = client.function_version(function_name, str(record.get("version", "")))
                if actual != record:
                    raise CommandError(f"preferred Lambda version does not match: {key}")
            else:
                record = client.publish_current_configuration(
                    function_name,
                    description=f"Expense Tracker adopted boundary {release_id}",
                )
            alias = client.create_alias(function_name, record["version"])
            created = True
        version, _revision = _validate_alias(alias, function_name)
        backend[key] = client.function_version(function_name, version)
    return backend, created


def validate_backend_versions(
    client: AWSClient,
    backend: dict[str, Any],
) -> None:
    if set(backend) != set(OUTPUT_KEYS):
        raise CommandError("target release backend function set is invalid")
    for key in OUTPUT_KEYS:
        record = backend[key]
        if key == "errorNotifier" and record is None:
            continue
        if not isinstance(record, dict):
            raise CommandError(f"target backend record is invalid: {key}")
        actual = client.function_version(
            str(record.get("functionName", "")),
            str(record.get("version", "")),
        )
        if actual != record:
            raise CommandError(f"target Lambda version does not match its manifest: {key}")


def promote_backend(
    client: AWSClient,
    backend: dict[str, Any],
) -> dict[str, dict[str, str] | None]:
    validate_backend_versions(client, backend)
    previous: dict[str, dict[str, str] | None] = {}
    changed: list[tuple[str, str]] = []
    try:
        for key in PROMOTION_ORDER:
            target = backend[key]
            if target is None:
                previous[key] = None
                continue
            function_name = str(target["functionName"])
            alias = client.alias(function_name)
            if alias is None:
                raise CommandError(f"live alias is missing for {function_name}")
            old_version, revision = _validate_alias(alias, function_name)
            previous[key] = client.function_version(function_name, old_version)
            target_version = str(target["version"])
            if old_version == target_version:
                continue
            client.update_alias(function_name, target_version, revision)
            changed.append((function_name, old_version))
    except BaseException as promotion_error:
        recovery_errors: list[str] = []
        for function_name, old_version in reversed(changed):
            try:
                current = client.alias(function_name)
                if current is None:
                    raise CommandError("live alias disappeared during recovery")
                _current_version, revision = _validate_alias(current, function_name)
                client.update_alias(function_name, old_version, revision)
            except Exception as recovery_error:
                recovery_errors.append(f"{function_name}: {recovery_error}")
        if recovery_errors:
            raise CommandError(
                "Lambda alias promotion failed and compensation was incomplete: "
                + "; ".join(recovery_errors)
            ) from promotion_error
        raise
    for key in OUTPUT_KEYS:
        previous.setdefault(key, None)
    return previous
