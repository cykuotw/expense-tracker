#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$SCRIPT_DIR/lib/format.sh"

if [[ ! -f "$SCRIPT_DIR/lib/config.local.sh" ]]; then
  fail "Missing local config: $SCRIPT_DIR/lib/config.local.sh"
  printf 'Create it with: cp %s %s\n' \
    "$SCRIPT_DIR/lib/config.local.sh.example" \
    "$SCRIPT_DIR/lib/config.local.sh" >&2
  exit 1
fi
source "$SCRIPT_DIR/lib/config.local.sh"

source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/terraform.sh"

: "${AWS_REGION:?set AWS_REGION}"
: "${TF_VARS_FILE:?set TF_VARS_FILE}"
if [[ ! -f "$(tf_vars_file_path)" ]]; then
  fail "missing TF_VARS_FILE: $(tf_vars_file_path)"
  exit 1
fi
BACKEND_ENV_DIR="$(dirname "$BACKEND_ENV_PATH")"
STAGE_DIR="$BUILD_ROOT/backend"
BACKEND_ARCHIVE="$BUILD_ROOT/backend-release.tar.gz"
NGINX_CONFIG_TEMPLATE="deployment/nginx/expense-tracker.conf"

step 'Deploy configuration'
printf '  AWS_REGION=%s\n' "$AWS_REGION"
printf '  TF_DIR=%s\n' "$TF_DIR"
printf '  TF_VARS_FILE=%s\n' "$TF_VARS_FILE"
printf '  APP_DIR=%s\n' "$APP_DIR"
printf '  BACKEND_ENV_PATH=%s\n' "$BACKEND_ENV_PATH"
printf '  BACKEND_ENV_SOURCE_FILE=%s\n' "$BACKEND_ENV_SOURCE_FILE"
printf '  SYSTEMD_SERVICE_NAME=%s\n' "$SYSTEMD_SERVICE_NAME"
printf '  GOOS=%s\n' "$GOOS"
printf '  GOARCH=%s\n' "$GOARCH"
printf '  BUILD_ROOT=%s\n' "$BUILD_ROOT"
printf '  STAGE_DIR=%s\n' "$STAGE_DIR"
printf '  FRONTEND_DIST_DIR=%s\n' "$FRONTEND_DIST_DIR"
printf '  MIGRATIONS_SRC_DIR=%s\n' "$MIGRATIONS_SRC_DIR"
printf '  ARTIFACT_KEY=%s\n' "$ARTIFACT_KEY"
printf '  BACKEND_ARCHIVE=%s\n' "$BACKEND_ARCHIVE"
printf '  NGINX_CONFIG_TEMPLATE=%s\n' "$NGINX_CONFIG_TEMPLATE"
printf '  API_HTTP_HEALTHCHECK_URL=%s\n' "$API_HTTP_HEALTHCHECK_URL"
printf '  API_HTTPS_HEALTHCHECK_URL=%s\n' "$API_HTTPS_HEALTHCHECK_URL"
printf '  CERTBOT_ENABLED=%s\n' "$CERTBOT_ENABLED"
printf '  CERTBOT_EMAIL=%s\n' "$CERTBOT_EMAIL"
printf '  CERTBOT_STAGING=%s\n' "$CERTBOT_STAGING"

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    fail "required file not found: $path"
    exit 1
  fi
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
line = f"{key}={value}"
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

rm -rf "$BUILD_ROOT"
mkdir -p "$STAGE_DIR/bin" "$STAGE_DIR/migrations" "$STAGE_DIR/systemd" "$STAGE_DIR/nginx"

require_file "$BACKEND_ENV_SOURCE_FILE"
require_file "$NGINX_CONFIG_TEMPLATE"

step 'Applying Terraform'
terraform_cmd init -input=false
terraform_cmd apply -auto-approve -input=false

