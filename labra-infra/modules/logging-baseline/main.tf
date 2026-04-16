locals {
  log_groups = toset([
    for suffix in var.log_group_suffixes : "/aws/labra/${var.name_prefix}/${suffix}"
  ])
}

resource "aws_cloudwatch_log_group" "baseline" {
  for_each = local.log_groups

  name              = each.key
  retention_in_days = var.log_retention_days
  tags              = var.tags
}
