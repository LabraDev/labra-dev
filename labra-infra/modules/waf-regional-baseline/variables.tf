variable "name_prefix" {
  type = string
}

variable "associate_resource_arn" {
  type    = string
  default = null
}

variable "tags" {
  type    = map(string)
  default = {}
}
