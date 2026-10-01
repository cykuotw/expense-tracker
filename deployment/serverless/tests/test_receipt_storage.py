from __future__ import annotations

import contextlib
import sys
import unittest
from unittest import mock
from pathlib import Path
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import workflow
from config import Config


class ReceiptStorageDeploymentTest(unittest.TestCase):
    def test_ocr_runtime_projection_disables_receipt_storage(self) -> None:
        config = SimpleNamespace(
            frontend_origin="https://app.example.com",
            backend=SimpleNamespace(ocr_capability_secret="test-secret"),
        )
        variables = Config.ocr_environment(config, "ocr-replay", "private-receipts")["Variables"]
        self.assertEqual(variables["RECEIPT_STORAGE_ENABLED"], "false")
        self.assertEqual(variables["RECEIPT_BUCKET"], "private-receipts")
        self.assertNotIn("DB_PASSWORD", variables)

    def test_private_storage_is_in_backend_and_all_targeted_updates(self) -> None:
        addresses = {
            "aws_s3_bucket.receipt",
            "aws_s3_bucket_public_access_block.receipt",
            "aws_s3_bucket_ownership_controls.receipt",
            "aws_s3_bucket_server_side_encryption_configuration.receipt",
            "aws_s3_bucket_lifecycle_configuration.receipt",
            "aws_s3_bucket_policy.receipt",
            "aws_iam_role_policy.worker",
            "aws_cloudwatch_metric_alarm.receipt_temporary_backlog",
            "aws_cloudwatch_log_metric_filter.receipt_storage_write_failure",
            "aws_cloudwatch_metric_alarm.receipt_storage_write_failure",
        }
        for scope in ("backend", "all"):
            with self.subTest(scope=scope):
                self.assertTrue(addresses <= set(workflow._infrastructure_targets(scope)))
        self.assertTrue(addresses.isdisjoint(workflow._infrastructure_targets("frontend")))

    def test_dark_storage_creates_safely_in_both_alias_plan_paths(self) -> None:
        for aliases in (False, True):
            with self.subTest(use_lambda_aliases=aliases):
                context = mock.MagicMock()
                context.terraform_root = ROOT / "infrastructure/tf"
                context.config.error_alerting_enabled = False
                terraform = mock.MagicMock()
                terraform.show_plan.return_value = {"resource_changes": [
                    {"address": "aws_s3_bucket.receipt", "change": {"actions": ["create"]}},
                    {"address": "aws_iam_role_policy.ocr", "change": {"actions": ["update"]}},
                    {"address": "aws_iam_role_policy.worker", "change": {"actions": ["update"]}},
                    {"address": "aws_s3_bucket_policy.receipt", "change": {"actions": ["update"]}},
                ]}
                with mock.patch.object(workflow, "_terraform", return_value=contextlib.nullcontext(Path("/tmp/variables"))) as variables, mock.patch.object(workflow, "Terraform", return_value=terraform), mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0):
                    workflow._apply_infrastructure_updates(context, "backend", use_lambda_aliases=aliases)
                self.assertEqual(variables.call_args.kwargs["use_lambda_aliases"], aliases)
                self.assertIn("aws_s3_bucket.receipt", terraform.plan.call_args.kwargs["targets"])
                self.assertIn("aws_iam_role_policy.worker", terraform.plan.call_args.kwargs["targets"])
                self.assertIn("aws_s3_bucket_policy.receipt", terraform.plan.call_args.kwargs["targets"])
                terraform.apply.assert_called_once()

    def test_receipt_bucket_has_private_worker_finalization_permissions(self) -> None:
        tf = (ROOT / "infrastructure/tf/receipt_storage.tf").read_text()
        ocr = (ROOT / "infrastructure/tf/ocr.tf").read_text()
        worker = (ROOT / "infrastructure/tf/backend.tf").read_text()
        self.assertIn('resource "aws_s3_bucket_public_access_block" "receipt"', tf)
        self.assertIn('block_public_policy     = true', tf)
        self.assertIn('resource "aws_s3_bucket_server_side_encryption_configuration" "receipt"', tf)
        self.assertIn('sse_algorithm = "AES256"', tf)
        self.assertIn('"aws:SecureTransport" = "false"', tf)
        self.assertIn('"s3:GetObject", "s3:GetObjectVersion"', tf)
        self.assertIn('"aws:PrincipalArn" = aws_iam_role.worker.arn', tf)
        self.assertNotIn('resource "aws_s3_bucket_versioning" "receipt"', tf)
        self.assertIn('prefix = "temporary/"', tf)
        self.assertIn('days = 1', tf)
        self.assertIn('"${aws_s3_bucket.receipt.arn}/temporary/*"', ocr)
        self.assertNotIn('s3:GetObject', ocr)
        self.assertNotIn('s3:ListBucket', ocr)
        self.assertIn('Action = ["s3:GetObject"]', worker)
        self.assertIn('Action = ["s3:PutObject"]', worker)
        self.assertIn('Action = ["s3:DeleteObject"]', worker)
        self.assertNotIn('s3:ListBucket', worker)
        self.assertNotIn('"${aws_s3_bucket.receipt.arn}/*"', worker)


if __name__ == "__main__":
    unittest.main()
