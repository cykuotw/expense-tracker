#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="${TF_DIR:-}"
FUNCTION_OUTPUT="${FUNCTION_OUTPUT:-}"
SECURITY_GROUP_OUTPUT="${SECURITY_GROUP_OUTPUT:-}"
AWS_REGION="${AWS_REGION:-${AWS_DEFAULT_REGION:-${TF_VAR_aws_region:-}}}"
AUTO_APPROVE="${AUTO_APPROVE:-false}"
FORCE_DETACH_LAMBDA_ENI="${FORCE_DETACH_LAMBDA_ENI:-false}"
LAMBDA_ENI_NORMAL_WAIT_SECONDS="${LAMBDA_ENI_NORMAL_WAIT_SECONDS:-1200}"
LAMBDA_ENI_EXTRA_WAIT_SECONDS="${LAMBDA_ENI_EXTRA_WAIT_SECONDS:-300}"
LAMBDA_ENI_POLL_SECONDS="${LAMBDA_ENI_POLL_SECONDS:-30}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

for command_name in aws grep jq mktemp realpath rm sleep terraform; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done
[[ -n "$TF_DIR" && -d "$TF_DIR" ]] || fail 'set TF_DIR to an existing Terraform directory'
[[ -n "$FUNCTION_OUTPUT" ]] || fail 'set FUNCTION_OUTPUT to the Terraform function-name output'
[[ -n "$SECURITY_GROUP_OUTPUT" ]] || fail 'set SECURITY_GROUP_OUTPUT to the Terraform security-group output'
if [[ -z "$AWS_REGION" ]]; then
  AWS_REGION="$(aws configure get region 2>/dev/null || true)"
fi
[[ -n "$AWS_REGION" ]] || fail 'set AWS_REGION, AWS_DEFAULT_REGION, or TF_VAR_aws_region'

TF_DIR="$(realpath "$TF_DIR")"
TF_DIR="$TF_DIR" RUNTIME_ENV_FILE="${RUNTIME_ENV_FILE:-}" "$SCRIPT_DIR/check-runtime-secret-boundary.sh"

function_name="$(terraform -chdir="$TF_DIR" output -raw "$FUNCTION_OUTPUT")"
security_group_id="$(terraform -chdir="$TF_DIR" output -raw "$SECURITY_GROUP_OUTPUT")"
[[ -n "$function_name" ]] || fail "Terraform output is empty: $FUNCTION_OUTPUT"
[[ "$security_group_id" == sg-* ]] || fail "invalid security group output: $SECURITY_GROUP_OUTPUT"

if [[ "$AUTO_APPROVE" != "true" ]]; then
  printf 'This deletes Lambda %s, waits for its ENIs, then destroys Terraform stack %s.\n' "$function_name" "$TF_DIR"
  printf 'Type the Lambda function name to continue: '
  read -r confirmation
  [[ "$confirmation" == "$function_name" ]] || fail 'destroy confirmation did not match the Lambda function name'
fi

function_lookup_error="$(mktemp)"
cleanup() {
  rm -f "$function_lookup_error"
}
trap cleanup EXIT
if aws lambda get-function --region "$AWS_REGION" --function-name "$function_name" >/dev/null 2>"$function_lookup_error"; then
  aws lambda delete-function --region "$AWS_REGION" --function-name "$function_name"
  printf 'Deleted Lambda function %s; retaining its execution role while ENIs retire.\n' "$function_name"
elif grep -Fq 'ResourceNotFoundException' "$function_lookup_error"; then
  printf 'Lambda function %s is already absent.\n' "$function_name"
else
  fail "unable to verify Lambda function $function_name before destroy"
fi

