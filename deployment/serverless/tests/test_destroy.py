from __future__ import annotations

import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.command import CommandError
from common.terraform import (
    require_create_only,
    require_cutover_only,
    require_non_destructive_update,
    require_restore_verification_cleanup,
    require_restore_verification_create,
    require_temporary_access_create,
)


def plan(*changes: tuple[str, list[str]]) -> dict:
    return {"resource_changes": [{"address": address, "change": {"actions": actions}} for address, actions in changes]}


class PlanGuardTest(unittest.TestCase):
    def test_initial_plan_rejects_updates(self) -> None:
        with self.assertRaises(CommandError):
            require_create_only(plan(("aws_instance.postgres", ["update"])))

    def test_cutover_accepts_only_temporary_access_deletes(self) -> None:
        value = plan(
            ("aws_eip.temporary_postgres[0]", ["delete"]),
            ("aws_eip_association.temporary_postgres[0]", ["delete"]),
            ("aws_vpc_security_group_ingress_rule.ssh[0]", ["delete"]),
        )
        self.assertEqual(len(require_cutover_only(value)), 3)

    def test_temporary_access_accepts_only_expected_creates(self) -> None:
        value = plan(
            ("aws_eip.temporary_postgres[0]", ["create"]),
            ("aws_eip_association.temporary_postgres[0]", ["create"]),
            ("aws_vpc_security_group_ingress_rule.ssh[0]", ["create"]),
        )
        self.assertEqual(len(require_temporary_access_create(value)), 3)

    def test_restore_verification_guards_only_temporary_resources(self) -> None:
        addresses = (
            "aws_security_group.restore_verification[0]",
            "aws_vpc_security_group_ingress_rule.restore_verification_ssh[0]",
            "aws_vpc_security_group_egress_rule.restore_verification_https[0]",
            "aws_vpc_security_group_egress_rule.restore_verification_dns[0]",
            "aws_instance.restore_verification[0]",
            "aws_eip.restore_verification[0]",
            "aws_eip_association.restore_verification[0]",
        )
        create_changes = tuple((address, ["create"]) for address in addresses)
        delete_changes = tuple((address, ["delete"]) for address in addresses)
        self.assertEqual(len(require_restore_verification_create(plan(*create_changes))), 7)
        self.assertEqual(len(require_restore_verification_cleanup(plan(*delete_changes))), 7)
        with self.assertRaises(CommandError):
            require_restore_verification_create(plan(("aws_instance.postgres", ["create"])))

    def test_update_accepts_only_create_and_in_place_update(self) -> None:
        value = plan(
            ("aws_apigatewayv2_route.google_register", ["create"]),
            ("aws_cloudfront_distribution.frontend", ["update"]),
        )
        self.assertEqual(len(require_non_destructive_update(value)), 2)

    def test_update_rejects_delete_and_replacement(self) -> None:
        with self.assertRaises(CommandError):
            require_non_destructive_update(plan(("aws_instance.postgres", ["delete"])))
        with self.assertRaises(CommandError):
            require_non_destructive_update(plan(("aws_instance.postgres", ["delete", "create"])))

    def test_update_allows_only_explicitly_allowlisted_route_deletions(self) -> None:
        retired_route = 'aws_apigatewayv2_route.authenticated_mutation["expire_invitation"]'
        value = require_non_destructive_update(
            plan((retired_route, ["delete"])),
            allowed_deletes=frozenset({retired_route}),
        )
        self.assertEqual(value[retired_route], ["delete"])

        with self.assertRaises(CommandError):
            require_non_destructive_update(
                plan(("aws_apigatewayv2_route.default", ["delete"])),
                allowed_deletes=frozenset({retired_route}),
            )
