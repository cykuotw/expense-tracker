#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SERVERLESS_ROOT="$(cd "$WORKER_ROOT/.." && pwd)"
TF_DIR="${TF_DIR:-$WORKER_ROOT/tf}"
RUNTIME_ENV_FILE="${RUNTIME_ENV_FILE:-}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

for command_name in aws jq terraform; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done
[[ -n "$RUNTIME_ENV_FILE" ]] || fail 'set RUNTIME_ENV_FILE to the protected worker runtime JSON file'

TF_DIR="$TF_DIR" RUNTIME_ENV_FILE="$RUNTIME_ENV_FILE" \
  "$SERVERLESS_ROOT/scripts/check-runtime-secret-boundary.sh"

function_name="$(terraform -chdir="$TF_DIR" output -raw worker_function_name)"
aws_region="$(terraform -chdir="$TF_DIR" output -raw worker_aws_region)"
[[ -n "$function_name" ]] || fail 'Terraform worker_function_name output is empty'
[[ -n "$aws_region" ]] || fail 'Terraform worker_aws_region output is empty'

current_concurrency="$(aws lambda get-function-concurrency \
  --region "$aws_region" \
  --function-name "$function_name" \
  --query 'ReservedConcurrentExecutions' \
  --output text)"
[[ "$current_concurrency" == "0" ]] \
  || fail "expected $function_name reserved concurrency 0 before activation; got $current_concurrency"

AWS_REGION="$aws_region" REQUIRED_RESERVED_CONCURRENCY=4 \
  "$SERVERLESS_ROOT/scripts/check-lambda-concurrency.sh"

aws lambda put-function-concurrency \
  --region "$aws_region" \
  --function-name "$function_name" \
  --reserved-concurrent-executions 3 \
  >/dev/null

live_concurrency="$(aws lambda get-function-concurrency \
  --region "$aws_region" \
  --function-name "$function_name" \
  --query 'ReservedConcurrentExecutions' \
  --output text)"
[[ "$live_concurrency" == "3" ]] \
  || fail "worker activation returned reserved concurrency $live_concurrency instead of 3"

TF_DIR="$TF_DIR" RUNTIME_ENV_FILE="$RUNTIME_ENV_FILE" \
  "$SERVERLESS_ROOT/scripts/check-runtime-secret-boundary.sh"
printf 'Activated worker %s with reserved concurrency %s in %s\n' \
  "$function_name" "$live_concurrency" "$aws_region"
