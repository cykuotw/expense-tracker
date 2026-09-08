# Sender has public Internet access; Delivery alone connects to PostgreSQL.
# Existing sender security-group resource names are retained for state compatibility.

resource "aws_iam_role" "sender" {
  name = "${local.resource_prefix}-sender-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = {
        Service = "lambda.amazonaws.com"
      }, Action = "sts:AssumeRole"
    }]
  })
  tags = merge(local.common_tags, {
    Component = "notifications"
  })
}

resource "aws_cloudwatch_log_group" "sender" {
  name              = "/aws/lambda/${local.resource_prefix}-sender"
  retention_in_days = 7
  tags = merge(local.common_tags, {
    Component = "notifications"
  })
}

resource "aws_iam_role_policy" "sender" {
  name = "${local.resource_prefix}-sender-runtime"
  role = aws_iam_role.sender.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [
      {
        Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "${aws_cloudwatch_log_group.sender.arn}:*"
      },
      {
        Effect = "Allow", Action = ["lambda:InvokeFunction"], Resource = aws_lambda_function.delivery.arn
      }
    ]
  })
}

resource "aws_lambda_function" "sender" {
  function_name                  = "${local.resource_prefix}-sender"
  role                           = aws_iam_role.sender.arn
  runtime                        = "provided.al2023"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  filename                       = var.sender_artifact_path
  source_code_hash               = filebase64sha256(var.sender_artifact_path)
  memory_size                    = 128
  timeout                        = 60
  reserved_concurrent_executions = 0
  depends_on                     = [aws_cloudwatch_log_group.sender, aws_iam_role_policy.sender]
  lifecycle {
    ignore_changes = [
      environment,
      filename,
      reserved_concurrent_executions,
      source_code_hash,
    ]
  }
  tags = merge(local.common_tags, {
    Component = "notifications"
  })
}

resource "aws_cloudwatch_event_rule" "sender" {
  name                = "${local.resource_prefix}-sender-tick"
  description         = "Run the web push sender every minute"
  schedule_expression = "rate(1 minute)"
  tags = merge(local.common_tags, {
    Component = "notifications"
  })
}

resource "aws_cloudwatch_event_target" "sender" {
  rule = aws_cloudwatch_event_rule.sender.name
  arn  = aws_lambda_function.sender.arn
}

resource "aws_lambda_permission" "sender_eventbridge" {
  statement_id  = "AllowEventBridgePushSender"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.sender.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.sender.arn
}

resource "aws_security_group" "sender" {
  name_prefix = "${local.resource_prefix}-sender-"
  description = "Expense Tracker web push sender Lambda"
  vpc_id      = var.vpc_id
  tags = merge(local.common_tags, {
    Name = "${local.resource_prefix}-sender", Component = "notifications"
  })
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_ingress_rule" "postgres_from_sender" {
  security_group_id            = aws_security_group.postgres.id
  referenced_security_group_id = aws_security_group.sender.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Push sender to PostgreSQL"
}

resource "aws_vpc_security_group_egress_rule" "sender_to_postgres" {
  security_group_id            = aws_security_group.sender.id
  referenced_security_group_id = aws_security_group.postgres.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
  description                  = "Push sender to PostgreSQL"
}

resource "aws_vpc_security_group_egress_rule" "sender_to_vpc_dns_udp" {
  security_group_id = aws_security_group.sender.id
  cidr_ipv4         = data.aws_vpc.selected.cidr_block
  ip_protocol       = "udp"
  from_port         = 53
  to_port           = 53
  description       = "VPC DNS resolution"
}

resource "aws_vpc_security_group_egress_rule" "sender_to_vpc_dns_tcp" {
  security_group_id = aws_security_group.sender.id
  cidr_ipv4         = data.aws_vpc.selected.cidr_block
  ip_protocol       = "tcp"
  from_port         = 53
  to_port           = 53
  description       = "VPC DNS resolution fallback"
}

resource "aws_iam_role" "delivery" {
  name = "${local.resource_prefix}-delivery-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = { Service = "lambda.amazonaws.com" }, Action = "sts:AssumeRole"
    }]
  })
  tags = merge(local.common_tags, { Component = "notifications" })
}

resource "aws_cloudwatch_log_group" "delivery" {
  name              = "/aws/lambda/${local.resource_prefix}-delivery"
  retention_in_days = 7
  tags              = merge(local.common_tags, { Component = "notifications" })
}

resource "aws_iam_role_policy" "delivery" {
  name = "${local.resource_prefix}-delivery-runtime"
  role = aws_iam_role.delivery.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [
      {
        Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "${aws_cloudwatch_log_group.delivery.arn}:*"
      },
      {
        Effect = "Allow", Action = local.lambda_network_actions, Resource = "*"
      }
    ]
  })
}

resource "aws_lambda_function" "delivery" {
  function_name                  = "${local.resource_prefix}-delivery"
  role                           = aws_iam_role.delivery.arn
  runtime                        = "provided.al2023"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  filename                       = var.delivery_artifact_path
  source_code_hash               = filebase64sha256(var.delivery_artifact_path)
  memory_size                    = 128
  timeout                        = 10
  reserved_concurrent_executions = 0
  vpc_config {
    subnet_ids = [var.subnet_id]
    # Retain the existing notification database-client security group identity.
    security_group_ids = [aws_security_group.sender.id]
  }
  depends_on = [aws_cloudwatch_log_group.delivery, aws_iam_role_policy.delivery]
  lifecycle {
    ignore_changes = [environment, filename, source_code_hash, reserved_concurrent_executions]
  }
  tags = merge(local.common_tags, { Component = "notifications" })
}
