data "aws_caller_identity" "current" {}
data "aws_route53_zone" "selected" {
  name         = var.hosted_zone_name
  private_zone = false
}

locals {
  resource_prefix = "${var.name_prefix}-${var.environment}"
  api_path        = "/api/v0"
  google_issuer   = "https://accounts.google.com"
  common_tags = merge(var.tags, {
    Project     = var.name_prefix
    Environment = var.environment
    ManagedBy   = "terraform"
    Deployment  = "serverless-phase-11"
  })
}
