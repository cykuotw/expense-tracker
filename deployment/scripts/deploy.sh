#!/usr/bin/env bash
set -euo pipefail

# ------------
# Init local deploy context
# ------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$SCRIPT_DIR/lib/format.sh"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/terraform.sh"

if [[ ! -f "$(tf_vars_file_path)" ]]; then
  fail "missing TF_VARS_FILE: $(tf_vars_file_path)"
  exit 1
fi

if ! CERTBOT_ENABLED="$(tf_var_string certbot_enabled 2>/dev/null)"; then
  fail "define certbot_enabled in $(tf_vars_file_path)"
  exit 1
fi
if ! CERTBOT_STAGING="$(tf_var_string certbot_staging 2>/dev/null)"; then
  fail "define certbot_staging in $(tf_vars_file_path)"
  exit 1
fi
if certbot_email_override="$(tf_var_string certbot_email 2>/dev/null)"; then
  CERTBOT_EMAIL="$certbot_email_override"
elif [[ "$CERTBOT_ENABLED" == "true" ]]; then
  fail "define certbot_email in $(tf_vars_file_path) when certbot_enabled=true"
  exit 1
else
  CERTBOT_EMAIL=""
fi

resolve_aws_region
BACKEND_ENV_DIR="$(dirname "$BACKEND_ENV_PATH")"
STAGE_DIR="$BUILD_ROOT/backend"
RUNTIME_STAGE_DIR="$STAGE_DIR/runtime"
DEPLOY_STAGE_DIR="$STAGE_DIR/deploy"
BACKEND_ARCHIVE="$BUILD_ROOT/backend-release.tar.gz"
NGINX_CONFIG_TEMPLATE="deployment/nginx/expense-tracker.conf"
SYSTEMD_UNIT_TEMPLATE="deployment/systemd/expense-tracker.service"
REMOTE_DEPLOY_SCRIPT_SRC="deployment/remote/deploy-release.sh"
RELEASE_MANIFEST_PATH="$DEPLOY_STAGE_DIR/release-manifest.env"
RUNTIME_ENV_SSM_PARAMETERS_PATH="$DEPLOY_STAGE_DIR/runtime-env-ssm.env"

# ------------
# Print deploy configuration
# ------------

step 'Deploy configuration'
printf '  AWS_REGION=%s\n' "$AWS_REGION"
printf '  TF_DIR=%s\n' "$TF_DIR"
printf '  TF_VARS_FILE=%s\n' "$TF_VARS_FILE"
printf '  APP_DIR=%s\n' "$APP_DIR"
printf '  BACKEND_ENV_PATH=%s\n' "$BACKEND_ENV_PATH"
printf '  BACKEND_URL=%s\n' "$BACKEND_URL"
printf '  API_URL=%s\n' "$API_URL"
printf '  SYSTEMD_SERVICE_NAME=%s\n' "$SYSTEMD_SERVICE_NAME"
printf '  GOOS=%s\n' "$GOOS"
printf '  GOARCH=%s\n' "$GOARCH"
printf '  BUILD_ROOT=%s\n' "$BUILD_ROOT"
printf '  STAGE_DIR=%s\n' "$STAGE_DIR"
printf '  RUNTIME_STAGE_DIR=%s\n' "$RUNTIME_STAGE_DIR"
printf '  DEPLOY_STAGE_DIR=%s\n' "$DEPLOY_STAGE_DIR"
printf '  FRONTEND_DIST_DIR=%s\n' "$FRONTEND_DIST_DIR"
printf '  MIGRATIONS_SRC_DIR=%s\n' "$MIGRATIONS_SRC_DIR"
printf '  ARTIFACT_KEY=%s\n' "$ARTIFACT_KEY"
printf '  BACKEND_ARCHIVE=%s\n' "$BACKEND_ARCHIVE"
printf '  NGINX_CONFIG_TEMPLATE=%s\n' "$NGINX_CONFIG_TEMPLATE"
printf '  SYSTEMD_UNIT_TEMPLATE=%s\n' "$SYSTEMD_UNIT_TEMPLATE"
printf '  REMOTE_DEPLOY_SCRIPT_SRC=%s\n' "$REMOTE_DEPLOY_SCRIPT_SRC"
printf '  RELEASE_MANIFEST_PATH=%s\n' "$RELEASE_MANIFEST_PATH"
printf '  RUNTIME_ENV_SSM_PARAMETERS_PATH=%s\n' "$RUNTIME_ENV_SSM_PARAMETERS_PATH"
printf '  API_HTTP_HEALTHCHECK_URL=%s\n' "$API_HTTP_HEALTHCHECK_URL"
printf '  API_HTTPS_HEALTHCHECK_URL=%s\n' "$API_HTTPS_HEALTHCHECK_URL"
printf '  CERTBOT_ENABLED=%s\n' "$CERTBOT_ENABLED"
printf '  CERTBOT_EMAIL=%s\n' "$CERTBOT_EMAIL"
printf '  CERTBOT_STAGING=%s\n' "$CERTBOT_STAGING"

