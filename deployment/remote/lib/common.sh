#!/usr/bin/env bash

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

delete_ssm_parameter_if_exists() {
  local name="$1"
  local output
  if output="$(aws --region "$AWS_REGION" ssm delete-parameter --name "$name" 2>&1)"; then
    return
  fi
  if [[ "$output" == *"ParameterNotFound"* ]]; then
    return
  fi
  fail "failed to delete temporary SSM parameter $name: $output"
  exit 1
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

# ------------
# Host helpers
# ------------

install_edge_packages() {
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
    fail "remote deploy script must run as root"
    exit 1
  fi
}

bootstrap_first_admin_if_requested() (
  set -euo pipefail

  if [[ -z "${FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME:-}" ]]; then
    step 'Skipping first-admin bootstrap'
    exit 0
  fi

  for key in \
    FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME \
    FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME \
    FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME \
    FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME
  do
    require_env "$key"
  done

  cleanup() {
    delete_ssm_parameter_if_exists "$FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME"
    delete_ssm_parameter_if_exists "$FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME"
    delete_ssm_parameter_if_exists "$FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME"
    delete_ssm_parameter_if_exists "$FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME"
    if [[ -n "${FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME:-}" ]]; then
      delete_ssm_parameter_if_exists "$FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME"
    fi
  }
  trap cleanup EXIT

  step 'Bootstrapping first admin user'
  require_file "$APP_DIR/bin/tracker-bootstrap-first-admin"
  export FIRST_ADMIN_EMAIL
  FIRST_ADMIN_EMAIL="$(fetch_ssm_parameter "$FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME")"
  export FIRST_ADMIN_PASSWORD
  FIRST_ADMIN_PASSWORD="$(fetch_ssm_parameter "$FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME")"
  export FIRST_ADMIN_FIRSTNAME
  FIRST_ADMIN_FIRSTNAME="$(fetch_ssm_parameter "$FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME")"
  export FIRST_ADMIN_LASTNAME
  FIRST_ADMIN_LASTNAME="$(fetch_ssm_parameter "$FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME")"
  export FIRST_ADMIN_NICKNAME="${FIRST_ADMIN_NICKNAME:-}"
  if [[ -n "${FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME:-}" ]]; then
    FIRST_ADMIN_NICKNAME="$(fetch_ssm_parameter "$FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME")"
  fi

  "$APP_DIR/bin/tracker-bootstrap-first-admin"
)
