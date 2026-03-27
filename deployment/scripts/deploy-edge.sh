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
resolve_certbot_settings
resolve_aws_region

STAGE_DIR="$BUILD_ROOT/edge"
RUNTIME_STAGE_DIR="$STAGE_DIR/runtime"
DEPLOY_STAGE_DIR="$STAGE_DIR/deploy"
DEPLOY_LIB_DIR="$DEPLOY_STAGE_DIR/lib"
EDGE_ARCHIVE="$BUILD_ROOT/edge-release.tar.gz"
NGINX_CONFIG_TEMPLATE="deployment/nginx/expense-tracker.conf"
REMOTE_DEPLOY_SCRIPT_SRC="deployment/remote/deploy-edge-release.sh"
REMOTE_DEPLOY_COMMON_SRC="deployment/remote/lib/common.sh"
RELEASE_MANIFEST_PATH="$DEPLOY_STAGE_DIR/release-manifest.env"

# ------------
# Prepare local staging directories
# ------------

rm -rf "$BUILD_ROOT"
mkdir -p \
  "$RUNTIME_STAGE_DIR/nginx" \
  "$DEPLOY_LIB_DIR"

require_file "$NGINX_CONFIG_TEMPLATE"
require_file "$REMOTE_DEPLOY_SCRIPT_SRC"
require_file "$REMOTE_DEPLOY_COMMON_SRC"

# ------------
# Resolve deploy contract
# ------------

INSTANCE_ID="$(terraform_output backend_instance_id)"
ARTIFACT_BUCKET="$(terraform_output artifact_bucket_name)"
API_FQDN="$(terraform_output api_fqdn)"
S3_URI="s3://$ARTIFACT_BUCKET/$EDGE_ARTIFACT_KEY"
if [[ -z "$API_HTTP_HEALTHCHECK_URL" ]]; then
  API_HTTP_HEALTHCHECK_URL="http://$API_FQDN${API_URL%/}/health"
fi
if [[ -z "$API_HTTPS_HEALTHCHECK_URL" ]]; then
  API_HTTPS_HEALTHCHECK_URL="https://$API_FQDN${API_URL%/}/health"
fi

# ------------
# Print deploy configuration
# ------------

step 'Edge deploy configuration'
printf '  AWS_REGION=%s\n' "$AWS_REGION"
printf '  TF_DIR=%s\n' "$TF_DIR"
printf '  TF_VARS_FILE=%s\n' "$TF_VARS_FILE"
printf '  BUILD_ROOT=%s\n' "$BUILD_ROOT"
printf '  STAGE_DIR=%s\n' "$STAGE_DIR"
printf '  RUNTIME_STAGE_DIR=%s\n' "$RUNTIME_STAGE_DIR"
printf '  DEPLOY_STAGE_DIR=%s\n' "$DEPLOY_STAGE_DIR"
printf '  EDGE_ARTIFACT_KEY=%s\n' "$EDGE_ARTIFACT_KEY"
printf '  EDGE_ARCHIVE=%s\n' "$EDGE_ARCHIVE"
printf '  INSTANCE_ID=%s\n' "$INSTANCE_ID"
printf '  ARTIFACT_BUCKET=%s\n' "$ARTIFACT_BUCKET"
printf '  API_FQDN=%s\n' "$API_FQDN"
printf '  NGINX_CONFIG_TEMPLATE=%s\n' "$NGINX_CONFIG_TEMPLATE"
printf '  REMOTE_DEPLOY_SCRIPT_SRC=%s\n' "$REMOTE_DEPLOY_SCRIPT_SRC"
printf '  REMOTE_DEPLOY_COMMON_SRC=%s\n' "$REMOTE_DEPLOY_COMMON_SRC"
printf '  RELEASE_MANIFEST_PATH=%s\n' "$RELEASE_MANIFEST_PATH"
printf '  API_HTTP_HEALTHCHECK_URL=%s\n' "$API_HTTP_HEALTHCHECK_URL"
printf '  API_HTTPS_HEALTHCHECK_URL=%s\n' "$API_HTTPS_HEALTHCHECK_URL"
printf '  CERTBOT_ENABLED=%s\n' "$CERTBOT_ENABLED"
printf '  CERTBOT_EMAIL=%s\n' "$CERTBOT_EMAIL"
printf '  CERTBOT_STAGING=%s\n' "$CERTBOT_STAGING"

# ------------
# Build edge artifacts
# ------------

step 'Building edge artifacts'
cp "$REMOTE_DEPLOY_SCRIPT_SRC" "$DEPLOY_STAGE_DIR/deploy-edge-release.sh"
chmod 755 "$DEPLOY_STAGE_DIR/deploy-edge-release.sh"
cp "$REMOTE_DEPLOY_COMMON_SRC" "$DEPLOY_LIB_DIR/common.sh"
python3 - "$NGINX_CONFIG_TEMPLATE" "$RUNTIME_STAGE_DIR/nginx/expense-tracker.conf" "$API_FQDN" <<'PY2'
import pathlib
import sys

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
api_fqdn = sys.argv[3]
text = source.read_text()
target.write_text(text.replace("__API_FQDN__", api_fqdn))
PY2

# ------------
# Package remote release metadata
# ------------

: > "$RELEASE_MANIFEST_PATH"
write_shell_var "$RELEASE_MANIFEST_PATH" AWS_REGION "$AWS_REGION"
write_shell_var "$RELEASE_MANIFEST_PATH" API_FQDN "$API_FQDN"
write_shell_var "$RELEASE_MANIFEST_PATH" CERTBOT_ENABLED "$CERTBOT_ENABLED"
write_shell_var "$RELEASE_MANIFEST_PATH" CERTBOT_EMAIL "$CERTBOT_EMAIL"
write_shell_var "$RELEASE_MANIFEST_PATH" CERTBOT_STAGING "$CERTBOT_STAGING"

# ------------
# Upload release artifacts
# ------------

tar -C "$STAGE_DIR" -czf "$EDGE_ARCHIVE" .

step "Uploading edge bundle to $S3_URI"
aws --region "$AWS_REGION" s3 cp "$EDGE_ARCHIVE" "$S3_URI"

# ------------
# Trigger remote deploy through SSM
# ------------

run_remote_release_via_ssm "$INSTANCE_ID" "$S3_URI" "deploy/deploy-edge-release.sh"

# ------------
# Verify deployed edge health
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

step 'Edge deploy complete'
