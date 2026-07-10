#!/usr/bin/env bash
set -euo pipefail

# ------------
# Init release context
# ------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/common.sh"

require_root

RELEASE_ROOT_INPUT="${1:-${RELEASE_ROOT:-}}"
if [[ -z "$RELEASE_ROOT_INPUT" ]]; then
  fail "usage: deploy-backend-release.sh <release-root>"
  exit 1
fi

RELEASE_ROOT="$(cd "$RELEASE_ROOT_INPUT" && pwd)"
RUNTIME_ROOT="$RELEASE_ROOT/runtime"
DEPLOY_ROOT="$RELEASE_ROOT/deploy"
RELEASE_MANIFEST_PATH="$DEPLOY_ROOT/release-manifest.env"
RUNTIME_ENV_SSM_PARAMETERS_PATH="$DEPLOY_ROOT/runtime-env-ssm.env"

if [[ ! -d "$RUNTIME_ROOT" ]]; then
  fail "runtime payload not found: $RUNTIME_ROOT"
  exit 1
fi
require_file "$RELEASE_MANIFEST_PATH"
require_file "$RUNTIME_ENV_SSM_PARAMETERS_PATH"

source "$RELEASE_MANIFEST_PATH"
source "$RUNTIME_ENV_SSM_PARAMETERS_PATH"

for key in \
  AWS_REGION \
  APP_DIR \
  BACKEND_ENV_DIR \
  BACKEND_ENV_PATH \
  BACKEND_URL \
  API_URL \
  SYSTEMD_SERVICE_NAME \
  DB_ADMIN_USERNAME \
  DB_MIGRATION_USERNAME \
  DB_ADMIN_PASSWORD_SSM_PARAMETER_NAME \
  DB_MIGRATION_PASSWORD_SSM_PARAMETER_NAME \
  RUNTIME_ENV_REQUIRED_KEYS
do
  require_env "$key"
done

SYSTEMD_UNIT_NAME="$SYSTEMD_SERVICE_NAME"
if [[ "$SYSTEMD_UNIT_NAME" != *.service ]]; then
  SYSTEMD_UNIT_NAME="${SYSTEMD_UNIT_NAME}.service"
fi
SYSTEMD_UNIT_PATH="/etc/systemd/system/$SYSTEMD_UNIT_NAME"
LOCAL_API_HEALTHCHECK_URL="http://$BACKEND_URL${API_URL%/}/health"

# ------------
# Prepare release directories
# ------------

step 'Preparing release directories'
mkdir -p "$APP_DIR" "$APP_DIR/bin" "$APP_DIR/migrations" "$BACKEND_ENV_DIR"
TMP_BACKEND_ENV_PATH="$(mktemp "$BACKEND_ENV_DIR/backend.env.XXXXXX")"
cleanup() {
  rm -f "$TMP_BACKEND_ENV_PATH"
}
trap cleanup EXIT

# ------------
# Install runtime payload
# ------------

stop_service_if_running "$SYSTEMD_UNIT_NAME"

step 'Installing runtime files'
cp -R "$RUNTIME_ROOT/bin/." "$APP_DIR/bin/"
cp -R "$RUNTIME_ROOT/migrations/." "$APP_DIR/migrations/"
chown -R expense-tracker:expense-tracker "$APP_DIR"

# ------------
# Render backend environment from SSM
# ------------

step 'Rendering backend environment from SSM'
: > "$TMP_BACKEND_ENV_PATH"
replace_env_key "$TMP_BACKEND_ENV_PATH" BACKEND_URL "$BACKEND_URL"
replace_env_key "$TMP_BACKEND_ENV_PATH" API_URL "$API_URL"

