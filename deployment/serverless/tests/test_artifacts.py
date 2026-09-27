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
        with mock.patch.object(artifacts, "run") as run:
            artifacts._go_build(
                Path("/repo"),
                "./backend/cmd/tracker-serverless",
                Path("/tmp/worker-bootstrap"),
            )

        command = run.call_args.args[0]
        self.assertIn("-buildvcs=false", command)
        self.assertIn("-trimpath", command)

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
