#!/usr/bin/env bash
set -euo pipefail

# ------------
# Init local deploy context
# ------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVERFUL_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SERVERFUL_ROOT/../../.." && pwd)"

source "$SERVERFUL_ROOT/shared/lib/format.sh"
source "$REPO_ROOT/deployment/backend/shared/lib/config.sh"
source "$SERVERFUL_ROOT/shared/lib/config.sh"
source "$SERVERFUL_ROOT/shared/lib/terraform.sh"
source "$SERVERFUL_ROOT/shared/lib/deploy-helpers.sh"

# ------------
# Resolve deploy context
# ------------

require_tf_vars_file
resolve_aws_region

# ------------
# Print deploy configuration
# ------------

step 'Infra apply configuration'
printf '  AWS_REGION=%s\n' "$AWS_REGION"
printf '  TF_DIR=%s\n' "$TF_DIR"
printf '  TF_VARS_FILE=%s\n' "$TF_VARS_FILE"

# ------------
# Apply Terraform
# ------------

step 'Applying Terraform'
terraform_cmd init -input=false
terraform_cmd apply -auto-approve -input=false

step 'Infra apply complete'
