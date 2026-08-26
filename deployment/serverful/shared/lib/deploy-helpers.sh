# ------------
# Terraform and deploy context helpers
# ------------

require_tf_vars_file() {
  if [[ ! -f "$(tf_vars_file_path)" ]]; then
    fail "missing TF_VARS_FILE: $(tf_vars_file_path)"
    exit 1
  fi
}

resolve_certbot_settings() {
  if ! CERTBOT_ENABLED="$(tf_var_string certbot_enabled 2>/dev/null)"; then
    fail "define certbot_enabled in $(tf_vars_file_path)"
    exit 1
  fi
  if ! CERTBOT_STAGING="$(tf_var_string certbot_staging 2>/dev/null)"; then
    fail "define certbot_staging in $(tf_vars_file_path)"
    exit 1
  fi
  if certbot_email_override="$(tf_var_string certbot_email 2>/dev/null)"; then
    CERTBOT_EMAIL="$certbot_email_override"
  elif [[ "$CERTBOT_ENABLED" == "true" ]]; then
    fail "define certbot_email in $(tf_vars_file_path) when certbot_enabled=true"
    exit 1
  else
    CERTBOT_EMAIL=""
  fi
}

resolve_first_admin_bootstrap_settings() {
  PROJECT_NAME="$(tf_var_string project_name)"
  DEPLOY_ENVIRONMENT="$(tf_var_string_optional environment)"
  DEPLOY_ENVIRONMENT="${DEPLOY_ENVIRONMENT:-dev}"
  FIRST_ADMIN_EMAIL="$(tf_var_string_optional first_admin_email)"
  FIRST_ADMIN_PASSWORD="$(tf_var_string_optional first_admin_password)"
  FIRST_ADMIN_FIRSTNAME="$(tf_var_string_optional first_admin_firstname)"
  FIRST_ADMIN_LASTNAME="$(tf_var_string_optional first_admin_lastname)"
  FIRST_ADMIN_NICKNAME="$(tf_var_string_optional first_admin_nickname)"
  FIRST_ADMIN_BOOTSTRAP_SSM_PARAMETER_PREFIX="/$PROJECT_NAME/$DEPLOY_ENVIRONMENT/deploy/first_admin"
  FIRST_ADMIN_BOOTSTRAP_ENABLED=false
  FIRST_ADMIN_BOOTSTRAP_PARAMETER_NAMES=()

  if [[ -n "$FIRST_ADMIN_EMAIL$FIRST_ADMIN_PASSWORD$FIRST_ADMIN_FIRSTNAME$FIRST_ADMIN_LASTNAME$FIRST_ADMIN_NICKNAME" ]]; then
    FIRST_ADMIN_BOOTSTRAP_ENABLED=true
    for required_var in FIRST_ADMIN_EMAIL FIRST_ADMIN_PASSWORD FIRST_ADMIN_FIRSTNAME FIRST_ADMIN_LASTNAME; do
      if [[ -z "${!required_var:-}" ]]; then
        fail "define $required_var in $(tf_vars_file_path) when bootstrapping the first admin during deploy"
        exit 1
      fi
    done
  fi
}

# ------------
# File and manifest helpers
# ------------

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    fail "required file not found: $path"
    exit 1
  fi
}

write_shell_var() {
  local file="$1"
  local key="$2"
  local value="$3"
  printf '%s=%q\n' "$key" "$value" >>"$file"
}

# ------------
# Shell and SSM payload helpers
# ------------

shell_quote() {
  printf '%q' "$1"
}

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  value="${value//$'\f'/\\f}"
  value="${value//$'\b'/\\b}"
  printf '%s' "$value"
}

build_ssm_parameters() {
  local commands=("$@")
  local json='{"commands":['
  local index

  for index in "${!commands[@]}"; do
    if (( index > 0 )); then
      json+=','
    fi
    json+="\"$(json_escape "${commands[$index]}")\""
  done

  json+=']}'
  printf '%s\n' "$json"
}

# ------------
# Remote release helpers
# ------------

