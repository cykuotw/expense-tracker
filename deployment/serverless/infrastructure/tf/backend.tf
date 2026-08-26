resource "aws_iam_role" "worker" {
  name = "${local.resource_prefix}-worker-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = {
        Service = "lambda.amazonaws.com"
      }, Action = "sts:AssumeRole"
    }]
  })
  tags = merge(local.common_tags, {
    Component = "backend"
  })
}
resource "aws_iam_role" "bootstrap" {
  name = "${local.resource_prefix}-bootstrap-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = {
        Service = "lambda.amazonaws.com"
      }, Action = "sts:AssumeRole"
    }]
  })
  tags = merge(local.common_tags, {
    Component = "database"
  })
}
resource "aws_cloudwatch_log_group" "worker" {
  name              = "/aws/lambda/${local.resource_prefix}-worker"
  retention_in_days = 7
  tags = merge(local.common_tags, {
    Component = "backend"
  })
}
resource "aws_cloudwatch_log_group" "bootstrap" {
  name              = "/aws/lambda/${local.resource_prefix}-bootstrap"
  retention_in_days = 7
  tags = merge(local.common_tags, {
    Component = "database"
  })
}
locals {
  lambda_network_actions = ["ec2:AssignPrivateIpAddresses", "ec2:CreateNetworkInterface", "ec2:DeleteNetworkInterface", "ec2:DescribeNetworkInterfaces", "ec2:DescribeSubnets", "ec2:UnassignPrivateIpAddresses"]
}
resource "aws_iam_role_policy" "worker" {
  name = "${local.resource_prefix}-worker-runtime"
  role = aws_iam_role.worker.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [
      {
        Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "${aws_cloudwatch_log_group.worker.arn}:*"
      },
      {
        Effect = "Allow", Action = local.lambda_network_actions, Resource = "*"
      }
    ]
  })
}
resource "aws_iam_role_policy" "bootstrap" {
  name = "${local.resource_prefix}-bootstrap-runtime"
  role = aws_iam_role.bootstrap.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [
      {
        Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "${aws_cloudwatch_log_group.bootstrap.arn}:*"
      },
      {
        Effect = "Allow", Action = local.lambda_network_actions, Resource = "*"
      }
    ]
  })
}
resource "aws_lambda_function" "worker" {
  function_name                  = "${local.resource_prefix}-worker"
  role                           = aws_iam_role.worker.arn
  runtime                        = "provided.al2023"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  filename                       = var.worker_artifact_path
  source_code_hash               = filebase64sha256(var.worker_artifact_path)
  memory_size                    = 128
  timeout                        = 15
  reserved_concurrent_executions = 0
  vpc_config {
    subnet_ids         = [var.subnet_id]
    security_group_ids = [aws_security_group.worker.id]
  }
  depends_on = [aws_cloudwatch_log_group.worker, aws_iam_role_policy.worker]
  lifecycle {
    ignore_changes = [environment, reserved_concurrent_executions]
  }
  tags = merge(local.common_tags, {
    Component = "backend"
  })
}
resource "aws_lambda_function" "bootstrap" {
  function_name                  = "${local.resource_prefix}-bootstrap"
  role                           = aws_iam_role.bootstrap.arn
  runtime                        = "provided.al2023"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  filename                       = var.bootstrap_artifact_path
  source_code_hash               = filebase64sha256(var.bootstrap_artifact_path)
  memory_size                    = 128
  timeout                        = 300
  reserved_concurrent_executions = 1
  vpc_config {
    subnet_ids         = [var.subnet_id]
    security_group_ids = [aws_security_group.bootstrap.id]
  }
  depends_on = [aws_cloudwatch_log_group.bootstrap, aws_iam_role_policy.bootstrap]
  lifecycle {
    ignore_changes = [environment]
  }
  tags = merge(local.common_tags, {
    Component = "database"
  })
}
