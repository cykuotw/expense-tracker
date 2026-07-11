output "serverless_postgres_instance_id" {
  value = aws_instance.postgres.id
}

output "serverless_postgres_host" {
  value = aws_instance.postgres.private_ip
}

output "serverless_postgres_temporary_public_ipv4" {
  value = try(aws_eip.temporary_postgres[0].public_ip, "")
}

output "serverless_postgres_port" {
  value = 5432
}

output "serverless_postgres_sslmode" {
  value = "disable"
}

output "serverless_postgres_vpc_id" {
  value = var.vpc_id
}

output "serverless_postgres_subnet_id" {
  value = var.subnet_id
}

output "serverless_postgres_vpc_ipv4_cidr" {
  value = data.aws_vpc.selected.cidr_block
}

output "serverless_postgres_security_group_id" {
  value = aws_security_group.postgres.id
}

output "serverless_postgres_worker_client_security_group_id" {
  value = aws_security_group.worker_client.id
}

output "serverless_postgres_bootstrap_client_security_group_id" {
  value = aws_security_group.bootstrap_client.id
}
