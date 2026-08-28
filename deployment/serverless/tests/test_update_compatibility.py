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
        self.assertEqual(
            terraform.plan.call_args.kwargs["targets"],
            (
                "aws_apigatewayv2_route.google_register",
                "aws_apigatewayv2_route.google_link",
                "aws_cloudfront_response_headers_policy.frontend_security",
                "aws_cloudfront_distribution.frontend",
            ),
        )
        terraform.apply.assert_called_once_with(plan_path)

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
