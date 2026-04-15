locals {
  resolved_domain_prefix = coalesce(var.domain_prefix, replace(var.name_prefix, "_", "-"))
}

resource "aws_cognito_user_pool" "platform" {
  name = "${var.name_prefix}-user-pool"

  auto_verified_attributes = ["email"]

  password_policy {
    minimum_length    = 12
    require_lowercase = true
    require_uppercase = true
    require_numbers   = true
    require_symbols   = true
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  tags = var.tags
}

resource "aws_cognito_user_pool_client" "platform" {
  name         = "${var.name_prefix}-app-client"
  user_pool_id = aws_cognito_user_pool.platform.id

  explicit_auth_flows = [
    "ALLOW_USER_PASSWORD_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
    "ALLOW_USER_SRP_AUTH"
  ]

  supported_identity_providers = ["COGNITO"]

  callback_urls = var.callback_urls
  logout_urls   = var.logout_urls

  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = ["email", "openid", "profile"]

  access_token_validity  = 60
  id_token_validity      = 60
  refresh_token_validity = 30

  token_validity_units {
    access_token  = "minutes"
    id_token      = "minutes"
    refresh_token = "days"
  }
}

resource "aws_cognito_user_pool_domain" "platform" {
  count = var.create_domain ? 1 : 0

  domain       = local.resolved_domain_prefix
  user_pool_id = aws_cognito_user_pool.platform.id
}
