output "alb_arn" {
  value = aws_lb.api.arn
}

output "alb_dns_name" {
  value = aws_lb.api.dns_name
}

output "alb_zone_id" {
  value = aws_lb.api.zone_id
}

output "api_listener_arn" {
  value = aws_lb_listener.api_http.arn
}

output "api_target_group_arn" {
  value = aws_lb_target_group.api.arn
}

output "service_names" {
  value = { for k, v in aws_ecs_service.service : k => v.name }
}

output "service_arns" {
  value = { for k, v in aws_ecs_service.service : k => v.id }
}

output "task_definition_arns" {
  value = { for k, v in aws_ecs_task_definition.service : k => v.arn }
}

output "task_execution_role_arn" {
  value = local.resolved_execution_role_arn
}
