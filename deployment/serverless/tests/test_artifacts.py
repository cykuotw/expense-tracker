from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from backend import artifacts


class ArtifactBuildTest(unittest.TestCase):
    def test_go_build_excludes_repository_wide_vcs_metadata(self) -> None:
        with tempfile.TemporaryDirectory() as temporary, mock.patch.object(artifacts, "run") as run:
            repo = Path(temporary)
            (repo / "go.mod").write_text("module test\n\ngo 1.27.3\n")
            artifacts._go_build(
                repo,
                "./backend/cmd/tracker-serverless",
                Path("/tmp/worker-bootstrap"),
            )

        command = run.call_args.args[0]
        self.assertIn("-buildvcs=false", command)
        self.assertIn("-trimpath", command)
        self.assertEqual(run.call_args.kwargs["env"]["GOTOOLCHAIN"], "go1.27.3")
        self.assertEqual(run.call_args.kwargs["env"]["GOROOT"], "")

    def test_build_uses_module_version_instead_of_inherited_toolchain(self) -> None:
        with tempfile.TemporaryDirectory() as temporary, mock.patch.dict(artifacts.os.environ, {"GOROOT": "/old/go", "GOTOOLCHAIN": "local"}), mock.patch.object(artifacts, "run") as run:
            repo = Path(temporary)
            (repo / "go.mod").write_text("module test\n\ngo 1.28.1\n")
            artifacts._go_build(repo, "./worker", repo / "bootstrap")
            self.assertEqual(run.call_args.kwargs["env"]["GOTOOLCHAIN"], "go1.28.1")
            self.assertEqual(run.call_args.kwargs["env"]["GOROOT"], "")
            self.assertEqual(artifacts.os.environ["GOROOT"], "/old/go")

    def test_missing_module_go_version_stops_before_build(self) -> None:
        with tempfile.TemporaryDirectory() as temporary, mock.patch.object(artifacts, "run") as run:
            repo = Path(temporary)
            (repo / "go.mod").write_text("module test\n")
            with self.assertRaisesRegex(artifacts.CommandError, "go.mod must declare a Go version"):
                artifacts._go_build(repo, "./worker", repo / "bootstrap")
            run.assert_not_called()

    def test_builds_only_selected_components(self) -> None:
        def fake_go_build(
            _repo_root: Path,
            package: str,
            destination: Path,
            _ldflags: str = "-s -w",
        ) -> None:
            destination.write_bytes(package.encode())

        with tempfile.TemporaryDirectory() as temporary, \
             mock.patch.object(artifacts, "validate_repository"), \
             mock.patch.object(
                 artifacts,
                 "_go_build",
                 side_effect=fake_go_build,
             ) as go_build:
            built = artifacts.build(
                Path("/repo"),
                Path(temporary) / "output",
                components=frozenset({"worker", "ocr"}),
            )

        self.assertEqual(set(built), {"worker", "ocr"})
        self.assertEqual(go_build.call_count, 2)

    def test_empty_selection_skips_repository_validation(self) -> None:
        with mock.patch.object(artifacts, "validate_repository") as validate:
            built = artifacts.build(
                Path("/repo"),
                Path("/unused"),
                components=frozenset(),
            )

        self.assertEqual(built, {})
        validate.assert_not_called()


if __name__ == "__main__":
    unittest.main()
