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
- CloudFront distribution for the configured frontend hostname
- Route 53 alias record for the frontend hostname
- Route 53 A record for the API hostname

## Certificate Requirement

Set `frontend_certificate_arn` to an ACM certificate ARN in `us-east-1`.
CloudFront can only use ACM certificates from `us-east-1`, even if the rest of the infrastructure lives in another AWS region.

## CloudFront Plan Note

This Terraform layout does not set `price_class` on the frontend distribution.
That keeps the distribution aligned with the CloudFront flat-rate plan flow you are using in the console.

## Database Credentials

Terraform provisions the RDS master login and stores its password in SSM for administrative bootstrap use.
Terraform also stores separate migration-user and app-user passwords in SSM.
The backend EC2 role is allowed to read those specific parameters so deploy can bootstrap roles and run migrations on-host without writing the admin credential into the runtime env file.

## Deployment Split

- Terraform manages infrastructure, DNS, and database credential locations.
- `deployment/scripts/deploy.sh` builds and deploys application releases.
