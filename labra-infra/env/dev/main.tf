terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = local.tags
  }
}

locals {
  component_suffix = var.component == "" ? "" : "-${var.component}"
  resource_prefix  = "${var.project_name}-${var.environment}${local.component_suffix}"
  tags = merge({
    Project      = var.project_name
    Environment  = var.environment
    Owner        = var.owner
    CostCenter   = var.cost_center
    ManagedBy    = "Terraform"
    Version      = var.roadmap_version
    RoadmapPhase = var.roadmap_phase
  }, var.extra_tags)
}

module "state_bootstrap" {
  count  = var.bootstrap_state_backend ? 1 : 0
  source = "../../modules/state-bootstrap"

  name_prefix       = local.resource_prefix
  state_bucket_name = var.state_bucket_name
  lock_table_name   = var.state_lock_table_name
  force_destroy     = var.state_bucket_force_destroy
  tags              = local.tags
}

module "kms_baseline" {
  count  = var.enable_foundation_modules ? 1 : 0
  source = "../../modules/kms-baseline"

  name_prefix             = local.resource_prefix
  enable_key_rotation     = var.kms_enable_key_rotation
  deletion_window_in_days = var.kms_deletion_window_in_days
  tags                    = local.tags
}

module "logging_baseline" {
  count  = var.enable_foundation_modules ? 1 : 0
  source = "../../modules/logging-baseline"

  name_prefix        = local.resource_prefix
  log_retention_days = var.logging_retention_days
  log_group_suffixes = var.logging_group_suffixes
  tags               = local.tags
}

module "vpc_baseline" {
  count  = var.enable_foundation_modules ? 1 : 0
  source = "../../modules/vpc-baseline"

  name_prefix        = local.resource_prefix
  vpc_cidr           = var.vpc_cidr
  az_count           = var.vpc_az_count
  enable_nat_gateway = var.vpc_enable_nat_gateway
  tags               = local.tags
}

module "security_groups_baseline" {
  count  = var.enable_foundation_modules ? 1 : 0
  source = "../../modules/security-groups-baseline"

  name_prefix = local.resource_prefix
  vpc_id      = module.vpc_baseline[0].vpc_id
  tags        = local.tags
}

module "iam_baseline" {
  count  = var.enable_foundation_modules ? 1 : 0
  source = "../../modules/iam-baseline"

  name_prefix              = local.resource_prefix
  enable_github_oidc_role  = var.iam_enable_github_oidc_role
  github_oidc_provider_arn = var.iam_github_oidc_provider_arn
  github_repository        = var.iam_github_repository
  tags                     = local.tags
}

module "secrets_baseline" {
  count  = var.enable_foundation_modules ? 1 : 0
  source = "../../modules/secrets-baseline"

  name_prefix               = local.resource_prefix
  create_placeholder_secret = var.secrets_create_placeholder_secret
  tags                      = local.tags
}

module "static_runtime" {
  source = "../../modules/static_runtime"

  name_prefix               = local.resource_prefix
  app_name                  = var.app_name
  build_type                = var.build_type
  region                    = var.aws_region
  bucket_name               = var.static_site_bucket_name
  default_root_object       = var.static_default_root_object
  price_class               = var.static_price_class
  enable_spa_routing        = var.static_enable_spa_routing
  force_destroy             = var.static_force_destroy
  release_prefix            = var.static_release_prefix
  release_retention_days    = var.static_release_retention_days
  noncurrent_retention_days = var.static_noncurrent_retention_days
  enable_alarms             = var.static_enable_alarms
  alarm_period_seconds      = var.static_alarm_period_seconds
  alarm_evaluation_periods  = var.static_alarm_evaluation_periods
  cf_5xx_rate_threshold     = var.static_cf_5xx_rate_threshold
  tags                      = local.tags
}