run_remote_release_via_ssm() {
  local instance_id="$1"
  local s3_uri="$2"
  local remote_script_path="$3"
  local aws_region_quoted
  local s3_uri_quoted
  local command_id
  local remote_script_path_literal
  local ssm_parameters
  local ssm_commands

  step 'Running remote release through SSM'
  aws_region_quoted="$(shell_quote "$AWS_REGION")"
  s3_uri_quoted="$(shell_quote "$s3_uri")"
  remote_script_path_literal="$remote_script_path"
  ssm_commands=(
    'set -euo pipefail'
    'TMP_RELEASE="$(mktemp /tmp/release-bundle.XXXXXX.tar.gz)"'
    'TMP_RELEASE_DIR="$(mktemp -d /tmp/release-bundle.XXXXXX)"'
    'cleanup() { rm -f "$TMP_RELEASE"; rm -rf "$TMP_RELEASE_DIR"; }'
    'trap cleanup EXIT'
    "aws --region $aws_region_quoted s3 cp $s3_uri_quoted \"\$TMP_RELEASE\""
    'tar -xzf "$TMP_RELEASE" -C "$TMP_RELEASE_DIR"'
    "chmod 755 \"\$TMP_RELEASE_DIR/$remote_script_path_literal\""
    "\"\$TMP_RELEASE_DIR/$remote_script_path_literal\" \"\$TMP_RELEASE_DIR\""
  )
  ssm_parameters="$(build_ssm_parameters "${ssm_commands[@]}")"
  command_id="$({
    aws --region "$AWS_REGION" ssm send-command \
      --instance-ids "$instance_id" \
      --document-name AWS-RunShellScript \
      --comment 'expense-tracker deploy' \
      --parameters "$ssm_parameters" \
      --query 'Command.CommandId' \
      --output text
  })"

  set +e
  aws --region "$AWS_REGION" ssm wait command-executed --command-id "$command_id" --instance-id "$instance_id"
  WAIT_EXIT=$?
  SSM_STATUS="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$command_id" --instance-id "$instance_id" --query 'Status' --output text)"
  STATUS_EXIT=$?
  SSM_RESPONSE_CODE="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$command_id" --instance-id "$instance_id" --query 'ResponseCode' --output text)"
  RESPONSE_CODE_EXIT=$?
  SSM_STDOUT="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$command_id" --instance-id "$instance_id" --query 'StandardOutputContent' --output text)"
  STDOUT_EXIT=$?
  SSM_STDERR="$(aws --region "$AWS_REGION" ssm get-command-invocation --command-id "$command_id" --instance-id "$instance_id" --query 'StandardErrorContent' --output text)"
  STDERR_EXIT=$?
  set -e

  if [[ $STDOUT_EXIT -eq 0 && -n "$SSM_STDOUT" && "$SSM_STDOUT" != "None" ]]; then
    printf 'Remote stdout:\n'
    printf '%s\n' "$SSM_STDOUT"
  fi

  if [[ $STDERR_EXIT -eq 0 && -n "$SSM_STDERR" && "$SSM_STDERR" != "None" ]]; then
    printf 'Remote stderr:\n' >&2
    printf '%s\n' "$SSM_STDERR" >&2
  fi

  if [[ $WAIT_EXIT -ne 0 || $STATUS_EXIT -ne 0 || $RESPONSE_CODE_EXIT -ne 0 || $STDOUT_EXIT -ne 0 || $STDERR_EXIT -ne 0 ]]; then
    fail "remote deploy command failed for command $command_id"
    exit 1
  fi

  if [[ "$SSM_STATUS" != "Success" || "$SSM_RESPONSE_CODE" != "0" ]]; then
    fail "remote deploy failed with SSM status=$SSM_STATUS response_code=$SSM_RESPONSE_CODE"
    exit 1
  fi
}

# ------------
# AWS parameter helpers
# ------------

create_ssm_parameter() {
  local name="$1"
  local type="$2"
  local value="$3"
  aws --region "$AWS_REGION" ssm put-parameter --name "$name" --type "$type" --value "$value" --overwrite >/dev/null
}

delete_ssm_parameter_if_exists() {
  local name="$1"
  local output
  if output="$(aws --region "$AWS_REGION" ssm delete-parameter --name "$name" 2>&1)"; then
    return
  fi
  if [[ "$output" == *"ParameterNotFound"* ]]; then
    return
  fi
  warn "failed to delete temporary SSM parameter $name: $output"
}
