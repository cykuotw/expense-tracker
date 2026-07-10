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

tf_var_string_optional() {
  local name="$1"
  local value=""
  if value="$(tf_var_string "$name" 2>/dev/null)"; then
    printf '%s\n' "$value"
    return
  fi
  printf '\n'
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

missing_deploy_outputs_hint() {
  printf "terraform state has no deploy outputs in %s; this usually means infra was destroyed, you just ran 'make destroy', or terraform apply has not been run yet. Run 'make deploy infra' to recreate infra outputs, or 'make deploy all' to recreate infra and continue the full deploy\n" "$TF_DIR" >&2
}

terraform_output() {
  local name="$1"
  local value
  local status
  if value="$(
    python3 - "$TF_DIR" "$name" <<'PY2'
import json
import subprocess
import sys

tf_dir = sys.argv[1]
name = sys.argv[2]
proc = subprocess.run(
    ["terraform", f"-chdir={tf_dir}", "output", "-json"],
    capture_output=True,
    text=True,
)
if proc.returncode != 0:
    raise SystemExit(2)
outputs = json.loads(proc.stdout)
if not outputs:
    raise SystemExit(3)
entry = outputs.get(name)
if entry is None:
    raise SystemExit(4)
value = entry.get("value")
if value in ("", None):
    raise SystemExit(5)
if isinstance(value, (dict, list)):
    print(json.dumps(value))
else:
    print(value)
PY2
  )"; then
    :
  else
    status=$?
    case "$status" in
      2)
        fail "failed to read terraform output '$name' from $TF_DIR; if you just ran 'make destroy', run 'make deploy infra' or 'make deploy all' first"
        ;;
      3)
        missing_deploy_outputs_hint
        ;;
      4)
        fail "missing required terraform output: $name; if infra was recently destroyed or changed, run 'make deploy infra' or 'make deploy all' to refresh state outputs"
        ;;
      5)
        fail "empty required terraform output: $name; check terraform state and outputs before deploying, or rerun 'make deploy infra' if infra was recently recreated"
        ;;
      *)
        fail "failed to resolve terraform output: $name"
        ;;
    esac
    exit 1
  fi
  printf '%s\n' "$value"
}

terraform_output_optional() {
  local name="$1"
  local value
  local status
  if value="$(
    python3 - "$TF_DIR" "$name" <<'PY2'
import json
import subprocess
import sys

tf_dir = sys.argv[1]
name = sys.argv[2]
proc = subprocess.run(
    ["terraform", f"-chdir={tf_dir}", "output", "-json"],
    capture_output=True,
    text=True,
)
if proc.returncode != 0:
    raise SystemExit(2)
outputs = json.loads(proc.stdout)
if not outputs:
    raise SystemExit(3)
entry = outputs.get(name)
if entry is None:
    raise SystemExit(0)
value = entry.get("value")
if value in ("", None):
    raise SystemExit(0)
if isinstance(value, (dict, list)):
    print(json.dumps(value))
else:
    print(value)
PY2
  )"; then
    :
  else
    status=$?
    case "$status" in
      2)
        fail "failed to read terraform outputs from $TF_DIR; if you just ran 'make destroy', run 'make deploy infra' or 'make deploy all' first"
        exit 1
        ;;
      3)
        missing_deploy_outputs_hint
        exit 1
        ;;
      *)
        printf '\n'
        return
        ;;
    esac
  fi
  printf '%s\n' "$value"
}
