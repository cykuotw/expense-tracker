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

output "db_username" {
  value       = aws_db_instance.db.username
  description = "RDS PostgreSQL master username"
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

output "db_password_ssm_parameter_name" {
  value       = aws_ssm_parameter.db_password.name
  description = "SSM parameter name containing the backend database password"
}
