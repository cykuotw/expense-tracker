# Local PostgreSQL monitoring uses the existing free S3 gateway path. Only state
# transitions create objects; the existing outside-VPC notifier sends Discord.
resource "aws_iam_role_policy" "database_alert_writer" {
  count = var.enable_database_connection_alerting ? 1 : 0
  name  = "${local.resource_prefix}-database-alert-write"
  role  = aws_iam_role.postgres_backup_writer.id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Action = ["s3:PutObject"], Resource = "${aws_s3_bucket.postgres_backup.arn}/database-alerts/*"
    }]
  })
}

resource "aws_iam_role_policy" "database_alert_reader" {
  count = var.enable_database_connection_alerting ? 1 : 0
  name  = "${local.resource_prefix}-database-alert-read"
  role  = aws_iam_role.error_notifier[0].id
  policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Action = ["s3:GetObject"], Resource = "${aws_s3_bucket.postgres_backup.arn}/database-alerts/*"
    }]
  })
}

resource "aws_lambda_permission" "database_alerts" {
  count          = var.enable_database_connection_alerting ? 1 : 0
  statement_id   = "AllowPrivateDatabaseAlertObjects"
  action         = "lambda:InvokeFunction"
  function_name  = aws_lambda_function.error_notifier[0].function_name
  qualifier      = var.use_lambda_aliases ? "live" : null
  principal      = "s3.amazonaws.com"
  source_account = var.expected_account_id
  source_arn     = aws_s3_bucket.postgres_backup.arn
}

resource "aws_s3_bucket_notification" "database_alerts" {
  count  = var.enable_database_connection_alerting ? 1 : 0
  bucket = aws_s3_bucket.postgres_backup.id
  lambda_function {
    lambda_function_arn = var.use_lambda_aliases ? local.error_notifier_live_arn : aws_lambda_function.error_notifier[0].arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "database-alerts/"
    filter_suffix       = ".json"
  }
  depends_on = [aws_lambda_permission.database_alerts, aws_iam_role_policy.database_alert_reader]
}
