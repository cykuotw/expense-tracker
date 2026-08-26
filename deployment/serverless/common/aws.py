from __future__ import annotations

import json
import time
from pathlib import Path
from typing import Any

from common.command import CommandError, run


class AWSClient:
    def __init__(self, region: str):
        self.region = region

    def call(self, service: str, *args: str, check: bool = True) -> str:
        result = run(["aws", service, *args, "--region", self.region], check=check)
        return result.stdout

    def json(self, service: str, *args: str) -> Any:
        output = self.call(service, *args, "--output", "json")
        return json.loads(output or "null")

    def identity(self) -> dict[str, Any]:
        return self.json("sts", "get-caller-identity")

    def _probe(self, service: str, *args: str, missing: tuple[str, ...]) -> bool:
        result = run(["aws", service, *args, "--region", self.region], check=False)
        if result.returncode == 0:
            return True
        detail = result.stderr or result.stdout
        if any(marker in detail for marker in missing):
            return False
        raise CommandError(f"unable to verify AWS resource absence: {detail.strip()}")

    def function_exists(self, function_name: str) -> bool:
        return self._probe("lambda", "get-function", "--function-name", function_name, missing=("ResourceNotFoundException",))

    def resource_exists(self, service: str, *args: str, missing: tuple[str, ...]) -> bool:
        return self._probe(service, *args, missing=missing)

    def delete_function(self, function_name: str) -> None:
        result = run(["aws", "lambda", "delete-function", "--function-name", function_name, "--region", self.region], check=False)
        if result.returncode and "ResourceNotFoundException" not in (result.stderr or result.stdout):
            raise CommandError(f"failed to delete Lambda {function_name}: {(result.stderr or result.stdout).strip()}")

    def publish_environment(self, function_name: str, path: Path) -> None:
        self.call("lambda", "update-function-configuration", "--function-name", function_name, "--environment", f"file://{path}")
        self.call("lambda", "wait", "function-updated", "--function-name", function_name)

    def publish_code(self, function_name: str, artifact: Path) -> None:
        self.call("lambda", "update-function-code", "--function-name", function_name, "--zip-file", f"fileb://{artifact}", "--architectures", "arm64", "--publish")
        self.call("lambda", "wait", "function-updated", "--function-name", function_name)

    def concurrency(self, function_name: str) -> int | None:
        result = self.json("lambda", "get-function-concurrency", "--function-name", function_name)
        return result.get("ReservedConcurrentExecutions")

    def activate_worker(self, function_name: str) -> None:
        current = self.concurrency(function_name)
        if current not in (0, 3):
            raise CommandError(f"unexpected worker reserved concurrency: {current}")
        limits = self.json("lambda", "get-account-settings")["AccountLimit"]
        if int(limits["ConcurrentExecutions"]) < 4:
            raise CommandError("Lambda account concurrency must be at least 4")
        if current == 0:
            self.call("lambda", "put-function-concurrency", "--function-name", function_name, "--reserved-concurrent-executions", "3")
        if self.concurrency(function_name) != 3:
            raise CommandError("worker activation did not reach reserved concurrency 3")

    def invoke_bootstrap(self, function_name: str, response_path: Path) -> dict[str, Any]:
        metadata = self.json("lambda", "invoke", "--function-name", function_name, "--cli-binary-format", "raw-in-base64-out", "--payload", '{"operation":"all"}', str(response_path))
        if "FunctionError" in metadata:
            raise CommandError("bootstrap Lambda returned FunctionError")
        response = json.loads(response_path.read_text())
        if response.get("status") != "ok" or response.get("operation") != "all":
            raise CommandError("bootstrap Lambda returned an unexpected response")
        return response

    def raw_endpoint(self, api_id: str, disabled: bool) -> None:
        self.call("apigatewayv2", "update-api", "--api-id", api_id, "--disable-execute-api-endpoint" if disabled else "--no-disable-execute-api-endpoint")
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            api = self.json("apigatewayv2", "get-api", "--api-id", api_id)
            if bool(api.get("DisableExecuteApiEndpoint")) is disabled:
                return
            time.sleep(3)
        raise CommandError("API raw-endpoint setting did not converge")
