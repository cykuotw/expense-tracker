data "aws_instance" "postgres" {
  filter {
    name   = "tag:Name"
    values = [var.postgres_name]
  }

  filter {
    name   = "instance-state-name"
    values = ["running"]
  }
}

data "aws_subnet" "postgres" {
  id = data.aws_instance.postgres.subnet_id
}

data "aws_security_group" "bootstrap_client" {
  filter {
    name   = "tag:Name"
    values = ["${var.postgres_name}-bootstrap"]
  }

  vpc_id = data.aws_subnet.postgres.vpc_id
}

locals {
  tags = merge(var.tags, {
    Project   = "expense-tracker"
    Component = "serverless-bootstrap"
    ManagedBy = "terraform"
  })
}

resource "aws_iam_role" "bootstrap" {
  name = "${var.function_name}-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })

  tags = local.tags
}

resource "aws_cloudwatch_log_group" "bootstrap" {
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = 7
  tags              = local.tags
}

resource "aws_iam_role_policy" "bootstrap" {
  name = "${var.function_name}-runtime"
  role = aws_iam_role.bootstrap.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "WriteBootstrapLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = "${aws_cloudwatch_log_group.bootstrap.arn}:*"
      },
      {
        Sid    = "ManageLambdaNetworkInterfaces"
        Effect = "Allow"
        Action = [
          "ec2:AssignPrivateIpAddresses",
          "ec2:CreateNetworkInterface",
          "ec2:DeleteNetworkInterface",
          "ec2:DescribeNetworkInterfaces",
          "ec2:DescribeSubnets",
          "ec2:UnassignPrivateIpAddresses",
        ]
        Resource = "*"
      },
    ]
  })
}

resource "aws_lambda_function" "bootstrap" {
  function_name = var.function_name
  role          = aws_iam_role.bootstrap.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = ["arm64"]

  filename         = var.artifact_path
  source_code_hash = filebase64sha256(var.artifact_path)

  memory_size                    = 128
  timeout                        = 300
  reserved_concurrent_executions = 1

  vpc_config {
    subnet_ids         = [data.aws_instance.postgres.subnet_id]
    security_group_ids = [data.aws_security_group.bootstrap_client.id]
  }

  environment {
    variables = {
      DB_PUBLIC_HOST      = data.aws_instance.postgres.private_ip
      DB_PORT             = "5432"
      DB_SSLMODE          = "disable"
      DB_MAINTENANCE_NAME = "postgres"
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.bootstrap,
    aws_iam_role_policy.bootstrap,
  ]

  lifecycle {
    ignore_changes = [environment]
  }

  tags = local.tags
}
