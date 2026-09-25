data "aws_caller_identity" "current" {}
data "aws_route53_zone" "selected" {
  name         = var.hosted_zone_name
  private_zone = false
}

locals {
  resource_prefix = "${var.name_prefix}-${var.environment}"
  api_path        = "/api/v0"
  google_issuer   = "https://accounts.google.com"
  worker_live_arn = "${aws_lambda_function.worker.arn}:live"
  worker_live_invoke_arn = replace(
    aws_lambda_function.worker.invoke_arn,
    "/invocations",
    ":live/invocations",
  )
  ocr_live_arn = "${aws_lambda_function.ocr.arn}:live"
  ocr_live_invoke_arn = replace(
    aws_lambda_function.ocr.invoke_arn,
    "/invocations",
    ":live/invocations",
  )
  sender_live_arn         = "${aws_lambda_function.sender.arn}:live"
  delivery_live_arn       = "${aws_lambda_function.delivery.arn}:live"
  error_notifier_live_arn = try("${aws_lambda_function.error_notifier[0].arn}:live", "")
  common_tags = merge(var.tags, {
    Project     = var.name_prefix
    Environment = var.environment
    ManagedBy   = "terraform"
    Deployment  = "serverless"
  })
}
