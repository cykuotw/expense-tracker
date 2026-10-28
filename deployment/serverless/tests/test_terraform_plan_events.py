from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import terraform
from common.command import CommandError

SCHEMA_ERROR = (
    "Failed to marshal plan to json: error marshaling prior state: schema version 1 "
    "for aws_instance.postgres in state does not match version 2 from the provider"
)


def events(*changes: tuple[str, str], **summary_overrides: object) -> str:
    counts = {"add": 0, "change": 0, "remove": 0, "operation": "plan"}
    messages = [{"type": "version", "ui": "1.3"}]
    for address, action in changes:
        messages.append({"type": "planned_change", "change": {"resource": {"addr": address}, "action": action}})
        counts["add"] += action in {"create", "replace"}
        counts["change"] += action == "update"
        counts["remove"] += action in {"delete", "replace"}
    counts.update(summary_overrides)
    messages.append({"type": "change_summary", "changes": counts})
    return "\n".join(json.dumps(message) for message in messages)


class PlanEventsTest(unittest.TestCase):
    def test_projection_preserves_actions_and_existing_destructive_guards(self) -> None:
        changes = (("aws_s3_bucket.frontend", "create"), ("aws_cloudfront_distribution.frontend", "update"),
                   ("aws_instance.postgres", "delete"), ("aws_lambda_permission.api_gateway", "replace"),
                   ("data.aws_caller_identity.current", "read"))
        plan = terraform._plan_actions_from_events(events(*changes))
        self.assertEqual(terraform.plan_actions(plan), {
            changes[0][0]: ["create"], changes[1][0]: ["update"], changes[2][0]: ["delete"],
            changes[3][0]: ["delete", "create"],
        })
        with self.assertRaises(CommandError):
            terraform.require_non_destructive_update(plan)
        with self.assertRaises(CommandError):
            terraform.require_non_destructive_update(plan, allowed_replacements=frozenset({changes[3][0]}))
        terraform.require_non_destructive_update(plan, allowed_deletes=frozenset({changes[2][0]}),
                                               allowed_replacements=frozenset({changes[3][0]}))

    def test_rejects_incomplete_inconsistent_and_unsupported_streams(self) -> None:
        valid = events(("aws_s3_bucket.frontend", "create"))
        cases = ["", "not json", "\n".join(valid.splitlines()[:-1]),
                 valid.replace('"ui": "1.3"', '"ui": "2.0"'),
                 events(("aws_s3_bucket.frontend", "create"), add=2),
                 events(("aws_s3_bucket.frontend", "import")),
                 events(("aws_s3_bucket.frontend", "forget")),
                 events(("aws_s3_bucket.frontend", "create"), ("aws_s3_bucket.frontend", "delete")),
                 events(action_invocation=1), events(**{"import": 1}),
                 valid + '\n{"type":"deferred_change"}',
                 valid + '\n{"type":"diagnostic","diagnostic":{"severity":"error"}}',
                 valid + '\n' + valid.splitlines()[-1],
                 valid.replace('"add": 1', '"add": true')]
        for value in cases:
            with self.subTest(stream=value), self.assertRaises(CommandError):
                terraform._plan_actions_from_events(value)
        self.assertEqual(terraform.plan_actions(terraform._plan_actions_from_events(events())), {})

    def test_fallback_uses_only_events_for_the_exact_saved_plan(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            plan = root / "saved.tfplan"
            client = terraform.Terraform(root, root / "variables.json")

            def run(args, **kwargs):
                if "plan" in args:
                    self.assertIn("-json", args)
                    self.assertIn("-refresh=false", args)
                    self.assertIn("-target=aws_s3_bucket.frontend", args)
                    plan.write_bytes(b"synthetic saved plan")
                    return subprocess.CompletedProcess(args, 0, events(("aws_s3_bucket.frontend", "create")), "")
                self.assertFalse(kwargs["check"])
                return subprocess.CompletedProcess(args, 1, "", SCHEMA_ERROR)

            with mock.patch.object(terraform, "run", side_effect=run):
                with self.assertRaises(CommandError):
                    client.show_plan(plan)
                client.plan(plan, targets=("aws_s3_bucket.frontend",))
                result = client.show_plan(plan)
                self.assertEqual(terraform.plan_actions(result), {"aws_s3_bucket.frontend": ["create"]})
                plan.write_bytes(b"changed")
                with self.assertRaisesRegex(CommandError, "saved plan changed"):
                    client.show_plan(plan)

    def test_normal_json_and_unrelated_failures_do_not_use_fallback(self) -> None:
        client = terraform.Terraform(Path("/tmp"), Path("/tmp/variables.json"))
        for result, expected in ((subprocess.CompletedProcess([], 0, '{"resource_changes":[]}', ""), {}),
                                 (subprocess.CompletedProcess([], 1, "", "permission denied"), None)):
            with mock.patch.object(terraform, "run", return_value=result):
                if expected is None:
                    with self.assertRaisesRegex(CommandError, "permission denied"):
                        client.show_plan(Path("plan.tfplan"))
                else:
                    self.assertEqual(terraform.plan_actions(client.show_plan(Path("plan.tfplan"))), expected)


if __name__ == "__main__":
    unittest.main()
