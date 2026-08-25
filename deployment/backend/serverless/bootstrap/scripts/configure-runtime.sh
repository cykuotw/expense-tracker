#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOOTSTRAP_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SERVERLESS_ROOT="$(cd "$BOOTSTRAP_ROOT/.." && pwd)"
REPO_ROOT="$(cd "$BOOTSTRAP_ROOT/../../../.." && pwd)"
TF_DIR="${TF_DIR:-$BOOTSTRAP_ROOT/tf}"
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
  DB_PUBLIC_HOST DB_PORT DB_SSLMODE DB_MAINTENANCE_NAME DB_NAME
  DB_ADMIN_USER DB_ADMIN_PASSWORD DB_MIGRATION_USER DB_MIGRATION_PASSWORD
  DB_RUNTIME_USER DB_RUNTIME_PASSWORD
)
for key in "${required_keys[@]}"; do
  jq -e --arg key "$key" '(.Variables | has($key)) and (.Variables[$key] | type == "string" and length > 0)' "$runtime_env_file" >/dev/null \
    || fail "missing non-empty string Variables.$key"
done

jq -e '
  .Variables.DB_MAINTENANCE_NAME == "postgres" and
  .Variables.DB_NAME != .Variables.DB_MAINTENANCE_NAME and
  ([.Variables.DB_ADMIN_USER, .Variables.DB_MIGRATION_USER, .Variables.DB_RUNTIME_USER] | unique | length) == 3
' "$runtime_env_file" >/dev/null || fail 'maintenance/application databases must differ and all database users must be distinct'
jq -e '
  [.Variables.DB_ADMIN_PASSWORD, .Variables.DB_MIGRATION_PASSWORD, .Variables.DB_RUNTIME_PASSWORD]
  | all(startswith("REPLACE_") | not)
' "$runtime_env_file" >/dev/null || fail 'replace all password placeholders before publishing runtime configuration'

first_admin_count="$(jq '[.Variables.FIRST_ADMIN_EMAIL, .Variables.FIRST_ADMIN_PASSWORD, .Variables.FIRST_ADMIN_FIRSTNAME, .Variables.FIRST_ADMIN_LASTNAME] | map(select(type == "string" and length > 0)) | length' "$runtime_env_file")"
first_admin_any_count="$(jq '[.Variables.FIRST_ADMIN_EMAIL, .Variables.FIRST_ADMIN_PASSWORD, .Variables.FIRST_ADMIN_FIRSTNAME, .Variables.FIRST_ADMIN_LASTNAME, .Variables.FIRST_ADMIN_NICKNAME] | map(select(type == "string" and length > 0)) | length' "$runtime_env_file")"
[[ "$first_admin_any_count" == "0" || "$first_admin_count" == "4" ]] \
  || fail 'first-admin values must be omitted together or all four required values must be non-empty strings'
if [[ "$first_admin_count" == "4" ]]; then
  jq -e '.Variables.FIRST_ADMIN_PASSWORD | startswith("REPLACE_") | not' "$runtime_env_file" >/dev/null \
    || fail 'replace the first-admin password placeholder before publishing runtime configuration'
fi

expected_db_host="$(terraform -chdir="$TF_DIR" output -raw bootstrap_db_host)"
runtime_db_host="$(jq -r '.Variables.DB_PUBLIC_HOST' "$runtime_env_file")"
[[ "$runtime_db_host" == "$expected_db_host" ]] \
  || fail 'Variables.DB_PUBLIC_HOST must equal the Terraform bootstrap_db_host output'

function_name="$(terraform -chdir="$TF_DIR" output -raw bootstrap_function_name)"
aws lambda update-function-configuration \
  --function-name "$function_name" \
  --environment "file://$runtime_env_file" \
  >/dev/null
aws lambda wait function-updated --function-name "$function_name"
TF_DIR="$TF_DIR" RUNTIME_ENV_FILE="$runtime_env_file" \
  "$SERVERLESS_ROOT/scripts/check-runtime-secret-boundary.sh"
printf 'Updated bootstrap runtime environment for %s\n' "$function_name"
