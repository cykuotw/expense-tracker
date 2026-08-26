provider "aws" {
  region              = var.aws_region
  allowed_account_ids = [var.expected_account_id]
}

provider "aws" {
  alias               = "us_east_1"
  region              = "us-east-1"
  allowed_account_ids = [var.expected_account_id]
}
