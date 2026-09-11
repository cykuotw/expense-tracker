resource "aws_iam_role" "error_notifier" {
  count = var.enable_error_alerting ? 1 : 0
  name  = "${local.resource_prefix}-error-notifier-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = {
        Service = "lambda.amazonaws.com"
      }, Action = "sts:AssumeRole"
    }]
  })
  tags = merge(local.common_tags, {
    Component = "observability"
  })
}

resource "aws_cloudwatch_log_group" "error_notifier" {
  count             = var.enable_error_alerting ? 1 : 0
  name              = "/aws/lambda/${local.resource_prefix}-error-notifier"
  retention_in_days = var.worker_log_retention_days
  tags = merge(local.common_tags, {
    Component = "observability"
  })
}

resource "aws_iam_role_policy" "error_notifier" {
  count = var.enable_error_alerting ? 1 : 0
  name  = "${local.resource_prefix}-error-notifier-runtime"
  role  = aws_iam_role.error_notifier[0].id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "${aws_cloudwatch_log_group.error_notifier[0].arn}:*"
    }]
  })
}

resource "aws_lambda_function" "error_notifier" {
  count                          = var.enable_error_alerting ? 1 : 0
  function_name                  = "${local.resource_prefix}-error-notifier"
  role                           = aws_iam_role.error_notifier[0].arn
  runtime                        = "provided.al2023"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  filename                       = var.notifier_artifact_path
  source_code_hash               = filebase64sha256(var.notifier_artifact_path)
  memory_size                    = 128
  timeout                        = 10
  reserved_concurrent_executions = 0
  depends_on                     = [aws_cloudwatch_log_group.error_notifier, aws_iam_role_policy.error_notifier]

  lifecycle {
    ignore_changes = [
      environment,
      filename,
      reserved_concurrent_executions,
      source_code_hash,
    ]
  }
  tags = merge(local.common_tags, {
    Component = "observability"
  })
}

resource "aws_lambda_permission" "worker_logs_error_notifier" {
  count          = var.enable_error_alerting ? 1 : 0
  statement_id   = "AllowWorkerCloudWatchLogs"
  action         = "lambda:InvokeFunction"
  function_name  = aws_lambda_function.error_notifier[0].function_name
  principal      = "logs.${var.aws_region}.amazonaws.com"
  source_account = var.expected_account_id
  source_arn     = "${aws_cloudwatch_log_group.worker.arn}:*"
}

resource "aws_cloudwatch_log_subscription_filter" "worker_error_notifier" {
  count           = var.enable_error_alerting ? 1 : 0
  name            = "${local.resource_prefix}-unexpected-server-errors"
  log_group_name  = aws_cloudwatch_log_group.worker.name
  filter_pattern  = "{ ($.alertable IS TRUE) && (($.event = \"unexpected_http_error\" && $.status >= 500) || $.event = \"panic_recovered\") }"
  destination_arn = aws_lambda_function.error_notifier[0].arn
  depends_on      = [aws_lambda_permission.worker_logs_error_notifier]
}
