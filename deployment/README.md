# Deployment

This directory is the active deployment layout.

## Design

This version uses:
- Terraform under `deployment/tf` for AWS infrastructure
- a single `deployment/scripts/deploy.sh` entrypoint for application releases
- Amazon RDS PostgreSQL managed by Terraform instead of same-host Docker PostgreSQL
- three database identities during deploy: an RDS admin login, a migration user, and a runtime app user

The intended flow is:
1. build frontend and backend locally
2. run `terraform apply` to provision or update AWS infrastructure and publish the database credential parameter names
3. read Terraform outputs for the EC2 instance ID, bucket names, database endpoints, usernames, and SSM parameter names
4. upload frontend assets and backend release artifacts
5. write the backend runtime env with the app-user credential only
6. run the backend update path through SSM
7. on the backend host, fetch the admin and migration passwords from SSM, bootstrap database roles and grants, and run migrations with the migration user
8. install or update nginx on the backend host to proxy `:80` to the Go API on `localhost:8080`
9. optionally run certbot on the backend host after nginx is live to provision HTTPS for the API domain

## Files

- `scripts/deploy.sh`: single deploy entrypoint
- `scripts/destroy.sh`: teardown entrypoint that empties versioned S3 buckets before Terraform destroy
- `systemd/expense-tracker.service`: backend service definition
- `backend.env.example`: backend runtime env template using the RDS endpoint
- `tf/`: Terraform infrastructure for the backend host, RDS, buckets, IAM, Elastic IP, DNS, and database credential locations

## Notes

- This layout intentionally does not manage application releases with Terraform.
- Terraform outputs are used by the deploy and destroy scripts so EC2 instance IDs, bucket names, database usernames, and SSM parameter names do not need to be duplicated in shell config.
- nginx runs on the backend host as a port 80 reverse proxy to the Go service on `localhost:8080`. If `CERTBOT_ENABLED=true`, deploy also runs `certbot --nginx` for the API domain, installs a renewal hook that reloads nginx, enables the certbot timer, and then checks both HTTP and HTTPS health endpoints.
- The backend env file on EC2 should contain only the runtime app credential. The admin and migration credentials should be fetched on demand from SSM during deploy and should not remain in the long-lived service env file.
