from __future__ import annotations

import sys
import contextlib
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import workflow


class WorkflowTest(unittest.TestCase):
    def test_fresh_deploy_runs_component_order_and_raw_cutover_last(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config.deployment.name_prefix = "expense-tracker"
        outputs = {
            "database_temporary_public_ipv4": "203.0.113.5",
            "database_host": "10.0.0.5",
            "worker_function_name": "worker",
            "bootstrap_function_name": "bootstrap",
            "api_id": "api",
            "raw_api_endpoint": "https://raw.example",
            "frontend_bucket_name": "bucket",
            "cloudfront_distribution_id": "distribution",
        }
        events: list[str] = []

        class FakeTerraform:
            apply_count = 0

            def __init__(self, *_args): pass
            def init(self): events.append("init")
            def validate(self): events.append("validate")
            def has_state(self): return False
            def plan(self, *_args): events.append("plan")
            def show_plan(self, *_args): return {}
            def apply(self, *_args):
                FakeTerraform.apply_count += 1
                events.append(f"apply-{FakeTerraform.apply_count}")
            def output(self): return outputs

        context.aws.json.return_value = {"DisableExecuteApiEndpoint": False}
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow.artifacts, "build", side_effect=lambda *args: events.append("artifacts")), \
             mock.patch.object(workflow, "_terraform", side_effect=lambda *args: contextlib.nullcontext(Path("/tmp/vars"))), \
             mock.patch.object(workflow, "Terraform", FakeTerraform), \
             mock.patch.object(workflow, "require_create_only", return_value={}), \
             mock.patch.object(workflow, "require_cutover_only", return_value={}), \
             mock.patch.object(workflow, "_assert_no_conflicts"), \
             mock.patch.object(workflow, "_confirm"), \
             mock.patch.object(workflow, "setup_database", side_effect=lambda *args: events.append("database")), \
             mock.patch.object(workflow.runtime, "configure_bootstrap", side_effect=lambda *args: events.append("bootstrap")), \
             mock.patch.object(workflow.runtime, "configure_worker", side_effect=lambda *args: events.append("worker")), \
             mock.patch.object(workflow, "verify_api", side_effect=lambda *args, **kwargs: events.append("api-verify")), \
             mock.patch.object(workflow, "publish_frontend", side_effect=lambda *args: events.append("frontend")), \
             mock.patch.object(workflow, "verify_frontend"), \
             mock.patch.object(workflow, "verify_session"), \
             mock.patch.object(workflow, "verify_google_exchange"):
            context.aws.raw_endpoint.side_effect = lambda *args: events.append("raw-cutover")
            workflow.deploy(context)
        self.assertLess(events.index("database"), events.index("bootstrap"))
        self.assertLess(events.index("bootstrap"), events.index("worker"))
        self.assertLess(events.index("worker"), events.index("frontend"))
        self.assertEqual(events[-2:], ["raw-cutover", "api-verify"])

    def test_all_update_is_ordered_and_fail_fast(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config = mock.MagicMock()
        context.aws = mock.MagicMock()
        outputs = {"worker_function_name": "worker"}
        events: list[str] = []
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", side_effect=lambda *args: events.append("state-repair")), \
             mock.patch.object(workflow, "_require_complete", return_value=outputs), \
             mock.patch.object(workflow.artifacts, "build", return_value={"bootstrap": Path("bootstrap.zip"), "worker": Path("worker.zip")}), \
             mock.patch.object(workflow.runtime, "update_bootstrap", side_effect=lambda *args: events.append("migrations")), \
             mock.patch.object(workflow, "_apply_infrastructure_updates", side_effect=lambda *args: events.append("infrastructure")), \
             mock.patch.object(workflow.runtime, "update_worker", side_effect=lambda *args: events.append("backend")), \
             mock.patch.object(workflow, "verify_api"), \
             mock.patch.object(workflow, "publish_frontend", side_effect=lambda *args: events.append("frontend")), \
             mock.patch.object(workflow, "verify_frontend"):
            workflow.update(context, "all")
        self.assertEqual(events, ["state-repair", "migrations", "infrastructure", "backend", "frontend"])

    def test_backend_scope_does_not_publish_frontend(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0), \
             mock.patch.object(workflow, "_require_complete", return_value={}), \
             mock.patch.object(workflow.artifacts, "build", return_value={"bootstrap": Path("b"), "worker": Path("w")}), \
             mock.patch.object(workflow, "_apply_infrastructure_updates"), \
             mock.patch.object(workflow.runtime, "update_worker"), \
             mock.patch.object(workflow, "verify_api"), \
             mock.patch.object(workflow, "publish_frontend") as publish:
            workflow.update(context, "backend")
        publish.assert_not_called()
