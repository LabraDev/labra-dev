variable "name_prefix" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}

variable "feature_enabled" {
  type    = bool
  default = true
}

variable "kill_switch_enabled" {
  type    = bool
  default = false
}

variable "create_runtime_role" {
  type    = bool
  default = true
}

variable "allowed_model_arns" {
  type    = list(string)
  default = ["*"]
}

variable "log_retention_days" {
  type    = number
  default = 30
}
