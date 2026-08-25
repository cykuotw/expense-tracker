output "frontend_bucket_name" {
  description = "Private S3 bucket containing frontend assets."
  value       = aws_s3_bucket.frontend.bucket
}

output "cloudfront_distribution_id" {
  description = "Dedicated frontend CloudFront distribution ID."
  value       = aws_cloudfront_distribution.frontend.id
}

output "cloudfront_domain_name" {
  description = "CloudFront-assigned distribution hostname."
  value       = aws_cloudfront_distribution.frontend.domain_name
}

output "frontend_fqdn" {
  description = "Exact frontend DNS hostname."
  value       = var.frontend_hostname
}

output "frontend_origin" {
  description = "Exact HTTPS frontend origin for worker runtime and CORS."
  value       = "https://${var.frontend_hostname}"
}
