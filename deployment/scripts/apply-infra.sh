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
