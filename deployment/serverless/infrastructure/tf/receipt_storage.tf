# Phase 7 is dark: the OCR runtime does not enable temporary writes until an
# explicit Keep receipt choice and the later association flow are available.
resource "aws_s3_bucket" "receipt" {
  bucket_prefix = "${substr(local.resource_prefix, 0, 40)}-receipt-"
  force_destroy = false

  tags = merge(local.common_tags, {
    Component = "private-receipts"
  })
}

# Omit aws_s3_bucket_versioning: new buckets are unversioned until a recovery
# requirement justifies noncurrent-version retention and purge complexity.
resource "aws_s3_bucket_public_access_block" "receipt" {
  bucket                  = aws_s3_bucket.receipt.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "receipt" {
  bucket = aws_s3_bucket.receipt.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "receipt" {
  bucket = aws_s3_bucket.receipt.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "receipt" {
  bucket = aws_s3_bucket.receipt.id
  rule {
    id     = "expire-abandoned-temporary-receipts"
    status = "Enabled"
    filter {
      prefix = "temporary/"
    }
    expiration {
      days = 1
    }
  }
}

resource "aws_s3_bucket_policy" "receipt" {
  bucket = aws_s3_bucket.receipt.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource  = [aws_s3_bucket.receipt.arn, "${aws_s3_bucket.receipt.arn}/*"]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      },
      {
        Sid       = "DenyReceiptReadsUntilAuthorizedViewing"
        Effect    = "Deny"
        Principal = "*"
        Action    = ["s3:GetObject", "s3:GetObjectVersion"]
        Resource  = "${aws_s3_bucket.receipt.arn}/*"
      }
    ]
  })
  depends_on = [aws_s3_bucket_public_access_block.receipt]
}

# Daily S3 storage metrics are intentionally coarse and contain no object keys.
resource "aws_cloudwatch_metric_alarm" "receipt_temporary_backlog" {
  alarm_name          = "${local.resource_prefix}-receipt-temporary-backlog"
  alarm_description   = "Review temporary receipt count before enabling receipt storage."
  namespace           = "AWS/S3"
  metric_name         = "NumberOfObjects"
  statistic           = "Average"
  period              = 86400
  evaluation_periods  = 1
  comparison_operator = "GreaterThanThreshold"
  threshold           = 100
  treat_missing_data  = "notBreaching"
  dimensions = {
    BucketName  = aws_s3_bucket.receipt.bucket
    StorageType = "AllStorageTypes"
  }
  tags = merge(local.common_tags, { Component = "private-receipts" })
}

resource "aws_cloudwatch_log_metric_filter" "receipt_storage_write_failure" {
  name           = "${local.resource_prefix}-receipt-storage-write-failure"
  log_group_name = aws_cloudwatch_log_group.ocr.name
  pattern        = "{ $.event = \"receipt_storage_write_failed\" }"

  metric_transformation {
    name      = "ReceiptStorageWriteFailures"
    namespace = "ExpenseTracker/OCR"
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "receipt_storage_write_failure" {
  alarm_name          = "${local.resource_prefix}-receipt-storage-write-failure"
  alarm_description   = "Temporary receipt writes failed; no receipt content is logged."
  namespace           = "ExpenseTracker/OCR"
  metric_name         = "ReceiptStorageWriteFailures"
  statistic           = "Sum"
  period              = 60
  evaluation_periods  = 1
  comparison_operator = "GreaterThanOrEqualToThreshold"
  threshold           = 1
  treat_missing_data  = "notBreaching"
  tags                = merge(local.common_tags, { Component = "private-receipts" })
}
