variable "aws_region" { type = string }
variable "expected_account_id" { type = string }
variable "name_prefix" { type = string }
variable "environment" { type = string }
variable "tags" { type = map(string) }
variable "vpc_id" { type = string }
variable "subnet_id" { type = string }
variable "key_pair_name" { type = string }
variable "operator_ssh_cidr" {
  type = string
  validation {
    condition     = can(cidrhost(var.operator_ssh_cidr, 0)) && var.operator_ssh_cidr != "0.0.0.0/0"
    error_message = "operator_ssh_cidr must be a restricted CIDR."
  }
}
variable "enable_temporary_public_access" { type = bool }
variable "enable_restore_verification" { type = bool }
variable "hosted_zone_name" { type = string }
variable "database_instance_type" { type = string }
variable "database_ami_id" {
  type     = string
  nullable = true
}
variable "worker_artifact_path" { type = string }
variable "notifier_artifact_path" { type = string }
variable "enable_error_alerting" {
  type        = bool
  description = "Create the optional Discord error-alerting resources."
  default     = false
}
variable "worker_log_retention_days" {
  type        = number
  description = "Number of days to retain Worker logs in CloudWatch Logs."
  default     = 3

  validation {
    condition = contains([
      1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731,
      1096, 1827, 2192, 2557, 2922, 3288, 3653,
    ], var.worker_log_retention_days)
    error_message = "worker_log_retention_days must be a retention period supported by CloudWatch Logs."
  }
}
variable "bootstrap_artifact_path" { type = string }
variable "sender_artifact_path" { type = string }
variable "delivery_artifact_path" { type = string }
variable "api_hostname" { type = string }
variable "frontend_hostname" { type = string }
variable "google_client_id" { type = string }
