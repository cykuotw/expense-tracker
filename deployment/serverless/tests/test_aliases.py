from __future__ import annotations

import sys
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from backend import aliases
from common.command import CommandError


def outputs() -> dict[str, str]:
    return {
        "worker_function_name": "worker",
        "bootstrap_function_name": "bootstrap",
        "sender_function_name": "sender",
        "delivery_function_name": "delivery",
        "error_notifier_function_name": "",
    }


def record(name: str, version: str) -> dict[str, str]:
    return {
        "functionName": name,
        "version": version,
        "qualifiedArn": f"arn:aws:lambda:ca-central-1:123:function:{name}:{version}",
        "codeSha256": f"hash-{name}-{version}",
    }


class AliasTest(unittest.TestCase):
    def test_adopts_latest_configuration_only_when_alias_is_missing(self) -> None:
        client = mock.MagicMock()
        client.alias.side_effect = [
            None,
            {"FunctionVersion": "2", "RevisionId": "b"},
            {"FunctionVersion": "3", "RevisionId": "c"},
            {"FunctionVersion": "4", "RevisionId": "d"},
        ]
        client.publish_current_configuration.return_value = record("worker", "1")
        client.create_alias.return_value = {
            "FunctionVersion": "1",
            "RevisionId": "a",
        }
        client.function_version.side_effect = lambda name, version: record(name, version)

        backend, created = aliases.ensure_live_aliases(
            client,
            outputs(),
            "20260919T184500Z-a1b2c3d4e5f6",
        )

        self.assertTrue(created)
        self.assertEqual(backend["worker"]["version"], "1")
        client.publish_current_configuration.assert_called_once()
        client.create_alias.assert_called_once_with("worker", "1")

    def test_fresh_aliases_use_exact_preferred_versions_without_republishing(self) -> None:
        client = mock.MagicMock()
        client.alias.return_value = None
        client.create_alias.side_effect = lambda _name, version: {
            "FunctionVersion": version,
            "RevisionId": f"revision-{version}",
        }
        client.function_version.side_effect = lambda name, version: record(name, version)
        preferred = {
            "worker": record("worker", "11"),
            "bootstrap": record("bootstrap", "12"),
            "sender": record("sender", "13"),
            "delivery": record("delivery", "14"),
            "errorNotifier": None,
        }

        backend, created = aliases.ensure_live_aliases(
            client,
            outputs(),
            "20260919T184500Z-a1b2c3d4e5f6",
            preferred=preferred,
        )

        self.assertTrue(created)
        self.assertEqual(backend, preferred)
        client.publish_current_configuration.assert_not_called()
        self.assertEqual(
            client.create_alias.call_args_list,
            [
                mock.call("worker", "11"),
                mock.call("bootstrap", "12"),
                mock.call("sender", "13"),
                mock.call("delivery", "14"),
            ],
        )

    def test_partial_promotion_failure_restores_changed_aliases(self) -> None:
        client = mock.MagicMock()
        backend = {
            "worker": record("worker", "11"),
            "bootstrap": record("bootstrap", "12"),
            "sender": record("sender", "13"),
            "delivery": record("delivery", "14"),
            "errorNotifier": None,
        }
        client.function_version.side_effect = lambda name, version: record(name, version)
        client.alias.side_effect = [
            {"FunctionVersion": "2", "RevisionId": "bootstrap-old"},
            {"FunctionVersion": "1", "RevisionId": "worker-old"},
            {"FunctionVersion": "12", "RevisionId": "bootstrap-new"},
        ]
        client.update_alias.side_effect = [
            {},
            CommandError("worker promotion failed"),
            {},
        ]

        with self.assertRaisesRegex(CommandError, "worker promotion failed"):
            aliases.promote_backend(client, backend)

        self.assertEqual(
            client.update_alias.call_args_list,
            [
                mock.call("bootstrap", "12", "bootstrap-old"),
                mock.call("worker", "11", "worker-old"),
                mock.call("bootstrap", "2", "bootstrap-new"),
            ],
        )

    def test_weighted_alias_is_rejected(self) -> None:
        client = mock.MagicMock()
        client.alias.return_value = {
            "FunctionVersion": "1",
            "RevisionId": "revision",
            "RoutingConfig": {"AdditionalVersionWeights": {"2": 0.1}},
        }
        with self.assertRaisesRegex(CommandError, "weighted"):
            aliases.current_backend(client, outputs())


if __name__ == "__main__":
    unittest.main()
