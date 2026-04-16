output "backend_service_role_arn" {
  value = aws_iam_role.backend_service.arn
}

output "deploy_runner_role_arn" {
  value = aws_iam_role.deploy_runner.arn
}

output "github_actions_role_arn" {
  value = var.enable_github_oidc_role ? aws_iam_role.github_actions[0].arn : null
}

output "github_oidc_provider_arn" {
  value = local.github_oidc_provider_arn_effective
}
