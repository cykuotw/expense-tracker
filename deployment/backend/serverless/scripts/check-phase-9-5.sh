#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
POSTGRES_SETUP="$SCRIPT_DIR/../postgres/scripts/setup-postgres.sh"
WORKER_ACTIVATE="$SCRIPT_DIR/../worker/scripts/activate.sh"
WORKER_VARIABLES="$SCRIPT_DIR/../worker/tf/variables.tf"
MAKEFILE="$REPO_ROOT/Makefile"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

for command_name in jq make mktemp; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done

test_root="$(mktemp -d)"
cleanup() {
  rm -rf "$test_root"
}
trap cleanup EXIT

cat > "$test_root/account-good.json" <<'JSON'
{"AccountLimit":{"ConcurrentExecutions":1000,"UnreservedConcurrentExecutions":996}}
JSON
cat > "$test_root/account-reduced.json" <<'JSON'
{"AccountLimit":{"ConcurrentExecutions":10,"UnreservedConcurrentExecutions":10}}
JSON

LAMBDA_ACCOUNT_SETTINGS_FILE="$test_root/account-good.json" \
  "$SCRIPT_DIR/check-lambda-concurrency.sh" >/dev/null
if LAMBDA_ACCOUNT_SETTINGS_FILE="$test_root/account-reduced.json" \
  "$SCRIPT_DIR/check-lambda-concurrency.sh" >/dev/null 2>&1; then
  fail 'reduced Lambda quota unexpectedly passed the preflight'
fi

mkdir -p "$test_root/tf"
printf '{"version":4,"resources":[]}\n' > "$test_root/tf/terraform.tfstate"
TF_DIR="$test_root/tf" "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null

sentinel_password="phase-9-5-password-$RANDOM-$RANDOM"
sentinel_jwt="phase-9-5-jwt-$RANDOM-$RANDOM"
jq -n \
  --arg password "$sentinel_password" \
  --arg jwt "$sentinel_jwt" \
  '{Variables:{DB_PASSWORD:$password,JWT_SECRET:$jwt}}' \
  > "$test_root/runtime.json"
RUNTIME_ENV_FILE="$test_root/runtime.json" TF_DIR="$test_root/tf" \
  "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null

printf '{"DB_PASSWORD":"not-a-runtime-secret"}\n' > "$test_root/tf/terraform.tfstate"
if RUNTIME_ENV_FILE="$test_root/runtime.json" TF_DIR="$test_root/tf" \
  "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null 2>&1; then
  fail 'sentinel secret unexpectedly passed the Terraform artifact check'
fi

printf '{"unrelated":"%s"}\n' "$sentinel_password" > "$test_root/tf/terraform.tfstate"
if RUNTIME_ENV_FILE="$test_root/runtime.json" TF_DIR="$test_root/tf" \
  "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null 2>&1; then
  fail 'sentinel value unexpectedly passed the Terraform state check'
fi

printf '{"version":4,"resources":[]}\n' > "$test_root/tf/terraform.tfstate"
printf 'captured value: %s\n' "$sentinel_jwt" > "$test_root/tf/phase-9-5.log"
if RUNTIME_ENV_FILE="$test_root/runtime.json" TF_DIR="$test_root/tf" \
  "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null 2>&1; then
  fail 'sentinel value unexpectedly passed the retained-log check'
fi
rm -f "$test_root/tf/phase-9-5.log"

mkdir -p "$test_root/plan-bin"
cat > "$test_root/plan-bin/terraform" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
plan_path="${!#}"
[[ "$plan_path" != *malformed.tfplan ]] || exit 1
cat "$plan_path"
MOCK
chmod +x "$test_root/plan-bin/terraform"
printf '{"planned":"%s"}\n' "$sentinel_password" > "$test_root/tf/sentinel.tfplan"
if PATH="$test_root/plan-bin:$PATH" RUNTIME_ENV_FILE="$test_root/runtime.json" TF_DIR="$test_root/tf" \
  "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null 2>&1; then
  fail 'sentinel value unexpectedly passed the saved-plan check'
