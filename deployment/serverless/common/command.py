from __future__ import annotations

import contextlib
import json
import os
import shutil
import subprocess
import tempfile
from collections.abc import Iterator, Mapping, Sequence
from pathlib import Path
from typing import Any


class CommandError(RuntimeError):
    pass


def require_tools(names: Sequence[str]) -> None:
    missing = [name for name in names if shutil.which(name) is None]
    if missing:
        raise CommandError(f"required commands not found: {', '.join(missing)}")


def run(
    args: Sequence[str], *, cwd: Path | None = None,
    env: Mapping[str, str] | None = None, input_text: str | None = None,
    check: bool = True, capture: bool = True,
) -> subprocess.CompletedProcess[str]:
    process_env = os.environ.copy()
    if env:
        process_env.update(env)
    result = subprocess.run(
        list(args), cwd=cwd, env=process_env, input=input_text,
        text=True, capture_output=capture, check=False,
    )
    if check and result.returncode:
        detail = (result.stderr or result.stdout or "command failed").strip()
        raise CommandError(f"{args[0]} failed ({result.returncode}): {detail}")
    return result


@contextlib.contextmanager
def protected_json(value: Any, *, prefix: str) -> Iterator[Path]:
    descriptor, name = tempfile.mkstemp(prefix=prefix, suffix=".json")
    path = Path(name)
    try:
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "w") as stream:
            json.dump(value, stream, separators=(",", ":"))
            stream.write("\n")
        yield path
    finally:
        path.unlink(missing_ok=True)


@contextlib.contextmanager
def deployment_lock(repo_root: Path) -> Iterator[None]:
    import fcntl

    lock_path = repo_root / ".git" / "serverless-deploy.lock"
    descriptor = os.open(lock_path, os.O_CREAT | os.O_RDWR, 0o600)
    try:
        try:
            fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise CommandError("another serverless deployment command is running") from exc
        yield
    finally:
        fcntl.flock(descriptor, fcntl.LOCK_UN)
        os.close(descriptor)
