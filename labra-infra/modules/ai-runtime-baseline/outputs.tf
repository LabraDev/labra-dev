output "ai_requests_log_group_name" {
  value = aws_cloudwatch_log_group.ai_requests.name
}

output "ai_feature_enabled_parameter_name" {
  value = aws_ssm_parameter.ai_feature_enabled.name
}

output "ai_kill_switch_parameter_name" {
  value = aws_ssm_parameter.ai_kill_switch.name
}

output "ai_runtime_role_arn" {
  value = try(aws_iam_role.ai_runtime[0].arn, null)
}

output "allowed_model_arns" {
  value = var.allowed_model_arns
}
