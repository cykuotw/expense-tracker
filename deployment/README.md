# Deployment

This directory is the active deployment layout.

## Design

This version uses:
- Terraform under `deployment/tf` for AWS infrastructure
- a single `deployment/scripts/deploy.sh` entrypoint for application releases
- Amazon RDS PostgreSQL managed by Terraform instead of same-host Docker PostgreSQL

The intended flow is:
1. build frontend and backend locally
2. run `terraform apply` to provision or update AWS infrastructure
3. read Terraform outputs for the EC2 instance ID and bucket names
4. upload frontend assets and backend release artifacts
5. run the backend update path through SSM
6. install or update nginx on the backend host to proxy `:80` to the Go API on `localhost:8080`
7. optionally run certbot on the backend host after nginx is live to provision HTTPS for the API domain
8. run migrations against the Terraform-managed RDS instance during deploy

## Files

- `scripts/deploy.sh`: single deploy entrypoint
- `scripts/destroy.sh`: teardown entrypoint that empties versioned S3 buckets before Terraform destroy
- `systemd/expense-tracker.service`: backend service definition
- `backend.env.example`: backend runtime env template using the RDS endpoint
- `tf/`: Terraform infrastructure for the backend host, RDS, buckets, IAM, Elastic IP, and optional DNS

## Notes

- This layout intentionally does not manage application releases with Terraform.
- Terraform outputs are used by the deploy and destroy scripts so EC2 instance IDs and bucket names do not need to be duplicated in shell config.
- nginx runs on the backend host as a port 80 reverse proxy to the Go service on `localhost:8080`. If `CERTBOT_ENABLED=true`, deploy also runs `certbot --nginx` for the API domain, installs a renewal hook that reloads nginx, enables the certbot timer, and then checks both HTTP and HTTPS health endpoints.
- The backend env file on EC2 should be populated with the RDS connection values from Terraform outputs.
