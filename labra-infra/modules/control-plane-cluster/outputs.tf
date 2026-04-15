output "cluster_name" {
  value = aws_ecs_cluster.control_plane.name
}

output "cluster_arn" {
  value = aws_ecs_cluster.control_plane.arn
}

output "service_discovery_namespace_id" {
  value = aws_service_discovery_private_dns_namespace.control_plane.id
}

output "service_log_group_names" {
  value = [for lg in aws_cloudwatch_log_group.services : lg.name]
}
