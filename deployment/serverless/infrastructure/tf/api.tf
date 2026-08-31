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
resource "aws_apigatewayv2_route" "google_register" {
  api_id             = aws_apigatewayv2_api.worker.id
  route_key          = "POST ${local.api_path}/auth/google/register"
  target             = "integrations/${aws_apigatewayv2_integration.worker.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.google.id
}
resource "aws_apigatewayv2_route" "google_link" {
  api_id             = aws_apigatewayv2_api.worker.id
  route_key          = "POST ${local.api_path}/account/google/link"
  target             = "integrations/${aws_apigatewayv2_integration.worker.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.google.id
}
resource "aws_apigatewayv2_route" "login" {
  api_id    = aws_apigatewayv2_api.worker.id
  route_key = "POST ${local.api_path}/login"
  target    = "integrations/${aws_apigatewayv2_integration.worker.id}"
}
resource "aws_apigatewayv2_route" "register" {
  api_id    = aws_apigatewayv2_api.worker.id
  route_key = "POST ${local.api_path}/register"
  target    = "integrations/${aws_apigatewayv2_integration.worker.id}"
}
resource "aws_apigatewayv2_route" "check_email" {
  api_id    = aws_apigatewayv2_api.worker.id
  route_key = "POST ${local.api_path}/checkEmail"
  target    = "integrations/${aws_apigatewayv2_integration.worker.id}"
}
resource "aws_apigatewayv2_route" "user_info" {
  api_id    = aws_apigatewayv2_api.worker.id
  route_key = "POST ${local.api_path}/userInfo"
  target    = "integrations/${aws_apigatewayv2_integration.worker.id}"
}
resource "aws_apigatewayv2_route" "invitation_exchange" {
  api_id    = aws_apigatewayv2_api.worker.id
  route_key = "POST ${local.api_path}/register/invitation/exchange"
  target    = "integrations/${aws_apigatewayv2_integration.worker.id}"
}

locals {
  authenticated_mutation_routes = {
    refresh                  = "POST ${local.api_path}/auth/refresh"
    logout                   = "POST ${local.api_path}/logout"
    update_account           = "PATCH ${local.api_path}/account"
    change_password          = "PATCH ${local.api_path}/account/password"
    create_group             = "POST ${local.api_path}/create_group"
    update_group             = "PUT ${local.api_path}/group/{groupid}"
    update_group_member      = "PUT ${local.api_path}/group_member"
    replace_group_members    = "PUT ${local.api_path}/group_members"
    archive_group            = "PUT ${local.api_path}/archive_group/{groupId}"
    create_expense           = "POST ${local.api_path}/create_expense"
    update_expense           = "PUT ${local.api_path}/expense/{expenseId}"
    delete_expense           = "PUT ${local.api_path}/delete_expense/{expenseId}"
    settle_expense           = "PUT ${local.api_path}/settle_expense/{groupId}"
    settle_balance           = "POST ${local.api_path}/settle_balance/{groupId}/{balanceId}"
    create_invitation        = "POST ${local.api_path}/invitations"
    update_admin_user_status = "PATCH ${local.api_path}/admin/users/{id}/status"
    update_admin_user_role   = "PATCH ${local.api_path}/admin/users/{id}/role"
    reissue_admin_invitation = "POST ${local.api_path}/admin/invitations/{id}/link"
    expire_admin_invitation  = "POST ${local.api_path}/admin/invitations/{id}/expire"
  }
}

resource "aws_apigatewayv2_route" "authenticated_mutation" {
  for_each = local.authenticated_mutation_routes

  api_id    = aws_apigatewayv2_api.worker.id
  route_key = each.value
  target    = "integrations/${aws_apigatewayv2_integration.worker.id}"
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
  default_route_settings {
    throttling_burst_limit = 40
    throttling_rate_limit  = 20
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.login.route_key
    throttling_burst_limit = 3
    throttling_rate_limit  = 1
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.register.route_key
    throttling_burst_limit = 3
    throttling_rate_limit  = 1
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.check_email.route_key
    throttling_burst_limit = 3
    throttling_rate_limit  = 1
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.user_info.route_key
    throttling_burst_limit = 3
    throttling_rate_limit  = 1
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.invitation_exchange.route_key
    throttling_burst_limit = 3
    throttling_rate_limit  = 1
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.google_exchange.route_key
    throttling_burst_limit = 3
    throttling_rate_limit  = 1
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.google_register.route_key
    throttling_burst_limit = 3
    throttling_rate_limit  = 1
  }
  route_settings {
    route_key              = aws_apigatewayv2_route.google_link.route_key
    throttling_burst_limit = 10
    throttling_rate_limit  = 5
  }
  dynamic "route_settings" {
    for_each = aws_apigatewayv2_route.authenticated_mutation
    content {
      route_key              = route_settings.value.route_key
      throttling_burst_limit = 10
      throttling_rate_limit  = 5
    }
  }
  tags = local.common_tags
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
