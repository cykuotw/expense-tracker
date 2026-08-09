#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$WORKER_ROOT/../../../.." && pwd)"
TF_DIR="${TF_DIR:-$WORKER_ROOT/tf}"
GOOGLE_ID_TOKEN_FILE="${GOOGLE_ID_TOKEN_FILE:-}"
COOKIE_JAR="$(mktemp)"
RESPONSE_FILE="$(mktemp)"
CURL_CONFIG="$(mktemp)"

cleanup() {
  rm -f "$COOKIE_JAR" "$RESPONSE_FILE" "$CURL_CONFIG"
}
trap cleanup EXIT
chmod 0600 "$COOKIE_JAR" "$RESPONSE_FILE" "$CURL_CONFIG"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

[[ -n "$GOOGLE_ID_TOKEN_FILE" ]] || fail 'set GOOGLE_ID_TOKEN_FILE to a protected file outside the repository'
[[ -f "$GOOGLE_ID_TOKEN_FILE" ]] || fail 'Google ID token file not found'
google_id_token_file="$(realpath "$GOOGLE_ID_TOKEN_FILE")"
case "$google_id_token_file" in
  "$REPO_ROOT" | "$REPO_ROOT"/*)
    fail 'Google ID token file must be outside the repository'
    ;;
esac
[[ "$(stat -c '%a' "$google_id_token_file")" == "600" ]] || fail 'Google ID token file must have mode 0600'

google_id_token="$(tr -d '\r\n' < "$google_id_token_file")"
[[ "$google_id_token" =~ ^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$ ]] \
  || fail 'Google ID token file must contain one JWT'
printf 'header = "Authorization: Bearer %s"\n' "$google_id_token" > "$CURL_CONFIG"
unset google_id_token

api_endpoint="$(terraform -chdir="$TF_DIR" output -raw worker_api_endpoint)"
frontend_origin="$(terraform -chdir="$TF_DIR" output -raw worker_frontend_origin)"
api_root="${api_endpoint%/}/api/v0"

csrf_status="$(curl -sS -o "$RESPONSE_FILE" -w '%{http_code}' \
  -c "$COOKIE_JAR" \
  -H "Origin: $frontend_origin" \
  "$api_root/auth/csrf")"
[[ "$csrf_status" == "200" ]] || fail "CSRF request returned HTTP $csrf_status, expected 200"
csrf_token="$(jq -er '.csrfToken | select(type == "string" and length > 0)' "$RESPONSE_FILE")" \
  || fail 'CSRF response did not contain csrfToken'

exchange_status="$(curl -sS -o "$RESPONSE_FILE" -w '%{http_code}' \
  --config "$CURL_CONFIG" \
  -X POST \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -H "Origin: $frontend_origin" \
  -H "X-CSRF-Token: $csrf_token" \
  "$api_root/auth/google/exchange")"
[[ "$exchange_status" == "200" ]] || fail "Google exchange returned HTTP $exchange_status, expected 200"

me_status="$(curl -sS -o "$RESPONSE_FILE" -w '%{http_code}' \
  -b "$COOKIE_JAR" \
  -H "Origin: $frontend_origin" \
  "$api_root/auth/me")"
[[ "$me_status" == "200" ]] || fail "auth/me returned HTTP $me_status after Google exchange, expected 200"

printf 'Google ID token exchange created a working application session\n'