fi
rm -f "$test_root/tf/sentinel.tfplan"
printf 'not a plan\n' > "$test_root/tf/malformed.tfplan"
if PATH="$test_root/plan-bin:$PATH" TF_DIR="$test_root/tf" \
  "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null 2>&1; then
  fail 'malformed saved plan unexpectedly passed the fail-closed check'
fi
rm -f "$test_root/tf/malformed.tfplan"

grep -Fq 'dnf install -y postgresql16 postgresql16-server postgresql16-contrib' "$POSTGRES_SETUP" \
  || fail 'Phase 7 setup does not install postgresql16-contrib'
grep -Fq "pg_available_extensions WHERE name = 'pgcrypto'" "$POSTGRES_SETUP" \
  || fail 'Phase 7 setup does not verify pgcrypto availability'
grep -Fq 'plan -refresh=false -input=false -out=' "$MAKEFILE" \
  || fail 'worker/bootstrap plans do not disable refresh and save the reviewed plan'
grep -Fq 'worker-activate:' "$MAKEFILE" \
  || fail 'Makefile does not expose the worker Bash activation path'
if grep -Fq 'WORKER_RESERVED_CONCURRENCY' "$MAKEFILE"; then
  fail 'Makefile still exposes Terraform-based worker activation'
fi
grep -Fq ".variables.reserved_concurrency.value == 0" "$MAKEFILE" \
  || fail 'worker Terraform apply does not reject an old activation plan'
grep -Fq 'reserved_concurrency must remain 0 in Terraform' "$WORKER_VARIABLES" \
  || fail 'worker Terraform does not reject activation concurrency'
grep -Fq 'put-function-concurrency' "$WORKER_ACTIVATE" \
  || fail 'worker activation does not set live Lambda concurrency'
grep -Fq 'get-function-concurrency' "$WORKER_ACTIVATE" \
  || fail 'worker activation does not verify live Lambda concurrency'
[[ -x "$WORKER_ACTIVATE" ]] || fail 'worker activation script is not executable'
grep -Fq 'destroy -refresh=false' "$SCRIPT_DIR/destroy-lambda-stack.sh" \
  || fail 'Lambda stack destroy does not disable refresh'
grep -Fq 'SERVERLESS_AWS_REGION = $(strip $(or $(AWS_REGION),$(AWS_DEFAULT_REGION),$(TF_VAR_aws_region),$(shell aws configure get region 2>/dev/null)))' "$MAKEFILE" \
  || fail 'Makefile does not resolve the serverless AWS region in the supported order'

make -C "$REPO_ROOT" -n \
  bootstrap-tf-init \
  bootstrap-tf-plan \
  bootstrap-tf-apply \
  bootstrap-tf-destroy \
  bootstrap-update-code \
  bootstrap-configure \
  bootstrap-invoke \
  lambda-concurrency-check \
  AWS_REGION=ca-test-1 > "$test_root/bootstrap-region.log"
[[ "$(grep -Fc 'AWS_REGION="ca-test-1"' "$test_root/bootstrap-region.log")" -ge 10 ]] \
  || fail 'Bootstrap commands do not consistently receive the resolved AWS_REGION'
[[ "$(grep -Fc 'TF_VAR_aws_region="ca-test-1"' "$test_root/bootstrap-region.log")" -ge 10 ]] \
  || fail 'Bootstrap commands do not consistently receive the resolved Terraform region'

env -u AWS_REGION -u TF_VAR_aws_region \
  make -C "$REPO_ROOT" -n bootstrap-tf-destroy AWS_DEFAULT_REGION=ca-fallback-1 \
  > "$test_root/bootstrap-region-fallback.log"
grep -Fq 'AWS_REGION="ca-fallback-1" TF_VAR_aws_region="ca-fallback-1"' \
  "$test_root/bootstrap-region-fallback.log" \
  || fail 'Bootstrap commands do not fall back to AWS_DEFAULT_REGION'

