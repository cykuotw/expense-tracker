#!/usr/bin/env bash
set -euo pipefail

# ------------
# Common helpers
# ------------

step() {
  printf '==> %s\n' "$1"
}

fail() {
  printf 'ERROR: %s\n' "$1" >&2
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    fail "missing required environment variable: $name"
    exit 1
  fi
}

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    fail "required file not found: $path"
    exit 1
  fi
}

env_key_has_value() {
  local file="$1"
  local key="$2"
  python3 - "$file" "$key" <<'PY2'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
key = sys.argv[2]
pattern = re.compile(rf"^{re.escape(key)}=(.*)$")

for line in path.read_text().splitlines():
    match = pattern.match(line)
    if not match:
        continue
    value = match.group(1).strip()
    if value not in ("", '""', "''"):
        raise SystemExit(0)
    break

raise SystemExit(1)
PY2
}

require_env_file_key() {
  local file="$1"
  local key="$2"
  if ! env_key_has_value "$file" "$key"; then
    fail "required runtime secret $key is missing from $file"
    exit 1
  fi
}

fetch_ssm_parameter() {
  local name="$1"
  local value
  value="$(aws --region "$AWS_REGION" ssm get-parameter --name "$name" --with-decryption --query 'Parameter.Value' --output text)"
  if [[ -z "$value" ]]; then
    fail "empty SSM parameter value for $name"
    exit 1
  fi
  printf '%s\n' "$value"
}

generate_secret() {
  python3 - <<'PY2'
import secrets

print(secrets.token_urlsafe(48))
PY2
}

fetch_or_create_ssm_parameter() {
  local name="$1"
  local value
  local output
  if value="$(aws --region "$AWS_REGION" ssm get-parameter --name "$name" --with-decryption --query 'Parameter.Value' --output text 2>/dev/null)"; then
    if [[ -z "$value" ]]; then
      fail "empty SSM parameter value for $name"
      exit 1
    fi
    printf '%s\n' "$value"
    return
  fi

  value="$(generate_secret)"
  if ! output="$(aws --region "$AWS_REGION" ssm put-parameter --name "$name" --type SecureString --value "$value" --query 'Version' --output text 2>&1)"; then
    if [[ "$output" == *"ParameterAlreadyExists"* ]]; then
      value="$(fetch_ssm_parameter "$name")"
      printf '%s\n' "$value"
      return
    fi
    fail "failed to create SSM parameter $name: $output"
    exit 1
  fi
  printf '%s\n' "$value"
}

replace_env_key() {
  local file="$1"
  local key="$2"
  local value="$3"
  python3 - "$file" "$key" "$value" <<'PY2'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
key = sys.argv[2]
value = sys.argv[3]
text = path.read_text()
escaped = value.replace("\\", "\\\\").replace('"', '\\"').replace("$", "\\$").replace("`", "\\`")
line = f'{key}="{escaped}"'
pattern = re.compile(rf"^{re.escape(key)}=.*$", re.MULTILINE)
if pattern.search(text):
    text = pattern.sub(line, text, count=1)
else:
    if text and not text.endswith("\n"):
        text += "\n"
    text += line + "\n"
path.write_text(text)
PY2
}

install_packages() {
  if command -v dnf >/dev/null 2>&1; then
    dnf install -y nginx certbot python3-certbot-nginx
  elif command -v yum >/dev/null 2>&1; then
    yum install -y nginx certbot python3-certbot-nginx
  else
    fail "neither dnf nor yum is available"
    exit 1
  fi
}

stop_service_if_running() {
  local unit_name="$1"
  if ! systemctl cat "$unit_name" >/dev/null 2>&1; then
    return
  fi
  if ! systemctl is-active --quiet "$unit_name"; then
    return
  fi

  step "Stopping $unit_name"
  systemctl stop "$unit_name"
}

require_root() {
  if [[ "$(id -u)" -ne 0 ]]; then
    fail "deploy-release.sh must run as root"
    exit 1
  fi
}

# ------------
# Init release context
# ------------

require_root

