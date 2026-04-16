locals {
  repo_names = toset([for r in var.repositories : trimspace(lower(r)) if trimspace(r) != ""])
}

resource "aws_ecr_repository" "repos" {
  for_each = local.repo_names

  name                 = "${var.name_prefix}-${each.key}"
  image_tag_mutability = var.mutable_tags ? "MUTABLE" : "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = var.scan_on_push
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = merge(var.tags, {
    Component = each.key
  })
}

resource "aws_ecr_lifecycle_policy" "repos" {
  for_each = aws_ecr_repository.repos

  repository = each.value.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Expire untagged images after 14 days"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 14
        }
        action = {
          type = "expire"
        }
      },
      {
        rulePriority = 2
        description  = "Retain only the latest tagged images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = var.max_images_per_repo
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}
