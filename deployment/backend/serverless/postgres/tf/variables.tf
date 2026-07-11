variable "aws_region" {
  description = "AWS region for the PostgreSQL EC2 instance."
  type        = string
}

variable "vpc_id" {
  description = "Existing VPC ID."
  type        = string
}

variable "subnet_id" {
  description = "Existing public subnet with an Internet Gateway default route."
  type        = string
}

variable "key_pair_name" {
  description = "Existing EC2 key-pair name used for operator SSH."
  type        = string
}

variable "enable_temporary_public_access" {
  description = "Temporarily attach an EIP and allow restricted SSH for initial PostgreSQL setup."
  type        = bool
  default     = false
}

variable "operator_ssh_cidr" {
  description = "Operator public IPv4 CIDR allowed to SSH only while temporary public access is enabled."
  type        = string
  default     = null
  nullable    = true

  validation {
    condition = var.operator_ssh_cidr == null || (
      can(cidrhost(var.operator_ssh_cidr, 0)) &&
      can(regex("^([0-9]{1,3}\\.){3}[0-9]{1,3}/[0-9]{1,2}$", var.operator_ssh_cidr)) &&
      var.operator_ssh_cidr != "0.0.0.0/0"
    )
    error_message = "operator_ssh_cidr must be null or a restricted IPv4 CIDR."
  }
}

variable "name_prefix" {
  description = "Name prefix for Phase 7 resources."
  type        = string
  default     = "expense-tracker-postgres"
}

variable "instance_type" {
  description = "ARM64 EC2 instance type."
  type        = string
  default     = "t4g.micro"
}

variable "ami_id" {
  description = "Optional explicit AL2023 ARM64 AMI ID."
  type        = string
  default     = null
  nullable    = true
}

variable "tags" {
  description = "Additional AWS resource tags."
  type        = map(string)
  default     = {}
}
