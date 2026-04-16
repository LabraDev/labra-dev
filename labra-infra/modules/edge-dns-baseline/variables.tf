variable "name_prefix" {
  type = string
}

variable "hosted_zone_id" {
  type = string
}

variable "api_domain_name" {
  type    = string
  default = null
}

variable "api_alb_dns_name" {
  type    = string
  default = null
}

variable "api_alb_zone_id" {
  type    = string
  default = null
}

variable "create_api_certificate" {
  type    = bool
  default = false
}

variable "frontend_domain_name" {
  type    = string
  default = null
}

variable "frontend_distribution_domain_name" {
  type    = string
  default = null
}

variable "frontend_distribution_zone_id" {
  type    = string
  default = "Z2FDTNDATAQYW2"
}

variable "tags" {
  type    = map(string)
  default = {}
}
