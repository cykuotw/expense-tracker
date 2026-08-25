# Deployment

Deployment tooling for the static frontend, the EC2-backed application, the PostgreSQL host, and the Lambda worker.

## Layout

```text
deployment/
  frontend/
    lib/
    scripts/
  backend/
    shared/
      lib/
    serverful/
      backend/
        remote/
        systemd/
      edge/
        nginx/
        remote/
      scripts/
      shared/
        lib/
      tf/
    serverless/
      bootstrap/
        build/
        scripts/
        tf/
      postgres/
        scripts/
        tf/
      worker/
        build/
        scripts/
        tf/
```

## Requirements

- authenticated AWS CLI with a configured region, or `AWS_REGION`/`AWS_DEFAULT_REGION` set
- standard regional Lambda concurrency quota `1000` before Phase 9 bootstrap/worker activation
- Terraform `>= 1.6`
- Go and the frontend package-manager dependencies
- `zip`, `unzip`, `file`, and `curl`
- `jq` and GNU `realpath`/`stat` for Lambda runtime and smoke-test scripts

## EC2 Application

- `make deploy`: deploy the EC2-backed application and frontend.
- `make deploy all`: apply infrastructure, deploy the backend and frontend, then deploy the nginx edge release.
- `make deploy infra`: apply EC2-backed Terraform.
- `make deploy backend`: build and release the backend.
- `make deploy frontend`: build and publish frontend assets and runtime configuration.
- `make deploy edge`: package and release nginx/certbot assets.
- `make deploy help`: show deploy command help.
- `make tf-init`, `make tf-plan`, and `make tf-apply`: manage EC2-backed Terraform.
- `make destroy`: empty managed versioned buckets and destroy EC2-backed Terraform resources.

## PostgreSQL Host

- `make postgres-tf-init`, `make postgres-tf-plan`, and `make postgres-tf-apply`: manage the PostgreSQL EC2 infrastructure; native setup installs PostgreSQL 16 plus the contrib extensions required by bootstrap.
- `SSH_PRIVATE_KEY=/path/to/key POSTGRES_PASSWORD_FILE=/path/to/password make postgres-setup`: install and configure PostgreSQL after temporary access is enabled.
- `make postgres-tf-destroy`: destroy the PostgreSQL Terraform resources.

## Lambda Worker

- `make worker-build`: build the ARM64 Lambda ZIP.
- `make worker-check-static-authorizer`: check the JWT route contract locally.
- `make worker-tf-init`, `make worker-tf-plan`, and `make worker-tf-apply`: create and apply the reviewed saved worker plan with request processing disabled at reserved concurrency `0`.
- `make worker-check-authorizer`: verify that the deployed API rejects missing and malformed Google credentials.
- `RUNTIME_ENV_FILE=/protected/worker-runtime.json make worker-configure`: publish the complete Lambda environment from a mode-`0600` file outside the repository.
- `RUNTIME_ENV_FILE=/protected/worker-runtime.json make worker-activate`: run the quota and state-boundary checks, then activate the configured worker at reserved concurrency `3` without a Terraform apply.
- `RUNTIME_ENV_FILE=/protected/worker-runtime.json make worker-check-secret-boundary`: verify protected values are absent from local Terraform artifacts.
- `RUNTIME_ENV_FILE=/protected/worker-runtime.json make worker-tf-destroy`: delete the worker Lambda first, wait for its ENIs, then destroy the remaining worker/API resources.
- `make worker-update-code`: rebuild and publish code to the existing Lambda function.
- `make worker-health`: check the deployed API Gateway-to-Lambda request path.
- `make worker-check-boundaries`: check deployed CORS, preflight, and password-route boundaries.
- `GOOGLE_ID_TOKEN_FILE=/protected/google-id-token make worker-google-exchange`: verify Google exchange and the resulting application session with a short-lived token stored in a mode-`0600` file outside the repository.

## Bootstrap Lambda

- `make bootstrap-build`: build the private ARM64 bootstrap Lambda ZIP with bundled migrations.
- `make bootstrap-tf-init`, `make bootstrap-tf-plan`, and `make bootstrap-tf-apply`: resolve the selected AWS region, run the Lambda quota preflight, and manage the private bootstrap Lambda through a reviewed saved plan without a Bootstrap `.tfvars` file.
- `RUNTIME_ENV_FILE=/protected/bootstrap-runtime.json make bootstrap-configure`: publish database and optional first-admin values from a mode-`0600` JSON file outside the repository.
- `make bootstrap-update-code`: rebuild and publish code without invoking database work.
- `make bootstrap-invoke`: directly invoke the single idempotent `all` operation.
- `RUNTIME_ENV_FILE=/protected/bootstrap-runtime.json make bootstrap-check-secret-boundary`: verify protected values are absent from local Terraform artifacts.
- `RUNTIME_ENV_FILE=/protected/bootstrap-runtime.json make bootstrap-tf-destroy`: delete bootstrap first, retain its role while ENIs retire, then destroy the remaining stack.

## Safety

- Review every Terraform plan before applying it. Apply, destroy, deploy, configuration, and code-update commands modify AWS resources.
- Keep credentials, private keys, runtime environment files, token files, and other secrets outside the repository and Terraform inputs.
- Worker/bootstrap plan and destroy targets deliberately use `-refresh=false`. Do not run plan/apply after runtime publication; use the checked worker activation and Lambda-first destroy commands plus independent AWS verification because this workflow does not detect remote drift.
- Saved `*.tfplan` files are local review artifacts, are Git-ignored, and are removed after a successful apply.
- Back up any required database data before destructive infrastructure changes.
- Do not enable Lambda request processing until its database and runtime configuration are ready.
