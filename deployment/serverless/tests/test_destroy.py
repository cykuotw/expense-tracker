from __future__ import annotations

import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.command import CommandError
from common.terraform import require_create_only, require_cutover_only, require_non_destructive_update


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
