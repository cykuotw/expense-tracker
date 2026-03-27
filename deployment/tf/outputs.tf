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

output "frontend_origin_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["frontend_origin"].name
  description = "SSM parameter name containing the backend runtime FRONTEND_ORIGIN value"
}

output "cors_allowed_origins_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["cors_allowed_origins"].name
  description = "SSM parameter name containing the backend runtime CORS_ALLOWED_ORIGINS value"
}

output "auth_cookie_domain_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["auth_cookie_domain"].name
  description = "SSM parameter name containing the backend runtime AUTH_COOKIE_DOMAIN value"
}

output "db_public_host_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["db_public_host"].name
  description = "SSM parameter name containing the backend runtime DB_PUBLIC_HOST value"
}

output "db_port_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["db_port"].name
  description = "SSM parameter name containing the backend runtime DB_PORT value"
}

output "db_user_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["db_user"].name
  description = "SSM parameter name containing the backend runtime DB_USER value"
}

output "db_name_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["db_name"].name
  description = "SSM parameter name containing the backend runtime DB_NAME value"
}

output "db_sslmode_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["db_sslmode"].name
  description = "SSM parameter name containing the backend runtime DB_SSLMODE value"
}

output "google_callback_url_ssm_parameter_name" {
  value       = aws_ssm_parameter.runtime_config["google_callback_url"].name
  description = "SSM parameter name containing the backend runtime GOOGLE_CALLBACK_URL value"
}

output "jwt_secret_ssm_parameter_name" {
  value       = local.jwt_secret_ssm_parameter_name
  description = "SSM parameter name containing the backend JWT signing secret"
}

output "refresh_jwt_secret_ssm_parameter_name" {
  value       = local.refresh_jwt_secret_ssm_parameter_name
  description = "SSM parameter name containing the backend refresh JWT signing secret"
}

output "third_party_session_secret_ssm_parameter_name" {
  value       = local.third_party_session_secret_ssm_parameter_name
  description = "SSM parameter name containing the backend third-party session secret"
}

output "google_client_id_ssm_parameter_name" {
  value       = try(aws_ssm_parameter.google_client_id[0].name, "")
  description = "Optional SSM parameter name containing the backend Google OAuth client ID"
}

output "google_client_secret_ssm_parameter_name" {
  value       = try(aws_ssm_parameter.google_client_secret[0].name, "")
  description = "Optional SSM parameter name containing the backend Google OAuth client secret"
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
