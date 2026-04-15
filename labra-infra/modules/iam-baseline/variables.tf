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

variable "github_oidc_provider_arn" {
  type    = string
  default = null
}

variable "github_repository" {
  type    = string
  default = null
}
