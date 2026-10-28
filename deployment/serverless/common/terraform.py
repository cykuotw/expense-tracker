from __future__ import annotations

import hashlib
import json
import re
from pathlib import Path
from typing import Any

from common.command import CommandError, run


class Terraform:
    def __init__(self, root: Path, variables: Path):
        self.root = root
        self.variables = variables
        self._plan_events: dict[Path, tuple[str, str]] = {}

    def _base(self) -> list[str]:
        return ["terraform", f"-chdir={self.root}"]

    def init(self) -> None:
        run([*self._base(), "init", "-input=false"])

    def validate(self) -> None:
        run([*self._base(), "validate"])

    def has_state(self) -> bool:
        if not (self.root / "terraform.tfstate").is_file():
            return False
        result = run([*self._base(), "state", "list"], check=False)
        if result.returncode:
            raise CommandError("Terraform state exists but cannot be read")
        return bool(result.stdout.strip())

    def output(self) -> dict[str, Any]:
        result = run([*self._base(), "output", "-json"])
        raw = json.loads(result.stdout)
        return {name: entry["value"] for name, entry in raw.items()}

    def plan(
        self,
        plan_path: Path,
        *,
        destroy: bool = False,
        targets: tuple[str, ...] = (),
    ) -> None:
        path = self._plan_path(plan_path)
        self._plan_events.pop(path, None)
        command = [*self._base(), "plan", "-json", "-refresh=false", "-input=false", f"-var-file={self.variables}", f"-out={plan_path}"]
        if destroy:
            command.insert(3, "-destroy")
        command.extend(f"-target={target}" for target in targets)
        result = run(command)
        self._plan_events[path] = (_plan_digest(path), result.stdout)

    def _plan_path(self, path: Path) -> Path:
        return (path if path.is_absolute() else self.root / path).resolve()

    def show_plan(self, plan_path: Path) -> dict[str, Any]:
        result = run([*self._base(), "show", "-json", str(plan_path)], check=False)
        if result.returncode == 0:
            return json.loads(result.stdout)
        detail = result.stderr or result.stdout
        cached = self._plan_events.get(self._plan_path(plan_path))
        # Targeting can leave unrelated resources at their previous provider schema.
        # Inspect only the actions emitted while producing this exact saved plan.
        legacy_schema = "error marshaling prior state" in detail and re.search(
            r"schema version \d+ for .+ in state does not match version \d+ from the provider", detail,
        )
        if not legacy_schema or cached is None:
            raise CommandError(f"terraform plan inspection failed: {detail.strip()}")
        digest, events = cached
        if _plan_digest(self._plan_path(plan_path)) != digest:
            raise CommandError("Terraform saved plan changed after planning; refusing event fallback")
        plan = _plan_actions_from_events(events)
        print("terraform_plan_inspection=planning_events reason=legacy_prior_state_schema", flush=True)
        return plan

    def apply(self, plan_path: Path) -> None:
        run([*self._base(), "apply", "-input=false", str(plan_path)], capture=False)

    def destroy(self) -> None:
        run([*self._base(), "destroy", "-refresh=false", "-input=false", "-auto-approve", f"-var-file={self.variables}"], capture=False)


def _plan_digest(path: Path) -> str:
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def _plan_actions_from_events(output: str) -> dict[str, Any]:
    """Return an action-only projection; never infer permission from partial logs."""
    actions = {
        "noop": ["no-op"], "read": ["read"], "create": ["create"],
        "update": ["update"], "delete": ["delete"], "replace": ["delete", "create"],
    }
    changes: dict[str, dict[str, Any]] = {}
    totals = {"add": 0, "change": 0, "remove": 0}
    summary = None
    try:
        messages = [json.loads(line) for line in output.splitlines() if line.strip()]
        if not messages or messages[0].get("type") != "version" or not re.fullmatch(r"1\.\d+", messages[0].get("ui", "")):
            raise ValueError("unsupported or missing Terraform UI version")
        for message in messages[1:]:
            kind = message.get("type")
            if message.get("@level") == "error" or kind in {"deferred_change", "version"}:
                raise ValueError("incomplete or unsupported plan events")
            if kind == "diagnostic" and message.get("diagnostic", {}).get("severity") == "error":
                raise ValueError("plan contains an error diagnostic")
            if kind == "planned_change":
                change = message["change"]
                address = change["resource"]["addr"]
                action = change["action"]
                if summary is not None or not isinstance(address, str) or not address or address in changes or action not in actions:
                    raise ValueError("unsupported or duplicate planned action")
                if change.get("previous_resource") or change.get("importing"):
                    raise ValueError("state moves and imports require full JSON inspection")
                changes[address] = {"address": address, "change": {"actions": actions[action]}}
                totals["add"] += action in {"create", "replace"}
                totals["change"] += action == "update"
                totals["remove"] += action in {"delete", "replace"}
            elif kind == "change_summary":
                if summary is not None:
                    raise ValueError("duplicate plan summary")
                summary = message["changes"]
                if summary.get("operation") != "plan" or any(
                    type(summary.get(key)) is not int or summary[key] != value
                    for key, value in totals.items()
                ) or summary.get("import", 0) != 0 or summary.get("action_invocation", 0) != 0:
                    raise ValueError("plan events do not match the action summary")
        if summary is None:
            raise ValueError("missing final plan summary")
    except (ValueError, KeyError, TypeError, AttributeError) as error:
        raise CommandError("cannot safely inspect Terraform planning events; refusing apply") from error
    return {"resource_changes": list(changes.values())}


