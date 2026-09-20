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

    def update_code(self, function_name: str, artifact: Path) -> None:
        self.call(
            "lambda",
            "update-function-code",
            "--function-name",
            function_name,
            "--zip-file",
            f"fileb://{artifact}",
            "--architectures",
            "arm64",
        )
        self.call("lambda", "wait", "function-updated", "--function-name", function_name)

    def update_environment(
        self,
        function_name: str,
        path: Path,
        *,
        description: str,
    ) -> None:
        self.call(
            "lambda",
            "update-function-configuration",
            "--function-name",
            function_name,
            "--environment",
            f"file://{path}",
            "--description",
            description,
        )
        self.call("lambda", "wait", "function-updated", "--function-name", function_name)

    def publish_version(self, function_name: str) -> dict[str, str]:
        response = self.json("lambda", "publish-version", "--function-name", function_name)
        version = str(response.get("Version", ""))
        arn = str(response.get("FunctionArn", ""))
        code_hash = str(response.get("CodeSha256", ""))
        if not version.isdigit() or version == "0" or not arn.endswith(f":{version}") or not code_hash:
            raise CommandError(f"Lambda returned an invalid published version for {function_name}")
        self.call(
            "lambda",
            "wait",
            "function-updated",
            "--function-name",
            function_name,
            "--qualifier",
            version,
        )
        return {
            "functionName": function_name,
            "version": version,
            "qualifiedArn": arn,
            "codeSha256": code_hash,
        }

    def publish_current_configuration(
        self,
        function_name: str,
        *,
        description: str,
    ) -> dict[str, str]:
        self.call(
            "lambda",
            "update-function-configuration",
            "--function-name",
            function_name,
            "--description",
            description,
        )
        self.call("lambda", "wait", "function-updated", "--function-name", function_name)
        return self.publish_version(function_name)

    def alias(self, function_name: str, alias_name: str = "live") -> dict[str, Any] | None:
        result = run(
            [
                "aws",
                "lambda",
                "get-alias",
                "--function-name",
                function_name,
                "--name",
                alias_name,
                "--output",
                "json",
                "--region",
                self.region,
            ],
            check=False,
        )
        if result.returncode:
            detail = result.stderr or result.stdout
            if "ResourceNotFoundException" in detail:
                return None
            raise CommandError(f"unable to read Lambda alias: {detail.strip()}")
        return json.loads(result.stdout or "null")

    def create_alias(
        self,
        function_name: str,
        function_version: str,
        alias_name: str = "live",
    ) -> dict[str, Any]:
        return self.json(
            "lambda",
            "create-alias",
            "--function-name",
            function_name,
            "--name",
            alias_name,
            "--function-version",
            function_version,
        )

    def update_alias(
        self,
        function_name: str,
        function_version: str,
        revision_id: str,
        alias_name: str = "live",
    ) -> dict[str, Any]:
        return self.json(
            "lambda",
            "update-alias",
            "--function-name",
            function_name,
            "--name",
            alias_name,
            "--function-version",
            function_version,
            "--revision-id",
            revision_id,
        )

    def function_version(
        self,
        function_name: str,
        version: str,
    ) -> dict[str, str]:
        response = self.json(
            "lambda",
            "get-function",
            "--function-name",
            function_name,
            "--qualifier",
            version,
        )
        configuration = response.get("Configuration", {})
        actual_version = str(configuration.get("Version", ""))
        arn = str(configuration.get("FunctionArn", ""))
        code_hash = str(configuration.get("CodeSha256", ""))
        if actual_version != version or not arn.endswith(f":{version}") or not code_hash:
            raise CommandError(
                f"Lambda version metadata is invalid for {function_name}:{version}"
            )
        return {
            "functionName": function_name,
            "version": version,
            "qualifiedArn": arn,
            "codeSha256": code_hash,
        }

    def list_versions(self, function_name: str) -> list[dict[str, Any]]:
        response = self.json(
            "lambda",
            "list-versions-by-function",
            "--function-name",
            function_name,
        )
        return list(response.get("Versions", []))

    def delete_version(self, function_name: str, version: str) -> None:
        if not version.isdigit() or version == "0":
            raise CommandError("refusing to delete a non-numbered Lambda version")
        self.call(
            "lambda",
            "delete-function",
            "--function-name",
            function_name,
            "--qualifier",
            version,
        )

    def put_parameter(self, name: str, value: str, *, overwrite: bool) -> None:
        arguments = [
            "put-parameter",
            "--name",
            name,
            "--type",
            "String",
            "--tier",
            "Standard",
            "--value",
            value,
        ]
        if overwrite:
            arguments.append("--overwrite")
        self.call("ssm", *arguments)

    def get_parameter(self, name: str) -> str | None:
        result = run(
            [
                "aws",
                "ssm",
                "get-parameter",
                "--name",
                name,
                "--output",
                "json",
                "--region",
                self.region,
            ],
            check=False,
        )
        if result.returncode:
            detail = result.stderr or result.stdout
            if "ParameterNotFound" in detail:
                return None
            raise CommandError(f"unable to read release metadata: {detail.strip()}")
        parsed = json.loads(result.stdout or "null")
        return str(parsed["Parameter"]["Value"])

    def parameters_by_path(self, path: str) -> list[dict[str, Any]]:
        response = self.json(
            "ssm",
            "get-parameters-by-path",
            "--path",
            path,
            "--recursive",
        )
        return list(response.get("Parameters", []))

    def delete_parameter(self, name: str) -> None:
        result = run(
            [
                "aws",
                "ssm",
                "delete-parameter",
                "--name",
                name,
                "--region",
                self.region,
            ],
            check=False,
        )
        if result.returncode and "ParameterNotFound" not in (
            result.stderr or result.stdout
        ):
            raise CommandError(
                f"failed to delete release metadata: {(result.stderr or result.stdout).strip()}"
            )

    def concurrency(self, function_name: str) -> int | None:
        result = self.json("lambda", "get-function-concurrency", "--function-name", function_name)
        return result.get("ReservedConcurrentExecutions")

    def activate_worker(self, function_name: str) -> None:
        current = self.concurrency(function_name)
        if current not in (0, 2, 3, 5):
            raise CommandError(f"unexpected worker reserved concurrency: {current}")
        limits = self.json("lambda", "get-account-settings")["AccountLimit"]
        if int(limits["ConcurrentExecutions"]) < 5:
            raise CommandError("Lambda account concurrency must be at least 5")
        if current != 5:
            self.call("lambda", "put-function-concurrency", "--function-name", function_name, "--reserved-concurrent-executions", "5")
        if self.concurrency(function_name) != 5:
            raise CommandError("worker activation did not reach reserved concurrency 5")

    def activate_notification_function(self, function_name: str) -> None:
        current = self.concurrency(function_name)
        if current not in (0, 1):
            raise CommandError(f"unexpected notification function reserved concurrency: {current}")
        limits = self.json("lambda", "get-account-settings")["AccountLimit"]
        if int(limits["ConcurrentExecutions"]) < 5:
            raise CommandError("Lambda account concurrency must be at least 5")
        if current == 0:
            self.call("lambda", "put-function-concurrency", "--function-name", function_name, "--reserved-concurrent-executions", "1")
        if self.concurrency(function_name) != 1:
            raise CommandError("notification function activation did not reach reserved concurrency 1")

    def pause_sender(self, function_name: str) -> int | None:
        if not self.function_exists(function_name):
            return None
        previous = self.concurrency(function_name)
        if previous not in (0, 1):
            raise CommandError(f"unexpected sender reserved concurrency: {previous}")
        if previous == 0:
            return previous
        try:
            self.call("lambda", "put-function-concurrency", "--function-name", function_name, "--reserved-concurrent-executions", "0")
            if self.concurrency(function_name) != 0:
                raise CommandError("sender pause did not reach reserved concurrency 0")
            # Existing invocations are not stopped by reserved concurrency. Both
            # supported sender versions have a maximum 60-second execution time.
            print("Waiting 60 seconds for in-flight push delivery before updating", flush=True)
            time.sleep(60)
        except BaseException as pause_error:
            try:
                self.restore_sender(function_name, previous)
            except Exception as recovery_error:
                raise CommandError(
                    "notification Sender pause failed and previous concurrency recovery also failed: "
                    f"{recovery_error}"
                ) from pause_error
            raise
        return previous

    def restore_sender(self, function_name: str, concurrency: int | None) -> None:
        if concurrency is None:
            return
        if concurrency not in (0, 1):
            raise CommandError(f"invalid previous sender reserved concurrency: {concurrency}")
        if self.concurrency(function_name) != concurrency:
            self.call(
                "lambda",
                "put-function-concurrency",
                "--function-name",
                function_name,
                "--reserved-concurrent-executions",
                str(concurrency),
            )
        if self.concurrency(function_name) != concurrency:
            raise CommandError(
                f"sender recovery did not restore reserved concurrency {concurrency}"
            )

    def invoke_bootstrap(
        self,
        function_name: str,
        response_path: Path,
        operation: str = "all",
    ) -> dict[str, Any]:
        payload = json.dumps({"operation": operation}, separators=(",", ":"))
        metadata = self.json("lambda", "invoke", "--function-name", function_name, "--cli-binary-format", "raw-in-base64-out", "--payload", payload, str(response_path))
        if "FunctionError" in metadata:
            raise CommandError("bootstrap Lambda returned FunctionError")
        response = json.loads(response_path.read_text())
        if response.get("status") != "ok" or response.get("operation") != operation:
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
