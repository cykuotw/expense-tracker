#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="${TF_DIR:-$(cd "$SCRIPT_DIR/../tf" && pwd)}"
RESPONSE_FILE="$(mktemp)"

cleanup() {
  rm -f "$RESPONSE_FILE"
}
trap cleanup EXIT

api_endpoint="$(terraform -chdir="$TF_DIR" output -raw worker_api_endpoint)"
exchange_url="${api_endpoint%/}/api/v0/auth/google/exchange"

missing_status="$(curl -sS -o "$RESPONSE_FILE" -w '%{http_code}' -X POST "$exchange_url")"
[[ "$missing_status" == "401" ]] || {
  printf 'error: missing Google token returned HTTP %s, expected 401\n' "$missing_status" >&2
  exit 1
}

malformed_status="$(curl -sS -o "$RESPONSE_FILE" -w '%{http_code}' -X POST \
  -H 'Authorization: Bearer not-a-jwt' \
  "$exchange_url")"
[[ "$malformed_status" == "401" ]] || {
  printf 'error: malformed Google token returned HTTP %s, expected 401\n' "$malformed_status" >&2
  exit 1
}

printf 'Google JWT authorizer rejected missing and malformed tokens with HTTP 401\n'
