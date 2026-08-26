tf_vars_file_path() {
  if [[ "$TF_VARS_FILE" = /* ]]; then printf '%s\n' "$TF_VARS_FILE"; else printf '%s\n' "$TF_DIR/$TF_VARS_FILE"; fi
}

tf_var_string() {
  local name="$1"
  python3 - "$name" "$(tf_vars_file_path)" <<'PY'
import pathlib
import re
import sys

name = sys.argv[1]
pattern = re.compile(rf'^\s*{re.escape(name)}\s*=\s*(?:"((?:[^"\\]|\\.)*)"|([^\s#]+))\s*(?:#.*)?$')
for line in pathlib.Path(sys.argv[2]).read_text().splitlines():
    match = pattern.match(line)
    if match:
        print(bytes(next(value for value in match.groups() if value is not None), "utf-8").decode("unicode_escape"))
        raise SystemExit(0)
raise SystemExit(1)
PY
}

resolve_aws_region() {
  if [[ -n "${AWS_REGION:-}" ]]; then return; fi
  AWS_REGION="$(tf_var_string aws_region 2>/dev/null || true)"
  [[ -n "$AWS_REGION" ]] || { fail "set AWS_REGION or define aws_region in $(tf_vars_file_path)"; exit 1; }
}

terraform_output() {
  local name="$1"
  local value
  value="$(terraform -chdir="$TF_DIR" output -raw "$name")" || { fail "failed to read Terraform output: $name"; exit 1; }
  [[ -n "$value" ]] || { fail "empty Terraform output: $name"; exit 1; }
  printf '%s\n' "$value"
}