# ------------
# Common helpers
# ------------

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    fail "required file not found: $path"
    exit 1
  fi
}

write_shell_var() {
  local file="$1"
  local key="$2"
  local value="$3"
  printf '%s=%q\n' "$key" "$value" >>"$file"
}

shell_quote() {
  printf '%q' "$1"
}

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  value="${value//$'\f'/\\f}"
  value="${value//$'\b'/\\b}"
  printf '%s' "$value"
}

build_ssm_parameters() {
  local commands=("$@")
  local json='{"commands":['
  local index

  for index in "${!commands[@]}"; do
    if (( index > 0 )); then
      json+=','
    fi
    json+="\"$(json_escape "${commands[$index]}")\""
  done

  json+=']}'
  printf '%s\n' "$json"
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
# Prepare local staging directories
# ------------

rm -rf "$BUILD_ROOT"
mkdir -p \
  "$RUNTIME_STAGE_DIR/bin" \
  "$RUNTIME_STAGE_DIR/migrations" \
  "$RUNTIME_STAGE_DIR/systemd" \
  "$RUNTIME_STAGE_DIR/nginx" \
  "$DEPLOY_STAGE_DIR"

# Validate static local inputs before provisioning or packaging anything.
require_file "$NGINX_CONFIG_TEMPLATE"
require_file "$SYSTEMD_UNIT_TEMPLATE"
require_file "$REMOTE_DEPLOY_SCRIPT_SRC"

# ------------
# Apply Terraform and resolve deploy contract
# ------------

# Apply infra first so the deploy uses the latest host, bucket, and SSM output values.
step 'Applying Terraform'
terraform_cmd init -input=false
terraform_cmd apply -auto-approve -input=false

INSTANCE_ID="$(terraform_output backend_instance_id)"
FRONTEND_BUCKET="$(terraform_output frontend_bucket_name)"
ARTIFACT_BUCKET="$(terraform_output artifact_bucket_name)"
DB_ADMIN_USERNAME="$(terraform_output db_admin_username)"
DB_ADMIN_PASSWORD_SSM_PARAMETER_NAME="$(terraform_output db_admin_password_ssm_parameter_name)"
DB_MIGRATION_USERNAME="$(terraform_output db_migration_username)"
DB_MIGRATION_PASSWORD_SSM_PARAMETER_NAME="$(terraform_output db_migration_password_ssm_parameter_name)"
FRONTEND_ORIGIN_SSM_PARAMETER_NAME="$(terraform_output frontend_origin_ssm_parameter_name)"
CORS_ALLOWED_ORIGINS_SSM_PARAMETER_NAME="$(terraform_output cors_allowed_origins_ssm_parameter_name)"
AUTH_COOKIE_DOMAIN_SSM_PARAMETER_NAME="$(terraform_output auth_cookie_domain_ssm_parameter_name)"
DB_PUBLIC_HOST_SSM_PARAMETER_NAME="$(terraform_output db_public_host_ssm_parameter_name)"
DB_PORT_SSM_PARAMETER_NAME="$(terraform_output db_port_ssm_parameter_name)"
DB_USER_SSM_PARAMETER_NAME="$(terraform_output db_user_ssm_parameter_name)"
DB_NAME_SSM_PARAMETER_NAME="$(terraform_output db_name_ssm_parameter_name)"
DB_PASSWORD_SSM_PARAMETER_NAME="$(terraform_output db_app_password_ssm_parameter_name)"
DB_SSLMODE_SSM_PARAMETER_NAME="$(terraform_output db_sslmode_ssm_parameter_name)"
GOOGLE_CALLBACK_URL_SSM_PARAMETER_NAME="$(terraform_output google_callback_url_ssm_parameter_name)"
JWT_SECRET_SSM_PARAMETER_NAME="$(terraform_output jwt_secret_ssm_parameter_name)"
REFRESH_JWT_SECRET_SSM_PARAMETER_NAME="$(terraform_output refresh_jwt_secret_ssm_parameter_name)"
THIRD_PARTY_SESSION_SECRET_SSM_PARAMETER_NAME="$(terraform_output third_party_session_secret_ssm_parameter_name)"
GOOGLE_CLIENT_ID_SSM_PARAMETER_NAME="$(terraform_output_optional google_client_id_ssm_parameter_name)"
GOOGLE_CLIENT_SECRET_SSM_PARAMETER_NAME="$(terraform_output_optional google_client_secret_ssm_parameter_name)"
API_FQDN="$(terraform_output api_fqdn)"
CLOUDFRONT_DISTRIBUTION_ID="$(terraform_output cloudfront_distribution_id)"
S3_URI="s3://$ARTIFACT_BUCKET/$ARTIFACT_KEY"
API_ORIGIN="https://$API_FQDN"
if [[ -z "$API_HTTP_HEALTHCHECK_URL" ]]; then
  API_HTTP_HEALTHCHECK_URL="http://$API_FQDN${API_URL%/}/health"
fi
if [[ -z "$API_HTTPS_HEALTHCHECK_URL" ]]; then
  API_HTTPS_HEALTHCHECK_URL="https://$API_FQDN${API_URL%/}/health"
fi

if [[ "$CERTBOT_ENABLED" == "true" && -z "$CERTBOT_EMAIL" ]]; then
  fail "CERTBOT_EMAIL must be set when CERTBOT_ENABLED=true"
  exit 1
fi

# ------------
# Build frontend and backend artifacts
# ------------

# Build the frontend locally with the live API origin that Terraform just resolved.
step 'Building frontend'
VITE_API_ORIGIN="$API_ORIGIN" VITE_API_PATH="$API_URL" make build-frontend

# Stage a split backend release: runtime files for APP_DIR and deploy metadata for the temp release root.
step 'Building backend'
make build-deploy-backend BUILD_DIR="$RUNTIME_STAGE_DIR/bin" GOOS="$GOOS" GOARCH="$GOARCH"

cp -R "$MIGRATIONS_SRC_DIR/." "$RUNTIME_STAGE_DIR/migrations/"
cp "$REMOTE_DEPLOY_SCRIPT_SRC" "$DEPLOY_STAGE_DIR/deploy-release.sh"
chmod 755 "$DEPLOY_STAGE_DIR/deploy-release.sh"
python3 - "$NGINX_CONFIG_TEMPLATE" "$RUNTIME_STAGE_DIR/nginx/expense-tracker.conf" "$API_FQDN" <<'PY2'
import pathlib
import sys

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
api_fqdn = sys.argv[3]
text = source.read_text()
target.write_text(text.replace("__API_FQDN__", api_fqdn))
PY2
python3 - "$SYSTEMD_UNIT_TEMPLATE" "$RUNTIME_STAGE_DIR/systemd/expense-tracker.service" "$APP_DIR" "$BACKEND_ENV_PATH" <<'PY2'
import pathlib
import sys

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
app_dir = sys.argv[3]
backend_env_path = sys.argv[4]
text = source.read_text()
text = text.replace("__APP_DIR__", app_dir)
text = text.replace("__BACKEND_ENV_PATH__", backend_env_path)
target.write_text(text)
PY2

# ------------
# Package remote runtime env contract
# ------------

# Package the runtime env contract as SSM parameter names so the host renders a complete backend.env
# from one source of truth instead of merging local overlays with host leftovers.
: > "$RUNTIME_ENV_SSM_PARAMETERS_PATH"
RUNTIME_ENV_REQUIRED_KEYS=(
  FRONTEND_ORIGIN
  CORS_ALLOWED_ORIGINS
  AUTH_COOKIE_DOMAIN
  DB_PUBLIC_HOST
  DB_PORT
  DB_USER
  DB_NAME
  DB_PASSWORD
  DB_SSLMODE
  GOOGLE_CALLBACK_URL
  JWT_SECRET
  REFRESH_JWT_SECRET
  THIRD_PARTY_SESSION_SECRET
)
RUNTIME_ENV_OPTIONAL_KEYS=(
  GOOGLE_CLIENT_ID
  GOOGLE_CLIENT_SECRET
)
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_REQUIRED_KEYS "${RUNTIME_ENV_REQUIRED_KEYS[*]}"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_OPTIONAL_KEYS "${RUNTIME_ENV_OPTIONAL_KEYS[*]}"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__FRONTEND_ORIGIN "$FRONTEND_ORIGIN_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__CORS_ALLOWED_ORIGINS "$CORS_ALLOWED_ORIGINS_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__AUTH_COOKIE_DOMAIN "$AUTH_COOKIE_DOMAIN_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__DB_PUBLIC_HOST "$DB_PUBLIC_HOST_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__DB_PORT "$DB_PORT_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__DB_USER "$DB_USER_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__DB_NAME "$DB_NAME_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__DB_PASSWORD "$DB_PASSWORD_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__DB_SSLMODE "$DB_SSLMODE_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__GOOGLE_CALLBACK_URL "$GOOGLE_CALLBACK_URL_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__JWT_SECRET "$JWT_SECRET_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__REFRESH_JWT_SECRET "$REFRESH_JWT_SECRET_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__THIRD_PARTY_SESSION_SECRET "$THIRD_PARTY_SESSION_SECRET_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__GOOGLE_CLIENT_ID "$GOOGLE_CLIENT_ID_SSM_PARAMETER_NAME"
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__GOOGLE_CLIENT_SECRET "$GOOGLE_CLIENT_SECRET_SSM_PARAMETER_NAME"

# ------------
# Package remote release metadata
# ------------

# Package the non-secret release context the host needs to finalize the deploy locally.
: > "$RELEASE_MANIFEST_PATH"
write_shell_var "$RELEASE_MANIFEST_PATH" AWS_REGION "$AWS_REGION"
write_shell_var "$RELEASE_MANIFEST_PATH" APP_DIR "$APP_DIR"
write_shell_var "$RELEASE_MANIFEST_PATH" BACKEND_ENV_DIR "$BACKEND_ENV_DIR"
write_shell_var "$RELEASE_MANIFEST_PATH" BACKEND_ENV_PATH "$BACKEND_ENV_PATH"
write_shell_var "$RELEASE_MANIFEST_PATH" BACKEND_URL "$BACKEND_URL"
write_shell_var "$RELEASE_MANIFEST_PATH" API_URL "$API_URL"
write_shell_var "$RELEASE_MANIFEST_PATH" SYSTEMD_SERVICE_NAME "$SYSTEMD_SERVICE_NAME"
write_shell_var "$RELEASE_MANIFEST_PATH" DB_ADMIN_USERNAME "$DB_ADMIN_USERNAME"
write_shell_var "$RELEASE_MANIFEST_PATH" DB_MIGRATION_USERNAME "$DB_MIGRATION_USERNAME"
write_shell_var "$RELEASE_MANIFEST_PATH" DB_ADMIN_PASSWORD_SSM_PARAMETER_NAME "$DB_ADMIN_PASSWORD_SSM_PARAMETER_NAME"
write_shell_var "$RELEASE_MANIFEST_PATH" DB_MIGRATION_PASSWORD_SSM_PARAMETER_NAME "$DB_MIGRATION_PASSWORD_SSM_PARAMETER_NAME"
write_shell_var "$RELEASE_MANIFEST_PATH" API_FQDN "$API_FQDN"
write_shell_var "$RELEASE_MANIFEST_PATH" CERTBOT_ENABLED "$CERTBOT_ENABLED"
write_shell_var "$RELEASE_MANIFEST_PATH" CERTBOT_EMAIL "$CERTBOT_EMAIL"
write_shell_var "$RELEASE_MANIFEST_PATH" CERTBOT_STAGING "$CERTBOT_STAGING"

# ------------
# Upload release artifacts
# ------------

# Bundle runtime and deploy metadata into a single artifact for the remote release step.
tar -C "$STAGE_DIR" -czf "$BACKEND_ARCHIVE" .

step "Uploading frontend dist to s3://$FRONTEND_BUCKET"
aws --region "$AWS_REGION" s3 sync "$FRONTEND_DIST_DIR/" "s3://$FRONTEND_BUCKET/" --delete

step "Invalidating CloudFront distribution $CLOUDFRONT_DISTRIBUTION_ID"
aws --region "$AWS_REGION" cloudfront create-invalidation   --distribution-id "$CLOUDFRONT_DISTRIBUTION_ID"   --paths '/*' >/dev/null

step "Uploading backend bundle to $S3_URI"
aws --region "$AWS_REGION" s3 cp "$BACKEND_ARCHIVE" "$S3_URI"

# ------------
# Trigger remote deploy through SSM
# ------------

# Keep the SSM payload minimal: download, unpack, and hand off to the packaged remote script.
step 'Running remote deploy through SSM'
AWS_REGION_QUOTED="$(shell_quote "$AWS_REGION")"
S3_URI_QUOTED="$(shell_quote "$S3_URI")"
SSM_COMMANDS=(
  'set -euo pipefail'
  'TMP_RELEASE="$(mktemp /tmp/backend-release.XXXXXX.tar.gz)"'
  'TMP_RELEASE_DIR="$(mktemp -d /tmp/backend-release.XXXXXX)"'
  'cleanup() { rm -f "$TMP_RELEASE"; rm -rf "$TMP_RELEASE_DIR"; }'
  'trap cleanup EXIT'
  "aws --region $AWS_REGION_QUOTED s3 cp $S3_URI_QUOTED \"\$TMP_RELEASE\""
  'tar -xzf "$TMP_RELEASE" -C "$TMP_RELEASE_DIR"'
  'chmod 755 "$TMP_RELEASE_DIR/deploy/deploy-release.sh"'
  '"$TMP_RELEASE_DIR/deploy/deploy-release.sh" "$TMP_RELEASE_DIR"'
)
SSM_PARAMETERS="$(build_ssm_parameters "${SSM_COMMANDS[@]}")"
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

# ------------
# Verify deployed API health
# ------------

if [[ "$CERTBOT_ENABLED" == "true" ]]; then
  step "Checking API health via HTTP redirect at $API_HTTP_HEALTHCHECK_URL"
  curl --fail --silent --show-error --location --retry 15 --retry-delay 2 --retry-connrefused "$API_HTTP_HEALTHCHECK_URL" >/dev/null

  step "Checking API health at $API_HTTPS_HEALTHCHECK_URL"
  curl --fail --silent --show-error --retry 15 --retry-delay 2 --retry-connrefused "$API_HTTPS_HEALTHCHECK_URL" >/dev/null
else
  step "Checking API health at $API_HTTP_HEALTHCHECK_URL"
  curl --fail --silent --show-error --retry 15 --retry-delay 2 --retry-connrefused "$API_HTTP_HEALTHCHECK_URL" >/dev/null
fi

step 'Deploy complete'
