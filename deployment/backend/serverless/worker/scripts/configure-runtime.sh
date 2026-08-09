#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$WORKER_ROOT/../../../.." && pwd)"
TF_DIR="${TF_DIR:-$WORKER_ROOT/tf}"
RUNTIME_ENV_FILE="${RUNTIME_ENV_FILE:-}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

[[ -n "$RUNTIME_ENV_FILE" ]] || fail 'set RUNTIME_ENV_FILE to a protected JSON file outside the repository'
[[ -f "$RUNTIME_ENV_FILE" ]] || fail "runtime environment file not found: $RUNTIME_ENV_FILE"

runtime_env_file="$(realpath "$RUNTIME_ENV_FILE")"
case "$runtime_env_file" in
  "$REPO_ROOT" | "$REPO_ROOT"/*)
    fail 'runtime environment file must be outside the repository'
    ;;
esac

[[ "$(stat -c '%a' "$runtime_env_file")" == "600" ]] || fail 'runtime environment file must have mode 0600'
jq -e '(.Variables | type == "object" and length > 0) and all(.Variables[]; type == "string")' "$runtime_env_file" >/dev/null \
  || fail 'expected JSON shape: {"Variables":{"KEY":"value"}} with string values'

required_keys=(
  MODE API_URL FRONTEND_ORIGIN CORS_ALLOWED_ORIGINS CORS_ALLOW_CREDENTIALS
  AUTH_COOKIE_DOMAIN AUTH_COOKIE_SECURE AUTH_COOKIE_SAME_SITE DB_PUBLIC_HOST
  DB_PORT DB_USER DB_PASSWORD DB_NAME DB_SSLMODE DB_MAX_OPEN_CONNS
  DB_MAX_IDLE_CONNS DB_CONN_MAX_LIFETIME_SECONDS DB_CONN_MAX_IDLE_TIME_SECONDS
  JWT_SECRET JWT_EXP REFRESH_JWT_SECRET REFRESH_JWT_EXP EXPENSES_PER_PAGE
  GOOGLE_OAUTH_ENABLED GOOGLE_CLIENT_ID GOOGLE_EXCHANGE_MODE
)
for key in "${required_keys[@]}"; do
  jq -e --arg key "$key" '(.Variables | has($key)) and (.Variables[$key] | type == "string")' "$runtime_env_file" >/dev/null \
    || fail "missing string Variables.$key"
done

required_nonempty_keys=(
  MODE API_URL FRONTEND_ORIGIN CORS_ALLOWED_ORIGINS CORS_ALLOW_CREDENTIALS
  AUTH_COOKIE_SECURE AUTH_COOKIE_SAME_SITE DB_PUBLIC_HOST DB_PORT DB_USER
  DB_PASSWORD DB_NAME DB_SSLMODE DB_MAX_OPEN_CONNS DB_MAX_IDLE_CONNS
  DB_CONN_MAX_LIFETIME_SECONDS DB_CONN_MAX_IDLE_TIME_SECONDS JWT_SECRET JWT_EXP
  REFRESH_JWT_SECRET REFRESH_JWT_EXP EXPENSES_PER_PAGE GOOGLE_OAUTH_ENABLED
  GOOGLE_CLIENT_ID GOOGLE_EXCHANGE_MODE
)
for key in "${required_nonempty_keys[@]}"; do
  jq -e --arg key "$key" '.Variables[$key] | length > 0' "$runtime_env_file" >/dev/null \
    || fail "Variables.$key must not be empty"
done

jq -e '.Variables.API_URL == "/api/v0"' "$runtime_env_file" >/dev/null \
  || fail 'Variables.API_URL must be /api/v0'
jq -e '.Variables.GOOGLE_OAUTH_ENABLED == "true"' "$runtime_env_file" >/dev/null \
  || fail 'Variables.GOOGLE_OAUTH_ENABLED must be true'
jq -e '.Variables.GOOGLE_EXCHANGE_MODE == "upstream_verified"' "$runtime_env_file" >/dev/null \
  || fail 'Variables.GOOGLE_EXCHANGE_MODE must be upstream_verified'
jq -e '.Variables.DB_MAX_OPEN_CONNS == "2" and .Variables.DB_MAX_IDLE_CONNS == "1"' "$runtime_env_file" >/dev/null \
  || fail 'worker DB pool must use DB_MAX_OPEN_CONNS=2 and DB_MAX_IDLE_CONNS=1'
jq -e '[.Variables.DB_PASSWORD, .Variables.JWT_SECRET, .Variables.REFRESH_JWT_SECRET] | all(startswith("REPLACE_") | not)' "$runtime_env_file" >/dev/null \
  || fail 'replace all secret placeholders before publishing runtime configuration'

expected_audience="$(terraform -chdir="$TF_DIR" output -raw worker_google_audience)"
runtime_audience="$(jq -r '.Variables.GOOGLE_CLIENT_ID' "$runtime_env_file")"
[[ "$runtime_audience" == "$expected_audience" ]] \
  || fail 'Variables.GOOGLE_CLIENT_ID must equal the Terraform worker_google_audience output'

expected_db_host="$(terraform -chdir="$TF_DIR" output -raw worker_db_host)"
runtime_db_host="$(jq -r '.Variables.DB_PUBLIC_HOST' "$runtime_env_file")"
[[ "$runtime_db_host" == "$expected_db_host" ]] \
  || fail 'Variables.DB_PUBLIC_HOST must equal the Terraform worker_db_host output'

function_name="$(terraform -chdir="$TF_DIR" output -raw worker_function_name)"
aws lambda update-function-configuration \
  --function-name "$function_name" \
  --environment "file://$runtime_env_file" \
  >/dev/null
aws lambda wait function-updated --function-name "$function_name"
printf 'Updated worker runtime environment for %s\n' "$function_name"
