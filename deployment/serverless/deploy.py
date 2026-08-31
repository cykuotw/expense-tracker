#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

SERVERLESS_ROOT = Path(__file__).resolve().parent
REPO_ROOT = SERVERLESS_ROOT.parents[1]
sys.path.insert(0, str(SERVERLESS_ROOT))

from common.command import CommandError  # noqa: E402
from config import ConfigError, initialize, load  # noqa: E402
from workflow import execute, make_context  # noqa: E402


DEFAULT_CONFIG = Path("~/.config/expense-tracker/deploy.json").expanduser()


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser(description="Expense Tracker unified serverless deployment")
    result.add_argument(
        "--action",
        choices=["auto", "init", "plan", "deploy", "update", "status", "backup-configure", "restore-verify", "backup-status", "backup-cleanup", "destroy"],
        default="auto",
    )
    result.add_argument("--scope", choices=["migrations", "backend", "frontend", "all"], default="all")
    return result


def main() -> int:
    arguments = parser().parse_args()
    config_path = Path(os.environ.get("DEPLOY_CONFIG_FILE", DEFAULT_CONFIG)).expanduser()
    try:
        if arguments.action == "init":
            initialize(config_path, REPO_ROOT)
            print(f"Created protected deployment config template: {config_path}")
            return 0
        config = load(config_path, REPO_ROOT)
        execute(make_context(config), arguments.action, arguments.scope)
        return 0
    except (ConfigError, CommandError, OSError, ValueError, RuntimeError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
