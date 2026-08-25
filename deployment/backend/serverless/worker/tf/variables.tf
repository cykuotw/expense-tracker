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
  description = "Worker Lambda function name."
  type        = string
  default     = "expense-tracker-worker"
}

variable "frontend_origin" {
  description = "Exact intended Phase 10 or already-deployed HTTPS frontend origin allowed by application-owned CORS."
  type        = string

  validation {
    condition     = can(regex("^https://[^/]+$", var.frontend_origin))
    error_message = "frontend_origin must be one HTTPS origin without a path or trailing slash."
  }
}

variable "google_client_id" {
  description = "Non-secret Google web client ID used as the JWT authorizer audience."
  type        = string

  validation {
    condition     = trimspace(var.google_client_id) != ""
    error_message = "google_client_id must not be empty."
  }
}

variable "hosted_zone_name" {
  description = "Existing public Route 53 hosted-zone name for the API custom domain."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$", var.hosted_zone_name))
    error_message = "hosted_zone_name must be a lowercase public DNS zone without a trailing dot."
  }
}

variable "api_hostname" {
  description = "Exact regional API Gateway custom hostname."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$", var.api_hostname))
    error_message = "api_hostname must be a lowercase DNS hostname without a scheme, path, or trailing dot."
  }
}

variable "reserved_concurrency" {
  description = "Terraform creates the worker disabled; scripts/activate.sh owns the live transition to 3 after bootstrap."
  type        = number
  default     = 0

  validation {
    condition     = var.reserved_concurrency == 0
    error_message = "reserved_concurrency must remain 0 in Terraform; use make worker-activate after bootstrap."
  }
}

variable "artifact_path" {
  description = "Worker Lambda ZIP built by scripts/build-worker.sh."
  type        = string
  default     = "../build/worker.zip"
}

variable "tags" {
  description = "Additional AWS resource tags."
  type        = map(string)
  default     = {}
}
