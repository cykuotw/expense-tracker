#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
FRONTEND_TF="$REPO_ROOT/deployment/frontend/tf"
WORKER_TF="$REPO_ROOT/deployment/backend/serverless/worker/tf"
WORKER_SCRIPTS="$REPO_ROOT/deployment/backend/serverless/worker/scripts"
FRONTEND_DEPLOY="$REPO_ROOT/deployment/frontend/scripts/deploy-frontend.sh"
MAKEFILE="$REPO_ROOT/Makefile"

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

for command_name in grep make mktemp node python3; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done

grep -Fq 'resource "aws_s3_bucket" "frontend"' "$FRONTEND_TF/main.tf" \
  || fail 'dedicated frontend Terraform does not create a private S3 bucket'
grep -Fq 'resource "aws_cloudfront_origin_access_control" "frontend"' "$FRONTEND_TF/main.tf" \
  || fail 'dedicated frontend Terraform does not use CloudFront OAC'
grep -Fq 'price_class         = "PriceClass_100"' "$FRONTEND_TF/main.tf" \
  || fail 'dedicated frontend Terraform does not use the MVP CloudFront price class'
grep -Fq 'provider          = aws.us_east_1' "$FRONTEND_TF/main.tf" \
  || fail 'frontend certificate is not owned in us-east-1'
grep -Fq 'resource "aws_apigatewayv2_domain_name" "api"' "$WORKER_TF/main.tf" \
  || fail 'worker Terraform does not create the API custom domain'
grep -Fq 'resource "aws_apigatewayv2_api_mapping" "api"' "$WORKER_TF/main.tf" \
  || fail 'worker Terraform does not create the root API mapping'
grep -Fq 'ignore_changes = [disable_execute_api_endpoint]' "$WORKER_TF/main.tf" \
  || fail 'worker Terraform does not protect the live raw-endpoint toggle'
grep -Fq 'FRONTEND_DEPLOY_MODE=serverless' "$MAKEFILE" \
  || fail 'Makefile does not expose the serverless frontend deployment mode'
grep -Fq 'worker_custom_api_origin' "$FRONTEND_DEPLOY" \
  || fail 'frontend deploy does not consume the worker custom API origin'
grep -Fq 'worker_google_audience' "$FRONTEND_DEPLOY" \
  || fail 'frontend deploy does not consume the worker authorizer audience'

test_root="$(mktemp -d)"
cleanup() {
  rm -rf "$test_root"
}
trap cleanup EXIT

mkdir -p "$test_root/bin" "$test_root/tf"
printf '{"version":4,"resources":[]}\n' > "$test_root/tf/terraform.tfstate"
printf 'False\n' > "$test_root/raw-disabled"

cat > "$test_root/bin/terraform" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"output -raw worker_api_id") printf 'phase-10-api-id\n' ;;
  *"output -raw worker_aws_region") printf 'ca-central-1\n' ;;
  *) printf 'unexpected terraform arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK

cat > "$test_root/bin/aws" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"apigatewayv2 get-api"*) cat "$MOCK_RAW_ENDPOINT_STATE" ;;
  *"apigatewayv2 update-api"*"--no-disable-execute-api-endpoint"*) printf 'False\n' > "$MOCK_RAW_ENDPOINT_STATE" ;;
  *"apigatewayv2 update-api"*"--disable-execute-api-endpoint"*) printf 'True\n' > "$MOCK_RAW_ENDPOINT_STATE" ;;
  *) printf 'unexpected aws arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK
chmod +x "$test_root/bin/terraform" "$test_root/bin/aws"

PATH="$test_root/bin:$PATH" \
  MOCK_RAW_ENDPOINT_STATE="$test_root/raw-disabled" \
  TF_DIR="$test_root/tf" \
  "$WORKER_SCRIPTS/set-raw-endpoint.sh" disabled > "$test_root/disable.log"
[[ "$(cat "$test_root/raw-disabled")" == "True" ]] \
  || fail 'checked cutover did not disable the raw endpoint'

PATH="$test_root/bin:$PATH" \
  MOCK_RAW_ENDPOINT_STATE="$test_root/raw-disabled" \
  TF_DIR="$test_root/tf" \
  "$WORKER_SCRIPTS/set-raw-endpoint.sh" enabled > "$test_root/enable.log"
[[ "$(cat "$test_root/raw-disabled")" == "False" ]] \
  || fail 'checked rollback did not enable the raw endpoint'

mkdir -p \
  "$test_root/deploy-bin" \
  "$test_root/frontend-tf" \
  "$test_root/worker-tf" \
  "$test_root/serverful-tf" \
  "$test_root/dist" \
  "$test_root/serverful-dist"
