#!/usr/bin/env bash
set -euo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOOTSTRAP_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TF_DIR="${TF_DIR:-$BOOTSTRAP_ROOT/tf}"
OUTPUT_FILE="${BOOTSTRAP_OUTPUT_FILE:-}"
METADATA_FILE="$(mktemp)"
REMOVE_OUTPUT=0

if [[ -z "$OUTPUT_FILE" ]]; then
  OUTPUT_FILE="$(mktemp)"
  REMOVE_OUTPUT=1
fi

cleanup() {
  rm -f "$METADATA_FILE"
  if [[ "$REMOVE_OUTPUT" == "1" ]]; then
    rm -f "$OUTPUT_FILE"
  fi
}
trap cleanup EXIT

function_name="$(terraform -chdir="$TF_DIR" output -raw bootstrap_function_name)"
aws lambda invoke \
  --function-name "$function_name" \
  --cli-binary-format raw-in-base64-out \
  --payload '{"operation":"all"}' \
  "$OUTPUT_FILE" \
  >"$METADATA_FILE"

jq -e 'has("FunctionError") | not' "$METADATA_FILE" >/dev/null || {
  printf 'error: bootstrap Lambda reported a function error\n' >&2
  exit 1
}
jq -e '.status == "ok" and .operation == "all"' "$OUTPUT_FILE" >/dev/null || {
  printf 'error: bootstrap Lambda returned an unexpected response\n' >&2
  exit 1
}

first_admin_status="$(jq -r '.first_admin_status' "$OUTPUT_FILE")"
printf 'Bootstrap operation succeeded for %s (first admin: %s)\n' "$function_name" "$first_admin_status"
if [[ "$REMOVE_OUTPUT" == "0" ]]; then
  printf 'Response written to %s\n' "$OUTPUT_FILE"
fi
