output "instance_id" {
  value = aws_instance.metadata_host.id
}

output "private_ip" {
  value = aws_instance.metadata_host.private_ip
}

output "availability_zone" {
  value = aws_instance.metadata_host.availability_zone
}

output "iam_role_arn" {
  value = var.create_instance_profile ? aws_iam_role.metadata_host[0].arn : null
}
