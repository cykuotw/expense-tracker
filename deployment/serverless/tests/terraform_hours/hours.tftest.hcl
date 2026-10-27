mock_provider "aws" {
  mock_resource "aws_instance" {
    defaults = { id = "i-00000000000000000", arn = "arn:aws:ec2:ca-central-1:123456789012:instance/i-00000000000000000" }
  }
  mock_resource "aws_lambda_function" {
    defaults = { arn = "arn:aws:lambda:ca-central-1:123456789012:function:delivery" }
  }
  mock_resource "aws_iam_role" {
    defaults = { arn = "arn:aws:iam::123456789012:role/hours-test" }
  }
  mock_resource "aws_scheduler_schedule_group" {
    defaults = { arn = "arn:aws:scheduler:ca-central-1:123456789012:schedule-group/hours-test" }
  }
  mock_resource "aws_sqs_queue" {
    defaults = { arn = "arn:aws:sqs:ca-central-1:123456789012:hours-test" }
  }
  mock_resource "aws_s3_bucket" {
    defaults = { arn = "arn:aws:s3:::hours-test-frontend" }
  }
}
variables {
  operating_hours = { configured = true, enabled = true, start = "09:00", stop = "22:00", safety_stop = "01:00", timezone = "America/Toronto"
  }
  activate_operating_hours = true
}
run "pre_alias" {
  command = apply
  assert {
    condition     = aws_scheduler_schedule.monthly_review[0].target[0].arn == "arn:aws:lambda:ca-central-1:123456789012:function:delivery"
    error_message = "Pre-alias invocation changed"
  }
  assert {
    condition     = length(aws_scheduler_schedule.database_hours) == 3
    error_message = "Missing discrete EC2 schedules"
  }
  assert {
    condition     = aws_scheduler_schedule.database_hours["safety-stop"].target[0].retry_policy[0].maximum_retry_attempts == 0
    error_message = "Unexpected delayed retry"
  }
}
run "alias_cutover" {
  command = apply
  variables {
    use_lambda_aliases = true
  }
  assert {
    condition     = aws_scheduler_schedule.monthly_review[0].target[0].arn == "arn:aws:lambda:ca-central-1:123456789012:function:delivery:live"
    error_message = "Alias cutover did not update target"
  }
  assert {
    condition     = aws_instance.postgres.id == run.pre_alias.database_id
    error_message = "Database changed during cutover"
  }
}
run "disable" {
  command = apply
  variables {
    use_lambda_aliases = true
    operating_hours    = { configured = true, enabled = false, start = "09:00", stop = "22:00", safety_stop = "01:00", timezone = "America/Toronto" }
  }
  assert {
    condition     = length(aws_scheduler_schedule.database_hours) == 0 && length(aws_scheduler_schedule.monthly_review) == 1
    error_message = "Disable must remove only EC2 schedules and preserve daily jobs"
  }
  assert {
    condition     = aws_instance.postgres.id == run.pre_alias.database_id
    error_message = "Database changed during disablement"
  }
}
