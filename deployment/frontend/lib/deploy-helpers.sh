require_tf_vars_file() {
  if [[ ! -f "$(tf_vars_file_path)" ]]; then
    fail "missing TF_VARS_FILE: $(tf_vars_file_path)"
    exit 1
  fi
}
