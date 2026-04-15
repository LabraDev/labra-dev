output "user_pool_id" {
  value = aws_cognito_user_pool.platform.id
}

output "user_pool_arn" {
  value = aws_cognito_user_pool.platform.arn
}

output "app_client_id" {
  value = aws_cognito_user_pool_client.platform.id
}

output "domain" {
  value = var.create_domain ? aws_cognito_user_pool_domain.platform[0].domain : null
}
