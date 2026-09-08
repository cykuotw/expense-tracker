from __future__ import annotations

import sys
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from common.aws import AWSClient


class AWSClientTest(unittest.TestCase):
    def test_pause_sender_drains_existing_invocations(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "function_exists", return_value=True), \
             mock.patch.object(client, "concurrency", return_value=1), \
             mock.patch.object(client, "call") as call, \
             mock.patch("common.aws.time.sleep") as sleep:
            client.pause_sender("sender")
        call.assert_called_once_with("lambda", "put-function-concurrency", "--function-name", "sender", "--reserved-concurrent-executions", "0")
        sleep.assert_called_once_with(60)

    def test_worker_activation_reduces_legacy_concurrency_to_two(self) -> None:
        client = AWSClient("ca-central-1")
        with mock.patch.object(client, "concurrency", side_effect=[3, 2]), mock.patch.object(
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
            "2",
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
