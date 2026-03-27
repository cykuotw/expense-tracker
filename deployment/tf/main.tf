locals {
  name_prefix                           = "${var.project_name}-${var.environment}"
  frontend_fqdn                         = "${var.frontend_subdomain}.${var.root_domain}"
  api_fqdn                              = "${var.api_subdomain}.${var.root_domain}"
  db_admin_password_ssm_parameter_name  = "/${var.project_name}/${var.environment}/db/admin_password"
  db_migration_password_ssm_parameter_name = "/${var.project_name}/${var.environment}/db/migration_password"
  db_app_password_ssm_parameter_name    = "/${var.project_name}/${var.environment}/db/app_password"
  jwt_secret_ssm_parameter_name         = "/${var.project_name}/${var.environment}/app/jwt_secret"
  refresh_jwt_secret_ssm_parameter_name = "/${var.project_name}/${var.environment}/app/refresh_jwt_secret"
  third_party_session_secret_ssm_parameter_name = "/${var.project_name}/${var.environment}/app/third_party_session_secret"
  google_client_id_ssm_parameter_name   = "/${var.project_name}/${var.environment}/app/google_client_id"
  google_client_secret_ssm_parameter_name = "/${var.project_name}/${var.environment}/app/google_client_secret"
  first_admin_bootstrap_ssm_parameter_prefix = "/${var.project_name}/${var.environment}/deploy/first_admin"
  frontend_origin_ssm_parameter_name    = "/${var.project_name}/${var.environment}/runtime/frontend_origin"
  cors_allowed_origins_ssm_parameter_name = "/${var.project_name}/${var.environment}/runtime/cors_allowed_origins"
  cors_allow_credentials_ssm_parameter_name = "/${var.project_name}/${var.environment}/runtime/cors_allow_credentials"
  auth_cookie_domain_ssm_parameter_name = "/${var.project_name}/${var.environment}/runtime/auth_cookie_domain"
  db_public_host_ssm_parameter_name     = "/${var.project_name}/${var.environment}/runtime/db_public_host"
  db_port_ssm_parameter_name            = "/${var.project_name}/${var.environment}/runtime/db_port"
  db_user_ssm_parameter_name            = "/${var.project_name}/${var.environment}/runtime/db_user"
  db_name_ssm_parameter_name            = "/${var.project_name}/${var.environment}/runtime/db_name"
  db_sslmode_ssm_parameter_name         = "/${var.project_name}/${var.environment}/runtime/db_sslmode"
  google_callback_url_ssm_parameter_name = "/${var.project_name}/${var.environment}/runtime/google_callback_url"
  jwt_secret_ssm_parameter_arn          = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter${local.jwt_secret_ssm_parameter_name}"
  refresh_jwt_secret_ssm_parameter_arn  = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter${local.refresh_jwt_secret_ssm_parameter_name}"
  third_party_session_secret_ssm_parameter_arn = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter${local.third_party_session_secret_ssm_parameter_name}"
  runtime_config_parameters = {
    frontend_origin = {
      name  = local.frontend_origin_ssm_parameter_name
      value = "https://${local.frontend_fqdn}"
    }
    cors_allowed_origins = {
      name  = local.cors_allowed_origins_ssm_parameter_name
      value = "https://${local.frontend_fqdn}"
    }
    cors_allow_credentials = {
      name  = local.cors_allow_credentials_ssm_parameter_name
      value = "true"
    }
    auth_cookie_domain = {
      name  = local.auth_cookie_domain_ssm_parameter_name
      value = ".${local.frontend_fqdn}"
    }
    db_public_host = {
      name  = local.db_public_host_ssm_parameter_name
      value = aws_db_instance.db.address
    }
    db_port = {
      name  = local.db_port_ssm_parameter_name
      value = tostring(var.db_port)
    }
    db_user = {
      name  = local.db_user_ssm_parameter_name
      value = var.db_app_username
    }
    db_name = {
      name  = local.db_name_ssm_parameter_name
      value = aws_db_instance.db.db_name
    }
    db_sslmode = {
      name  = local.db_sslmode_ssm_parameter_name
      value = "require"
    }
    google_callback_url = {
      name  = local.google_callback_url_ssm_parameter_name
      value = "https://${local.api_fqdn}/api/v0/auth/google/callback"
    }
  }
  common_tags = merge(
    {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "terraform"
    },
    var.tags,
  )
}

