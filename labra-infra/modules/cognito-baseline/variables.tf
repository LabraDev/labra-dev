variable "name_prefix" {
  type = string
}

variable "callback_urls" {
  type    = list(string)
  default = ["http://localhost:5173/dashboard"]
}

variable "logout_urls" {
  type    = list(string)
  default = ["http://localhost:5173/login"]
}

variable "create_domain" {
  type    = bool
  default = true
}

variable "domain_prefix" {
  type    = string
  default = null
}

variable "tags" {
  type    = map(string)
  default = {}
}