printf 'aws_region = "ca-central-1"\n' > "$test_root/frontend-tf/terraform.tfvars"
printf 'aws_region = "ca-central-1"\n' > "$test_root/serverful-tf/terraform.tfvars"
cat > "$test_root/deploy-bin/terraform" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"-chdir=$MOCK_FRONTEND_TF output -json"*)
    printf '%s\n' '{"frontend_bucket_name":{"value":"phase-10-bucket"},"cloudfront_distribution_id":{"value":"phase-10-distribution"}}'
    ;;
  *"-chdir=$MOCK_WORKER_TF output -json"*)
    printf '%s\n' '{"worker_custom_api_origin":{"value":"https://api.example.com"},"worker_google_audience":{"value":"phase-10-client.apps.googleusercontent.com"}}'
    ;;
  *"-chdir=$MOCK_SERVERFUL_TF output -json"*)
    printf '%s\n' '{"frontend_bucket_name":{"value":"serverful-bucket"},"cloudfront_distribution_id":{"value":"serverful-distribution"},"api_fqdn":{"value":"legacy-api.example.test"}}'
    ;;
  *) printf 'unexpected terraform deploy arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK
cat > "$test_root/deploy-bin/make" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "build-frontend" ]] || { printf 'unexpected make arguments: %s\n' "$*" >&2; exit 1; }
MOCK
cat > "$test_root/deploy-bin/aws" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  *"s3 sync"* | *"cloudfront create-invalidation"*) ;;
  *) printf 'unexpected aws deploy arguments: %s\n' "$*" >&2; exit 1 ;;
esac
MOCK
chmod +x "$test_root/deploy-bin/terraform" "$test_root/deploy-bin/make" "$test_root/deploy-bin/aws"

PATH="$test_root/deploy-bin:$PATH" \
  MOCK_FRONTEND_TF="$test_root/frontend-tf" \
  MOCK_WORKER_TF="$test_root/worker-tf" \
  MOCK_SERVERFUL_TF="$test_root/serverful-tf" \
  AWS_REGION=ca-central-1 \
  FRONTEND_DEPLOY_MODE=serverless \
  TF_DIR="$test_root/frontend-tf" \
  WORKER_TF_DIR="$test_root/worker-tf" \
  FRONTEND_DIST_DIR="$test_root/dist" \
  "$FRONTEND_DEPLOY" > "$test_root/frontend-deploy.log"
grep -Fq '"apiOrigin": "https://api.example.com"' "$test_root/dist/runtime-config.js" \
  || fail 'serverless frontend runtime config does not use the worker custom API origin'
grep -Fq '"googleOAuthEnabled": true' "$test_root/dist/runtime-config.js" \
  || fail 'serverless frontend runtime config does not enable Google OAuth'
grep -Fq '"googleClientId": "phase-10-client.apps.googleusercontent.com"' "$test_root/dist/runtime-config.js" \
  || fail 'serverless frontend runtime config does not use the worker authorizer audience'

PATH="$test_root/deploy-bin:$PATH" \
  MOCK_FRONTEND_TF="$test_root/frontend-tf" \
  MOCK_WORKER_TF="$test_root/worker-tf" \
  MOCK_SERVERFUL_TF="$test_root/serverful-tf" \
  AWS_REGION=ca-central-1 \
  FRONTEND_DEPLOY_MODE=serverful \
  TF_DIR="$test_root/serverful-tf" \
  FRONTEND_DIST_DIR="$test_root/serverful-dist" \
  API_ORIGIN= \
  GOOGLE_OAUTH_ENABLED=false \
  GOOGLE_CLIENT_ID= \
  "$FRONTEND_DEPLOY" > "$test_root/serverful-deploy.log"
grep -Fq '"apiOrigin": "https://legacy-api.example.test"' "$test_root/serverful-dist/runtime-config.js" \
  || fail 'default frontend deployment no longer preserves the serverful API origin'
grep -Fq '"googleOAuthEnabled": false' "$test_root/serverful-dist/runtime-config.js" \
  || fail 'default frontend deployment no longer preserves serverful Google OAuth configuration'

make -C "$REPO_ROOT" -n \
  frontend-tf-init frontend-tf-plan frontend-tf-apply frontend-tf-destroy \
  serverless-frontend-deploy worker-disable-raw-endpoint worker-enable-raw-endpoint \
  > "$test_root/make.log"
grep -Fq 'set-raw-endpoint.sh disabled' "$test_root/make.log" \
  || fail 'Make dry-run does not expose raw-endpoint disable'
grep -Fq 'set-raw-endpoint.sh enabled' "$test_root/make.log" \
  || fail 'Make dry-run does not expose raw-endpoint rollback'

printf 'Phase 10 local contract checks passed.\n'
