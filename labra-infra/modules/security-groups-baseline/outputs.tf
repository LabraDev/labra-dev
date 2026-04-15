output "frontend_security_group_id" {
  value = aws_security_group.frontend.id
}

output "api_security_group_id" {
  value = aws_security_group.api.id
}

output "internal_security_group_id" {
  value = aws_security_group.internal.id
}
