#!/usr/bin/env bash
set -euo pipefail

TF_DIR="${TF_DIR:-}"
RUNTIME_ENV_FILE="${RUNTIME_ENV_FILE:-}"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

[[ -n "$TF_DIR" ]] || fail 'set TF_DIR to the worker or bootstrap Terraform directory'
[[ -d "$TF_DIR" ]] || fail "Terraform directory not found: $TF_DIR"

for command_name in base64 find grep mktemp realpath rm; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done
TF_DIR="$(realpath "$TF_DIR")"

forbidden_key_pattern='DB_PASSWORD|DB_ADMIN_PASSWORD|DB_MIGRATION_PASSWORD|DB_RUNTIME_PASSWORD|JWT_SECRET|REFRESH_JWT_SECRET|FIRST_ADMIN_PASSWORD'
artifacts=()
while IFS= read -r -d '' artifact; do
  artifacts+=("$artifact")
done < <(
  find "$TF_DIR" -maxdepth 1 -type f \
    \( -name '*.tfstate' -o -name '*.tfstate.*' -o -name '*.tfplan' -o -name '*.plan' -o -name '*.log' \) \
    -print0
)

scan_files=()
temporary_files=()
cleanup() {
  if ((${#temporary_files[@]} > 0)); then
    rm -f -- "${temporary_files[@]}"
  fi
}
trap cleanup EXIT

for artifact in "${artifacts[@]}"; do
  case "$artifact" in
    *.tfplan | *.plan)
      command -v terraform >/dev/null 2>&1 || fail 'required command not found for saved-plan inspection: terraform'
      rendered_plan="$(mktemp)"
      temporary_files+=("$rendered_plan")
      if ! terraform -chdir="$TF_DIR" show -json "$artifact" >"$rendered_plan"; then
        fail "could not inspect saved Terraform plan: $artifact"
      fi
      scan_files+=("$rendered_plan")
      ;;
    *)
      scan_files+=("$artifact")
      ;;
  esac
done

for index in "${!artifacts[@]}"; do
  if grep -aE "$forbidden_key_pattern" "${scan_files[$index]}" >/dev/null; then
    artifact="${artifacts[$index]}"
    fail "protected runtime key detected in Terraform artifact: $artifact"
  fi
done

if [[ -n "$RUNTIME_ENV_FILE" ]]; then
  command -v jq >/dev/null 2>&1 || fail 'required command not found: jq'
  [[ -f "$RUNTIME_ENV_FILE" ]] || fail "runtime environment file not found: $RUNTIME_ENV_FILE"
  while IFS= read -r encoded_value; do
    secret_value="$(printf '%s' "$encoded_value" | base64 --decode)"
    [[ -n "$secret_value" ]] || continue
    for index in "${!artifacts[@]}"; do
      if grep -aF -- "$secret_value" "${scan_files[$index]}" >/dev/null; then
        artifact="${artifacts[$index]}"
        fail "protected runtime value detected in Terraform artifact: $artifact"
      fi
    done
  done < <(
    jq -r '
      .Variables
      | to_entries[]
      | select(.key | test("(?:PASSWORD|SECRET)$"))
      | select(.value | type == "string" and length > 0)
      | .value
      | @base64
    ' "$RUNTIME_ENV_FILE"
  )
fi

printf 'Runtime secret boundary check passed for %s (%s artifact(s) checked)\n' "$TF_DIR" "${#artifacts[@]}"
