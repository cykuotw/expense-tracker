#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$SCRIPT_DIR/lib/format.sh"
source "$SCRIPT_DIR/lib/config.sh"
source "$SCRIPT_DIR/lib/terraform.sh"

if [[ ! -f "$(tf_vars_file_path)" ]]; then
  fail "missing TF_VARS_FILE: $(tf_vars_file_path)"
  exit 1
fi
resolve_aws_region

step 'Destroy configuration'
printf '  AWS_REGION=%s\n' "$AWS_REGION"
printf '  TF_DIR=%s\n' "$TF_DIR"
printf '  TF_VARS_FILE=%s\n' "$TF_VARS_FILE"

list_delete_batch() {
  local bucket="$1"
  local key_marker="${2:-}"
  local version_id_marker="${3:-}"

  if [[ -n "$key_marker" ]]; then
    aws --region "$AWS_REGION" s3api list-object-versions       --bucket "$bucket"       --key-marker "$key_marker"       --version-id-marker "$version_id_marker"       --output json
  else
    aws --region "$AWS_REGION" s3api list-object-versions       --bucket "$bucket"       --output json
  fi
}

build_delete_payload() {
  python3 -c '
import json
import sys
response = json.load(sys.stdin)
objects = []
for item in response.get("Versions", []):
    objects.append({"Key": item["Key"], "VersionId": item["VersionId"]})
for item in response.get("DeleteMarkers", []):
    objects.append({"Key": item["Key"], "VersionId": item["VersionId"]})
if not objects:
    sys.exit(0)
print(json.dumps({"Objects": objects, "Quiet": True}))
'
}

next_markers() {
  python3 -c '
import json
import sys
response = json.load(sys.stdin)
print(response.get("NextKeyMarker", ""))
print(response.get("NextVersionIdMarker", ""))
'
}

empty_bucket() {
  local bucket="$1"
  local response
  local delete_payload
  local markers=()
  local key_marker=""
  local version_id_marker=""

  step "Emptying s3://$bucket"

  if ! aws --region "$AWS_REGION" s3api head-bucket --bucket "$bucket" >/dev/null 2>&1; then
    warn "Bucket already missing or inaccessible, skipping: s3://$bucket"
    return 0
  fi

  aws --region "$AWS_REGION" s3 rm "s3://$bucket" --recursive >/dev/null || true

  while :; do
    if ! response="$(list_delete_batch "$bucket" "$key_marker" "$version_id_marker" 2>/dev/null)"; then
      warn "Could not list remaining object versions for s3://$bucket; continuing to terraform destroy"
      return 0
    fi

    delete_payload="$(build_delete_payload <<<"$response" || true)"

    if [[ -n "$delete_payload" ]]; then
      if ! aws --region "$AWS_REGION" s3api delete-objects         --bucket "$bucket"         --delete "$delete_payload" >/dev/null; then
        warn "Could not delete some versioned objects from s3://$bucket; continuing to terraform destroy"
        return 0
      fi
    fi

    mapfile -t markers < <(next_markers <<<"$response")
    key_marker="${markers[0]:-}"
    version_id_marker="${markers[1]:-}"

    if [[ -z "$key_marker" ]]; then
      break
    fi
  done
}

step 'Loading Terraform outputs'
terraform_cmd init -input=false >/dev/null
FRONTEND_BUCKET="$(terraform_output frontend_bucket_name)"
ARTIFACT_BUCKET="$(terraform_output artifact_bucket_name)"

empty_bucket "$FRONTEND_BUCKET"
empty_bucket "$ARTIFACT_BUCKET"

step 'Destroying Terraform-managed infrastructure'
terraform_cmd destroy -auto-approve -input=false

step 'Destroy complete'
