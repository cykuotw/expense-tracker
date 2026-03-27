tf_vars_file_path() {
  if [[ "$TF_VARS_FILE" = /* ]]; then
    printf '%s\n' "$TF_VARS_FILE"
  else
    printf '%s\n' "$TF_DIR/$TF_VARS_FILE"
  fi
}

tf_var_string() {
  local name="$1"
  python3 - "$name" "$(tf_vars_file_path)" <<'PY2'
import pathlib
import re
import sys

name = sys.argv[1]
path = pathlib.Path(sys.argv[2])
pattern = re.compile(rf'^\s*{re.escape(name)}\s*=\s*(?:"((?:[^"\\]|\\.)*)"|([^\s#]+))\s*(?:#.*)?$')

for line in path.read_text().splitlines():
    match = pattern.match(line)
    if not match:
        continue
    quoted, bare = match.groups()
    value = quoted if quoted is not None else bare
    value = bytes(value, "utf-8").decode("unicode_escape")
    print(value)
    raise SystemExit(0)

raise SystemExit(1)
PY2
}

resolve_aws_region() {
  if [[ -n "${AWS_REGION:-}" ]]; then
    return
  fi

  if ! AWS_REGION="$(tf_var_string aws_region 2>/dev/null)"; then
    fail "set AWS_REGION or define aws_region in $(tf_vars_file_path)"
    exit 1
  fi

  if [[ -z "$AWS_REGION" ]]; then
    fail "empty aws_region in $(tf_vars_file_path)"
    exit 1
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

terraform_output_optional() {
  local name="$1"
  local value
  if ! value="$(terraform -chdir="$TF_DIR" output -raw "$name" 2>/dev/null)"; then
    fail "missing terraform output: $name"
    exit 1
  fi
  printf '%s\n' "$value"
}