for key in $RUNTIME_ENV_REQUIRED_KEYS; do
  parameter_name_var="RUNTIME_ENV_SSM_PARAMETER_NAME__${key}"
  require_env "$parameter_name_var"
  parameter_name="${!parameter_name_var}"

  case "$key" in
    JWT_SECRET|REFRESH_JWT_SECRET|THIRD_PARTY_SESSION_SECRET)
      replace_env_key "$TMP_BACKEND_ENV_PATH" "$key" "$(fetch_or_create_ssm_parameter "$parameter_name")"
      ;;
    *)
      replace_env_key "$TMP_BACKEND_ENV_PATH" "$key" "$(fetch_ssm_parameter "$parameter_name")"
      ;;
  esac
done

for key in ${RUNTIME_ENV_OPTIONAL_KEYS:-}; do
  parameter_name_var="RUNTIME_ENV_SSM_PARAMETER_NAME__${key}"
  parameter_name="${!parameter_name_var:-}"
  if [[ -z "$parameter_name" ]]; then
    continue
  fi
  replace_env_key "$TMP_BACKEND_ENV_PATH" "$key" "$(fetch_ssm_parameter "$parameter_name")"
done

for key in $RUNTIME_ENV_REQUIRED_KEYS; do
  require_env_file_key "$TMP_BACKEND_ENV_PATH" "$key"
done

mv "$TMP_BACKEND_ENV_PATH" "$BACKEND_ENV_PATH"
chown root:root "$BACKEND_ENV_PATH"
chmod 600 "$BACKEND_ENV_PATH"

# ------------
# Resolve deploy-time database credentials
# ------------

step 'Resolving deploy-time database credentials from SSM'
DB_ADMIN_PASSWORD="$(fetch_ssm_parameter "$DB_ADMIN_PASSWORD_SSM_PARAMETER_NAME")"
DB_MIGRATION_PASSWORD="$(fetch_ssm_parameter "$DB_MIGRATION_PASSWORD_SSM_PARAMETER_NAME")"

# ------------
# Install service configuration
# ------------

step 'Installing service configuration'
require_file "$RUNTIME_ROOT/systemd/expense-tracker.service"
cp "$RUNTIME_ROOT/systemd/expense-tracker.service" "$SYSTEMD_UNIT_PATH"
systemctl daemon-reload

# ------------
# Bootstrap database and run migrations
# ------------

step 'Bootstrapping database and running migrations'
set -a
source "$BACKEND_ENV_PATH"
set +a
export DB_ADMIN_USER="$DB_ADMIN_USERNAME"
export DB_ADMIN_PASSWORD
export DB_MIGRATION_USER="$DB_MIGRATION_USERNAME"
export DB_MIGRATION_PASSWORD
export DB_APP_USER="$DB_USER"
export DB_APP_PASSWORD="$DB_PASSWORD"
cd "$APP_DIR"
"$APP_DIR/bin/tracker-db-bootstrap"
DB_USER="$DB_MIGRATION_USERNAME" DB_PASSWORD="$DB_MIGRATION_PASSWORD" "$APP_DIR/bin/tracker-migrate" up
bootstrap_first_admin_if_requested
unset DB_ADMIN_USER DB_ADMIN_PASSWORD DB_MIGRATION_USER DB_MIGRATION_PASSWORD DB_APP_USER DB_APP_PASSWORD

# ------------
# Restart backend service
# ------------

step "Restarting $SYSTEMD_UNIT_NAME"
systemctl enable "$SYSTEMD_UNIT_NAME"
systemctl restart "$SYSTEMD_UNIT_NAME"
systemctl status "$SYSTEMD_UNIT_NAME" --no-pager

# ------------
# Verify backend health
# ------------

step "Checking backend health at $LOCAL_API_HEALTHCHECK_URL"
BACKEND_HEALTH_READY=false
for attempt in $(seq 1 15); do
  if curl --fail --silent "$LOCAL_API_HEALTHCHECK_URL" >/dev/null 2>&1; then
    BACKEND_HEALTH_READY=true
    break
  fi
  sleep 2
done

if [[ "$BACKEND_HEALTH_READY" != "true" ]]; then
  curl --fail --silent --show-error "$LOCAL_API_HEALTHCHECK_URL" >/dev/null
fi
