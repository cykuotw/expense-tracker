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

data "aws_security_group" "worker_client" {
  filter {
    name   = "tag:Name"
    values = ["${var.postgres_name}-worker"]
  }

  vpc_id = data.aws_subnet.postgres.vpc_id
}

locals {
  api_path      = "/api/v0"
  google_issuer = "https://accounts.google.com"
  tags = merge(var.tags, {
    Project   = "expense-tracker"
    Component = "serverless-worker"
    ManagedBy = "terraform"
  })
}

resource "aws_iam_role" "worker" {
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

resource "aws_cloudwatch_log_group" "worker" {
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = 7
  tags              = local.tags
}

resource "aws_iam_role_policy" "worker" {
  name = "${var.function_name}-runtime"
  role = aws_iam_role.worker.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "WriteWorkerLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = "${aws_cloudwatch_log_group.worker.arn}:*"
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

resource "aws_lambda_function" "worker" {
  function_name = var.function_name
  role          = aws_iam_role.worker.arn
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  architectures = ["arm64"]

  filename         = var.artifact_path
  source_code_hash = filebase64sha256(var.artifact_path)

  memory_size                    = 128
  timeout                        = 15
  reserved_concurrent_executions = var.reserved_concurrency

  vpc_config {
    subnet_ids         = [data.aws_instance.postgres.subnet_id]
    security_group_ids = [data.aws_security_group.worker_client.id]
  }

  environment {
    variables = {
      MODE                   = "release"
      API_URL                = local.api_path
      FRONTEND_ORIGIN        = var.frontend_origin
      CORS_ALLOWED_ORIGINS   = var.frontend_origin
      CORS_ALLOW_CREDENTIALS = "true"
      AUTH_COOKIE_DOMAIN     = ""
      AUTH_COOKIE_SECURE     = "true"
      AUTH_COOKIE_SAME_SITE  = "none"
      DB_PUBLIC_HOST         = data.aws_instance.postgres.private_ip
      DB_PORT                = "5432"
      DB_SSLMODE             = "disable"
      DB_MAX_OPEN_CONNS      = "2"
      DB_MAX_IDLE_CONNS      = "1"
      GOOGLE_OAUTH_ENABLED   = "true"
      GOOGLE_CLIENT_ID       = var.google_client_id
      GOOGLE_EXCHANGE_MODE   = "upstream_verified"
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.worker,
    aws_iam_role_policy.worker,
  ]

  lifecycle {
    ignore_changes = [environment]
  }

  tags = local.tags
}

resource "aws_apigatewayv2_api" "worker" {
  name                         = "${var.function_name}-http-api"
  protocol_type                = "HTTP"
  disable_execute_api_endpoint = false
  tags                         = local.tags
}

resource "aws_apigatewayv2_integration" "worker" {
  api_id                 = aws_apigatewayv2_api.worker.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.worker.invoke_arn
  integration_method     = "POST"
  payload_format_version = "2.0"
  timeout_milliseconds   = 15000
}

resource "aws_apigatewayv2_authorizer" "google" {
  api_id           = aws_apigatewayv2_api.worker.id
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  name             = "google-id-token"

  jwt_configuration {
    audience = [var.google_client_id]
    issuer   = local.google_issuer
  }
}

resource "aws_apigatewayv2_route" "google_exchange" {
  api_id             = aws_apigatewayv2_api.worker.id
  route_key          = "POST ${local.api_path}/auth/google/exchange"
  target             = "integrations/${aws_apigatewayv2_integration.worker.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.google.id
}

resource "aws_apigatewayv2_route" "default" {
  api_id             = aws_apigatewayv2_api.worker.id
  route_key          = "$default"
  target             = "integrations/${aws_apigatewayv2_integration.worker.id}"
  authorization_type = "NONE"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.worker.id
  name        = "$default"
  auto_deploy = true
  tags        = local.tags
}

resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowHTTPAPIInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.worker.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.worker.execution_arn}/*/*"
}
