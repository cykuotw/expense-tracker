output "worker_function_name" {
  value = aws_lambda_function.worker.function_name
}

output "worker_api_endpoint" {
  value = aws_apigatewayv2_api.worker.api_endpoint
}

output "worker_google_authorizer_id" {
  value = aws_apigatewayv2_authorizer.google.id
}

output "worker_google_audience" {
  value = var.google_client_id
}

output "worker_frontend_origin" {
  value = var.frontend_origin
}

output "worker_db_host" {
  value = data.aws_instance.postgres.private_ip
}

output "worker_security_group_id" {
  value = data.aws_security_group.worker_client.id
}

output "worker_reserved_concurrency" {
  value = var.reserved_concurrency
}
