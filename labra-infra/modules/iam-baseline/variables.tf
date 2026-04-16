variable "name_prefix" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}

variable "enable_github_oidc_role" {
  type    = bool
  default = false
}

variable "create_github_oidc_provider" {
  type    = bool
  default = false
}

variable "github_oidc_provider_arn" {
  type    = string
  default = null
}

variable "github_oidc_client_ids" {
  type    = list(string)
  default = ["sts.amazonaws.com"]
}

variable "github_oidc_thumbprints" {
  type = list(string)
  default = [
    "6938fd4d98bab03faadb97b34396831e3780aea1"
  ]
}

variable "github_repository" {
  type    = string
  default = null
}
