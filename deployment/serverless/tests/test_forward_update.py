from __future__ import annotations

import contextlib
import sys
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import workflow
from test_workflow import managed_manifest


class ForwardUpdateTest(unittest.TestCase):
    def test_all_scope_can_repair_unhealthy_application_on_complete_infrastructure(self) -> None:
        context = mock.MagicMock()
        outputs = {"worker_function_name": "worker", "database_instance_id": "i-test"}
        context.aws.json.return_value = {"Reservations": [{"Instances": [{"State": {"Name": "running"}}]}]}
        for require_healthy in (True, False):
            with self.subTest(require_healthy=require_healthy), \
                 mock.patch.object(workflow, "_terraform", return_value=contextlib.nullcontext(Path("/tmp/vars"))), \
                 mock.patch.object(workflow, "Terraform"), \
                 mock.patch.object(workflow, "_state", return_value=("complete", outputs)), \
                 mock.patch.object(workflow, "verify_api", side_effect=workflow.CommandError("API unavailable")), \
                 mock.patch("builtins.print") as printed:
                if require_healthy:
                    with self.assertRaisesRegex(workflow.CommandError, "healthy complete"):
                        workflow._require_complete(context)
                else:
                    self.assertEqual(workflow._require_complete(context, require_healthy=False), outputs)
                    self.assertIn("warning=existing_application_unhealthy", printed.call_args.args[0])

    def run_update(self, failure: str | None, *, live_version: int = 39) -> None:
        context = mock.MagicMock()
        context.repo_root = ROOT.parents[1]
        context.serverless_root = ROOT
        context.terraform_root = ROOT / "infrastructure/tf"
        context.config.error_alerting_enabled = False
        context.aws.pause_sender.return_value = 1
        outputs = {
            "receipt_bucket_name": "private-receipts",
            "bootstrap_function_name": "bootstrap",
            "worker_function_name": "worker",
            "sender_function_name": "sender",
            "delivery_function_name": "delivery",
        }
        previous = managed_manifest()
        previous["database"]["migrationVersion"] = 39
        previous["frontend"] = {"snapshotPrefix": "previous"}
        store = mock.MagicMock()
        built = {name: Path(name) for name in workflow.artifacts.ALL_COMPONENTS}
        policy = workflow.migration_policy.validate_repository(context.repo_root)
        target = {"functionName": "worker", "version": "2"}
        with contextlib.ExitStack() as stack:
            for patcher in (
                mock.patch.object(workflow, "preflight"),
                mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0),
                mock.patch.object(workflow, "repository_release_identity", return_value=("20261019T120000Z-0123456789ab", "0" * 40, "2026-10-19T12:00:00Z")),
                mock.patch.object(workflow, "ReleaseStore", return_value=store),
                mock.patch.object(workflow, "_adopt_existing_release", return_value=previous),
                mock.patch.object(workflow, "_migration_state", return_value=(live_version, False)),
                mock.patch.object(workflow.artifacts, "build", return_value=built),
                mock.patch.object(workflow, "_apply_infrastructure_updates", return_value=None),
                mock.patch.object(workflow.runtime, "publish_error_notifier_release"),
                mock.patch.object(workflow.runtime, "publish_worker_release", return_value=target),
                mock.patch.object(workflow.runtime, "publish_ocr_release", return_value=target),
                mock.patch.object(workflow.runtime, "publish_notification_releases", return_value=(target, target)),
                mock.patch.object(workflow.migration_policy, "validate_repository", return_value=policy),
            ):
                stack.enter_context(patcher)
            require_complete = stack.enter_context(mock.patch.object(workflow, "_require_complete", return_value=outputs))
            bootstrap = stack.enter_context(mock.patch.object(workflow.runtime, "publish_bootstrap_release", return_value=(target, {"migration_version": 43, "migration_dirty": False})))
            promote = stack.enter_context(mock.patch.object(workflow.aliases, "promote_backend"))
            verify_api = stack.enter_context(mock.patch.object(workflow, "verify_api"))
            publish_frontend = stack.enter_context(mock.patch.object(workflow, "publish_frontend", return_value={"snapshotPrefix": "new"}))
            verify_frontend = stack.enter_context(mock.patch.object(workflow, "verify_frontend"))
            restore = stack.enter_context(mock.patch.object(workflow, "restore_frontend"))
            cleanup = stack.enter_context(mock.patch.object(workflow, "_automatic_release_cleanup"))
            rollback_check = stack.enter_context(mock.patch.object(workflow.migration_policy.Manifest, "validate_application_rollback", side_effect=AssertionError("forward updates must not require rollback")))
            stages = {"aliases": promote, "api": verify_api, "frontend_publish": publish_frontend, "frontend_verify": verify_frontend}
            if failure:
                stages[failure].side_effect = RuntimeError(f"{failure} failed")
                with self.assertRaisesRegex(RuntimeError, f"{failure} failed"):
                    workflow.update(context, "all")
            else:
                workflow.update(context, "all")

        bootstrap.assert_called_once()
        require_complete.assert_called_once_with(context, require_healthy=False)
        self.assertEqual(bootstrap.call_args.kwargs["expected_schema_when_reused"], 43)
        promote.assert_called_once()
        self.assertFalse(promote.call_args.kwargs["restore_on_failure"])
        restore.assert_not_called()
        rollback_check.assert_not_called()
        context.aws.restore_sender.assert_called_once_with("sender", 1)
        if failure:
            store.set_current.assert_not_called()
            self.assertEqual(store.put_candidate.call_args.args[0]["status"], "failed")
            cleanup.assert_not_called()
        else:
            verify_frontend.assert_called_once_with(context.config, receipt_bucket="private-receipts")
            manifest = store.put_release.call_args.args[0]
            self.assertEqual(manifest["database"]["migrationVersion"], 43)
            self.assertEqual(manifest["backend"]["worker"], target)
            self.assertEqual(manifest["frontend"]["snapshotPrefix"], "new")
            store.set_current.assert_called_once()
            cleanup.assert_called_once()

    def test_update_applies_category_migration_without_rollback_requirement(self) -> None:
        self.run_update(None)

    def test_retry_uses_already_migrated_live_state_with_stale_release_record(self) -> None:
        self.run_update(None, live_version=40)

    def test_failures_keep_partial_and_completed_activations_for_forward_fix(self) -> None:
        for stage in ("aliases", "api", "frontend_publish", "frontend_verify"):
            with self.subTest(stage=stage):
                self.run_update(stage)
