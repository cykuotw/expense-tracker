# Deployment

This directory is the active deployment layout for frontend publication and the current EC2-backed backend runtime.

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
```

## Ownership

- `deployment/frontend/` owns static frontend publication and deployed `runtime-config.js` generation.
- `deployment/backend/shared/` owns backend deploy defaults that are not specific to the current EC2/serverful release path.
- `deployment/backend/serverful/` owns the current EC2-backed backend path: backend release packaging, remote host release scripts, systemd, nginx/certbot edge release assets, serverful helper scripts, and serverful Terraform.

## Commands

- `make deploy`: defaults to `make deploy app`.
- `make deploy app`: deploys the current serverful backend, then publishes the frontend.
- `make deploy all`: applies serverful infrastructure, deploys the serverful backend, publishes the frontend, then deploys the edge/nginx release.
- `make deploy infra`: applies Terraform under `deployment/backend/serverful/tf`.
- `make deploy backend`: builds and releases the current serverful backend only.
- `make deploy frontend`: builds frontend assets, writes deployed `runtime-config.js`, publishes to S3, and invalidates CloudFront.
- `make deploy edge`: packages and releases nginx/certbot edge assets only.
- `make deploy help`: prints deploy command help.
- `make destroy`: empties versioned frontend/artifact buckets, then destroys serverful Terraform-managed infrastructure.
- `make tf-init`, `make tf-plan`, and `make tf-apply`: run Terraform against `deployment/backend/serverful/tf`.

## Current Flow

1. `deployment/backend/serverful/scripts/apply-infra.sh` provisions or updates current serverful AWS infrastructure.
2. `deployment/backend/serverful/scripts/deploy-backend.sh` reads Terraform outputs, stages backend runtime and deploy payloads, uploads the backend artifact, and runs the remote backend release through SSM.
3. The remote backend release installs runtime files, renders the backend env from SSM, bootstraps database roles, runs migrations, optionally creates the first admin user, and restarts the Go service.
4. `deployment/frontend/scripts/deploy-frontend.sh` builds the frontend, writes deployed `runtime-config.js` from Terraform outputs and deploy-time defaults, uploads assets, and invalidates CloudFront.
5. `deployment/backend/serverful/scripts/deploy-edge.sh` stages nginx/TLS payloads, uploads the edge artifact, runs the remote edge release through SSM, and checks API health.

## Notes

- Terraform is limited to infrastructure, DNS, buckets, IAM, current database credential locations, and deploy-time outputs. Application release artifacts are handled by deploy scripts.
- The current serverful Terraform still contains legacy/current RDS resources and outputs. This layout move does not redesign or remove those resources.
- Set `TF_VARS_FILE` only when not using the default `terraform.tfvars` inside `deployment/backend/serverful/tf`; `AWS_REGION` is inferred from the selected tfvars file unless overridden in the shell.
- `APP_DIR`, `BACKEND_ENV_PATH`, and `SYSTEMD_SERVICE_NAME` are part of the serverful backend deploy contract.
- `deployment/frontend/scripts/deploy-frontend.sh` owns deployed frontend runtime config and does not mutate backend runtime configuration.
