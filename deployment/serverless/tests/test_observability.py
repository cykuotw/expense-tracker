from __future__ import annotations

import json
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "tests/fixtures/worker-observability.jsonl"


class ObservabilityInfrastructureTest(unittest.TestCase):
    def test_optional_notifier_has_narrow_runtime_and_no_fixed_cost_dependencies(self) -> None:
        source = (ROOT / "infrastructure/tf/error_alerting.tf").read_text()

        self.assertEqual(source.count("var.enable_error_alerting ? 1 : 0"), 6)
        self.assertIn("memory_size                    = 128", source)
        self.assertIn("timeout                        = 10", source)
        self.assertIn("reserved_concurrent_executions = 0", source)
        self.assertNotIn("vpc_config", source)
        self.assertIn("retention_in_days = var.worker_log_retention_days", source)
        self.assertNotIn("ssm:", source)
        self.assertIn("source_account = var.expected_account_id", source)
        self.assertIn('source_arn     = "${aws_cloudwatch_log_group.worker.arn}:*"', source)
        for fixed_cost_service in (
            "aws_cloudwatch_metric_alarm",
            "aws_sns_",
            "aws_secretsmanager_",
            "aws_nat_gateway",
            "aws_kinesis_firehose",
            "aws_kms_key",
        ):
            self.assertNotIn(fixed_cost_service, source)

    def test_subscription_filter_matches_only_canonical_alertable_failures(self) -> None:
        events = [json.loads(line) for line in FIXTURE.read_text().splitlines()]

        selected = [
            event
            for event in events
            if event.get("alertable") is True
            and (
                (
                    event.get("event") == "unexpected_http_error"
                    and isinstance(event.get("status"), int)
                    and event["status"] >= 500
                )
                or event.get("event") == "panic_recovered"
            )
        ]

        self.assertEqual(
            {event["event"] for event in selected},
            {"unexpected_http_error", "panic_recovered"},
        )
        self.assertEqual(len(selected), 2)
        source = (ROOT / "infrastructure/tf/error_alerting.tf").read_text()
        self.assertIn('$.alertable IS TRUE', source)
        self.assertIn('$.event = \\"unexpected_http_error\\"', source)
        self.assertIn('$.status >= 500', source)
        self.assertIn('$.event = \\"panic_recovered\\"', source)
        self.assertNotIn("error_notifier.name", source)

    def test_worker_retention_is_configurable_validated_and_defaults_to_three_days(self) -> None:
        variables = (ROOT / "infrastructure/tf/variables.tf").read_text()
        worker = variables.split('variable "worker_log_retention_days" {', 1)[1].split(
            'variable "bootstrap_artifact_path"', 1
        )[0]
        backend = (ROOT / "infrastructure/tf/backend.tf").read_text()
        worker_group = backend.split('resource "aws_cloudwatch_log_group" "worker" {', 1)[1].split(
            'resource "aws_cloudwatch_log_group" "bootstrap"', 1
        )[0]
        bootstrap_group = backend.split('resource "aws_cloudwatch_log_group" "bootstrap" {', 1)[1].split(
            "locals {", 1
        )[0]
        notifications = (ROOT / "infrastructure/tf/notifications.tf").read_text()

        self.assertIn("default     = 3", worker)
        self.assertIn("contains([", worker)
        self.assertIn("3653", worker)
        self.assertIn('name              = "/aws/lambda/${local.resource_prefix}-worker"', worker_group)
        self.assertIn("retention_in_days = var.worker_log_retention_days", worker_group)
        self.assertIn("retention_in_days = 7", bootstrap_group)
        self.assertEqual(notifications.count("retention_in_days = 7"), 2)
        self.assertNotIn("worker_log_retention_days", notifications)

    def test_worker_logging_permission_is_scoped_to_publishing_to_its_log_group(self) -> None:
        backend = (ROOT / "infrastructure/tf/backend.tf").read_text()
        policy = backend.split('resource "aws_iam_role_policy" "worker" {', 1)[1].split(
            'resource "aws_iam_role_policy" "bootstrap"', 1
        )[0]

        self.assertIn(
            'Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "${aws_cloudwatch_log_group.worker.arn}:*"',
            policy,
        )
        self.assertEqual(policy.count("logs:"), 2)
        self.assertNotIn("logs:*", policy)
        self.assertNotIn("logs:CreateLogGroup", policy)
        self.assertNotIn('Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "*"', policy)

    def test_investigation_queries_match_synthetic_log_fields(self) -> None:
        events = [json.loads(line) for line in FIXTURE.read_text().splitlines()]
        fields = set().union(*(event.keys() for event in events))
        guide = (ROOT / "README.md").read_text()

        expected_query_fields = {
            "@timestamp",
            "alertable",
            "api_gateway_request_id",
            "aws_request_id",
            "diagnostic_message",
            "error_category",
            "error_code",
            "event",
            "level",
            "method",
            "request_id",
            "route",
            "status",
        }
        self.assertEqual(expected_query_fields - {"@timestamp"} - fields, set())
        for field in expected_query_fields:
            self.assertIn(field, guide)
        for fragment in (
            'filter alertable = true',
            'filter request_id = "REPLACE_WITH_REQUEST_ID"',
            'filter event = "unexpected_http_error" and status >= 500',
            'stats count(*) as failures by route, error_code',
            'filter event = "panic_recovered"',
            'filter api_gateway_request_id = "REPLACE_WITH_API_GATEWAY_REQUEST_ID"',
            'or aws_request_id = "REPLACE_WITH_AWS_REQUEST_ID"',
        ):
            self.assertIn(fragment, guide)

    def test_alertable_fixture_events_exclude_sensitive_canaries(self) -> None:
        events = [json.loads(line) for line in FIXTURE.read_text().splitlines()]
        alertable = [event for event in events if event.get("alertable") is True]
        canary = next(event for event in events if event["event"] == "fixture_sensitive_canary")

        self.assertEqual(
            {event["event"] for event in alertable},
            {"unexpected_http_error", "panic_recovered"},
        )
        self.assertFalse(canary["alertable"])
        serialized_alertable = json.dumps(alertable)
        for value in (
            "CANARY_AUTHORIZATION_VALUE",
            "CANARY_RAW_REQUEST_TARGET",
            "CANARY_USER_IDENTITY",
        ):
            self.assertIn(value, json.dumps(canary))
            self.assertNotIn(value, serialized_alertable)


if __name__ == "__main__":
    unittest.main()
