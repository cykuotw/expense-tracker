from __future__ import annotations

import tempfile
import time
from pathlib import Path

from common.command import CommandError


def instance_state(context, outputs: dict) -> str:
    result = context.aws.json("ec2", "describe-instances", "--instance-ids", str(outputs["database_instance_id"]))
    return result["Reservations"][0]["Instances"][0]["State"]["Name"]


def require_running(context, outputs: dict) -> None:
    state = instance_state(context, outputs)
    if state != "running":
        raise CommandError(f"database is {state}; run ACTION=database-start and wait for readiness before deployment/bootstrap/backup configuration")


def wait_ready(context, outputs: dict, *, timeout: int = 300) -> None:
    started = time.monotonic()
    with tempfile.TemporaryDirectory(prefix="expense-db-ready-") as temporary:
        while time.monotonic() - started < timeout:
            try:
                # Database-backed read; HTTP /health alone is insufficient.
                remaining = timeout - (time.monotonic() - started)
                if remaining <= 0:
                    break
                context.aws.invoke_bootstrap(str(outputs["bootstrap_function_name"]), Path(temporary) / "readiness.json", "migration-state", timeout_seconds=min(15, remaining))
                print(f"database_ready=true readiness_seconds={time.monotonic() - started:.2f}", flush=True)
                return
            except CommandError:
                print("database_ready=false retry_in_seconds=10", flush=True)
                time.sleep(min(10, max(0, timeout - (time.monotonic() - started))))
    raise CommandError(f"database readiness timed out after {timeout} seconds; inspect EC2 status and PostgreSQL journal, then rerun ACTION=database-start")


def operate(context, outputs: dict, operation: str) -> None:
    state = instance_state(context, outputs)
    desired = "running" if operation == "start" else "stopped"
    if state not in {"running", "stopped", "pending", "stopping"}:
        raise CommandError(f"database state {state} does not support manual {operation}")
    if operation == "start":
        if state == "stopping":
            raise CommandError("database is stopping; wait for stopped, then rerun ACTION=database-start")
        if state == "stopped":
            context.aws.call("ec2", "start-instances", "--instance-ids", str(outputs["database_instance_id"]))
        wait_ready(context, outputs)
    else:
        if state == "pending":
            raise CommandError("database is starting; wait for running before ACTION=database-stop")
        if state == "running":
            context.aws.call("ec2", "stop-instances", "--instance-ids", str(outputs["database_instance_id"]))
        deadline = time.monotonic() + 300
        while instance_state(context, outputs) != desired:
            if time.monotonic() >= deadline:
                raise CommandError("normal database stop timed out; inspect EC2; do not force stop")
            time.sleep(10)
        print("database_state=stopped", flush=True)
