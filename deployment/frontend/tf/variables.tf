variable "aws_region" {
  description = "AWS region for the frontend S3 bucket and Route 53 operations."
  type        = string
}

variable "hosted_zone_name" {
  description = "Existing public Route 53 hosted-zone name."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$", var.hosted_zone_name))
    error_message = "hosted_zone_name must be a lowercase public DNS zone without a trailing dot."
  }
}

variable "frontend_hostname" {
  description = "Exact frontend hostname served by CloudFront."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$", var.frontend_hostname))
    error_message = "frontend_hostname must be a lowercase DNS hostname without a scheme, path, or trailing dot."
  }
}

variable "project_name" {
  description = "Stable project name used in resource names and tags."
  type        = string
  default     = "expense-tracker"
}

variable "environment" {
  description = "Deployment environment used in resource names and tags."
  type        = string
  default     = "serverless"
}

variable "tags" {
  description = "Additional AWS resource tags."
  type        = map(string)
  default     = {}
}
