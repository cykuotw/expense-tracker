#!/usr/bin/env bash
set -euo pipefail

# ------------
# Init local deploy context
# ------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$SCRIPT_DIR/lib/format.sh"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/terraform.sh"
source "$SCRIPT_DIR/lib/deploy-helpers.sh"

# ------------
# Resolve deploy context
# ------------

require_tf_vars_file
resolve_aws_region
resolve_first_admin_bootstrap_settings

BACKEND_ENV_DIR="$(dirname "$BACKEND_ENV_PATH")"
STAGE_DIR="$BUILD_ROOT/backend"
RUNTIME_STAGE_DIR="$STAGE_DIR/runtime"
DEPLOY_STAGE_DIR="$STAGE_DIR/deploy"
DEPLOY_LIB_DIR="$DEPLOY_STAGE_DIR/lib"
BACKEND_ARCHIVE="$BUILD_ROOT/backend-release.tar.gz"
SYSTEMD_UNIT_TEMPLATE="deployment/systemd/expense-tracker.service"
REMOTE_DEPLOY_SCRIPT_SRC="deployment/remote/deploy-backend-release.sh"
REMOTE_DEPLOY_COMMON_SRC="deployment/remote/lib/common.sh"
RELEASE_MANIFEST_PATH="$DEPLOY_STAGE_DIR/release-manifest.env"
RUNTIME_ENV_SSM_PARAMETERS_PATH="$DEPLOY_STAGE_DIR/runtime-env-ssm.env"

# ------------
# Common helpers
# ------------

cleanup_first_admin_bootstrap_parameters() {
  local name
  for name in "${FIRST_ADMIN_BOOTSTRAP_PARAMETER_NAMES[@]}"; do
    delete_ssm_parameter_if_exists "$name"
  done
}

trap cleanup_first_admin_bootstrap_parameters EXIT

# ------------
# Prepare local staging directories
# ------------

rm -rf "$BUILD_ROOT"
mkdir -p \
  "$RUNTIME_STAGE_DIR/bin" \
  "$RUNTIME_STAGE_DIR/migrations" \
  "$RUNTIME_STAGE_DIR/systemd" \
  "$DEPLOY_LIB_DIR"

require_file "$SYSTEMD_UNIT_TEMPLATE"
require_file "$REMOTE_DEPLOY_SCRIPT_SRC"
require_file "$REMOTE_DEPLOY_COMMON_SRC"

# ------------
# Resolve deploy contract
# ------------

INSTANCE_ID="$(terraform_output backend_instance_id)"
ARTIFACT_BUCKET="$(terraform_output artifact_bucket_name)"
DB_ADMIN_USERNAME="$(terraform_output db_admin_username)"
DB_ADMIN_PASSWORD_SSM_PARAMETER_NAME="$(terraform_output db_admin_password_ssm_parameter_name)"
DB_MIGRATION_USERNAME="$(terraform_output db_migration_username)"
DB_MIGRATION_PASSWORD_SSM_PARAMETER_NAME="$(terraform_output db_migration_password_ssm_parameter_name)"
FRONTEND_ORIGIN_SSM_PARAMETER_NAME="$(terraform_output frontend_origin_ssm_parameter_name)"
CORS_ALLOWED_ORIGINS_SSM_PARAMETER_NAME="$(terraform_output cors_allowed_origins_ssm_parameter_name)"
CORS_ALLOW_CREDENTIALS_SSM_PARAMETER_NAME="$(terraform_output cors_allow_credentials_ssm_parameter_name)"
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
S3_URI="s3://$ARTIFACT_BUCKET/$ARTIFACT_KEY"

# ------------
# Print deploy configuration
# ------------

step 'Backend deploy configuration'
printf '  AWS_REGION=%s\n' "$AWS_REGION"
printf '  TF_DIR=%s\n' "$TF_DIR"
printf '  TF_VARS_FILE=%s\n' "$TF_VARS_FILE"
printf '  PROJECT_NAME=%s\n' "$PROJECT_NAME"
printf '  DEPLOY_ENVIRONMENT=%s\n' "$DEPLOY_ENVIRONMENT"
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
printf '  MIGRATIONS_SRC_DIR=%s\n' "$MIGRATIONS_SRC_DIR"
printf '  ARTIFACT_KEY=%s\n' "$ARTIFACT_KEY"
printf '  BACKEND_ARCHIVE=%s\n' "$BACKEND_ARCHIVE"
printf '  INSTANCE_ID=%s\n' "$INSTANCE_ID"
printf '  ARTIFACT_BUCKET=%s\n' "$ARTIFACT_BUCKET"
printf '  SYSTEMD_UNIT_TEMPLATE=%s\n' "$SYSTEMD_UNIT_TEMPLATE"
printf '  REMOTE_DEPLOY_SCRIPT_SRC=%s\n' "$REMOTE_DEPLOY_SCRIPT_SRC"
printf '  REMOTE_DEPLOY_COMMON_SRC=%s\n' "$REMOTE_DEPLOY_COMMON_SRC"
printf '  RELEASE_MANIFEST_PATH=%s\n' "$RELEASE_MANIFEST_PATH"
printf '  RUNTIME_ENV_SSM_PARAMETERS_PATH=%s\n' "$RUNTIME_ENV_SSM_PARAMETERS_PATH"
printf '  FIRST_ADMIN_BOOTSTRAP_ENABLED=%s\n' "$FIRST_ADMIN_BOOTSTRAP_ENABLED"
printf '  FIRST_ADMIN_BOOTSTRAP_SSM_PARAMETER_PREFIX=%s\n' "$FIRST_ADMIN_BOOTSTRAP_SSM_PARAMETER_PREFIX"

