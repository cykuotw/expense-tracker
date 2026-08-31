output "database_instance_id" {
  value = aws_instance.postgres.id
}
output "database_volume_id" {
  value = aws_instance.postgres.root_block_device[0].volume_id
}
output "database_launch_template_id" {
  value = aws_launch_template.postgres.id
}
output "database_host" {
  value = aws_instance.postgres.private_ip
}
output "database_network_interface_id" {
  value = aws_instance.postgres.primary_network_interface_id
}
output "database_temporary_public_ipv4" {
  value = try(aws_eip.temporary_postgres[0].public_ip, "")
}
output "postgres_backup_bucket_name" {
  value = aws_s3_bucket.postgres_backup.bucket
}
output "restore_verification_temporary_public_ipv4" {
  value = try(aws_eip.restore_verification[0].public_ip, "")
}
output "vpc_ipv4_cidr" {
  value = data.aws_vpc.selected.cidr_block
}
output "database_security_group_id" {
  value = aws_security_group.postgres.id
}
output "worker_security_group_id" {
  value = aws_security_group.worker.id
}
output "bootstrap_security_group_id" {
  value = aws_security_group.bootstrap.id
}
output "worker_function_name" {
  value = aws_lambda_function.worker.function_name
}
output "bootstrap_function_name" {
  value = aws_lambda_function.bootstrap.function_name
}
output "worker_role_name" {
  value = aws_iam_role.worker.name
}
output "bootstrap_role_name" {
  value = aws_iam_role.bootstrap.name
}
output "worker_log_group_name" {
  value = aws_cloudwatch_log_group.worker.name
}
output "bootstrap_log_group_name" {
  value = aws_cloudwatch_log_group.bootstrap.name
}
output "api_id" {
  value = aws_apigatewayv2_api.worker.id
}
output "raw_api_endpoint" {
  value = aws_apigatewayv2_api.worker.api_endpoint
}
output "api_origin" {
  value = "https://${var.api_hostname}"
}
output "api_hostname" {
  value = var.api_hostname
}
output "api_certificate_arn" {
  value = aws_acm_certificate.api.arn
}
output "frontend_bucket_name" {
  value = aws_s3_bucket.frontend.bucket
}
output "cloudfront_distribution_id" {
  value = aws_cloudfront_distribution.frontend.id
}
output "cloudfront_domain_name" {
  value = aws_cloudfront_distribution.frontend.domain_name
}
output "frontend_origin" {
  value = "https://${var.frontend_hostname}"
}
output "frontend_hostname" {
  value = var.frontend_hostname
}
output "frontend_certificate_arn" {
  value = aws_acm_certificate.frontend.arn
}
output "hosted_zone_id" {
  value = data.aws_route53_zone.selected.zone_id
}
