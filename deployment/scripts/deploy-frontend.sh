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

# ------------
# Build frontend artifacts
# ------------

step 'Building frontend'
VITE_API_ORIGIN="$API_ORIGIN" VITE_API_PATH="$API_URL" make build-frontend

# ------------
# Publish frontend assets
# ------------

step "Uploading frontend dist to s3://$FRONTEND_BUCKET"
aws --region "$AWS_REGION" s3 sync "$FRONTEND_DIST_DIR/" "s3://$FRONTEND_BUCKET/" --delete

step "Invalidating CloudFront distribution $CLOUDFRONT_DISTRIBUTION_ID"
aws --region "$AWS_REGION" cloudfront create-invalidation --distribution-id "$CLOUDFRONT_DISTRIBUTION_ID" --paths '/*' >/dev/null

step 'Frontend deploy complete'
