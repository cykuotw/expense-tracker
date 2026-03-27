variable "aws_region" {
  type        = string
  description = "AWS region for the deployment"
}

variable "project_name" {
  type        = string
  description = "Project name used for resource naming"
}

variable "environment" {
  type        = string
  description = "Environment name used in tags and resource names"
  default     = "dev"
}

variable "instance_type" {
  type        = string
  description = "EC2 instance type for the backend host"
  default     = "t3.micro"
}

variable "db_instance_class" {
  type        = string
  description = "RDS instance class for the PostgreSQL database"
  default     = "db.t4g.micro"
}

variable "db_allocated_storage" {
  type        = number
  description = "Initial allocated storage for the RDS PostgreSQL instance in GiB"
  default     = 20
}

variable "db_max_allocated_storage" {
  type        = number
  description = "Maximum autoscaled storage for the RDS PostgreSQL instance in GiB"
  default     = 100
}

variable "db_name" {
  type        = string
  description = "Primary PostgreSQL database name"
}

variable "db_admin_username" {
  type        = string
  description = "Master username for the RDS PostgreSQL instance"
}

variable "db_admin_password" {
  type        = string
  description = "Master password for the RDS PostgreSQL instance"
  sensitive   = true
}

variable "db_migration_username" {
  type        = string
  description = "Username used for schema migrations and database bootstrap"
}

variable "db_migration_password" {
  type        = string
  description = "Password used for the migration database user"
  sensitive   = true
}

variable "db_app_username" {
  type        = string
  description = "Username used by the running backend service"
}

variable "db_app_password" {
  type        = string
  description = "Password used by the runtime application database user"
  sensitive   = true
}

variable "first_admin_email" {
  type        = string
  description = "Optional email address for bootstrapping the first application admin during deploy"
  default     = ""
}

variable "first_admin_password" {
  type        = string
  description = "Optional password for bootstrapping the first application admin during deploy"
  sensitive   = true
  default     = ""
}

variable "first_admin_firstname" {
  type        = string
  description = "Optional first name for bootstrapping the first application admin during deploy"
  default     = ""
}

variable "first_admin_lastname" {
  type        = string
  description = "Optional last name for bootstrapping the first application admin during deploy"
  default     = ""
}

variable "first_admin_nickname" {
  type        = string
  description = "Optional nickname for bootstrapping the first application admin during deploy"
  default     = ""
}

variable "google_client_id" {
  type        = string
  description = "Optional Google OAuth client ID for the backend runtime"
  default     = ""
}

variable "google_client_secret" {
  type        = string
  description = "Optional Google OAuth client secret for the backend runtime"
  sensitive   = true
  default     = ""
}

variable "db_port" {
  type        = number
  description = "PostgreSQL port for the RDS instance"
  default     = 5432
}

variable "db_backup_retention_period" {
  type        = number
  description = "Automated backup retention period for the RDS instance in days"
  default     = 7
}

variable "db_skip_final_snapshot" {
  type        = bool
  description = "Whether Terraform should skip the final RDS snapshot during destroy"
  default     = true
}

variable "key_name" {
  type        = string
  description = "Optional EC2 key pair name"
  default     = null
}

variable "frontend_bucket_name" {
  type        = string
  description = "S3 bucket name for frontend assets"
}

variable "artifact_bucket_name" {
  type        = string
  description = "S3 bucket name for backend release artifacts"
}

variable "root_domain" {
  type        = string
  description = "Route 53 hosted zone domain"
}

variable "frontend_subdomain" {
  type        = string
  description = "Frontend hostname relative to the root domain"
}

variable "api_subdomain" {
  type        = string
  description = "API hostname relative to the root domain"
}

variable "create_frontend_dns_record" {
  type        = bool
  description = "Whether to create the frontend Route 53 alias record"
  default     = true
}

variable "create_api_dns_record" {
  type        = bool
  description = "Whether to create the API Route 53 A record"
  default     = true
}

variable "frontend_certificate_arn" {
  type        = string
  description = "ACM certificate ARN for the frontend CloudFront distribution. Must be in us-east-1."
}

variable "app_dir" {
  type        = string
  description = "Application directory on the EC2 host"
  default     = "/opt/expense-tracker"
}

variable "backend_env_dir" {
  type        = string
  description = "Directory containing the backend environment file on the EC2 host"
  default     = "/etc/expense-tracker"
}

variable "certbot_enabled" {
  type        = bool
  description = "Whether deploy should request or renew the API TLS certificate through certbot"
  default     = false
}

variable "certbot_email" {
  type        = string
  description = "Email address used for certbot ACME registration when certbot is enabled"
  default     = ""
}

variable "certbot_staging" {
  type        = bool
  description = "Whether deploy should use Let's Encrypt staging when running certbot"
  default     = false
}

variable "ingress_cidr_blocks" {
  type        = list(string)
  description = "IPv4 CIDR blocks allowed to access the backend host"
  default     = ["0.0.0.0/0"]
}

variable "tags" {
  type        = map(string)
  description = "Additional tags to apply to resources"
  default     = {}
}
