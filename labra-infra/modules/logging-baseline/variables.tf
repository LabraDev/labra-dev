variable "name_prefix" {
  type = string
}

variable "log_retention_days" {
  type    = number
  default = 14
}

variable "log_group_suffixes" {
  type    = list(string)
  default = ["api", "deploy-runner", "webhook", "auth"]
}

variable "tags" {
  type    = map(string)
  default = {}
}
