locals {
  api_domain    = trimspace(coalesce(var.api_domain_name, ""))
  api_alb_dns   = trimspace(coalesce(var.api_alb_dns_name, ""))
  api_alb_zone  = trimspace(coalesce(var.api_alb_zone_id, ""))
  frontend_name = trimspace(coalesce(var.frontend_domain_name, ""))
  frontend_dns  = trimspace(coalesce(var.frontend_distribution_domain_name, ""))

  create_api_alias      = local.api_domain != "" && local.api_alb_dns != "" && local.api_alb_zone != ""
  create_frontend_alias = local.frontend_name != "" && local.frontend_dns != ""
}

resource "aws_route53_record" "api_alias" {
  count = local.create_api_alias ? 1 : 0

  zone_id = var.hosted_zone_id
  name    = local.api_domain
  type    = "A"

  alias {
    name                   = local.api_alb_dns
    zone_id                = local.api_alb_zone
    evaluate_target_health = true
  }
}

resource "aws_route53_record" "frontend_alias" {
  count = local.create_frontend_alias ? 1 : 0

  zone_id = var.hosted_zone_id
  name    = local.frontend_name
  type    = "A"

  alias {
    name                   = local.frontend_dns
    zone_id                = var.frontend_distribution_zone_id
    evaluate_target_health = false
  }
}

resource "aws_acm_certificate" "api" {
  count = var.create_api_certificate && local.api_domain != "" ? 1 : 0

  domain_name       = local.api_domain
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = merge(var.tags, {
    Component = "api-certificate"
  })
}

resource "aws_route53_record" "api_cert_validation" {
  for_each = var.create_api_certificate && local.api_domain != "" ? {
    for dvo in aws_acm_certificate.api[0].domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  } : {}

  zone_id = var.hosted_zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 60
  records = [each.value.record]
}

resource "aws_acm_certificate_validation" "api" {
  count = var.create_api_certificate && local.api_domain != "" ? 1 : 0

  certificate_arn         = aws_acm_certificate.api[0].arn
  validation_record_fqdns = [for record in aws_route53_record.api_cert_validation : record.fqdn]
}
