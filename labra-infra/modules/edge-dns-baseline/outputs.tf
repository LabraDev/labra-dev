output "api_alias_fqdn" {
  value = local.create_api_alias ? aws_route53_record.api_alias[0].fqdn : null
}

output "frontend_alias_fqdn" {
  value = local.create_frontend_alias ? aws_route53_record.frontend_alias[0].fqdn : null
}

output "api_certificate_arn" {
  value = var.create_api_certificate && local.api_domain != "" ? aws_acm_certificate.api[0].arn : null
}

output "api_certificate_validated" {
  value = var.create_api_certificate && local.api_domain != "" ? aws_acm_certificate_validation.api[0].id : null
}
