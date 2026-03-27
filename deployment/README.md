# Deployment

This directory is the active deployment layout.

## Design

This version uses:
- Terraform under `deployment/tf` for AWS infrastructure
- a thin `deployment/scripts/deploy.sh` wrapper that orchestrates focused infra, backend, frontend, and edge deploy entrypoints
- versioned remote backend and edge release scripts bundled inside their respective deployment artifacts
- Amazon RDS PostgreSQL managed by Terraform instead of same-host Docker PostgreSQL
- three database identities during deploy: an RDS admin login, a migration user, and a runtime app user

The intended flow is:
1. optionally run `scripts/apply-infra.sh` to provision or update AWS infrastructure and publish the database credential parameter names
2. run `scripts/deploy-backend.sh` to read Terraform outputs, stage the backend release into backend `runtime/` and `deploy/` payloads, upload the backend artifact, and trigger the remote backend release through SSM
3. on the backend host, read the packaged backend manifest and runtime env parameter manifest from the temp `deploy/` payload, render a complete runtime env from SSM, fetch the deploy-time database credentials from SSM, install the backend runtime contents, bootstrap database roles and grants, run migrations, optionally create the first admin user from deploy-time SSM parameters, and restart the Go service
4. run `scripts/deploy-frontend.sh` to build the frontend with the live API origin from Terraform outputs, upload frontend assets, and invalidate CloudFront
5. run `scripts/deploy-edge.sh` to stage the nginx/TLS payload, upload the edge artifact, and trigger the remote edge release through SSM
6. on the backend host, read the packaged edge manifest from the temp `deploy/` payload, install nginx config, reconcile certbot when enabled, and restart or reload nginx
7. optionally use `scripts/deploy.sh` for app-only deploys, or `scripts/deploy.sh all` for the full infra + app + edge flow

## Files

- `scripts/deploy.sh`: thin wrapper with `all`, `app`, `infra`, `frontend`, `backend`, `edge`, and `help`; defaults to `app`
- `scripts/apply-infra.sh`: Terraform apply entrypoint
- `scripts/deploy-backend.sh`: backend packaging, artifact upload, and backend-only SSM release
- `scripts/deploy-frontend.sh`: frontend build, S3 publish, and CloudFront invalidation
- `scripts/deploy-edge.sh`: edge packaging, artifact upload, SSM release, and external edge health checks
- `scripts/destroy.sh`: teardown entrypoint that empties versioned S3 buckets before Terraform destroy
- `remote/deploy-backend-release.sh`: host-side backend release routine packaged inside the backend artifact and executed through SSM
- `remote/deploy-edge-release.sh`: host-side edge release routine packaged inside the edge artifact and executed through SSM
- `remote/lib/common.sh`: shared helper functions sourced by the remote backend and edge release scripts
- `deploy/release-manifest.env`: generated at deploy time inside the release bundle with non-secret host-side release context
- `deploy/runtime-env-ssm.env`: generated manifest of runtime env keys to SSM parameter names used to render `backend.env` on the host
- `runtime/`: staged runtime payload copied into `APP_DIR` for backend deploys and into `/etc/nginx` for edge deploys
- `systemd/expense-tracker.service`: systemd unit template rendered at package time with the manifest-aligned app and env paths
- `backend.env.example`: backend runtime env template using the RDS endpoint
- `tf/`: Terraform infrastructure for the backend host, RDS, buckets, IAM, Elastic IP, DNS, and database credential locations

## Notes

- This layout intentionally does not manage application releases with Terraform.
- Terraform outputs are used by the deploy and destroy scripts so EC2 instance IDs, bucket names, database usernames, and SSM parameter names do not need to be duplicated in shell config.
- `scripts/apply-infra.sh` owns Terraform init/apply, `scripts/deploy-backend.sh` owns backend release packaging and remote app orchestration, `scripts/deploy-frontend.sh` owns frontend publication, and `scripts/deploy-edge.sh` owns nginx/certbot edge orchestration.
- `scripts/deploy.sh` preserves a single entrypoint while defaulting to app-only deploys and still letting CI/CD target infra, app, frontend, backend, or edge independently.
- `scripts/deploy-backend.sh` owns the runtime env contract and host-side backend env finalization; `scripts/deploy-edge.sh` owns nginx and certbot reconciliation; `scripts/deploy-frontend.sh` does not mutate backend runtime configuration.
- To seed the first admin during a backend deploy, define `first_admin_email`, `first_admin_password`, `first_admin_firstname`, and `first_admin_lastname` in the selected Terraform tfvars file before running `make deploy` or `make deploy backend`; `first_admin_nickname` is optional.
- Those first-admin inputs are written only to temporary SSM parameters for the lifetime of the deploy and are deleted after the remote bootstrap step, so they do not land in the packaged artifact or long-lived runtime env file.
- `APP_DIR`, `BACKEND_ENV_PATH`, and `SYSTEMD_SERVICE_NAME` are part of the backend deploy contract. The local packaging step renders the staged systemd unit from those values, and the remote backend release script installs that rendered unit using the configured service name.
- The local deploy and destroy scripts keep shared defaults in `deployment/scripts/lib/config.sh`. They no longer require a per-machine `config.local.sh`; set `TF_VARS_FILE` explicitly only when you are not using the default `deployment/tf/terraform.tfvars`, and `AWS_REGION` will be inferred from the selected tfvars file unless you override it in the shell.
- nginx runs on the backend host as a port 80 reverse proxy to the Go service on `localhost:8080`. If `CERTBOT_ENABLED=true`, edge deploy runs `certbot --nginx` for the API domain, installs a renewal hook that reloads nginx, enables the certbot timer, and then checks both HTTP and HTTPS health endpoints.
- The backend artifact includes a non-secret `deploy/` payload and a `runtime/` payload for app runtime files. The separate edge artifact includes the nginx payload and edge release metadata. The host renders the full long-lived backend env file from SSM during backend deploy instead of merging a local overlay with host-managed leftovers.
- Terraform now defines stable SSM parameter names for `JWT_SECRET`, `REFRESH_JWT_SECRET`, and `THIRD_PARTY_SESSION_SECRET`, and the remote backend deploy script creates those parameters on first deploy if they do not exist yet.
- Terraform also publishes the deploy-managed non-secret runtime values such as frontend origin, CORS credential policy, DB connection coordinates, cookie domain, and Google callback URL into SSM so deploy can read one runtime config source of truth.
- Optional Google OAuth credentials can also be stored in SSM via Terraform when `google_client_id` and `google_client_secret` are set; if they are left empty, deploy omits those keys from the rendered runtime env.
