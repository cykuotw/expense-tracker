resource "aws_apigatewayv2_api" "worker" {
  name                         = "${local.resource_prefix}-http-api"
  protocol_type                = "HTTP"
  disable_execute_api_endpoint = false
  tags = merge(local.common_tags, {
    Component = "backend"
  })
  lifecycle {
    ignore_changes = [disable_execute_api_endpoint]
  }
}
resource "aws_acm_certificate" "api" {
  domain_name       = var.api_hostname
  validation_method = "DNS"
  tags = merge(local.common_tags, {
    Component = "backend"
  })
  lifecycle {
    create_before_destroy = true
    precondition {
      condition     = var.api_hostname == var.hosted_zone_name || endswith(var.api_hostname, ".${var.hosted_zone_name}")
      error_message = "api_hostname must belong to hosted_zone_name."
    }

  }
}
resource "aws_route53_record" "api_certificate_validation" {
  for_each = {
    for option in aws_acm_certificate.api.domain_validation_options : option.domain_name => {
      name = option.resource_record_name, record = option.resource_record_value, type = option.resource_record_type
    }
  }
  allow_overwrite = true
  zone_id         = data.aws_route53_zone.selected.zone_id
  name            = each.value.name
  type            = each.value.type
  ttl             = 60
  records         = [each.value.record]
}
resource "aws_acm_certificate_validation" "api" {
  certificate_arn         = aws_acm_certificate.api.arn
  validation_record_fqdns = [for record in aws_route53_record.api_certificate_validation : record.fqdn]
}
resource "aws_apigatewayv2_domain_name" "api" {
  domain_name = var.api_hostname
  domain_name_configuration {
    certificate_arn = aws_acm_certificate_validation.api.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
  tags = merge(local.common_tags, {
    Component = "backend"
  })
}
resource "aws_apigatewayv2_integration" "worker" {
  api_id                 = aws_apigatewayv2_api.worker.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.worker.invoke_arn
  integration_method     = "POST"
  payload_format_version = "2.0"
  timeout_milliseconds   = 15000
}
resource "aws_apigatewayv2_authorizer" "google" {
  api_id           = aws_apigatewayv2_api.worker.id
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  name             = "google-id-token"
  jwt_configuration {
    audience = [var.google_client_id]
    issuer   = local.google_issuer
  }
}
resource "aws_apigatewayv2_route" "google_exchange" {
  api_id             = aws_apigatewayv2_api.worker.id
  route_key          = "POST ${local.api_path}/auth/google/exchange"
  target             = "integrations/${aws_apigatewayv2_integration.worker.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.google.id
}
resource "aws_apigatewayv2_route" "default" {
  api_id             = aws_apigatewayv2_api.worker.id
  route_key          = "$default"
  target             = "integrations/${aws_apigatewayv2_integration.worker.id}"
  authorization_type = "NONE"
}
resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.worker.id
  name        = "$default"
  auto_deploy = true
  tags        = local.common_tags
}
resource "aws_apigatewayv2_api_mapping" "api" {
  api_id      = aws_apigatewayv2_api.worker.id
  domain_name = aws_apigatewayv2_domain_name.api.id
  stage       = aws_apigatewayv2_stage.default.id
}
resource "aws_route53_record" "api" {
  zone_id = data.aws_route53_zone.selected.zone_id
  name    = var.api_hostname
  type    = "A"
  alias {
    name                   = aws_apigatewayv2_domain_name.api.domain_name_configuration[0].target_domain_name
    zone_id                = aws_apigatewayv2_domain_name.api.domain_name_configuration[0].hosted_zone_id
    evaluate_target_health = false
  }
}
resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowHTTPAPIInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.worker.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.worker.execution_arn}/*/*"
}
