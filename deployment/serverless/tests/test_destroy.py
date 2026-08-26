from __future__ import annotations

import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.terraform import require_create_only, require_cutover_only


def plan(*changes: tuple[str, list[str]]) -> dict:
    return {"resource_changes": [{"address": address, "change": {"actions": actions}} for address, actions in changes]}


class PlanGuardTest(unittest.TestCase):
    def test_initial_plan_rejects_updates(self) -> None:
        with self.assertRaises(Exception):
            require_create_only(plan(("aws_instance.postgres", ["update"])))

    def test_cutover_accepts_only_temporary_access_deletes(self) -> None:
        value = plan(
            ("aws_eip.temporary_postgres[0]", ["delete"]),
            ("aws_eip_association.temporary_postgres[0]", ["delete"]),
            ("aws_vpc_security_group_ingress_rule.ssh[0]", ["delete"]),
        )
        self.assertEqual(len(require_cutover_only(value)), 3)
