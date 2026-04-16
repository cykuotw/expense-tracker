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
# Resolve deploy contract
# ------------

require_tf_vars_file
resolve_aws_region

FRONTEND_BUCKET="$(terraform_output frontend_bucket_name)"
CLOUDFRONT_DISTRIBUTION_ID="$(terraform_output cloudfront_distribution_id)"
API_FQDN="$(terraform_output api_fqdn)"
API_ORIGIN="https://$API_FQDN"

case "${GOOGLE_OAUTH_ENABLED,,}" in
  ""|false)
    GOOGLE_OAUTH_ENABLED="false"
    ;;
  true)
    GOOGLE_OAUTH_ENABLED="true"
    ;;
  *)
    fail 'GOOGLE_OAUTH_ENABLED must be either "true" or "false".'
    exit 1
    ;;
esac

if [[ "$GOOGLE_OAUTH_ENABLED" == "true" && -z "${GOOGLE_CLIENT_ID//[[:space:]]/}" ]]; then
  fail 'GOOGLE_CLIENT_ID is required when GOOGLE_OAUTH_ENABLED=true.'
  exit 1
fi

# ------------
# Print deploy configuration
# ------------

step 'Frontend deploy configuration'
printf '  AWS_REGION=%s\n' "$AWS_REGION"
printf '  TF_DIR=%s\n' "$TF_DIR"
printf '  TF_VARS_FILE=%s\n' "$TF_VARS_FILE"
printf '  FRONTEND_DIST_DIR=%s\n' "$FRONTEND_DIST_DIR"
printf '  FRONTEND_BUCKET=%s\n' "$FRONTEND_BUCKET"
printf '  CLOUDFRONT_DISTRIBUTION_ID=%s\n' "$CLOUDFRONT_DISTRIBUTION_ID"
printf '  API_ORIGIN=%s\n' "$API_ORIGIN"
printf '  API_URL=%s\n' "$API_URL"
printf '  GOOGLE_OAUTH_ENABLED=%s\n' "$GOOGLE_OAUTH_ENABLED"
printf '  GOOGLE_CLIENT_ID_SET=%s\n' "$([[ -n "$GOOGLE_CLIENT_ID" ]] && printf yes || printf no)"

# ------------
# Build frontend artifacts
# ------------

step 'Building frontend'
make build-frontend

RUNTIME_CONFIG_PATH="$FRONTEND_DIST_DIR/runtime-config.js"
RUNTIME_CONFIG_PATH="$RUNTIME_CONFIG_PATH" \
  API_ORIGIN="$API_ORIGIN" \
  API_URL="$API_URL" \
  GOOGLE_OAUTH_ENABLED="$GOOGLE_OAUTH_ENABLED" \
  GOOGLE_CLIENT_ID="$GOOGLE_CLIENT_ID" \
  node <<'EOF'
const fs = require("fs");

function parseBoolean(value, name) {
    const normalized = (value ?? "").trim().toLowerCase();
    if (normalized === "") {
        return false;
    }
    if (normalized === "true") {
        return true;
    }
    if (normalized === "false") {
        return false;
    }
    throw new Error(`${name} must be either "true" or "false".`);
}

const googleOAuthEnabled = parseBoolean(
    process.env.GOOGLE_OAUTH_ENABLED,
    "GOOGLE_OAUTH_ENABLED",
);
const googleClientId = (process.env.GOOGLE_CLIENT_ID ?? "").trim();

if (googleOAuthEnabled && googleClientId === "") {
    throw new Error("GOOGLE_CLIENT_ID is required when GOOGLE_OAUTH_ENABLED=true.");
}

const config = {
    apiOrigin: process.env.API_ORIGIN ?? "",
    apiPath: process.env.API_URL ?? "",
    googleOAuthEnabled,
    googleClientId,
};

const runtimeConfig = `window.__APP_CONFIG__ = Object.freeze(${JSON.stringify(config, null, 4)});\n`;
fs.writeFileSync(process.env.RUNTIME_CONFIG_PATH, runtimeConfig);
EOF

# ------------
# Publish frontend assets
# ------------

step "Uploading frontend dist to s3://$FRONTEND_BUCKET"
aws --region "$AWS_REGION" s3 sync "$FRONTEND_DIST_DIR/" "s3://$FRONTEND_BUCKET/" --delete

step "Invalidating CloudFront distribution $CLOUDFRONT_DISTRIBUTION_ID"
aws --region "$AWS_REGION" cloudfront create-invalidation --distribution-id "$CLOUDFRONT_DISTRIBUTION_ID" --paths '/*' >/dev/null

step 'Frontend deploy complete'
