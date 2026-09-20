from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.aws import AWSClient


class AWSClientTest(unittest.TestCase):
    def test_publish_version_returns_exact_immutable_record(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(
            client,
            "json",
            return_value={
                "Version": "7",
                "FunctionArn": "arn:aws:lambda:ca-central-1:123:function:worker:7",
                "CodeSha256": "hash",
            },
        ), mock.patch.object(client, "call") as call:
            record = client.publish_version("worker")

        self.assertEqual(record["version"], "7")
        self.assertEqual(record["codeSha256"], "hash")
        call.assert_called_once_with(
            "lambda",
            "wait",
            "function-updated",
            "--function-name",
            "worker",
            "--qualifier",
            "7",
        )

    def test_alias_update_uses_optimistic_revision(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "json", return_value={}) as call:
            client.update_alias("worker", "8", "revision-7")

        call.assert_called_once_with(
            "lambda",
            "update-alias",
            "--function-name",
            "worker",
            "--name",
            "live",
            "--function-version",
            "8",
            "--revision-id",
            "revision-7",
        )

    def test_release_parameters_explicitly_use_standard_tier(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "call") as call:
            client.put_parameter("/release", "{}", overwrite=False)
            client.put_parameter("/current", "{}", overwrite=True)

        self.assertEqual(
            call.call_args_list,
            [
                mock.call(
                    "ssm",
                    "put-parameter",
                    "--name",
                    "/release",
                    "--type",
                    "String",
                    "--tier",
                    "Standard",
                    "--value",
                    "{}",
                ),
                mock.call(
                    "ssm",
                    "put-parameter",
                    "--name",
                    "/current",
                    "--type",
                    "String",
                    "--tier",
                    "Standard",
                    "--value",
                    "{}",
                    "--overwrite",
                ),
            ],
        )

    def test_bootstrap_invocation_supports_read_only_state_operation(self) -> None:
        client = AWSClient("ca-central-1")

        def invoke(*args: str) -> dict[str, object]:
            Path(args[-1]).write_text(
                '{"status":"ok","operation":"migration-state",'
                '"migration_version":36,"migration_dirty":false}'
            )
            return {}

        with tempfile.TemporaryDirectory() as temporary, mock.patch.object(
            client,
            "json",
            side_effect=invoke,
        ) as call:
            response = client.invoke_bootstrap(
                "bootstrap",
                Path(temporary) / "response.json",
                operation="migration-state",
            )

        self.assertEqual(response["migration_version"], 36)
        self.assertIn('{"operation":"migration-state"}', call.call_args.args)

    def test_pause_sender_drains_existing_invocations(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "function_exists", return_value=True), \
             mock.patch.object(client, "concurrency", side_effect=[1, 0]), \
             mock.patch.object(client, "call") as call, \
             mock.patch("common.aws.time.sleep") as sleep:
            previous = client.pause_sender("sender")
        self.assertEqual(previous, 1)
        call.assert_called_once_with("lambda", "put-function-concurrency", "--function-name", "sender", "--reserved-concurrent-executions", "0")
        sleep.assert_called_once_with(60)

    def test_restore_sender_returns_to_previous_concurrency(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "concurrency", side_effect=[0, 1]), mock.patch.object(
            client, "call"
        ) as call:
            client.restore_sender("sender", 1)

        call.assert_called_once_with(
            "lambda",
            "put-function-concurrency",
            "--function-name",
            "sender",
            "--reserved-concurrent-executions",
            "1",
        )

    def test_pause_sender_preserves_already_paused_state(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "function_exists", return_value=True), mock.patch.object(
            client, "concurrency", return_value=0
        ), mock.patch.object(client, "call") as call:
            previous = client.pause_sender("sender")

        self.assertEqual(previous, 0)
        call.assert_not_called()

    def test_pause_failure_restores_previous_concurrency(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "function_exists", return_value=True), \
             mock.patch.object(client, "concurrency", side_effect=[1, 1, 1, 1]), \
             mock.patch.object(client, "call") as call:
            with self.assertRaisesRegex(Exception, "sender pause did not reach"):
                client.pause_sender("sender")

        self.assertEqual(
            call.call_args_list,
            [
                mock.call("lambda", "put-function-concurrency", "--function-name", "sender", "--reserved-concurrent-executions", "0"),
            ],
        )

    def test_worker_activation_raises_legacy_concurrency_to_five(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "concurrency", side_effect=[2, 5]), mock.patch.object(
            client,
            "json",
            return_value={"AccountLimit": {"ConcurrentExecutions": 5}},
        ), mock.patch.object(client, "call") as call:
            client.activate_worker("worker")

        call.assert_called_once_with(
            "lambda",
            "put-function-concurrency",
            "--function-name",
            "worker",
            "--reserved-concurrent-executions",
            "5",
        )

    def test_sender_activation_keeps_single_concurrency(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "concurrency", side_effect=[0, 1]), mock.patch.object(
            client,
            "json",
            return_value={"AccountLimit": {"ConcurrentExecutions": 5}},
        ), mock.patch.object(client, "call") as call:
            client.activate_notification_function("sender")

        call.assert_called_once_with(
            "lambda",
            "put-function-concurrency",
            "--function-name",
            "sender",
            "--reserved-concurrent-executions",
            "1",
        )


if __name__ == "__main__":
    unittest.main()
