# Terraform Infrastructure

This Terraform layout provisions the active AWS infrastructure for the project.

## Managed Resources

- EC2 backend host in the configured AWS region
- Elastic IP for the backend host
- security groups for backend and RDS connectivity
- IAM role and instance profile for SSM, database credential reads, and artifact bucket access
- frontend S3 bucket in the configured AWS region
- artifact S3 bucket in the configured AWS region
- RDS PostgreSQL instance in the configured AWS region
- SSM SecureString parameters for the database admin, migration, and app passwords
- SSM String parameters for deploy-managed runtime config such as frontend origin, CORS credential policy, cookie domain, DB coordinates, and Google callback URL
- SSM SecureString parameters for optional Google OAuth credentials
- CloudFront distribution for the configured frontend hostname
- Route 53 alias record for the frontend hostname
- Route 53 A record for the API hostname

## Certificate Requirement

Set `frontend_certificate_arn` to an ACM certificate ARN in `us-east-1`.
CloudFront can only use ACM certificates from `us-east-1`, even if the rest of the infrastructure lives in another AWS region.

## CloudFront Plan Note

This Terraform layout does not set `price_class` on the frontend distribution.
That keeps the distribution aligned with the CloudFront flat-rate plan flow you are using in the console.

## Runtime Secrets

Terraform provisions the RDS master login and stores its password in SSM for administrative bootstrap use.
Terraform also stores separate migration-user and app-user passwords in SSM.
Terraform publishes the deploy-managed non-secret runtime config in SSM as well so the host can render a complete backend env from one parameter source.
Terraform also defines stable SSM parameter names for `JWT_SECRET`, `REFRESH_JWT_SECRET`, and `THIRD_PARTY_SESSION_SECRET`.
The remote deploy script creates those SecureString parameters on first deploy if they do not already exist.
If Google OAuth credentials are provided in `terraform.tfvars`, Terraform stores those in SSM as well.
The backend EC2 role is allowed to read those specific parameters so deploy can finalize runtime configuration on-host without copying a local secret-bearing env file into the release artifact or depending on stale host-side leftovers.
The backend EC2 role is also allowed to read and delete temporary deploy-time first-admin bootstrap parameters under the scoped `/project/environment/deploy/first_admin/*` prefix.
The selected tfvars file can also carry the first-admin bootstrap inputs; the local deploy script reads those values directly from tfvars, then hands them to the remote host through temporary SSM parameters instead of persisting them in Terraform-managed runtime config.

## Deployment Split

- Terraform manages infrastructure, DNS, and database credential locations.
- `deployment/scripts/deploy.sh` builds and deploys application releases.
