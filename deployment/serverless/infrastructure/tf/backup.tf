resource "aws_s3_bucket" "postgres_backup" {
  bucket_prefix = "${substr(local.resource_prefix, 0, 40)}-pgb-"
  force_destroy = false

  tags = merge(local.common_tags, {
    Component = "database-backup"
    Recovery  = "postgres-logical-dump"
  })
}

resource "aws_s3_bucket_public_access_block" "postgres_backup" {
  bucket                  = aws_s3_bucket.postgres_backup.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "postgres_backup" {
  bucket = aws_s3_bucket.postgres_backup.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "postgres_backup" {
  bucket = aws_s3_bucket.postgres_backup.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "postgres_backup" {
  bucket = aws_s3_bucket.postgres_backup.id

  rule {
    id     = "expire-daily-logical-dumps-after-90-days"
    status = "Enabled"

    filter {
      prefix = "daily/"
    }

    expiration {
      days = 90
    }
  }
}

resource "aws_s3_bucket_policy" "postgres_backup" {
  bucket = aws_s3_bucket.postgres_backup.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "DenyInsecureTransport"
      Effect    = "Deny"
      Principal = "*"
      Action    = "s3:*"
      Resource  = [aws_s3_bucket.postgres_backup.arn, "${aws_s3_bucket.postgres_backup.arn}/*"]
      Condition = {
        Bool = {
          "aws:SecureTransport" = "false"
        }
      }
    }]
  })

  depends_on = [aws_s3_bucket_public_access_block.postgres_backup]
}

data "aws_iam_policy_document" "postgres_backup_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "postgres_backup_writer" {
  name               = "${local.resource_prefix}-postgres-backup-writer"
  assume_role_policy = data.aws_iam_policy_document.postgres_backup_assume_role.json

  tags = merge(local.common_tags, {
    Component = "database-backup"
  })
}

resource "aws_iam_role_policy" "postgres_backup_writer" {
  name = "${local.resource_prefix}-postgres-backup-write"
  role = aws_iam_role.postgres_backup_writer.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetBucketLocation"]
        Resource = aws_s3_bucket.postgres_backup.arn
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = aws_s3_bucket.postgres_backup.arn
        Condition = {
          StringLike = {
            "s3:prefix" = ["daily/*"]
          }
        }
      },
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.postgres_backup.arn}/daily/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.postgres_backup.arn}/status/*"
      },
    ]
  })
}

resource "aws_iam_instance_profile" "postgres_backup_writer" {
  name = "${local.resource_prefix}-postgres-backup-writer"
  role = aws_iam_role.postgres_backup_writer.name
}

resource "aws_iam_role" "postgres_restore_verifier" {
  name               = "${local.resource_prefix}-postgres-restore-verifier"
  assume_role_policy = data.aws_iam_policy_document.postgres_backup_assume_role.json

  tags = merge(local.common_tags, {
    Component = "database-restore-verification"
  })
}

resource "aws_iam_role_policy" "postgres_restore_verifier" {
  name = "${local.resource_prefix}-postgres-restore-verify"
  role = aws_iam_role.postgres_restore_verifier.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetBucketLocation", "s3:ListBucket"]
        Resource = aws_s3_bucket.postgres_backup.arn
      },
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject"]
        Resource = "${aws_s3_bucket.postgres_backup.arn}/daily/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.postgres_backup.arn}/verification/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:DeleteObject"]
        Resource = "${aws_s3_bucket.postgres_backup.arn}/verification/restore-failure.txt"
      },
    ]
  })
}

resource "aws_iam_instance_profile" "postgres_restore_verifier" {
  name = "${local.resource_prefix}-postgres-restore-verifier"
  role = aws_iam_role.postgres_restore_verifier.name
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id            = var.vpc_id
  service_name      = "com.amazonaws.${var.aws_region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = [local.selected_route_table_id]

  tags = merge(local.common_tags, {
    Component = "database-backup"
  })
}