normal_deadline="$LAMBDA_ENI_NORMAL_WAIT_SECONDS"
final_deadline="$((LAMBDA_ENI_NORMAL_WAIT_SECONDS + LAMBDA_ENI_EXTRA_WAIT_SECONDS))"
elapsed=0
eni_json='[]'
while :; do
  eni_json="$(aws ec2 describe-network-interfaces \
    --region "$AWS_REGION" \
    --filters "Name=group-id,Values=$security_group_id" \
    --query 'NetworkInterfaces[].{Id:NetworkInterfaceId,Status:Status,Type:InterfaceType,Description:Description,Groups:Groups[].GroupId,AttachmentId:Attachment.AttachmentId}' \
    --output json)"
  eni_count="$(jq 'length' <<<"$eni_json")"
  if [[ "$eni_count" == "0" ]]; then
    printf 'All Lambda ENIs for %s are absent after %ss.\n' "$function_name" "$elapsed"
    break
  fi

  eni_status="$(jq -r 'map(.Id + ":" + .Status + ":" + .Type) | join(",")' <<<"$eni_json")"
  if (( elapsed >= final_deadline )); then
    printf 'Lambda ENI wait expired after %ss; attempting bounded manual cleanup: %s\n' "$elapsed" "$eni_status"
    break
  fi
  if (( elapsed >= normal_deadline )); then
    printf 'AWS normal ENI window expired; additional wait remaining=%ss; status=%s\n' "$((final_deadline - elapsed))" "$eni_status"
  else
    printf 'Waiting for Lambda ENIs: normal=%ss additional=%ss status=%s\n' \
      "$((normal_deadline - elapsed))" "$LAMBDA_ENI_EXTRA_WAIT_SECONDS" "$eni_status"
  fi
  sleep "$LAMBDA_ENI_POLL_SECONDS"
  elapsed="$((elapsed + LAMBDA_ENI_POLL_SECONDS))"
done

if [[ "$(jq 'length' <<<"$eni_json")" != "0" ]]; then
  while IFS= read -r eni; do
    eni_id="$(jq -r '.Id' <<<"$eni")"
    eni_status="$(jq -r '.Status' <<<"$eni")"
    eni_description="$(jq -r '.Description // ""' <<<"$eni")"
    attachment_id="$(jq -r '.AttachmentId // ""' <<<"$eni")"
    owns_security_group="$(jq --arg group "$security_group_id" '(.Groups | length == 1) and (.Groups[0] == $group)' <<<"$eni")"
    [[ "$eni_description" == "AWS Lambda VPC ENI-$function_name" ]] \
      || fail "refusing manual cleanup for ENI $eni_id with unexpected description"
    [[ "$owns_security_group" == "true" ]] \
      || fail "refusing manual cleanup for ENI $eni_id because its security groups are not exclusive to $security_group_id"

    if [[ "$eni_status" == "available" && -z "$attachment_id" ]]; then
      aws ec2 delete-network-interface --region "$AWS_REGION" --network-interface-id "$eni_id"
      printf 'Deleted available orphaned Lambda ENI %s.\n' "$eni_id"
      continue
    fi

    if aws ec2 delete-network-interface --region "$AWS_REGION" --network-interface-id "$eni_id" >/dev/null 2>&1; then
      printf 'Deleted Lambda ENI %s without detaching.\n' "$eni_id"
      continue
    fi

    [[ "$FORCE_DETACH_LAMBDA_ENI" == "true" ]] \
      || fail "ENI $eni_id is still attached; rerun with FORCE_DETACH_LAMBDA_ENI=true only after reviewing ownership"
    [[ -n "$attachment_id" ]] || fail "ENI $eni_id has no detachable attachment ID"
    aws ec2 detach-network-interface --region "$AWS_REGION" --attachment-id "$attachment_id" --force
    aws ec2 wait network-interface-available --region "$AWS_REGION" --network-interface-ids "$eni_id"
    aws ec2 delete-network-interface --region "$AWS_REGION" --network-interface-id "$eni_id"
    printf 'Force-detached and deleted Lambda ENI %s.\n' "$eni_id"
  done < <(jq -c '.[]' <<<"$eni_json")
fi

for attempt in {1..12}; do
  remaining_enis="$(aws ec2 describe-network-interfaces \
    --region "$AWS_REGION" \
    --filters "Name=group-id,Values=$security_group_id" \
    --query 'length(NetworkInterfaces)' \
    --output text)"
  [[ "$remaining_enis" == "0" ]] && break
  (( attempt < 12 )) || fail "Lambda ENIs still exist for security group $security_group_id after cleanup"
  sleep 5
done

terraform -chdir="$TF_DIR" destroy -refresh=false -input=false -auto-approve
printf 'Destroyed Lambda Terraform stack %s after ENI cleanup.\n' "$TF_DIR"
