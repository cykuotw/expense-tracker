output "bootstrap_function_name" {
  value = aws_lambda_function.bootstrap.function_name
}

output "bootstrap_db_host" {
  value = data.aws_instance.postgres.private_ip
}

output "bootstrap_security_group_id" {
  value = data.aws_security_group.bootstrap_client.id
}

output "bootstrap_reserved_concurrency" {
  value = aws_lambda_function.bootstrap.reserved_concurrent_executions
}
