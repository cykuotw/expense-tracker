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
    serverless/
      postgres/
        scripts/
        tf/
```

## Ownership

- `deployment/frontend/` owns static frontend publication and deployed `runtime-config.js` generation.
- `deployment/backend/shared/` owns backend deploy defaults that are not specific to the current EC2/serverful release path.
- `deployment/backend/serverful/` owns the current EC2-backed backend path: backend release packaging, remote host release scripts, systemd, nginx/certbot edge release assets, serverful helper scripts, and serverful Terraform.
- `deployment/backend/serverless/postgres/` owns the minimal native PostgreSQL EC2 used by future worker/bootstrap Lambda paths.

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
- `make postgres-tf-init`, `make postgres-tf-plan`, `make postgres-tf-apply`, and `postgres-tf-destroy`: manage the minimal PostgreSQL EC2 Terraform. Apply changes AWS resources and requires operator approval.
- `SSH_PRIVATE_KEY=/path/to/key POSTGRES_PASSWORD_FILE=/path/to/password make postgres-setup`: installs and configures native PostgreSQL 16 through the temporary EIP after temporary public access is enabled and applied.

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

## Minimal PostgreSQL EC2

The Phase 7 MVP creates one AL2023 ARM64 `t4g.micro` with one encrypted `10GB` gp3 root volume. Automatic public IPv4 assignment is disabled. PostgreSQL port `5432` accepts only the worker/bootstrap client security groups and is never exposed to a public CIDR.

Temporary public access defaults to false. For initial setup, set `enable_temporary_public_access = true` and provide a restricted `operator_ssh_cidr`; Terraform then attaches a separate EIP and opens only TCP `22` from that CIDR. After setup and verification, set the flag back to false. The follow-up plan must remove only the EIP association, EIP, and temporary SSH rule—not the EC2 instance.

The setup script installs native `postgresql16` and `postgresql16-server`, keeps data at `/var/lib/pgsql/data`, configures SCRAM access for the selected VPC CIDR, and uses the built-in `postgres` superuser. The authoritative password file and SSH private key stay outside Terraform and the repository.

This MVP deliberately has no separate data disk or backup. Terminating or replacing the EC2 instance deletes the root volume and PostgreSQL data. Handle any required data export manually before destructive Terraform changes.
