#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="${TF_DIR:-$(cd "$SCRIPT_DIR/../tf" && pwd)}"
HEADERS_FILE="$(mktemp)"
RESPONSE_FILE="$(mktemp)"

cleanup() {
  rm -f "$HEADERS_FILE" "$RESPONSE_FILE"
}
trap cleanup EXIT

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

api_endpoint="$(terraform -chdir="$TF_DIR" output -raw worker_api_endpoint)"
frontend_origin="$(terraform -chdir="$TF_DIR" output -raw worker_frontend_origin)"
api_root="${api_endpoint%/}/api/v0"

preflight_status="$(curl -sS -D "$HEADERS_FILE" -o "$RESPONSE_FILE" -w '%{http_code}' \
  -X OPTIONS \
  -H "Origin: $frontend_origin" \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: Authorization, X-CSRF-Token' \
  "$api_root/auth/google/exchange")"
[[ "$preflight_status" == "204" ]] \
  || fail "Google exchange preflight returned HTTP $preflight_status, expected 204"
grep -Fqi "Access-Control-Allow-Origin: $frontend_origin" "$HEADERS_FILE" \
  || fail 'Google exchange preflight did not return the configured CORS origin'

password_status="$(curl -sS -D "$HEADERS_FILE" -o "$RESPONSE_FILE" -w '%{http_code}' \
  -X POST \
  -H "Origin: $frontend_origin" \
  -H 'Content-Type: application/json' \
  --data '{"email":"boundary@example.invalid","password":"not-a-real-password"}' \
  "$api_root/login")"
[[ "$password_status" == "403" ]] \
  || fail "password login without CSRF returned HTTP $password_status, expected application 403"
grep -Fqi "Access-Control-Allow-Origin: $frontend_origin" "$HEADERS_FILE" \
  || fail 'password boundary response did not return the configured CORS origin'

printf 'CORS preflight and password login reached the application through the unauthenticated default route\n'