def plan_actions(plan: dict[str, Any]) -> dict[str, list[str]]:
    result: dict[str, list[str]] = {}
    for change in plan.get("resource_changes", []):
        if change.get("mode", "managed") == "data":
            continue
        actions = change.get("change", {}).get("actions", [])
        if actions not in (["no-op"], ["read"]):
            result[change["address"]] = actions
    return result


def require_create_only(plan: dict[str, Any]) -> dict[str, list[str]]:
    actions = plan_actions(plan)
    unsafe = {address: action for address, action in actions.items() if action != ["create"]}
    if unsafe:
        raise CommandError(f"initial plan contains non-create actions: {unsafe}")
    return actions


def require_cutover_only(plan: dict[str, Any]) -> dict[str, list[str]]:
    actions = plan_actions(plan)
    allowed = {
        "aws_eip.temporary_postgres",
        "aws_eip_association.temporary_postgres",
        "aws_vpc_security_group_ingress_rule.ssh",
    }
    unsafe = {}
    for address, action in actions.items():
        normalized_address = address[:-3] if address.endswith("[0]") else address
        if normalized_address not in allowed or action != ["delete"]:
            unsafe[address] = action
    if unsafe or not actions:
        raise CommandError(f"cutover plan is not the expected temporary-access deletion: {actions}")
    return actions


def require_temporary_access_create(plan: dict[str, Any]) -> dict[str, list[str]]:
    actions = plan_actions(plan)
    allowed = {
        "aws_eip.temporary_postgres",
        "aws_eip_association.temporary_postgres",
        "aws_vpc_security_group_ingress_rule.ssh",
    }
    unsafe = {}
    for address, action in actions.items():
        normalized_address = address[:-3] if address.endswith("[0]") else address
        if normalized_address not in allowed or action != ["create"]:
            unsafe[address] = action
    if unsafe or not actions:
        raise CommandError(f"temporary-access plan is not the expected creation: {actions}")
    return actions


def _require_restore_verification_plan(plan: dict[str, Any], expected_action: list[str]) -> dict[str, list[str]]:
    actions = plan_actions(plan)
    allowed = {
        "aws_security_group.restore_verification",
        "aws_vpc_security_group_ingress_rule.restore_verification_ssh",
        "aws_vpc_security_group_egress_rule.restore_verification_https",
        "aws_vpc_security_group_egress_rule.restore_verification_dns",
        "aws_instance.restore_verification",
        "aws_eip.restore_verification",
        "aws_eip_association.restore_verification",
    }
    unsafe = {}
    for address, action in actions.items():
        normalized_address = address[:-3] if address.endswith("[0]") else address
        if normalized_address not in allowed or action != expected_action:
            unsafe[address] = action
    if unsafe or not actions:
        raise CommandError(f"restore-verification plan is not the expected {expected_action}: {actions}")
    return actions


def require_restore_verification_create(plan: dict[str, Any]) -> dict[str, list[str]]:
    return _require_restore_verification_plan(plan, ["create"])


def require_restore_verification_cleanup(plan: dict[str, Any]) -> dict[str, list[str]]:
    return _require_restore_verification_plan(plan, ["delete"])


def require_non_destructive_update(
    plan: dict[str, Any],
    *,
    allowed_deletes: frozenset[str] = frozenset(),
    allowed_replacements: frozenset[str] = frozenset(),
) -> dict[str, list[str]]:
    actions = plan_actions(plan)
    allowed_actions = (["create"], ["update"])
    unsafe = {
        address: action
        for address, action in actions.items()
        if action not in allowed_actions
        and not (address in allowed_deletes and action == ["delete"])
        and not (
            address in allowed_replacements
            and action in (["delete", "create"], ["create", "delete"])
        )
    }
    if unsafe:
        raise CommandError(f"update plan contains destructive or replacement actions: {unsafe}")
    return actions
