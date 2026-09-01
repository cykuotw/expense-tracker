from __future__ import annotations

import contextlib
import sys
import unittest
from pathlib import Path
from unittest import mock


ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import workflow


class UpdateCompatibilityTest(unittest.TestCase):
    def test_pre_update_health_check_allows_not_yet_deployed_contracts(self) -> None:
        context = mock.MagicMock()
        context.terraform_root = Path("/repo/deployment/serverless/infrastructure/tf")
        outputs = {"api_id": "api"}

        with mock.patch.object(
            workflow,
            "_terraform",
            return_value=contextlib.nullcontext(Path("/tmp/variables")),
        ), mock.patch.object(
            workflow,
            "_state",
            return_value=("complete", outputs),
        ), mock.patch.object(workflow, "verify_api") as verify_api, mock.patch.object(
            workflow,
            "verify_frontend",
        ):
            self.assertEqual(workflow._require_complete(context), outputs)

        verify_api.assert_called_once_with(
            context.config,
            require_patch_cors=False,
            require_google_register_authorizer=False,
            require_google_link_authorizer=False,
        )

    def test_all_scope_applies_only_non_destructive_infrastructure_targets(self) -> None:
        context = mock.MagicMock()
        context.terraform_root = Path("/repo/deployment/serverless/infrastructure/tf")
        terraform = mock.MagicMock()
        terraform.show_plan.return_value = {
            "resource_changes": [
                {
                    "address": "aws_apigatewayv2_route.google_register",
                    "change": {"actions": ["create"]},
                }
            ]
        }

        with mock.patch.object(
            workflow,
            "_terraform",
            return_value=contextlib.nullcontext(Path("/tmp/variables")),
        ), mock.patch.object(
            workflow,
            "Terraform",
            return_value=terraform,
        ):
            workflow._apply_infrastructure_updates(context, "all")

        terraform.init.assert_called_once_with()
        terraform.validate.assert_called_once_with()
        plan_path = terraform.plan.call_args.args[0]
        targets = terraform.plan.call_args.kwargs["targets"]
        self.assertIn("aws_apigatewayv2_route.google_register", targets)
        self.assertIn("aws_apigatewayv2_route.invitation_lookup", targets)
        self.assertIn(
            'aws_apigatewayv2_route.authenticated_mutation["expire_invitation"]',
            targets,
        )
        self.assertIn("aws_cloudfront_distribution.frontend", targets)
        self.assertIn("aws_ec2_instance_connect_endpoint.operator_access", targets)
        self.assertIn("aws_security_group.operator_access", targets)
        self.assertIn("aws_s3_bucket.postgres_backup", targets)
        self.assertIn("aws_iam_instance_profile.postgres_backup_writer", targets)
        self.assertNotIn("aws_instance.postgres", targets)
        terraform.apply.assert_called_once_with(plan_path)

    def test_backup_profile_is_attached_in_place_when_absent(self) -> None:
        context = mock.MagicMock()
        context.aws.json.side_effect = [
            {"IamInstanceProfileAssociations": []},
            {
                "IamInstanceProfileAssociations": [
                    {
                        "State": "associated",
                        "IamInstanceProfile": {
                            "Arn": "arn:aws:iam::123456789012:instance-profile/backup-writer",
                        },
                    }
                ]
            },
        ]
        outputs = {
            "database_instance_id": "i-database",
            "postgres_backup_writer_instance_profile_name": "backup-writer",
        }

        workflow._ensure_postgres_backup_profile(context, outputs)

        context.aws.call.assert_called_once_with(
            "ec2",
            "associate-iam-instance-profile",
            "--instance-id",
            "i-database",
            "--iam-instance-profile",
            "Name=backup-writer",
        )

    def test_backup_profile_does_not_replace_an_unexpected_profile(self) -> None:
        context = mock.MagicMock()
        context.aws.json.return_value = {
            "IamInstanceProfileAssociations": [
                {
                    "State": "associated",
                    "IamInstanceProfile": {
                        "Arn": "arn:aws:iam::123456789012:instance-profile/other-profile",
                    },
                }
            ]
        }
        outputs = {
            "database_instance_id": "i-database",
            "postgres_backup_writer_instance_profile_name": "backup-writer",
        }

        with self.assertRaisesRegex(workflow.CommandError, "unexpected IAM"):
            workflow._ensure_postgres_backup_profile(context, outputs)
        context.aws.call.assert_not_called()

    def test_infrastructure_update_skips_apply_when_plan_is_empty(self) -> None:
        context = mock.MagicMock()
        context.terraform_root = Path("/repo/deployment/serverless/infrastructure/tf")
        terraform = mock.MagicMock()
        terraform.show_plan.return_value = {"resource_changes": []}

        with mock.patch.object(
            workflow,
            "_terraform",
            return_value=contextlib.nullcontext(Path("/tmp/variables")),
        ), mock.patch.object(
            workflow,
            "Terraform",
            return_value=terraform,
        ):
            workflow._apply_infrastructure_updates(context, "backend")

        terraform.apply.assert_not_called()


if __name__ == "__main__":
    unittest.main()
