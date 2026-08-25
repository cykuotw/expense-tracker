data "aws_caller_identity" "current" {}

data "aws_route53_zone" "selected" {
  name         = var.hosted_zone_name
  private_zone = false
}

locals {
  name_prefix = "${var.project_name}-${var.environment}"
  origin_id   = "${local.name_prefix}-frontend"
  tags = merge(var.tags, {
    Project   = var.project_name
    Component = "frontend"
    ManagedBy = "terraform"
  })
}

resource "aws_s3_bucket" "frontend" {
  bucket        = "${local.name_prefix}-frontend-${data.aws_caller_identity.current.account_id}"
  force_destroy = true
  tags          = local.tags
}

resource "aws_s3_bucket_public_access_block" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_acm_certificate" "frontend" {
  provider          = aws.us_east_1
  domain_name       = var.frontend_hostname
  validation_method = "DNS"
  tags              = local.tags

  lifecycle {
    create_before_destroy = true

    precondition {
      condition     = var.frontend_hostname == var.hosted_zone_name || endswith(var.frontend_hostname, ".${var.hosted_zone_name}")
      error_message = "frontend_hostname must belong to hosted_zone_name."
    }
  }
}

resource "aws_route53_record" "frontend_certificate_validation" {
  for_each = {
    for option in aws_acm_certificate.frontend.domain_validation_options : option.domain_name => {
      name   = option.resource_record_name
      record = option.resource_record_value
      type   = option.resource_record_type
    }
  }

  allow_overwrite = true
  zone_id         = data.aws_route53_zone.selected.zone_id
  name            = each.value.name
  type            = each.value.type
  ttl             = 60
  records         = [each.value.record]
}

resource "aws_acm_certificate_validation" "frontend" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.frontend.arn
  validation_record_fqdns = [for record in aws_route53_record.frontend_certificate_validation : record.fqdn]
}

resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = local.origin_id
  description                       = "CloudFront access control for the Expense Tracker frontend bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "frontend" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = "${local.name_prefix} frontend"
  default_root_object = "index.html"
  aliases             = [var.frontend_hostname]
  price_class         = "PriceClass_100"

  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = local.origin_id
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = local.origin_id
    viewer_protocol_policy = "redirect-to-https"
    compress               = true

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }
  }

  custom_error_response {
    error_code         = 403
    response_code      = 200
    response_page_path = "/index.html"
  }

  custom_error_response {
    error_code         = 404
    response_code      = 200
    response_page_path = "/index.html"
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.frontend.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  tags = local.tags
}

resource "aws_s3_bucket_policy" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "AllowCloudFrontRead"
      Effect = "Allow"
      Principal = {
        Service = "cloudfront.amazonaws.com"
      }
      Action   = "s3:GetObject"
      Resource = "${aws_s3_bucket.frontend.arn}/*"
      Condition = {
        StringEquals = {
          "AWS:SourceArn" = aws_cloudfront_distribution.frontend.arn
        }
      }
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.frontend]
}

resource "aws_route53_record" "frontend" {
  zone_id = data.aws_route53_zone.selected.zone_id
  name    = var.frontend_hostname
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.frontend.domain_name
    zone_id                = aws_cloudfront_distribution.frontend.hosted_zone_id
    evaluate_target_health = false
  }
}
