variable "aws_region" {
  description = "AWS region shared with the Phase 7 PostgreSQL EC2 instance."
  type        = string
}

variable "postgres_name" {
  description = "Stable Phase 7 PostgreSQL EC2 Name tag and client security-group prefix."
  type        = string
  default     = "expense-tracker-postgres"
}

variable "function_name" {
  description = "Private operator-invoked bootstrap Lambda function name."
  type        = string
  default     = "expense-tracker-bootstrap"
}

variable "artifact_path" {
  description = "Bootstrap Lambda ZIP built by scripts/build-bootstrap.sh."
  type        = string
  default     = "../build/bootstrap.zip"
}

variable "tags" {
  description = "Additional AWS resource tags."
  type        = map(string)
  default     = {}
}
