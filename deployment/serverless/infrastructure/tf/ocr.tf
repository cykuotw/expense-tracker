resource "aws_dynamodb_table" "ocr_replay" {
  name         = "${local.resource_prefix}-ocr-replay"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "request_id"

  attribute {
    name = "request_id"
    type = "S"
  }

  ttl {
    attribute_name = "expires_at"
    enabled        = true
  }

  server_side_encryption {
    enabled = true
  }

  tags = merge(local.common_tags, {
    Component = "ocr"
  })
}

resource "aws_iam_role" "ocr" {
  name = "${local.resource_prefix}-ocr-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = {
        Service = "lambda.amazonaws.com"
      }, Action = "sts:AssumeRole"
    }]
  })
  tags = merge(local.common_tags, {
    Component = "ocr"
  })
}

resource "aws_cloudwatch_log_group" "ocr" {
  name              = "/aws/lambda/${local.resource_prefix}-ocr"
  retention_in_days = var.worker_log_retention_days
  tags = merge(local.common_tags, {
    Component = "ocr"
  })
}

resource "aws_iam_role_policy" "ocr" {
  name = "${local.resource_prefix}-ocr-runtime"
  role = aws_iam_role.ocr.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [
      {
        Effect = "Allow", Action = ["logs:CreateLogStream", "logs:PutLogEvents"], Resource = "${aws_cloudwatch_log_group.ocr.arn}:*"
      },
      {
        Effect = "Allow", Action = ["dynamodb:PutItem"], Resource = aws_dynamodb_table.ocr_replay.arn
      }
    ]
  })
}

resource "aws_lambda_function" "ocr" {
  function_name                  = "${local.resource_prefix}-ocr"
  role                           = aws_iam_role.ocr.arn
  runtime                        = "provided.al2023"
  handler                        = "bootstrap"
  architectures                  = ["arm64"]
  filename                       = var.ocr_artifact_path
  source_code_hash               = filebase64sha256(var.ocr_artifact_path)
  memory_size                    = 512
  timeout                        = 12
  reserved_concurrent_executions = 0
  depends_on                     = [aws_cloudwatch_log_group.ocr, aws_iam_role_policy.ocr]

  lifecycle {
    ignore_changes = [
      environment,
      filename,
      reserved_concurrent_executions,
      source_code_hash,
    ]
  }

  tags = merge(local.common_tags, {
    Component = "ocr"
  })
}

resource "aws_cloudwatch_metric_alarm" "ocr_runtime" {
  alarm_name          = "${local.resource_prefix}-ocr-errors-or-throttles"
  alarm_description   = "OCR Lambda errors or throttles require investigation before provider calls are enabled."
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  threshold           = 1
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "signal"
    expression  = "errors + throttles"
    label       = "OCR errors and throttles"
    return_data = true
  }

  metric_query {
    id = "errors"
    metric {
      metric_name = "Errors"
      namespace   = "AWS/Lambda"
      period      = 60
      stat        = "Sum"
      dimensions = {
        FunctionName = aws_lambda_function.ocr.function_name
      }
    }
  }

  metric_query {
    id = "throttles"
    metric {
      metric_name = "Throttles"
      namespace   = "AWS/Lambda"
      period      = 60
      stat        = "Sum"
      dimensions = {
        FunctionName = aws_lambda_function.ocr.function_name
      }
    }
  }

  tags = merge(local.common_tags, {
    Component = "ocr"
  })
}
