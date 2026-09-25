from __future__ import annotations

import sys
import contextlib
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import workflow


RELEASE_ID = "20260919T115959Z-0123456789ab"


def managed_manifest() -> dict[str, object]:
    function = lambda name: {
        "functionName": name,
        "version": "1",
        "qualifiedArn": f"arn:aws:lambda:ca-central-1:123:function:{name}:1",
        "codeSha256": "hash",
    }
    return {
        "releaseId": RELEASE_ID,
        "commitSha": "0" * 40,
        "createdAt": "2026-09-19T11:59:59Z",
        "database": {
            "migrationVersion": 35,
            "dirty": False,
            "migrationManifestDigest": "sha256:" + "0" * 64,
        },
        "backend": {
            "worker": function("worker"),
            "ocr": function("ocr"),
            "bootstrap": function("bootstrap"),
            "sender": function("sender"),
            "delivery": function("delivery"),
            "errorNotifier": None,
        },
        "frontend": None,
    }


class WorkflowTest(unittest.TestCase):
    def test_fresh_plan_uses_pre_alias_infrastructure_projection(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        terraform = mock.MagicMock()
        terraform.has_state.return_value = False
        terraform.show_plan.return_value = {}

        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_migration_preflight"), \
             mock.patch.object(workflow.artifacts, "build"), \
             mock.patch.object(
                 workflow,
                 "_terraform",
                 return_value=contextlib.nullcontext(Path("/tmp/vars")),
             ) as variables, \
             mock.patch.object(workflow, "Terraform", return_value=terraform), \
             mock.patch.object(workflow, "require_create_only", return_value={}):
            workflow.plan(context)

        variables.assert_called_once_with(
            context,
            True,
            use_lambda_aliases=False,
        )


    def test_state_accepts_current_worker_concurrency_after_api_cutover(self) -> None:
        context = mock.MagicMock()
        terraform = mock.MagicMock()
        outputs = {
            "database_host": "10.0.0.5",
            "worker_function_name": "worker",
            "bootstrap_function_name": "bootstrap",
            "api_id": "api",
            "raw_api_endpoint": "https://raw.example",
            "frontend_bucket_name": "bucket",
            "cloudfront_distribution_id": "distribution",
        }
        terraform.has_state.return_value = True
        terraform.output.return_value = outputs
        context.aws.concurrency.return_value = 5
        context.aws.json.return_value = {"DisableExecuteApiEndpoint": True}

        state, detected_outputs = workflow._state(context, terraform)

        self.assertEqual(state, "complete")
        self.assertEqual(detected_outputs, outputs)

    def test_state_requires_api_cutover_at_current_worker_concurrency(self) -> None:
        context = mock.MagicMock()
        terraform = mock.MagicMock()
        terraform.has_state.return_value = True
        terraform.output.return_value = {
            "database_host": "10.0.0.5",
            "worker_function_name": "worker",
            "bootstrap_function_name": "bootstrap",
            "api_id": "api",
            "raw_api_endpoint": "https://raw.example",
            "frontend_bucket_name": "bucket",
            "cloudfront_distribution_id": "distribution",
        }
        context.aws.concurrency.return_value = 5
        context.aws.json.return_value = {"DisableExecuteApiEndpoint": False}

        state, _ = workflow._state(context, terraform)

        self.assertEqual(state, "infra_ready_private")

    def test_preflight_does_not_require_nat(self) -> None:
        context = mock.MagicMock()
        context.config.deployment.account_id = "123"
        context.config.error_alerting_enabled = False
        context.config.aws.hosted_zone_name = "example.com"
        context.aws.identity.return_value = {"Account": "123"}
        context.aws.json.side_effect = [
            {"HostedZones": [{"Name": "example.com."}]},
            {"AccountLimit": {"ConcurrentExecutions": 8}},
        ]
        with mock.patch.object(workflow, "require_tools"), \
             mock.patch.object(workflow, "require_node_22"), \
             mock.patch.object(workflow, "require_supported_pnpm"):
            workflow.preflight(context, mutation=True)
        for call in context.aws.json.call_args_list:
            self.assertNotIn("describe-route-tables", call.args)

    def test_backup_status_reports_latest_backup_and_restore_marker(self) -> None:
        context = mock.MagicMock()
        context.terraform_root = Path("/repo/deployment/serverless/infrastructure/tf")
        terraform = mock.MagicMock()
        terraform.has_state.return_value = True
        terraform.output.return_value = {"postgres_backup_bucket_name": "backup-bucket"}
        context.aws.json.side_effect = [
            {"Contents": [{"Key": "daily/expense_tracker-20260831T000000Z.dump", "LastModified": "2026-08-31T00:00:00+00:00", "Size": 42}]},
            {"LastModified": "2026-08-31T01:00:00+00:00"},
        ]
        context.aws.resource_exists.return_value = True
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_terraform", return_value=contextlib.nullcontext(Path("/tmp/variables"))), \
             mock.patch.object(workflow, "Terraform", return_value=terraform), \
             mock.patch("builtins.print") as printed:
            workflow.backup_status(context)
        self.assertIn(mock.call("latest_postgres_backup=daily/expense_tracker-20260831T000000Z.dump"), printed.call_args_list)
        self.assertIn(mock.call("latest_restore_verification_time=2026-08-31T01:00:00+00:00"), printed.call_args_list)
    def test_require_node_22_accepts_the_declared_version(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="v22.23.2\n")):
            workflow.require_node_22()

    def test_require_node_22_rejects_an_older_version(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="v22.23.1\n")):
            with self.assertRaisesRegex(workflow.CommandError, "Node 22.23.2 through 22.x"):
                workflow.require_node_22()

    def test_require_node_22_rejects_an_unsupported_version(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="v23.0.0\n")):
            with self.assertRaisesRegex(workflow.CommandError, "Node 22.23.2 through 22.x"):
                workflow.require_node_22()

    def test_require_supported_pnpm_accepts_the_minimum_version(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="11.25.0\n")):
            workflow.require_supported_pnpm()

    def test_require_supported_pnpm_accepts_pnpm_12(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="12.3.4\n")):
            workflow.require_supported_pnpm()

    def test_require_supported_pnpm_rejects_an_older_version(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="11.24.9\n")):
            with self.assertRaisesRegex(workflow.CommandError, "pnpm 11.25.0 through 12.x"):
                workflow.require_supported_pnpm()

    def test_require_supported_pnpm_rejects_pnpm_13(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="13.0.0\n")):
            with self.assertRaisesRegex(workflow.CommandError, "pnpm 11.25.0 through 12.x"):
                workflow.require_supported_pnpm()

    def test_require_supported_pnpm_rejects_an_unparseable_version(self) -> None:
        with mock.patch.object(workflow, "run", return_value=mock.Mock(stdout="not-a-version\n")):
            with self.assertRaisesRegex(workflow.CommandError, "unable to parse pnpm version"):
                workflow.require_supported_pnpm()

    def test_fresh_deploy_runs_component_order_and_raw_cutover_last(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config.deployment.name_prefix = "expense-tracker"
        context.config.observability.discord_webhook_url = None
        context.config.error_alerting_enabled = False
        outputs = {
            "database_temporary_public_ipv4": "203.0.113.5",
            "database_host": "10.0.0.5",
            "worker_function_name": "worker",
            "ocr_function_name": "ocr",
            "ocr_replay_table_name": "ocr-replay",
            "bootstrap_function_name": "bootstrap",
            "sender_function_name": "sender",
            "delivery_function_name": "delivery",
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
        with contextlib.ExitStack() as stack:
            for patcher in (
                mock.patch.object(workflow, "preflight"),
                mock.patch.object(workflow, "_migration_preflight", side_effect=lambda *args: events.append("migration-policy")),
                mock.patch.object(workflow.artifacts, "build", side_effect=lambda *args: events.append("artifacts") or {"bootstrap": Path("b"), "worker": Path("w"), "ocr": Path("o"), "sender": Path("s"), "delivery": Path("d"), "notifier": Path("n")}),
                mock.patch.object(workflow, "_terraform", side_effect=lambda *args, **kwargs: contextlib.nullcontext(Path("/tmp/vars"))),
                mock.patch.object(workflow, "Terraform", FakeTerraform),
                mock.patch.object(workflow, "require_create_only", return_value={}),
                mock.patch.object(workflow, "require_cutover_only", return_value={}),
                mock.patch.object(workflow, "_assert_no_conflicts"),
                mock.patch.object(workflow, "_confirm"),
                mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")),
                mock.patch.object(workflow, "ReleaseStore"),
                mock.patch.object(workflow, "setup_database", side_effect=lambda *args: events.append("database")),
                mock.patch.object(workflow.runtime, "publish_bootstrap_release", side_effect=lambda *args: (events.append("bootstrap") or ({"functionName": "bootstrap", "version": "1"}, {}))),
                mock.patch.object(workflow.runtime, "publish_error_notifier_release", side_effect=lambda *args: events.append("notifier")),
                mock.patch.object(workflow.runtime, "publish_worker_release", side_effect=lambda *args: events.append("worker") or {"functionName": "worker", "version": "1"}),
                mock.patch.object(workflow.runtime, "publish_ocr_release", side_effect=lambda *args: events.append("ocr") or {"functionName": "ocr", "version": "1"}),
                mock.patch.object(workflow.runtime, "publish_notification_releases", side_effect=lambda *args, **kwargs: events.append("sender") or ({"functionName": "sender", "version": "1"}, {"functionName": "delivery", "version": "1"})),
                mock.patch.object(workflow.aliases, "ensure_live_aliases", return_value=({"worker": {}, "ocr": {}, "bootstrap": {}, "sender": {}, "delivery": {}, "errorNotifier": None}, True)),
                mock.patch.object(workflow.aliases, "promote_backend"),
                mock.patch.object(workflow, "_apply_infrastructure_updates", return_value=outputs),
                mock.patch.object(workflow, "_database_record", return_value={"migrationVersion": 35, "dirty": False, "migrationManifestDigest": "sha256:" + "0" * 64}),
                mock.patch.object(workflow, "_automatic_release_cleanup"),
                mock.patch.object(workflow, "verify_api", side_effect=lambda *args, **kwargs: events.append("api-verify")),
                mock.patch.object(workflow, "publish_frontend", side_effect=lambda *args: events.append("frontend") or {"snapshotPrefix": "releases/20260919T120000Z-0123456789ab/frontend", "snapshotManifestKey": "releases/20260919T120000Z-0123456789ab/frontend/snapshot.json", "snapshotDigest": "sha256:" + "1" * 64}),
                mock.patch.object(workflow, "verify_frontend"),
                mock.patch.object(workflow, "verify_session"),
                mock.patch.object(workflow, "verify_google_exchange"),
            ):
                stack.enter_context(patcher)
            context.aws.raw_endpoint.side_effect = lambda *args: events.append("raw-cutover")
            workflow.deploy(context)
        self.assertLess(events.index("artifacts"), events.index("migration-policy"))
        self.assertLess(events.index("migration-policy"), events.index("database"))
        self.assertLess(events.index("database"), events.index("bootstrap"))
        self.assertLess(events.index("bootstrap"), events.index("notifier"))
        self.assertLess(events.index("notifier"), events.index("worker"))
        self.assertLess(events.index("worker"), events.index("ocr"))
        self.assertLess(events.index("bootstrap"), events.index("worker"))
        self.assertLess(events.index("worker"), events.index("sender"))
        self.assertLess(events.index("worker"), events.index("frontend"))
        self.assertEqual(events[-2:], ["raw-cutover", "api-verify"])

    def test_fresh_deploy_records_failed_candidate_without_moving_current(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        outputs = {"bootstrap_function_name": "bootstrap"}
        store = mock.MagicMock()

        with contextlib.ExitStack() as stack:
            for patcher in (
                mock.patch.object(workflow, "preflight"),
                mock.patch.object(workflow.artifacts, "build", return_value={"bootstrap": Path("b"), "worker": Path("w"), "sender": Path("s"), "delivery": Path("d"), "notifier": Path("n")}),
                mock.patch.object(workflow, "_terraform", return_value=contextlib.nullcontext(Path("/tmp/vars"))),
                mock.patch.object(workflow, "Terraform"),
                mock.patch.object(workflow, "_state", return_value=("infra_ready_private", outputs)),
                mock.patch.object(workflow, "_migration_preflight"),
                mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")),
                mock.patch.object(workflow, "ReleaseStore", return_value=store),
                mock.patch.object(workflow.runtime, "publish_bootstrap_release", side_effect=RuntimeError("bootstrap failed")),
            ):
                stack.enter_context(patcher)
            with self.assertRaisesRegex(RuntimeError, "bootstrap failed"):
                workflow.deploy(context)

        self.assertEqual(store.put_candidate.call_count, 2)
        failed = store.put_candidate.call_args.args[0]
        self.assertEqual(failed["status"], "failed")
        self.assertEqual(failed["errorType"], "RuntimeError")
        self.assertTrue(store.put_candidate.call_args.kwargs["overwrite"])
        store.set_current.assert_not_called()

    def test_all_update_is_ordered_and_fail_fast(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config = mock.MagicMock()
        context.config.observability.discord_webhook_url = None
        context.config.error_alerting_enabled = False
        context.aws = mock.MagicMock()
        outputs = {
            "worker_function_name": "worker",
            "sender_function_name": "sender",
            "delivery_function_name": "delivery",
        }
        events: list[str] = []
        policy = mock.MagicMock(migrations=(), baseline_version=35)
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_migration_preflight", side_effect=lambda *args: events.append("migration-policy")), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", side_effect=lambda *args: events.append("state-repair")), \
             mock.patch.object(workflow, "_require_complete", return_value=outputs), \
             mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")), \
             mock.patch.object(workflow, "ReleaseStore"), \
             mock.patch.object(workflow, "_adopt_existing_release", return_value=managed_manifest()), \
             mock.patch.object(workflow.migration_policy, "validate_repository", return_value=policy), \
             mock.patch.object(workflow, "_database_record", return_value=managed_manifest()["database"]), \
             mock.patch.object(workflow, "_automatic_release_cleanup"), \
             mock.patch.object(workflow.artifacts, "build", return_value={"bootstrap": Path("bootstrap.zip"), "worker": Path("worker.zip"), "ocr": Path("ocr.zip"), "sender": Path("sender.zip"), "delivery": Path("delivery.zip"), "notifier": Path("notifier.zip")}), \
             mock.patch.object(context.aws, "pause_sender", side_effect=lambda *args: events.append("pause-sender") or 1), \
             mock.patch.object(workflow.runtime, "publish_bootstrap_release", side_effect=lambda *args: (events.append("migrations") or ({"functionName": "bootstrap"}, {}))), \
             mock.patch.object(workflow, "_apply_infrastructure_updates", side_effect=lambda *args: events.append("infrastructure")), \
             mock.patch.object(workflow.runtime, "publish_error_notifier_release", side_effect=lambda *args: events.append("notifier")), \
             mock.patch.multiple(
                 workflow.runtime,
                 publish_worker_release=mock.Mock(side_effect=lambda *args: events.append("backend") or {"functionName": "worker"}),
                 publish_ocr_release=mock.Mock(side_effect=lambda *args: events.append("ocr") or {"functionName": "ocr"}),
             ), \
             mock.patch.object(workflow.runtime, "publish_notification_releases", side_effect=lambda *args, **kwargs: events.append("sender") or ({"functionName": "sender"}, {"functionName": "delivery"})) as publish_notifications, \
             mock.patch.object(workflow.aliases, "promote_backend"), \
             mock.patch.object(workflow, "verify_api"), \
             mock.patch.object(workflow, "publish_frontend", side_effect=lambda *args: events.append("frontend") or {"snapshotPrefix": "p"}), \
             mock.patch.object(workflow, "verify_frontend"):
            workflow.update(context, "all")
        self.assertEqual(events, ["migration-policy", "state-repair", "infrastructure", "migrations", "pause-sender", "notifier", "backend", "ocr", "sender", "frontend"])
        self.assertFalse(publish_notifications.call_args.kwargs["activate"])
        context.aws.activate_notification_function.assert_called_once_with("delivery")
        context.aws.restore_sender.assert_called_once_with("sender", 1)

    def test_backend_scope_does_not_publish_frontend(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config.observability.discord_webhook_url = None
        context.config.error_alerting_enabled = False
        context.aws.pause_sender.return_value = 1
        events: list[str] = []
        policy = mock.MagicMock(migrations=(), baseline_version=35)
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_migration_preflight") as migration_preflight, \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0), \
             mock.patch.object(
                 workflow,
                 "_require_complete",
                 return_value={
                     "delivery_function_name": "delivery",
                     "sender_function_name": "sender",
                 },
             ), \
             mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")), \
             mock.patch.object(workflow, "ReleaseStore"), \
             mock.patch.object(workflow, "_adopt_existing_release", return_value=managed_manifest()), \
             mock.patch.object(workflow.migration_policy, "validate_repository", return_value=policy), \
             mock.patch.object(workflow, "_database_record", return_value=managed_manifest()["database"]), \
             mock.patch.object(workflow, "_automatic_release_cleanup"), \
             mock.patch.object(workflow.artifacts, "build", return_value={"bootstrap": Path("b"), "worker": Path("w"), "ocr": Path("o"), "sender": Path("s"), "delivery": Path("d"), "notifier": Path("n")}), \
             mock.patch.object(workflow, "_apply_infrastructure_updates", return_value=None), \
             mock.patch.object(workflow.runtime, "publish_bootstrap_release", side_effect=lambda *args: (events.append("migrations") or ({"functionName": "bootstrap"}, {}))), \
             mock.patch.object(workflow.runtime, "publish_error_notifier_release", side_effect=lambda *args: events.append("notifier")), \
             mock.patch.multiple(
                 workflow.runtime,
                 publish_worker_release=mock.Mock(side_effect=lambda *args: events.append("backend") or {"functionName": "worker"}),
                 publish_ocr_release=mock.Mock(side_effect=lambda *args: events.append("ocr") or {"functionName": "ocr"}),
             ), \
             mock.patch.object(workflow.runtime, "publish_notification_releases", side_effect=lambda *args, **kwargs: events.append("sender") or ({"functionName": "sender"}, {"functionName": "delivery"})) as publish_notifications, \
             mock.patch.object(workflow.aliases, "promote_backend"), \
             mock.patch.object(workflow, "verify_api"), \
             mock.patch.object(workflow, "publish_frontend") as publish:
            workflow.update(context, "backend")
        publish.assert_not_called()
        self.assertFalse(publish_notifications.call_args.kwargs["activate"])
        context.aws.activate_notification_function.assert_called_once_with(
            "delivery"
        )
        context.aws.restore_sender.assert_called_once_with(
            "sender",
            1,
        )
        migration_preflight.assert_called_once_with(
            context,
            "backend",
            {
                "delivery_function_name": "delivery",
                "sender_function_name": "sender",
            },
        )
        self.assertEqual(events, ["migrations", "notifier", "backend", "ocr", "sender"])

    def test_first_release_history_cutover_requires_all_scope(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        store = mock.MagicMock()
        store.get_current.return_value = None

        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_require_complete", return_value={}), \
             mock.patch.object(workflow, "_migration_preflight"), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0), \
             mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")), \
             mock.patch.object(workflow, "ReleaseStore", return_value=store), \
             mock.patch.object(workflow.artifacts, "build") as build:
            with self.assertRaisesRegex(
                workflow.CommandError,
                "first release-history cutover.*SCOPE=all",
            ):
                workflow.update(context, "frontend")

        build.assert_not_called()
        store.put_candidate.assert_not_called()

    def test_committed_candidate_cleanup_failure_is_nonfatal(self) -> None:
        store = mock.MagicMock()
        store.delete_candidate.side_effect = RuntimeError("SSM unavailable")

        with mock.patch("builtins.print") as printed:
            workflow._delete_committed_candidate(
                store,
                "20260919T120000Z-0123456789ab",
            )

        store.delete_candidate.assert_called_once_with(
            "20260919T120000Z-0123456789ab"
        )
        warning = printed.call_args.args[0]
        self.assertIn("release_committed_but_candidate_cleanup_failed", warning)
        self.assertIn("error_type=RuntimeError", warning)

    def test_incomplete_adoption_still_requires_all_scope(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        store = mock.MagicMock()
        store.get_current.return_value = {
            "currentReleaseId": "20260919T115959Z-0123456789ab"
        }
        adopted = managed_manifest()
        adopted["operation"] = "adopt"
        adopted["frontend"] = None
        store.get_release.return_value = adopted

        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_require_complete", return_value={}), \
             mock.patch.object(workflow, "_migration_preflight"), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0), \
             mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")), \
             mock.patch.object(workflow, "ReleaseStore", return_value=store), \
             mock.patch.object(workflow.artifacts, "build") as build:
            with self.assertRaisesRegex(
                workflow.CommandError,
                "first release-history cutover.*SCOPE=all",
            ):
                workflow.update(context, "backend")

        build.assert_not_called()
        store.put_candidate.assert_not_called()

    def test_migration_policy_preflight_covers_only_migration_bearing_scopes(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        manifest = mock.MagicMock()
        manifest.has_maintenance_required.return_value = False
        manifest.summary.return_value = "migration summary"
        with mock.patch.object(workflow.migration_policy, "validate_repository", return_value=manifest) as validate:
            for scope in ("migrations", "backend", "all"):
                with self.subTest(scope=scope):
                    workflow._migration_preflight(context, scope)
            workflow._migration_preflight(context, "frontend")

        self.assertEqual(validate.call_count, 3)
        self.assertEqual(manifest.has_maintenance_required.call_count, 3)

    def test_preflight_allows_applied_maintenance_migration(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.aws.function_exists.return_value = True
        context.aws.invoke_bootstrap.return_value = {
            "migration_version": 36,
            "migration_dirty": False,
        }
        manifest = mock.MagicMock()
        manifest.has_maintenance_required.return_value = True
        manifest.summary.return_value = "migration summary"
        outputs = {"bootstrap_function_name": "bootstrap"}
        with mock.patch.object(
            workflow.migration_policy,
            "validate_repository",
            return_value=manifest,
        ):
            workflow._migration_preflight(context, "backend", outputs)

        manifest.validate_pending.assert_called_once_with(36, False)
        context.aws.invoke_bootstrap.assert_called_once()

    def test_migration_policy_failure_precedes_remote_update_mutations(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config = mock.MagicMock()
        context.aws = mock.MagicMock()
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_require_complete", return_value={"bootstrap_function_name": "bootstrap"}), \
             mock.patch.object(workflow, "ReleaseStore"), \
             mock.patch.object(workflow, "_migration_preflight", side_effect=ValueError("unsafe migration")), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary") as repair, \
             mock.patch.object(workflow.artifacts, "build") as build, \
             mock.patch.object(workflow.runtime, "update_bootstrap") as bootstrap:
            with self.assertRaisesRegex(ValueError, "unsafe migration"):
                workflow.update(context, "all")

        repair.assert_not_called()
        build.assert_not_called()
        context.aws.pause_sender.assert_not_called()
        bootstrap.assert_not_called()

    def test_migration_failure_does_not_pause_sender(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config = mock.MagicMock()
        context.aws = mock.MagicMock()
        outputs = {"worker_function_name": "worker", "sender_function_name": "sender"}
        policy = mock.MagicMock(migrations=(), baseline_version=35)
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_migration_preflight"), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0), \
             mock.patch.object(workflow, "_require_complete", return_value=outputs), \
             mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")), \
             mock.patch.object(workflow, "ReleaseStore"), \
             mock.patch.object(workflow, "_adopt_existing_release", return_value=managed_manifest()), \
             mock.patch.object(workflow.migration_policy, "validate_repository", return_value=policy), \
             mock.patch.object(workflow, "_apply_infrastructure_updates"), \
             mock.patch.object(workflow.artifacts, "build", return_value={"bootstrap": Path("b")}), \
             mock.patch.object(workflow.runtime, "publish_bootstrap_release", side_effect=RuntimeError("migration failed")):
            with self.assertRaisesRegex(RuntimeError, "migration failed"):
                workflow.update(context, "backend")

        context.aws.pause_sender.assert_not_called()
        context.aws.restore_sender.assert_not_called()

    def test_backend_failure_restores_previous_sender_concurrency(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.serverless_root = Path("/repo/deployment/serverless")
        context.terraform_root = context.serverless_root / "infrastructure/tf"
        context.config = mock.MagicMock()
        context.config.error_alerting_enabled = False
        context.aws = mock.MagicMock()
        context.aws.pause_sender.return_value = 1
        outputs = {"worker_function_name": "worker", "sender_function_name": "sender"}
        built = {
            "bootstrap": Path("b"),
            "worker": Path("w"),
            "sender": Path("s"),
            "delivery": Path("d"),
            "notifier": Path("n"),
        }
        policy = mock.MagicMock(migrations=(), baseline_version=35)
        store = mock.MagicMock()
        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_migration_preflight"), \
             mock.patch.object(workflow.runtime, "repair_secret_boundary", return_value=0), \
             mock.patch.object(workflow, "_require_complete", return_value=outputs), \
             mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")), \
             mock.patch.object(workflow, "ReleaseStore", return_value=store), \
             mock.patch.object(workflow, "_adopt_existing_release", return_value=managed_manifest()), \
             mock.patch.object(workflow.migration_policy, "validate_repository", return_value=policy), \
             mock.patch.object(workflow.artifacts, "build", return_value=built), \
             mock.patch.object(workflow.runtime, "publish_bootstrap_release", return_value=({"functionName": "bootstrap"}, {})), \
             mock.patch.object(workflow, "_apply_infrastructure_updates"), \
             mock.patch.object(workflow.runtime, "publish_error_notifier_release"), \
             mock.patch.object(workflow.runtime, "publish_worker_release", side_effect=RuntimeError("worker failed")):
            with self.assertRaisesRegex(RuntimeError, "worker failed"):
                workflow.update(context, "backend")

        context.aws.pause_sender.assert_called_once_with("sender")
        context.aws.restore_sender.assert_called_once_with("sender", 1)
        self.assertEqual(store.put_candidate.call_count, 2)
        failed_candidate = store.put_candidate.call_args.args[0]
        self.assertEqual(failed_candidate["status"], "failed")
        self.assertEqual(failed_candidate["errorType"], "RuntimeError")
        self.assertTrue(store.put_candidate.call_args.kwargs["overwrite"])

    def test_backend_rollback_keeps_schema_forward_and_writes_composite_release(self) -> None:
        context = mock.MagicMock()
        context.repo_root = Path("/repo")
        context.config.deployment.name_prefix = "expense"
        outputs = {"sender_function_name": "sender"}
        current = managed_manifest()
        current["database"]["migrationVersion"] = 36  # type: ignore[index]
        target = managed_manifest()
        target["releaseId"] = "20260918T120000Z-abcdefabcdef"
        target["database"]["migrationVersion"] = 35  # type: ignore[index]
        store = mock.MagicMock()
        store.get_current.return_value = {"currentReleaseId": current["releaseId"]}
        store.get_release.side_effect = [current, target]
        store.put_release.return_value = "sha256:" + "3" * 64
        policy = mock.MagicMock()
        context.aws.pause_sender.return_value = 1

        with contextlib.ExitStack() as stack:
            for patcher in (
                mock.patch.object(workflow, "preflight"),
                mock.patch.object(workflow, "_require_complete", return_value=outputs),
                mock.patch.object(workflow, "ReleaseStore", return_value=store),
                mock.patch.object(workflow.migration_policy, "validate_repository", return_value=policy),
                mock.patch.object(workflow.migration_policy, "repository_digest", return_value=current["database"]["migrationManifestDigest"]),  # type: ignore[index]
                mock.patch.object(workflow, "_migration_state", return_value=(36, False)),
                mock.patch.object(workflow, "_confirm"),
                mock.patch.object(workflow, "repository_release_identity", return_value=("20260919T120000Z-0123456789ab", "0" * 40, "2026-09-19T12:00:00Z")),
                mock.patch.object(workflow.aliases, "promote_backend"),
                mock.patch.object(workflow, "_automatic_release_cleanup"),
                mock.patch.object(workflow, "verify_api"),
            ):
                stack.enter_context(patcher)
            workflow.activate_release(
                context,
                "rollback",
                "backend",
                str(target["releaseId"]),
            )

        policy.validate_application_rollback.assert_called_once_with(35, 36, False)
        written = store.put_release.call_args.args[0]
        self.assertEqual(written["operation"], "rollback")
        self.assertEqual(written["sourceReleaseId"], target["releaseId"])
        self.assertEqual(written["database"]["migrationVersion"], 36)
        self.assertEqual(written["backend"], target["backend"])
        self.assertEqual(written["frontend"], current["frontend"])
        context.aws.restore_sender.assert_called_once_with("sender", 1)

    def test_backend_activation_rejects_error_notifier_topology_change(self) -> None:
        context = mock.MagicMock()
        current = managed_manifest()
        target = managed_manifest()
        target["releaseId"] = "20260918T120000Z-abcdefabcdef"
        target["backend"]["errorNotifier"] = {  # type: ignore[index]
            "functionName": "notifier",
            "version": "1",
            "qualifiedArn": "arn:aws:lambda:ca-central-1:123:function:notifier:1",
            "codeSha256": "hash",
        }
        store = mock.MagicMock()
        store.get_current.return_value = {
            "currentReleaseId": current["releaseId"]
        }
        store.get_release.side_effect = [current, target]

        with mock.patch.object(workflow, "preflight"), \
             mock.patch.object(workflow, "_require_complete", return_value={}), \
             mock.patch.object(workflow, "ReleaseStore", return_value=store):
            with self.assertRaisesRegex(
                workflow.CommandError,
                "cannot change the Error Notifier.*topology",
            ):
                workflow.activate_release(
                    context,
                    "rollback",
                    "backend",
                    str(target["releaseId"]),
                )

        context.aws.pause_sender.assert_not_called()

    def test_cleanup_does_not_delete_snapshot_carried_by_retained_release(self) -> None:
        context = mock.MagicMock()
        context.config.deployment.name_prefix = "expense"
        outputs = {"frontend_bucket_name": "bucket"}
        shared_frontend = {
            "snapshotPrefix": "releases/20260901T120000Z-000000000001/frontend",
            "snapshotManifestKey": "releases/20260901T120000Z-000000000001/frontend/snapshot.json",
            "snapshotDigest": "sha256:" + "1" * 64,
        }
        current = managed_manifest()
        current["createdAt"] = "2026-09-19T12:00:00Z"
        current["frontend"] = shared_frontend
        old = managed_manifest()
        old["releaseId"] = "20260101T120000Z-000000000001"
        old["createdAt"] = "2026-01-01T12:00:00Z"
        old["frontend"] = shared_frontend
        store = mock.MagicMock()
        store.get_current.return_value = {
            "currentReleaseId": current["releaseId"],
            "previousReleaseId": None,
        }
        store.list_releases.return_value = [current, old]
        store.list_candidates.return_value = []
        store.release_path.side_effect = lambda release_id: f"/releases/{release_id}"
        context.aws.list_versions.return_value = []
        context.aws.json.return_value = {"Contents": []}

        with (
            mock.patch.object(workflow, "preflight"),
            mock.patch.object(workflow, "_require_complete", return_value=outputs),
            mock.patch.object(workflow, "ReleaseStore", return_value=store),
            mock.patch.object(workflow.aliases, "current_backend", return_value=current["backend"]),
            mock.patch.object(
                workflow,
                "load_snapshot_descriptor",
                return_value={"assets": [], "mutableFiles": []},
            ) as load_descriptor,
        ):
            workflow.cleanup_releases(
                context,
                apply=True,
                require_confirmation=False,
            )

        self.assertEqual(
            load_descriptor.call_args_list,
            [
                mock.call(context.aws, "bucket", shared_frontend),
            ],
        )

        s3_removals = [
            call.args
            for call in context.aws.call.call_args_list
            if call.args[:2] == ("s3", "rm")
        ]
        self.assertEqual(s3_removals, [])
        context.aws.delete_parameter.assert_called_once_with(
            "/releases/20260101T120000Z-000000000001"
        )
