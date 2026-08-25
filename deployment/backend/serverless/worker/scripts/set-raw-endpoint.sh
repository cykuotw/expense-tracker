#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SERVERLESS_ROOT="$(cd "$WORKER_ROOT/.." && pwd)"
TF_DIR="${TF_DIR:-$WORKER_ROOT/tf}"
desired_state="${1:-}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

case "$desired_state" in
  enabled)
    desired_disabled="False"
    update_flag="--no-disable-execute-api-endpoint"
    ;;
  disabled)
    desired_disabled="True"
    update_flag="--disable-execute-api-endpoint"
    ;;
  *)
    fail 'usage: set-raw-endpoint.sh enabled|disabled'
    ;;
esac

for command_name in aws terraform; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done

TF_DIR="$TF_DIR" RUNTIME_ENV_FILE="${RUNTIME_ENV_FILE:-}" \
  "$SERVERLESS_ROOT/scripts/check-runtime-secret-boundary.sh" >/dev/null

api_id="$(terraform -chdir="$TF_DIR" output -raw worker_api_id)"
aws_region="$(terraform -chdir="$TF_DIR" output -raw worker_aws_region)"
[[ -n "$api_id" ]] || fail 'Terraform worker_api_id output is empty'
[[ -n "$aws_region" ]] || fail 'Terraform worker_aws_region output is empty'

current_disabled="$(aws apigatewayv2 get-api \
  --region "$aws_region" \
  --api-id "$api_id" \
  --query 'DisableExecuteApiEndpoint' \
  --output text)"

if [[ "${current_disabled,,}" != "${desired_disabled,,}" ]]; then
  aws apigatewayv2 update-api \
    --region "$aws_region" \
    --api-id "$api_id" \
    "$update_flag" \
    >/dev/null
fi

live_disabled="$(aws apigatewayv2 get-api \
  --region "$aws_region" \
  --api-id "$api_id" \
  --query 'DisableExecuteApiEndpoint' \
  --output text)"
[[ "${live_disabled,,}" == "${desired_disabled,,}" ]] \
  || fail "raw execute-api endpoint did not become $desired_state"

printf 'Raw execute-api endpoint is %s for API %s in %s\n' \
  "$desired_state" "$api_id" "$aws_region"
