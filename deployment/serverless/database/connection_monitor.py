"""EC2 runtime: poll local PostgreSQL and relay state changes through private S3.

Uses only the Python standard library, psql and the already installed AWS CLI.
No credentials, query text or application rows are persisted in alert events.
"""
from __future__ import annotations

import argparse
import datetime
import fcntl
import json
import os
import subprocess
import sys
import tempfile
import time
import uuid
from pathlib import Path

THRESHOLD = 80
SQL = """SELECT json_build_object(
 'connections', count(*) FILTER (WHERE backend_type = 'client backend' AND pid <> pg_backend_pid()),
 'max_connections', current_setting('max_connections')::int)
 FROM pg_stat_activity;"""


def collect():
    result = subprocess.run(
        ["sudo", "-u", "postgres", "env", "PGCONNECT_TIMEOUT=3", "PGOPTIONS=-c statement_timeout=3000",
         "psql", "-X", "-qAt", "--dbname=postgres", "--set=ON_ERROR_STOP=1", "--command", SQL],
        capture_output=True, text=True, timeout=8, check=True,
    )
    sample = json.loads(result.stdout)
    used, maximum = sample["connections"], sample["max_connections"]
    if type(used) is not int or type(maximum) is not int or not 0 <= used < maximum <= 1_000_000:
        raise ValueError("invalid connection sample")
    return used, maximum


def transition(state, sample, now):
    """Require two fresh samples for alarm, recovery or collection failure."""
    used, maximum = sample if sample is not None else (None, None)
    desired = "CHECK_FAILED" if sample is None else "ALARM" if used * 100 >= maximum * THRESHOLD else "OK"
    streak = state.get("streak", 0) + 1 if state.get("candidate") == desired and 0 <= now - state.get("sampled_at", 0) <= 180 else 1
    state.update(candidate=desired, streak=min(streak, 2), sampled_at=now)
    if streak < 2 or desired == state.get("reported", "OK"):
        return
    event = {
        "version": 1, "event_id": str(uuid.uuid4()), "state": desired,
        "connections": used, "max_connections": maximum, "threshold_percent": THRESHOLD,
        "observed_at": datetime.datetime.fromtimestamp(now, datetime.timezone.utc).isoformat(),
    }
    state.update(reported=desired, pending=event)


def save(path, state):
    descriptor, temporary = tempfile.mkstemp(dir=path.parent, prefix=".state-")
    try:
        with os.fdopen(descriptor, "w") as stream:
            json.dump(state, stream)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)
    finally:
        Path(temporary).unlink(missing_ok=True)


def deliver(bucket, region, event):
    # Stable object keys make an ambiguous upload retry identifiable.
    result = subprocess.run(
        ["aws", "s3", "cp", "-", f"s3://{bucket}/database-alerts/{event['event_id']}.json",
         "--region", region, "--sse", "AES256", "--content-type", "application/json", "--only-show-errors"],
        input=json.dumps(event), capture_output=True, text=True, timeout=15,
        env={**os.environ, "AWS_MAX_ATTEMPTS": "2", "AWS_PAGER": ""},
    )
    if result.returncode:
        raise RuntimeError("database alert relay unavailable")


def tick(path, bucket, region):
    state = json.loads(path.read_text()) if path.exists() else {}
    if state.get("pending"):
        deliver(bucket, region, state["pending"])
        del state["pending"]
        save(path, state)
    try:
        sample = collect()
    except (OSError, subprocess.SubprocessError, ValueError, KeyError):
        sample = None
    transition(state, sample, time.time())
    save(path, state)  # Preserve an undelivered transition before attempting I/O.
    if state.get("pending"):
        deliver(bucket, region, state["pending"])
        del state["pending"]
        save(path, state)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--bucket", required=True)
    parser.add_argument("--region", required=True)
    args = parser.parse_args()
    directory = Path("/var/lib/expense-tracker-db-alerts")
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.umask(0o077)
    with (directory / "monitor.lock").open("w") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            return
        tick(directory / "state.json", args.bucket, args.region)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        # Never print subprocess stderr, payloads or environment variables.
        print("database connection monitor failed; pending delivery is retained", file=sys.stderr)
        sys.exit(1)
