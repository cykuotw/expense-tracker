output "backend_instance_id" {
  value       = aws_instance.backend.id
  description = "EC2 instance ID for the backend host"
}

output "backend_public_ip" {
  value       = aws_eip.backend.public_ip
  description = "Elastic IP attached to the backend host"
}

output "frontend_bucket_name" {
  value       = aws_s3_bucket.frontend.bucket
  description = "S3 bucket for frontend assets"
}

output "artifact_bucket_name" {
  value       = aws_s3_bucket.artifacts.bucket
  description = "S3 bucket for backend release artifacts"
}

output "db_host" {
  value       = aws_db_instance.db.address
  description = "RDS PostgreSQL hostname for the backend configuration"
}

output "db_port" {
  value       = aws_db_instance.db.port
  description = "RDS PostgreSQL port for the backend configuration"
}

output "db_name" {
  value       = aws_db_instance.db.db_name
  description = "RDS PostgreSQL database name"
}

output "db_admin_username" {
  value       = aws_db_instance.db.username
  description = "RDS PostgreSQL master username reserved for administration and bootstrap"
}

output "db_admin_password_ssm_parameter_name" {
  value       = aws_ssm_parameter.db_admin_password.name
  description = "SSM parameter name containing the RDS PostgreSQL master password"
}

output "db_migration_username" {
  value       = var.db_migration_username
  description = "PostgreSQL username used for migrations"
}

output "db_migration_password_ssm_parameter_name" {
  value       = aws_ssm_parameter.db_migration_password.name
  description = "SSM parameter name containing the migration database password"
}

output "db_app_username" {
  value       = var.db_app_username
  description = "PostgreSQL username used by the running backend service"
}

output "db_app_password_ssm_parameter_name" {
  value       = aws_ssm_parameter.db_app_password.name
  description = "SSM parameter name containing the backend runtime database password"
}

output "cloudfront_distribution_id" {
  value       = aws_cloudfront_distribution.frontend.id
  description = "CloudFront distribution ID for the frontend"
}

output "cloudfront_domain_name" {
  value       = aws_cloudfront_distribution.frontend.domain_name
  description = "CloudFront domain name for the frontend"
}

output "frontend_fqdn" {
  value       = local.frontend_fqdn
  description = "Frontend DNS name"
}

output "api_fqdn" {
  value       = var.create_api_dns_record ? aws_route53_record.api[0].fqdn : local.api_fqdn
  description = "API DNS name"
}
