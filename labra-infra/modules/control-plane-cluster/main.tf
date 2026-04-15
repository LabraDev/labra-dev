resource "aws_ecs_cluster" "control_plane" {
  name = "${var.name_prefix}-control-plane"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = var.tags
}

resource "aws_service_discovery_private_dns_namespace" "control_plane" {
  name        = "${replace(var.name_prefix, "-", ".")}.internal"
  description = "Service discovery namespace for Labra control-plane services"
  vpc         = var.vpc_id

  tags = var.tags
}

resource "aws_cloudwatch_log_group" "services" {
  for_each = toset([
    "/aws/labra/${var.name_prefix}/control-api",
    "/aws/labra/${var.name_prefix}/deploy-orchestrator",
    "/aws/labra/${var.name_prefix}/webhook-ingestor"
  ])

  name              = each.key
  retention_in_days = 14
  tags              = var.tags
}
