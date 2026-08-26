#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$FRONTEND_ROOT/../../.." && pwd)"
source "$FRONTEND_ROOT/lib/format.sh"
source "$FRONTEND_ROOT/lib/config.sh"
source "$FRONTEND_ROOT/lib/terraform.sh"
source "$FRONTEND_ROOT/lib/deploy-helpers.sh"

require_tf_vars_file
resolve_aws_region
FRONTEND_BUCKET="$(terraform_output frontend_bucket_name)"
CLOUDFRONT_DISTRIBUTION_ID="$(terraform_output cloudfront_distribution_id)"
if [[ -z "$API_ORIGIN" ]]; then API_ORIGIN="https://$(terraform_output api_fqdn)"; fi
case "${GOOGLE_OAUTH_ENABLED,,}" in false|'') GOOGLE_OAUTH_ENABLED=false ;; true) ;; *) fail 'GOOGLE_OAUTH_ENABLED must be true or false'; exit 1 ;; esac
if [[ "$GOOGLE_OAUTH_ENABLED" == true && -z "${GOOGLE_CLIENT_ID//[[:space:]]/}" ]]; then
  fail 'GOOGLE_CLIENT_ID is required when Google OAuth is enabled'
  exit 1
fi

step 'Building frontend'
make -C "$REPO_ROOT" build-frontend
RUNTIME_CONFIG_PATH="$FRONTEND_DIST_DIR/runtime-config.js"
RUNTIME_CONFIG_PATH="$RUNTIME_CONFIG_PATH" API_ORIGIN="$API_ORIGIN" API_URL="$API_URL" GOOGLE_OAUTH_ENABLED="$GOOGLE_OAUTH_ENABLED" GOOGLE_CLIENT_ID="$GOOGLE_CLIENT_ID" node <<'EOF'
const fs = require("fs");
const config = {
    apiOrigin: process.env.API_ORIGIN ?? "",
    apiPath: process.env.API_URL ?? "",
    googleOAuthEnabled: process.env.GOOGLE_OAUTH_ENABLED === "true",
    googleClientId: process.env.GOOGLE_CLIENT_ID ?? "",
};
fs.writeFileSync(process.env.RUNTIME_CONFIG_PATH, "window.__APP_CONFIG__ = Object.freeze(" + JSON.stringify(config, null, 4) + ");\n");
EOF
step "Uploading frontend dist to s3://$FRONTEND_BUCKET"
aws --region "$AWS_REGION" s3 sync "$FRONTEND_DIST_DIR/" "s3://$FRONTEND_BUCKET/" --delete
step "Invalidating CloudFront distribution $CLOUDFRONT_DISTRIBUTION_ID"
aws --region "$AWS_REGION" cloudfront create-invalidation --distribution-id "$CLOUDFRONT_DISTRIBUTION_ID" --paths '/*' >/dev/null
