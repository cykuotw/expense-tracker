# Deployment

This directory is the active deployment layout.

## Design

This version uses:
- Terraform under `deployment/tf` for AWS infrastructure
- a single `deployment/scripts/deploy.sh` entrypoint for application releases
- a versioned remote release script bundled inside the backend deployment artifact
- Amazon RDS PostgreSQL managed by Terraform instead of same-host Docker PostgreSQL
- three database identities during deploy: an RDS admin login, a migration user, and a runtime app user

The intended flow is:
1. build frontend and backend locally
2. run `terraform apply` to provision or update AWS infrastructure and publish the database credential parameter names
3. read Terraform outputs for the EC2 instance ID, bucket names, database endpoints, usernames, and SSM parameter names
4. stage the backend release into two parts: a `runtime/` payload with binaries, migrations, nginx config, and a rendered systemd unit; and a `deploy/` payload with the packaged remote release script, release manifest, and an SSM runtime env parameter manifest
5. upload frontend assets and the backend release artifact
6. trigger a minimal SSM bootstrap that downloads the backend artifact from S3 into a temp location, unpacks it, and runs the packaged remote release script against that temp release root
7. on the backend host, read the packaged manifest and runtime env parameter manifest from the temp `deploy/` payload, render a complete runtime env from SSM, fetch the deploy-time database credentials from SSM, install the `runtime/` contents, bootstrap database roles and grants, run migrations, and restart services
8. install or update nginx on the backend host to proxy `:80` to the Go API on `localhost:8080`
9. optionally run certbot on the backend host after nginx is live to provision HTTPS for the API domain

## Files

- `scripts/deploy.sh`: single deploy entrypoint
- `scripts/destroy.sh`: teardown entrypoint that empties versioned S3 buckets before Terraform destroy
- `remote/deploy-release.sh`: host-side release routine packaged inside the backend artifact and executed through SSM
- `deploy/release-manifest.env`: generated at deploy time inside the release bundle with non-secret host-side release context
- `deploy/runtime-env-ssm.env`: generated manifest of runtime env keys to SSM parameter names used to render `backend.env` on the host
- `runtime/`: staged runtime payload copied into `APP_DIR` during deploy, with service and nginx files installed into `/etc`
- `systemd/expense-tracker.service`: systemd unit template rendered at package time with the manifest-aligned app and env paths
- `backend.env.example`: backend runtime env template using the RDS endpoint
- `tf/`: Terraform infrastructure for the backend host, RDS, buckets, IAM, Elastic IP, DNS, and database credential locations

## Notes

- This layout intentionally does not manage application releases with Terraform.
- Terraform outputs are used by the deploy and destroy scripts so EC2 instance IDs, bucket names, database usernames, and SSM parameter names do not need to be duplicated in shell config.
- The local deploy script owns build, artifact packaging, uploads, and minimal SSM orchestration. The packaged remote release script owns host-side install of `runtime/` into `APP_DIR`, manifest loading, env finalization, database bootstrap, migrations, and service restarts.
- `APP_DIR`, `BACKEND_ENV_PATH`, and `SYSTEMD_SERVICE_NAME` are part of the deploy contract. The local packaging step renders the staged systemd unit from those values, and the remote release script installs that rendered unit using the configured service name.
- The local deploy and destroy scripts keep shared defaults in `deployment/scripts/lib/config.sh`. They no longer require a per-machine `config.local.sh`; set `TF_VARS_FILE` explicitly only when you are not using the default `deployment/tf/terraform.tfvars`, and `AWS_REGION` will be inferred from the selected tfvars file unless you override it in the shell.
- nginx runs on the backend host as a port 80 reverse proxy to the Go service on `localhost:8080`. If `CERTBOT_ENABLED=true`, deploy also runs `certbot --nginx` for the API domain, installs a renewal hook that reloads nginx, enables the certbot timer, and then checks both HTTP and HTTPS health endpoints.
- The backend artifact includes a non-secret `deploy/` payload and a `runtime/` payload. The host now renders the full long-lived service env file from SSM during deploy instead of merging a local overlay with host-managed leftovers, and deploy-only files stay in the temp release directory instead of `APP_DIR`.
- Terraform now defines stable SSM parameter names for `JWT_SECRET`, `REFRESH_JWT_SECRET`, and `THIRD_PARTY_SESSION_SECRET`, and the remote deploy script creates those parameters on first deploy if they do not exist yet.
- Terraform also publishes the deploy-managed non-secret runtime values such as frontend origin, DB connection coordinates, cookie domain, and Google callback URL into SSM so deploy can read one runtime config source of truth.
- Optional Google OAuth credentials can also be stored in SSM via Terraform when `google_client_id` and `google_client_secret` are set; if they are left empty, deploy omits those keys from the rendered runtime env.