# ------------
# Build backend artifacts
# ------------

step 'Building backend'
make build-deploy-backend BUILD_DIR="$RUNTIME_STAGE_DIR/bin" GOOS="$GOOS" GOARCH="$GOARCH"

cp -R "$MIGRATIONS_SRC_DIR/." "$RUNTIME_STAGE_DIR/migrations/"
cp "$REMOTE_DEPLOY_SCRIPT_SRC" "$DEPLOY_STAGE_DIR/deploy-backend-release.sh"
chmod 755 "$DEPLOY_STAGE_DIR/deploy-backend-release.sh"
cp "$REMOTE_DEPLOY_COMMON_SRC" "$DEPLOY_LIB_DIR/common.sh"
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

: > "$RUNTIME_ENV_SSM_PARAMETERS_PATH"
RUNTIME_ENV_REQUIRED_KEYS=(
  FRONTEND_ORIGIN
  CORS_ALLOWED_ORIGINS
  CORS_ALLOW_CREDENTIALS
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
write_shell_var "$RUNTIME_ENV_SSM_PARAMETERS_PATH" RUNTIME_ENV_SSM_PARAMETER_NAME__CORS_ALLOW_CREDENTIALS "$CORS_ALLOW_CREDENTIALS_SSM_PARAMETER_NAME"
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

if [[ "$FIRST_ADMIN_BOOTSTRAP_ENABLED" == "true" ]]; then
  step 'Packaging deploy-time first-admin bootstrap inputs'
  FIRST_ADMIN_BOOTSTRAP_SESSION_ID="$(python3 - <<'PY2'
import uuid

print(uuid.uuid4())
PY2
)"
  FIRST_ADMIN_BOOTSTRAP_BASE_PARAMETER_NAME="${FIRST_ADMIN_BOOTSTRAP_SSM_PARAMETER_PREFIX%/}/$FIRST_ADMIN_BOOTSTRAP_SESSION_ID"
  FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME="$FIRST_ADMIN_BOOTSTRAP_BASE_PARAMETER_NAME/email"
  FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME="$FIRST_ADMIN_BOOTSTRAP_BASE_PARAMETER_NAME/password"
  FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME="$FIRST_ADMIN_BOOTSTRAP_BASE_PARAMETER_NAME/firstname"
  FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME="$FIRST_ADMIN_BOOTSTRAP_BASE_PARAMETER_NAME/lastname"
  FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME=""

  create_ssm_parameter "$FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME" String "$FIRST_ADMIN_EMAIL"
  create_ssm_parameter "$FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME" SecureString "$FIRST_ADMIN_PASSWORD"
  create_ssm_parameter "$FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME" String "$FIRST_ADMIN_FIRSTNAME"
  create_ssm_parameter "$FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME" String "$FIRST_ADMIN_LASTNAME"
  FIRST_ADMIN_BOOTSTRAP_PARAMETER_NAMES=(
    "$FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME"
    "$FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME"
    "$FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME"
    "$FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME"
  )

  write_shell_var "$RELEASE_MANIFEST_PATH" FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME "$FIRST_ADMIN_EMAIL_SSM_PARAMETER_NAME"
  write_shell_var "$RELEASE_MANIFEST_PATH" FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME "$FIRST_ADMIN_PASSWORD_SSM_PARAMETER_NAME"
  write_shell_var "$RELEASE_MANIFEST_PATH" FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME "$FIRST_ADMIN_FIRSTNAME_SSM_PARAMETER_NAME"
  write_shell_var "$RELEASE_MANIFEST_PATH" FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME "$FIRST_ADMIN_LASTNAME_SSM_PARAMETER_NAME"

  if [[ -n "$FIRST_ADMIN_NICKNAME" ]]; then
    FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME="$FIRST_ADMIN_BOOTSTRAP_BASE_PARAMETER_NAME/nickname"
    create_ssm_parameter "$FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME" String "$FIRST_ADMIN_NICKNAME"
    FIRST_ADMIN_BOOTSTRAP_PARAMETER_NAMES+=("$FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME")
    write_shell_var "$RELEASE_MANIFEST_PATH" FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME "$FIRST_ADMIN_NICKNAME_SSM_PARAMETER_NAME"
  fi
fi

# ------------
# Upload release artifacts
# ------------

tar -C "$STAGE_DIR" -czf "$BACKEND_ARCHIVE" .

step "Uploading backend bundle to $S3_URI"
aws --region "$AWS_REGION" s3 cp "$BACKEND_ARCHIVE" "$S3_URI"

# ------------
# Trigger remote deploy through SSM
# ------------

run_remote_release_via_ssm "$INSTANCE_ID" "$S3_URI" "deploy/deploy-backend-release.sh"

step 'Backend deploy complete'
