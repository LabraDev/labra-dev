output "resource_prefix" {
  value = local.resource_prefix
}

output "tags" {
  value = local.tags
}

output "roadmap_phase" {
  value = var.roadmap_phase
}

output "roadmap_version" {
  value = var.roadmap_version
}

output "app_name" {
  value = var.app_name
}

output "build_type" {
  value = var.build_type
}

output "state_bucket_name" {
  value = try(module.state_bootstrap[0].state_bucket_name, null)
}

output "state_lock_table_name" {
  value = try(module.state_bootstrap[0].lock_table_name, null)
}

output "static_bucket_name" {
  value = module.static_runtime.bucket_name
}

output "static_distribution_id" {
  value = module.static_runtime.distribution_id
}

output "static_site_url" {
  value = module.static_runtime.site_url
}

output "static_release_prefix" {
  value = module.static_runtime.release_prefix
}

output "static_alarm_names" {
  value = module.static_runtime.alarm_names
}

output "kms_key_arn" {
  value = try(module.kms_baseline[0].kms_key_arn, null)
}

output "log_group_names" {
  value = try(module.logging_baseline[0].log_group_names, [])
}

output "vpc_id" {
  value = try(module.vpc_baseline[0].vpc_id, null)
}

output "vpc_public_subnet_ids" {
  value = try(module.vpc_baseline[0].public_subnet_ids, [])
}

output "vpc_private_subnet_ids" {
  value = try(module.vpc_baseline[0].private_subnet_ids, [])
}

output "frontend_security_group_id" {
  value = try(module.security_groups_baseline[0].frontend_security_group_id, null)
}

output "api_security_group_id" {
  value = try(module.security_groups_baseline[0].api_security_group_id, null)
}

output "internal_security_group_id" {
  value = try(module.security_groups_baseline[0].internal_security_group_id, null)
}

output "backend_service_role_arn" {
  value = try(module.iam_baseline[0].backend_service_role_arn, null)
}

output "deploy_runner_role_arn" {
  value = try(module.iam_baseline[0].deploy_runner_role_arn, null)
}

output "github_actions_role_arn" {
  value = try(module.iam_baseline[0].github_actions_role_arn, null)
}

output "platform_secret_arn" {
  value = try(module.secrets_baseline[0].platform_secret_arn, null)
}

output "cognito_user_pool_id" {
  value = try(module.cognito_baseline[0].user_pool_id, null)
}

output "cognito_app_client_id" {
  value = try(module.cognito_baseline[0].app_client_id, null)
}

output "cognito_domain" {
  value = try(module.cognito_baseline[0].domain, null)
}

output "control_plane_cluster_name" {
  value = try(module.control_plane_cluster[0].cluster_name, null)
}

output "control_plane_cluster_arn" {
  value = try(module.control_plane_cluster[0].cluster_arn, null)
}

output "control_plane_service_namespace_id" {
  value = try(module.control_plane_cluster[0].service_discovery_namespace_id, null)
}

output "control_plane_service_log_groups" {
  value = try(module.control_plane_cluster[0].service_log_group_names, [])
}

output "runner_contract" {
  value = {
    enabled                 = var.runner_enabled
    launch_type             = var.runner_launch_type
    region                  = var.aws_region
    container_image         = var.runner_container_image
    timeout_seconds         = var.runner_timeout_seconds
    ephemeral_storage_gib   = var.runner_ephemeral_storage_gib
    assign_public_ip        = var.runner_assign_public_ip
    subnet_ids              = var.runner_subnet_ids
    security_group_ids      = var.runner_security_group_ids
    task_cpu                = var.runner_task_cpu
    task_memory             = var.runner_task_memory
    log_retention_days      = var.runner_log_retention_days
    execution_role_name     = var.runner_execution_role_name
    task_role_name          = var.runner_task_role_name
    contract_schema_version = "1.0"
  }
}
