variable "name_prefix" {
  type = string
}

variable "description" {
  type    = string
  default = "KMS key for Labra platform encryption at rest"
}

variable "enable_key_rotation" {
  type    = bool
  default = true
}

variable "deletion_window_in_days" {
  type    = number
  default = 7
}

variable "tags" {
  type    = map(string)
  default = {}
}
