variable "operating_hours" {
  type = object({
    configured  = bool
    enabled     = bool
    start       = string
    stop        = string
    safety_stop = string
    timezone    = string
  })
  default = {
    configured  = false, enabled = false, start = "09:00", stop = "22:00",
    safety_stop = "01:00", timezone = "America/Toronto"
  }
}

# The deployer stages schedule changes disabled, drains old invocations, then
# enables the validated configuration. Normal deployments leave this false;
# ACTION=hours-apply is the explicit schedule activation boundary.
variable "activate_operating_hours" {
  type    = bool
  default = false
}

locals {
  database_actions = var.operating_hours.enabled ? {
    start       = { time = var.operating_hours.start, operation = "startInstances" }
    stop        = { time = var.operating_hours.stop, operation = "stopInstances" }
    safety-stop = { time = var.operating_hours.safety_stop, operation = "stopInstances" }
  } : {}
  review_minute = (tonumber(split(":", var.operating_hours.start)[0]) * 60 + tonumber(split(":", var.operating_hours.start)[1]) + 30) % 1440
}

resource "aws_scheduler_schedule_group" "database_hours" {
  count = var.operating_hours.configured ? 1 : 0
  name  = "${local.resource_prefix}-database-hours"
  tags  = merge(local.common_tags, { Component = "database-hours" })
}

resource "aws_sqs_queue" "database_hours_failures" {
  count                     = var.operating_hours.configured ? 1 : 0
  name                      = "${local.resource_prefix}-database-hours-failures"
  message_retention_seconds = 1209600
  sqs_managed_sse_enabled   = true
  tags                      = merge(local.common_tags, { Component = "database-hours" })
}

resource "aws_iam_role" "database_hours" {
  count = var.operating_hours.configured ? 1 : 0
  name  = "${local.resource_prefix}-database-hours"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = { Service = "scheduler.amazonaws.com" },
      Action = "sts:AssumeRole", Condition = {
        StringEquals = { "aws:SourceAccount" = var.expected_account_id },
        ArnEquals    = { "aws:SourceArn" = aws_scheduler_schedule_group.database_hours[0].arn }
      }
    }]
  })
  tags = merge(local.common_tags, { Component = "database-hours" })
}

resource "aws_iam_role_policy" "database_hours" {
  count = var.operating_hours.configured ? 1 : 0
  name  = "database-hours"
  role  = aws_iam_role.database_hours[0].id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [
      { Effect = "Allow", Action = ["ec2:StartInstances", "ec2:StopInstances"], Resource = aws_instance.postgres.arn },
      { Effect = "Allow", Action = "lambda:InvokeFunction", Resource = var.use_lambda_aliases ? local.delivery_live_arn : aws_lambda_function.delivery.arn },
      { Effect = "Allow", Action = "sqs:SendMessage", Resource = aws_sqs_queue.database_hours_failures[0].arn }
    ]
  })
}

resource "aws_scheduler_schedule" "database_hours" {
  for_each                     = local.database_actions
  name                         = "${local.resource_prefix}-database-${each.key}"
  group_name                   = aws_scheduler_schedule_group.database_hours[0].name
  schedule_expression          = "cron(${tonumber(split(":", each.value.time)[1])} ${tonumber(split(":", each.value.time)[0])} * * ? *)"
  schedule_expression_timezone = var.operating_hours.timezone
  state                        = var.activate_operating_hours ? "ENABLED" : "DISABLED"
  flexible_time_window { mode = "OFF" }
  target {
    arn      = "arn:aws:scheduler:::aws-sdk:ec2:${each.value.operation}"
    role_arn = aws_iam_role.database_hours[0].arn
    # Normal EC2 stop only. Never Force, SkipOsShutdown or termination.
    input = jsonencode({ InstanceIds = [aws_instance.postgres.id] })
    retry_policy {
      maximum_event_age_in_seconds = 60
      maximum_retry_attempts       = 0
    }
    dead_letter_config { arn = aws_sqs_queue.database_hours_failures[0].arn }
  }
  depends_on = [aws_iam_role_policy.database_hours]
}

# Retained when hours are disabled: daily jobs stay aligned with the last window.
resource "aws_scheduler_schedule" "monthly_review" {
  count                        = var.operating_hours.configured ? 1 : 0
  name                         = "${local.resource_prefix}-monthly-review-local"
  group_name                   = aws_scheduler_schedule_group.database_hours[0].name
  schedule_expression          = "cron(${local.review_minute % 60} ${floor(local.review_minute / 60)} * * ? *)"
  schedule_expression_timezone = var.operating_hours.timezone
  state                        = var.activate_operating_hours ? "ENABLED" : "DISABLED"
  flexible_time_window { mode = "OFF" }
  target {
    arn      = var.use_lambda_aliases ? local.delivery_live_arn : aws_lambda_function.delivery.arn
    role_arn = aws_iam_role.database_hours[0].arn
    input    = jsonencode({ action = "publish_monthly_reviews" })
    retry_policy {
      maximum_event_age_in_seconds = 60
      maximum_retry_attempts       = 0
    }
    dead_letter_config { arn = aws_sqs_queue.database_hours_failures[0].arn }
  }
  depends_on = [aws_iam_role_policy.database_hours]
}

resource "aws_iam_role_policy" "sender_operating_hours" {
  name = "${local.resource_prefix}-sender-operating-hours"
  role = aws_iam_role.sender.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect   = "Allow", Action = "s3:GetObject",
      Resource = "${aws_s3_bucket.frontend.arn}/operating-hours.json"
    }]
  })
}
