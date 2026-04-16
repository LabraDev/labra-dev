output "repository_names" {
  value = { for k, v in aws_ecr_repository.repos : k => v.name }
}

output "repository_arns" {
  value = { for k, v in aws_ecr_repository.repos : k => v.arn }
}

output "repository_urls" {
  value = { for k, v in aws_ecr_repository.repos : k => v.repository_url }
}
