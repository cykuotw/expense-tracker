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
  state               = "DISABLED"
  tags = merge(local.common_tags, {
    Component = "notifications"
  })
}

resource "aws_cloudwatch_event_target" "sender" {
  rule = aws_cloudwatch_event_rule.sender.name
  arn  = var.use_lambda_aliases ? local.sender_live_arn : aws_lambda_function.sender.arn
}

resource "aws_lambda_permission" "sender_eventbridge" {
  statement_id  = "AllowEventBridgePushSender"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.sender.function_name
  qualifier     = var.use_lambda_aliases ? "live" : null
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.sender.arn
}