INSTANCE_ID="$(terraform_output backend_instance_id)"
FRONTEND_BUCKET="$(terraform_output frontend_bucket_name)"
ARTIFACT_BUCKET="$(terraform_output artifact_bucket_name)"
DB_HOST="$(terraform_output db_host)"
DB_PORT="$(terraform_output db_port)"
DB_NAME="$(terraform_output db_name)"
DB_USERNAME="$(terraform_output db_username)"
DB_PASSWORD_SSM_PARAMETER_NAME="$(terraform_output db_password_ssm_parameter_name)"
FRONTEND_FQDN="$(terraform_output frontend_fqdn)"
API_FQDN="$(terraform_output api_fqdn)"
CLOUDFRONT_DISTRIBUTION_ID="$(terraform_output cloudfront_distribution_id)"
S3_URI="s3://$ARTIFACT_BUCKET/$ARTIFACT_KEY"
FRONTEND_ORIGIN="https://$FRONTEND_FQDN"
API_ORIGIN="https://$API_FQDN"
if [[ -z "$API_HTTP_HEALTHCHECK_URL" ]]; then
  API_HTTP_HEALTHCHECK_URL="http://$API_FQDN/api/v0/health"
fi
if [[ -z "$API_HTTPS_HEALTHCHECK_URL" ]]; then
  API_HTTPS_HEALTHCHECK_URL="https://$API_FQDN/api/v0/health"
fi

CERTBOT_STAGING_FLAG=""
if [[ "$CERTBOT_STAGING" == "true" ]]; then
  CERTBOT_STAGING_FLAG="--staging"
fi
if [[ "$CERTBOT_ENABLED" == "true" && -z "$CERTBOT_EMAIL" ]]; then
  fail "CERTBOT_EMAIL must be set when CERTBOT_ENABLED=true"
  exit 1
fi

DB_PASSWORD="$(aws --region "$AWS_REGION" ssm get-parameter --name "$DB_PASSWORD_SSM_PARAMETER_NAME" --with-decryption --query "Parameter.Value" --output text)"
if [[ -z "$DB_PASSWORD" ]]; then
  fail "empty DB password from SSM parameter $DB_PASSWORD_SSM_PARAMETER_NAME"
  exit 1
fi

BACKEND_ENV_LOCAL="$BUILD_ROOT/backend.env"
cp "$BACKEND_ENV_SOURCE_FILE" "$BACKEND_ENV_LOCAL"
replace_env_key "$BACKEND_ENV_LOCAL" FRONTEND_ORIGIN "$FRONTEND_ORIGIN"
replace_env_key "$BACKEND_ENV_LOCAL" CORS_ALLOWED_ORIGINS "$FRONTEND_ORIGIN"
replace_env_key "$BACKEND_ENV_LOCAL" AUTH_COOKIE_DOMAIN ".$FRONTEND_FQDN"
replace_env_key "$BACKEND_ENV_LOCAL" DB_PUBLIC_HOST "$DB_HOST"
replace_env_key "$BACKEND_ENV_LOCAL" DB_PORT "$DB_PORT"
replace_env_key "$BACKEND_ENV_LOCAL" DB_USER "$DB_USERNAME"
replace_env_key "$BACKEND_ENV_LOCAL" DB_NAME "$DB_NAME"
replace_env_key "$BACKEND_ENV_LOCAL" DB_PASSWORD "$DB_PASSWORD"
replace_env_key "$BACKEND_ENV_LOCAL" DB_SSLMODE "require"
replace_env_key "$BACKEND_ENV_LOCAL" GOOGLE_CALLBACK_URL "$API_ORIGIN/api/v0/auth/google/callback"

step 'Building frontend'
VITE_API_ORIGIN="$API_ORIGIN" VITE_API_PATH="/api/v0" make build-frontend

step 'Building backend'
make build-deploy-backend BUILD_DIR="$STAGE_DIR/bin" GOOS="$GOOS" GOARCH="$GOARCH"

