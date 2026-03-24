tf_vars_file_path() {
  if [[ "$TF_VARS_FILE" = /* ]]; then
    printf '%s\n' "$TF_VARS_FILE"
  else
    printf '%s\n' "$TF_DIR/$TF_VARS_FILE"
  fi
}

terraform_cmd() {
  local subcommand="$1"
  shift

  case "$subcommand" in
    apply|plan|destroy)
      if [[ -n "$TF_VARS_FILE" ]]; then
        terraform -chdir="$TF_DIR" "$subcommand" "$@" -var-file="$TF_VARS_FILE"
      else
        terraform -chdir="$TF_DIR" "$subcommand" "$@"
      fi
      ;;
    *)
      terraform -chdir="$TF_DIR" "$subcommand" "$@"
      ;;
  esac
}

terraform_output() {
  local name="$1"
  local value
  if ! value="$(terraform -chdir="$TF_DIR" output -raw "$name" 2>/dev/null)"; then
    fail "missing terraform output: $name"
    exit 1
  fi
  if [[ -z "$value" ]]; then
    fail "empty terraform output: $name"
    exit 1
  fi
  printf '%s\n' "$value"
}