mkdir -p "$test_root/activate-bin" "$test_root/activate-tf"
printf '{"version":4,"resources":[]}\n' > "$test_root/activate-tf/terraform.tfstate"
cat > "$test_root/activate-bin/terraform" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"output -raw worker_function_name") printf 'phase-9-5-mock-worker\n' ;;
  *"output -raw worker_aws_region") printf 'ca-central-1\n' ;;
  *) printf 'unexpected terraform activation arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK
cat > "$test_root/activate-bin/aws" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"lambda get-function-concurrency"*)
    if [[ -f "$MOCK_ACTIVATION_MARKER" ]]; then printf '3\n'; else printf '0\n'; fi
    ;;
  *"lambda put-function-concurrency"*"--reserved-concurrent-executions 3"*)
    : > "$MOCK_ACTIVATION_MARKER"
    ;;
  *) printf 'unexpected aws activation arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK
chmod +x "$test_root/activate-bin/terraform" "$test_root/activate-bin/aws"
PATH="$test_root/activate-bin:$PATH" \
  MOCK_ACTIVATION_MARKER="$test_root/worker-activated" \
  LAMBDA_ACCOUNT_SETTINGS_FILE="$test_root/account-good.json" \
  TF_DIR="$test_root/activate-tf" \
  RUNTIME_ENV_FILE="$test_root/runtime.json" \
  "$WORKER_ACTIVATE" > "$test_root/activate.log"
[[ -f "$test_root/worker-activated" ]] || fail 'worker Bash activation did not reserve concurrency 3'
if grep -aF -e "$sentinel_password" -e "$sentinel_jwt" "$test_root/activate.log" >/dev/null; then
  fail 'worker activation output exposed a sentinel runtime value'
fi
RUNTIME_ENV_FILE="$test_root/runtime.json" TF_DIR="$test_root/activate-tf" \
  "$SCRIPT_DIR/check-runtime-secret-boundary.sh" >/dev/null

mkdir -p "$test_root/mock-bin" "$test_root/destroy-tf"
printf '{"version":4,"resources":[]}\n' > "$test_root/destroy-tf/terraform.tfstate"
cat > "$test_root/mock-bin/terraform" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"output -raw mock_function") printf 'phase-9-5-mock-function\n' ;;
  *"output -raw mock_security_group") printf 'sg-0123456789abcdef0\n' ;;
  *"destroy -refresh=false -input=false -auto-approve") : > "$MOCK_DESTROY_MARKER" ;;
  *) printf 'unexpected terraform mock arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK
cat > "$test_root/mock-bin/aws" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"lambda get-function"*) exit 0 ;;
  *"lambda delete-function"*) : > "$MOCK_DELETE_MARKER" ;;
  *"ec2 describe-network-interfaces"*"length(NetworkInterfaces)"*) printf '0\n' ;;
  *"ec2 describe-network-interfaces"*) printf '[]\n' ;;
  *) printf 'unexpected aws mock arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK
chmod +x "$test_root/mock-bin/terraform" "$test_root/mock-bin/aws"
PATH="$test_root/mock-bin:$PATH" \
  MOCK_DELETE_MARKER="$test_root/lambda-deleted" \
  MOCK_DESTROY_MARKER="$test_root/terraform-destroyed" \
  AWS_REGION=ca-central-1 \
  AUTO_APPROVE=true \
  TF_DIR="$test_root/destroy-tf" \
  FUNCTION_OUTPUT=mock_function \
  SECURITY_GROUP_OUTPUT=mock_security_group \
  RUNTIME_ENV_FILE="$test_root/runtime.json" \
  "$SCRIPT_DIR/destroy-lambda-stack.sh" >"$test_root/destroy.log"
[[ -f "$test_root/lambda-deleted" ]] || fail 'Lambda-first cleanup did not delete the function'
[[ -f "$test_root/terraform-destroyed" ]] || fail 'Lambda-first cleanup did not finish Terraform destroy'
if grep -aF -e "$sentinel_password" -e "$sentinel_jwt" "$test_root/destroy.log" >/dev/null; then
  fail 'destroy output exposed a sentinel runtime value'
fi

printf 'Phase 9.5 local contract checks passed.\n'
