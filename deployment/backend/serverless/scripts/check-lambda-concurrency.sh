#!/usr/bin/env bash
set -euo pipefail

STANDARD_ACCOUNT_CONCURRENCY="${STANDARD_ACCOUNT_CONCURRENCY:-1000}"
REQUIRED_RESERVED_CONCURRENCY="${REQUIRED_RESERVED_CONCURRENCY:-4}"
LAMBDA_ACCOUNT_SETTINGS_FILE="${LAMBDA_ACCOUNT_SETTINGS_FILE:-}"
AWS_REGION="${AWS_REGION:-${AWS_DEFAULT_REGION:-${TF_VAR_aws_region:-}}}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

command -v jq >/dev/null 2>&1 || fail 'required command not found: jq'

if [[ -n "$LAMBDA_ACCOUNT_SETTINGS_FILE" ]]; then
  [[ -f "$LAMBDA_ACCOUNT_SETTINGS_FILE" ]] || fail "account settings file not found: $LAMBDA_ACCOUNT_SETTINGS_FILE"
  account_settings="$(<"$LAMBDA_ACCOUNT_SETTINGS_FILE")"
else
  command -v aws >/dev/null 2>&1 || fail 'required command not found: aws'
  if [[ -z "$AWS_REGION" ]]; then
    AWS_REGION="$(aws configure get region 2>/dev/null || true)"
  fi
  [[ -n "$AWS_REGION" ]] || fail 'set AWS_REGION, AWS_DEFAULT_REGION, or TF_VAR_aws_region'
  account_settings="$(aws lambda get-account-settings --region "$AWS_REGION" --output json)"
fi

total_concurrency="$(jq -er '.AccountLimit.ConcurrentExecutions | numbers' <<<"$account_settings")" \
  || fail 'Lambda account settings do not include AccountLimit.ConcurrentExecutions'
unreserved_concurrency="$(jq -er '.AccountLimit.UnreservedConcurrentExecutions | numbers' <<<"$account_settings")" \
  || fail 'Lambda account settings do not include AccountLimit.UnreservedConcurrentExecutions'

(( total_concurrency >= STANDARD_ACCOUNT_CONCURRENCY )) || fail \
  "Lambda regional concurrency quota is $total_concurrency; request the standard $STANDARD_ACCOUNT_CONCURRENCY quota before Phase 9"
(( unreserved_concurrency >= REQUIRED_RESERVED_CONCURRENCY )) || fail \
  "only $unreserved_concurrency Lambda concurrency units remain available; Phase 8/9 require $REQUIRED_RESERVED_CONCURRENCY"

printf 'Lambda concurrency preflight passed: total=%s, available=%s, planned-reserved=%s\n' \
  "$total_concurrency" "$unreserved_concurrency" "$REQUIRED_RESERVED_CONCURRENCY"