data "aws_vpc" "default" {
  default = true
}

data "aws_caller_identity" "current" {}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }
}

data "aws_route53_zone" "root" {
  count        = var.create_api_dns_record || var.create_frontend_dns_record ? 1 : 0
  name         = var.root_domain
  private_zone = false
}

resource "aws_s3_bucket" "frontend" {
  bucket = var.frontend_bucket_name
  tags   = local.common_tags
}

resource "aws_s3_bucket_versioning" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  versioning_configuration {
    status = "Enabled"
  }
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

resource "aws_s3_bucket" "artifacts" {
  bucket = var.artifact_bucket_name
  tags   = local.common_tags
}

resource "aws_s3_bucket_versioning" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_security_group" "backend" {
  name        = "${local.name_prefix}-backend"
  description = "Backend host security group"
  vpc_id      = data.aws_vpc.default.id
  tags        = local.common_tags

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = var.ingress_cidr_blocks
  }

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.ingress_cidr_blocks
  }

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ingress_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "db" {
  name        = "${local.name_prefix}-db"
  description = "RDS PostgreSQL security group"
  vpc_id      = data.aws_vpc.default.id
  tags        = local.common_tags

  ingress {
    description     = "PostgreSQL from backend"
    from_port       = var.db_port
    to_port         = var.db_port
    protocol        = "tcp"
    security_groups = [aws_security_group.backend.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_db_subnet_group" "db" {
  name       = "${local.name_prefix}-db"
  subnet_ids = data.aws_subnets.default.ids
  tags       = local.common_tags
}

resource "aws_iam_role" "backend" {
  name = "${local.name_prefix}-backend-role"
  tags = local.common_tags

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "ssm_core" {
  role       = aws_iam_role.backend.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy" "artifact_access" {
  name = "${local.name_prefix}-artifact-access"
  role = aws_iam_role.backend.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.artifacts.arn,
          "${aws_s3_bucket.artifacts.arn}/*"
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy" "db_credentials_access" {
  name = "${local.name_prefix}-db-credentials-access"
  role = aws_iam_role.backend.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter"
        ]
        Resource = compact(concat(
          [
            aws_ssm_parameter.db_admin_password.arn,
            aws_ssm_parameter.db_migration_password.arn,
            aws_ssm_parameter.db_app_password.arn,
            local.jwt_secret_ssm_parameter_arn,
            local.refresh_jwt_secret_ssm_parameter_arn,
            local.third_party_session_secret_ssm_parameter_arn,
          ],
          [for parameter in values(aws_ssm_parameter.runtime_config) : parameter.arn],
          [
            try(aws_ssm_parameter.google_client_id[0].arn, null),
            try(aws_ssm_parameter.google_client_secret[0].arn, null),
          ],
        ))
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:PutParameter"
        ]
        Resource = [
          local.jwt_secret_ssm_parameter_arn,
          local.refresh_jwt_secret_ssm_parameter_arn,
          local.third_party_session_secret_ssm_parameter_arn,
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:DeleteParameter"
        ]
        Resource = [
          "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter${local.first_admin_bootstrap_ssm_parameter_prefix}/*",
        ]
      }
    ]
  })
}

resource "aws_iam_instance_profile" "backend" {
  name = "${local.name_prefix}-backend-profile"
  role = aws_iam_role.backend.name
}

