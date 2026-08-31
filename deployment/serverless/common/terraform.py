from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from common.command import CommandError, run


class Terraform:
    def __init__(self, root: Path, variables: Path):
        self.root = root
        self.variables = variables

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
        command = [*self._base(), "plan", "-refresh=false", "-input=false", f"-var-file={self.variables}", f"-out={plan_path}"]
        if destroy:
            command.insert(3, "-destroy")
        command.extend(f"-target={target}" for target in targets)
        run(command)

    def show_plan(self, plan_path: Path) -> dict[str, Any]:
        return json.loads(run([*self._base(), "show", "-json", str(plan_path)]).stdout)

    def apply(self, plan_path: Path) -> None:
        run([*self._base(), "apply", "-input=false", str(plan_path)], capture=False)

    def destroy(self) -> None:
        run([*self._base(), "destroy", "-refresh=false", "-input=false", "-auto-approve", f"-var-file={self.variables}"], capture=False)


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
    plan: dict[str, Any], *, allowed_deletes: frozenset[str] = frozenset()
) -> dict[str, list[str]]:
    actions = plan_actions(plan)
    allowed_actions = (["create"], ["update"])
    unsafe = {
        address: action
        for address, action in actions.items()
        if action not in allowed_actions
        and not (address in allowed_deletes and action == ["delete"])
    }
    if unsafe:
        raise CommandError(f"update plan contains destructive or replacement actions: {unsafe}")
    return actions
