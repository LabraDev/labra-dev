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
}

variable "aws_region" {
  description = "Customer AWS region for Terraform operations."
  type        = string
  default     = "us-west-2"
}

variable "role_name" {
  description = "Name for the IAM role Labra will assume."
  type        = string
  default     = "LabraCustomerDeployRole"
}

variable "platform_principal_arn" {
  description = "IAM role ARN in the Labra platform account trusted to assume this role."
  type        = string

  validation {
    condition     = can(regex("^arn:aws:iam::[0-9]{12}:role\\/[A-Za-z0-9+=,.@_\\/-]+$", var.platform_principal_arn))
    error_message = "platform_principal_arn must look like arn:aws:iam::<account-id>:role/<role-name>."
  }
}

variable "external_id" {
  description = "External ID issued by Labra to prevent confused-deputy attacks."
  type        = string

  validation {
    condition     = length(trimspace(var.external_id)) >= 8 && length(trimspace(var.external_id)) <= 128
    error_message = "external_id must be between 8 and 128 characters."
  }
}

variable "managed_policy_arns" {
  description = "Managed policies to attach to the customer role."
  type        = list(string)
  default     = ["arn:aws:iam::aws:policy/ReadOnlyAccess"]
}

variable "max_session_duration_seconds" {
  description = "Maximum STS session duration in seconds."
  type        = number
  default     = 3600
}

variable "permissions_boundary_arn" {
  description = "Optional permissions boundary ARN."
  type        = string
  default     = null
}

variable "tags" {
  description = "Optional tags to apply to the role."
  type        = map(string)
  default = {
    ManagedBy = "Terraform"
    Project   = "labra-customer-onboarding"
    Purpose   = "cross-account-assume-role"
  }
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "assume_role_trust" {
  statement {
    sid     = "AllowLabraPlatformAssumeRole"
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = [var.platform_principal_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "sts:ExternalId"
      values   = [var.external_id]
    }
  }
}

resource "aws_iam_role" "labra_customer_role" {
  name                 = var.role_name
  assume_role_policy   = data.aws_iam_policy_document.assume_role_trust.json
  max_session_duration = var.max_session_duration_seconds
  permissions_boundary = var.permissions_boundary_arn
  tags                 = var.tags
}

resource "aws_iam_role_policy_attachment" "managed" {
  for_each = toset(var.managed_policy_arns)

  role       = aws_iam_role.labra_customer_role.name
  policy_arn = each.value
}

output "customer_role_arn" {
  description = "ARN to submit in Labra AWS connection settings."
  value       = aws_iam_role.labra_customer_role.arn
}

output "customer_role_name" {
  description = "Role name created for Labra."
  value       = aws_iam_role.labra_customer_role.name
}

output "customer_account_id" {
  description = "AWS account ID where this role exists."
  value       = data.aws_caller_identity.current.account_id
}