cp -R "$MIGRATIONS_SRC_DIR/." "$STAGE_DIR/migrations/"
cp deployment/systemd/expense-tracker.service "$STAGE_DIR/systemd/expense-tracker.service"
python3 - "$NGINX_CONFIG_TEMPLATE" "$STAGE_DIR/nginx/expense-tracker.conf" "$API_FQDN" <<'PY2'
import pathlib
import sys

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
api_fqdn = sys.argv[3]
text = source.read_text()
target.write_text(text.replace("__API_FQDN__", api_fqdn))
PY2
tar -C "$STAGE_DIR" -czf "$BACKEND_ARCHIVE" .

step "Uploading frontend dist to s3://$FRONTEND_BUCKET"
aws --region "$AWS_REGION" s3 sync "$FRONTEND_DIST_DIR/" "s3://$FRONTEND_BUCKET/" --delete

step "Invalidating CloudFront distribution $CLOUDFRONT_DISTRIBUTION_ID"
aws --region "$AWS_REGION" cloudfront create-invalidation   --distribution-id "$CLOUDFRONT_DISTRIBUTION_ID"   --paths '/*' >/dev/null

step "Uploading backend bundle to $S3_URI"
aws --region "$AWS_REGION" s3 cp "$BACKEND_ARCHIVE" "$S3_URI"

BACKEND_ENV_B64="$(base64 < "$BACKEND_ENV_LOCAL" | tr -d '\n')"

read -r -d '' REMOTE_COMMANDS <<EOF_REMOTE || true
set -euo pipefail
sudo mkdir -p '$APP_DIR'
sudo mkdir -p '$BACKEND_ENV_DIR'
aws --region '$AWS_REGION' s3 cp '$S3_URI' /tmp/backend-release.tar.gz
sudo tar -xzf /tmp/backend-release.tar.gz -C '$APP_DIR'
rm -f /tmp/backend-release.tar.gz
printf '%s' '$BACKEND_ENV_B64' | base64 -d | sudo tee '$BACKEND_ENV_PATH' >/dev/null
sudo chown root:root '$BACKEND_ENV_PATH'
sudo chmod 600 '$BACKEND_ENV_PATH'
sudo chown -R expense-tracker:expense-tracker '$APP_DIR'
sudo cp '$APP_DIR/systemd/expense-tracker.service' /etc/systemd/system/expense-tracker.service
if command -v dnf >/dev/null 2>&1; then
  sudo dnf install -y nginx certbot python3-certbot-nginx
else
  sudo yum install -y nginx certbot python3-certbot-nginx
fi
sudo rm -f /etc/nginx/conf.d/default.conf
sudo cp '$APP_DIR/nginx/expense-tracker.conf' /etc/nginx/conf.d/expense-tracker.conf
sudo nginx -t
sudo systemctl daemon-reload
set -a
source '$BACKEND_ENV_PATH'
set +a
cd '$APP_DIR'
'$APP_DIR/bin/tracker-migrate' up
sudo systemctl enable '$SYSTEMD_SERVICE_NAME'
sudo systemctl restart '$SYSTEMD_SERVICE_NAME'
sudo systemctl status '$SYSTEMD_SERVICE_NAME' --no-pager
sudo systemctl enable nginx
sudo systemctl restart nginx
sudo systemctl status nginx --no-pager
if [[ '$CERTBOT_ENABLED' == 'true' ]]; then
  sudo mkdir -p /etc/letsencrypt/renewal-hooks/deploy
  cat <<'EOF_CERTBOT_HOOK' | sudo tee /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh >/dev/null
