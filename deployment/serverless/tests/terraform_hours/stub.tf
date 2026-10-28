terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 6.23"
    }
  }
}
provider "aws" {
  region = "ca-central-1"
}
variable "expected_account_id" {
  default = "123456789012"
}
variable "use_lambda_aliases" {
  default = false
}
locals {
  resource_prefix   = "hours-test"
  common_tags       = {}
  delivery_live_arn = "arn:aws:lambda:ca-central-1:123456789012:function:delivery:live"
}
resource "aws_instance" "postgres" {
  ami           = "ami-00000000000000000"
  instance_type = "t4g.micro"
}
resource "aws_iam_role" "sender" {
  name               = "hours-test-sender"
  assume_role_policy = jsonencode({ Version = "2012-10-17", Statement = [] })
}
resource "aws_lambda_function" "delivery" {
  function_name = "delivery"
  role          = "arn:aws:iam::123456789012:role/delivery"
  runtime       = "provided.al2023"
  handler       = "bootstrap"
  filename      = "unused.zip"
}
resource "aws_s3_bucket" "frontend" {
  bucket = "hours-test-frontend"
}
output "database_id" {
  value = aws_instance.postgres.id
}
