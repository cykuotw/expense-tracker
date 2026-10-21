# Improvement evidence is separate from retained product receipts. Capture stays
# disabled in runtime projections until consent/purge verification is complete.
resource "aws_s3_bucket" "ocr_evidence" {
  bucket_prefix = "${substr(local.resource_prefix, 0, 35)}-ocr-evidence-"
  force_destroy = false
  tags          = merge(local.common_tags, { Component = "ocr-evidence" })
}

resource "aws_s3_bucket_public_access_block" "ocr_evidence" {
  bucket                  = aws_s3_bucket.ocr_evidence.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "ocr_evidence" {
  bucket = aws_s3_bucket.ocr_evidence.id
  rule { object_ownership = "BucketOwnerEnforced" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "ocr_evidence" {
  bucket = aws_s3_bucket.ocr_evidence.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}

# Unversioned: opt-out deletion must not leave hidden noncurrent copies.
resource "aws_s3_bucket_lifecycle_configuration" "ocr_evidence" {
  bucket = aws_s3_bucket.ocr_evidence.id
  rule {
    id     = "expire-unconfirmed-evidence"
    status = "Enabled"
    filter { prefix = "temporary/" }
    expiration { days = 1 }
  }
}

resource "aws_iam_role" "ocr_evidence_replay" {
  name = "${local.resource_prefix}-ocr-evidence-replay"
  assume_role_policy = jsonencode({
    Version = "2012-10-17", Statement = [{
      Effect = "Allow", Principal = { AWS = "arn:aws:iam::${var.expected_account_id}:root" }, Action = "sts:AssumeRole"
    }]
  })
  tags = merge(local.common_tags, { Component = "ocr-evidence" })
}

resource "aws_iam_role_policy" "ocr_evidence_replay" {
  role = aws_iam_role.ocr_evidence_replay.id
  policy = jsonencode({ Version = "2012-10-17", Statement = [
    { Effect = "Allow", Action = ["s3:GetObject"], Resource = "${aws_s3_bucket.ocr_evidence.arn}/confirmed/*" },
    { Effect = "Allow", Action = ["s3:ListBucket"], Resource = aws_s3_bucket.ocr_evidence.arn, Condition = { StringLike = { "s3:prefix" = ["confirmed/*"] } } }
  ] })
}

resource "aws_iam_role_policy" "ocr_evidence_writer" {
  role = aws_iam_role.ocr.id
  policy = jsonencode({ Version = "2012-10-17", Statement = [
    { Effect = "Allow", Action = ["s3:PutObject"], Resource = "${aws_s3_bucket.ocr_evidence.arn}/temporary/*" }
  ] })
}

resource "aws_iam_role_policy" "ocr_evidence_worker" {
  role = aws_iam_role.worker.id
  policy = jsonencode({ Version = "2012-10-17", Statement = [
    { Effect = "Allow", Action = ["s3:GetObject"], Resource = "${aws_s3_bucket.ocr_evidence.arn}/temporary/*" },
    { Effect = "Allow", Action = ["s3:PutObject"], Resource = "${aws_s3_bucket.ocr_evidence.arn}/confirmed/*" },
    { Effect = "Allow", Action = ["s3:DeleteObject"], Resource = "${aws_s3_bucket.ocr_evidence.arn}/*" },
    { Effect = "Allow", Action = ["s3:ListBucket"], Resource = aws_s3_bucket.ocr_evidence.arn }
  ] })
}

resource "aws_s3_bucket_policy" "ocr_evidence" {
  bucket = aws_s3_bucket.ocr_evidence.id
  policy = jsonencode({ Version = "2012-10-17", Statement = [
    { Sid = "DenyInsecureTransport", Effect = "Deny", Principal = "*", Action = "s3:*", Resource = [aws_s3_bucket.ocr_evidence.arn, "${aws_s3_bucket.ocr_evidence.arn}/*"], Condition = { Bool = { "aws:SecureTransport" = "false" } } },
    { Sid = "DenyEvidenceReadsExceptOfflineReplay", Effect = "Deny", Principal = "*", Action = ["s3:GetObject", "s3:GetObjectVersion"], Resource = "${aws_s3_bucket.ocr_evidence.arn}/confirmed/*", Condition = { ArnNotEquals = { "aws:PrincipalArn" = aws_iam_role.ocr_evidence_replay.arn } } },
    { Sid = "DenyTemporaryEvidenceReadsExceptWorker", Effect = "Deny", Principal = "*", Action = ["s3:GetObject", "s3:GetObjectVersion"], Resource = "${aws_s3_bucket.ocr_evidence.arn}/temporary/*", Condition = { ArnNotEquals = { "aws:PrincipalArn" = aws_iam_role.worker.arn } } }
  ] })
  depends_on = [aws_s3_bucket_public_access_block.ocr_evidence]
}

output "ocr_evidence_bucket_name" { value = aws_s3_bucket.ocr_evidence.bucket }
output "ocr_evidence_replay_role_arn" { value = aws_iam_role.ocr_evidence_replay.arn }