#!/usr/bin/env bash
set -euo pipefail
systemctl reload nginx
EOF_CERTBOT_HOOK
  sudo chmod 755 /etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh
  if systemctl list-unit-files | grep -q '^certbot-renew.timer'; then
    sudo systemctl enable --now certbot-renew.timer
  elif systemctl list-unit-files | grep -q '^certbot.timer'; then
    sudo systemctl enable --now certbot.timer
  fi
  if [[ '$CERTBOT_STAGING' != 'true' ]] && sudo certbot certificates --cert-name '$API_FQDN' 2>/dev/null | grep -q 'INVALID: TEST_CERT'; then
    sudo certbot delete --non-interactive --cert-name '$API_FQDN'
  fi
  sudo certbot --nginx --non-interactive --agree-tos --email '$CERTBOT_EMAIL' -d '$API_FQDN' --redirect $CERTBOT_STAGING_FLAG
  sudo systemctl reload nginx
  sudo systemctl status nginx --no-pager
  if systemctl list-unit-files | grep -q '^certbot-renew.timer'; then
    sudo systemctl status certbot-renew.timer --no-pager
  elif systemctl list-unit-files | grep -q '^certbot.timer'; then
    sudo systemctl status certbot.timer --no-pager
  fi
fi
EOF_REMOTE

step 'Running remote deploy through SSM'
SSM_PARAMETERS="$(REMOTE_COMMANDS="$REMOTE_COMMANDS" python3 - <<'PY2'
import json
import os

commands = [line for line in os.environ["REMOTE_COMMANDS"].splitlines() if line.strip()]
print(json.dumps({"commands": commands}))
PY2
)"
COMMAND_ID="$({
  aws --region "$AWS_REGION" ssm send-command     --instance-ids "$INSTANCE_ID"     --document-name AWS-RunShellScript     --comment 'expense-tracker deploy'     --parameters "$SSM_PARAMETERS"     --query 'Command.CommandId'     --output text
} )"

set +e
aws --region "$AWS_REGION" ssm wait command-executed --command-id "$COMMAND_ID" --instance-id "$INSTANCE_ID"
WAIT_EXIT=$?
SSM_STATUS="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$COMMAND_ID" --instance-id "$INSTANCE_ID" --query 'Status' --output text)"
STATUS_EXIT=$?
SSM_RESPONSE_CODE="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$COMMAND_ID" --instance-id "$INSTANCE_ID" --query 'ResponseCode' --output text)"
RESPONSE_CODE_EXIT=$?
SSM_STDOUT="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$COMMAND_ID" --instance-id "$INSTANCE_ID" --query 'StandardOutputContent' --output text)"
STDOUT_EXIT=$?
SSM_STDERR="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$COMMAND_ID" --instance-id "$INSTANCE_ID" --query 'StandardErrorContent' --output text)"
STDERR_EXIT=$?
set -e

if [[ $STDOUT_EXIT -eq 0 && -n "$SSM_STDOUT" && "$SSM_STDOUT" != "None" ]]; then
  printf 'Remote stdout:\n'
  printf '%s\n' "$SSM_STDOUT"
fi

if [[ $STDERR_EXIT -eq 0 && -n "$SSM_STDERR" && "$SSM_STDERR" != "None" ]]; then
  printf 'Remote stderr:\n' >&2
  printf '%s\n' "$SSM_STDERR" >&2
fi

if [[ $WAIT_EXIT -ne 0 || $STATUS_EXIT -ne 0 || $RESPONSE_CODE_EXIT -ne 0 || $STDOUT_EXIT -ne 0 || $STDERR_EXIT -ne 0 ]]; then
  fail "remote deploy command failed for command $COMMAND_ID"
  exit 1
fi

if [[ "$SSM_STATUS" != "Success" || "$SSM_RESPONSE_CODE" != "0" ]]; then
  fail "remote deploy failed with SSM status=$SSM_STATUS response_code=$SSM_RESPONSE_CODE"
  exit 1
fi

step "Checking API health at $API_HTTP_HEALTHCHECK_URL"
curl --fail --silent --show-error --retry 15 --retry-delay 2 --retry-connrefused "$API_HTTP_HEALTHCHECK_URL" >/dev/null

if [[ "$CERTBOT_ENABLED" == "true" ]]; then
  step "Checking API health at $API_HTTPS_HEALTHCHECK_URL"
  curl --fail --silent --show-error --retry 15 --retry-delay 2 --retry-connrefused "$API_HTTPS_HEALTHCHECK_URL" >/dev/null
fi

step 'Deploy complete'