RELEASE_ROOT_INPUT="${1:-${RELEASE_ROOT:-}}"
if [[ -z "$RELEASE_ROOT_INPUT" ]]; then
  fail "usage: deploy-release.sh <release-root>"
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

: "${CERTBOT_EMAIL:=}"

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
  RUNTIME_ENV_REQUIRED_KEYS \
  API_FQDN \
  CERTBOT_ENABLED \
  CERTBOT_STAGING
do
  require_env "$key"
done

if [[ "$CERTBOT_ENABLED" == "true" && -z "$CERTBOT_EMAIL" ]]; then
  fail "CERTBOT_EMAIL must be set when CERTBOT_ENABLED=true"
  exit 1
fi

SYSTEMD_UNIT_NAME="$SYSTEMD_SERVICE_NAME"
if [[ "$SYSTEMD_UNIT_NAME" != *.service ]]; then
  SYSTEMD_UNIT_NAME="${SYSTEMD_UNIT_NAME}.service"
fi
SYSTEMD_UNIT_PATH="/etc/systemd/system/$SYSTEMD_UNIT_NAME"

CERTBOT_STAGING_FLAG=""
if [[ "$CERTBOT_STAGING" == "true" ]]; then
  CERTBOT_STAGING_FLAG="--staging"
fi

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

step 'Installing runtime packages'
install_packages

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
# Install service and nginx configuration
# ------------

step 'Installing service and nginx configuration'
require_file "$RUNTIME_ROOT/systemd/expense-tracker.service"
require_file "$RUNTIME_ROOT/nginx/expense-tracker.conf"
cp "$RUNTIME_ROOT/systemd/expense-tracker.service" "$SYSTEMD_UNIT_PATH"
rm -f /etc/nginx/conf.d/default.conf
cp "$RUNTIME_ROOT/nginx/expense-tracker.conf" /etc/nginx/conf.d/expense-tracker.conf
nginx -t
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
unset DB_ADMIN_USER DB_ADMIN_PASSWORD DB_MIGRATION_USER DB_MIGRATION_PASSWORD DB_APP_USER DB_APP_PASSWORD

# ------------
# Restart runtime services
# ------------

step "Restarting $SYSTEMD_UNIT_NAME"
systemctl enable "$SYSTEMD_UNIT_NAME"
systemctl restart "$SYSTEMD_UNIT_NAME"
systemctl status "$SYSTEMD_UNIT_NAME" --no-pager

step 'Restarting nginx'
systemctl enable nginx
systemctl restart nginx
systemctl status nginx --no-pager

# ------------
# Ensure certbot certificate
# ------------

if [[ "$CERTBOT_ENABLED" == "true" ]]; then
  step "Ensuring certbot certificate for $API_FQDN"
  mkdir -p /etc/letsencrypt/renewal-hooks/deploy
  cat <<'EOF_CERTBOT_HOOK' > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
#!/usr/bin/env bash
set -euo pipefail
systemctl reload nginx
EOF_CERTBOT_HOOK
  chmod 755 /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
  if systemctl list-unit-files | grep -q '^certbot-renew.timer'; then
    systemctl enable --now certbot-renew.timer
  elif systemctl list-unit-files | grep -q '^certbot.timer'; then
    systemctl enable --now certbot.timer
  fi
  if [[ "$CERTBOT_STAGING" != "true" ]] && certbot certificates --cert-name "$API_FQDN" 2>/dev/null | grep -q 'INVALID: TEST_CERT'; then
    certbot delete --non-interactive --cert-name "$API_FQDN"
  fi
  certbot --nginx --non-interactive --agree-tos --email "$CERTBOT_EMAIL" -d "$API_FQDN" --redirect $CERTBOT_STAGING_FLAG
  systemctl reload nginx
  systemctl status nginx --no-pager
  if systemctl list-unit-files | grep -q '^certbot-renew.timer'; then
    systemctl status certbot-renew.timer --no-pager
  elif systemctl list-unit-files | grep -q '^certbot.timer'; then
    systemctl status certbot.timer --no-pager
  fi
fi