resource "aws_instance" "backend" {
  ami                    = data.aws_ami.amazon_linux.id
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.backend.id]
  iam_instance_profile   = aws_iam_instance_profile.backend.name
  key_name               = var.key_name

  user_data = templatefile("${path.module}/templates/user_data.sh.tftpl", {
    app_dir         = var.app_dir
    backend_env_dir = var.backend_env_dir
  })

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-backend"
  })
}

resource "aws_eip" "backend" {
  domain   = "vpc"
  instance = aws_instance.backend.id
  tags     = local.common_tags
}

resource "aws_db_instance" "db" {
  identifier                 = "${local.name_prefix}-db"
  engine                     = "postgres"
  instance_class             = var.db_instance_class
  allocated_storage          = var.db_allocated_storage
  max_allocated_storage      = var.db_max_allocated_storage
  storage_type               = "gp3"
  storage_encrypted          = true
  db_name                    = var.db_name
  username                   = var.db_admin_username
  password                   = var.db_admin_password
  port                       = var.db_port
  backup_retention_period    = var.db_backup_retention_period
  db_subnet_group_name       = aws_db_subnet_group.db.name
  vpc_security_group_ids     = [aws_security_group.db.id]
  publicly_accessible        = false
  multi_az                   = false
  auto_minor_version_upgrade = true
  deletion_protection        = false
  skip_final_snapshot        = var.db_skip_final_snapshot
  apply_immediately          = true
  tags                       = local.common_tags
}

resource "aws_ssm_parameter" "db_admin_password" {
  name      = local.db_admin_password_ssm_parameter_name
  type      = "SecureString"
  value     = var.db_admin_password
  overwrite = true
  tags      = local.common_tags
}

resource "aws_ssm_parameter" "db_migration_password" {
  name      = local.db_migration_password_ssm_parameter_name
  type      = "SecureString"
  value     = var.db_migration_password
  overwrite = true
  tags      = local.common_tags
}

resource "aws_ssm_parameter" "db_app_password" {
  name      = local.db_app_password_ssm_parameter_name
  type      = "SecureString"
  value     = var.db_app_password
  overwrite = true
  tags      = local.common_tags
}

resource "aws_ssm_parameter" "runtime_config" {
  for_each  = local.runtime_config_parameters
  name      = each.value.name
  type      = "String"
  value     = each.value.value
  overwrite = true
  tags      = local.common_tags
}

resource "aws_ssm_parameter" "google_client_id" {
  count     = var.google_client_id != "" ? 1 : 0
  name      = local.google_client_id_ssm_parameter_name
  type      = "SecureString"
  value     = var.google_client_id
  overwrite = true
  tags      = local.common_tags
}

resource "aws_ssm_parameter" "google_client_secret" {
  count     = var.google_client_secret != "" ? 1 : 0
  name      = local.google_client_secret_ssm_parameter_name
  type      = "SecureString"
  value     = var.google_client_secret
  overwrite = true
  tags      = local.common_tags
}

resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "${local.name_prefix}-frontend"
  description                       = "CloudFront access control for the frontend bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "frontend" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = "${local.name_prefix} frontend"
  default_root_object = "index.html"
  aliases             = [local.frontend_fqdn]

  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = aws_s3_bucket.frontend.id
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = aws_s3_bucket.frontend.id
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
    acm_certificate_arn      = var.frontend_certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  tags = local.common_tags
}

resource "aws_s3_bucket_policy" "frontend" {
  bucket = aws_s3_bucket.frontend.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
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
      }
    ]
  })
}

resource "aws_route53_record" "frontend" {
  count   = var.create_frontend_dns_record ? 1 : 0
  zone_id = data.aws_route53_zone.root[0].zone_id
  name    = local.frontend_fqdn
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.frontend.domain_name
    zone_id                = aws_cloudfront_distribution.frontend.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "api" {
  count   = var.create_api_dns_record ? 1 : 0
  zone_id = data.aws_route53_zone.root[0].zone_id
  name    = local.api_fqdn
  type    = "A"
  ttl     = 300
  records = [aws_eip.backend.public_ip]
}
