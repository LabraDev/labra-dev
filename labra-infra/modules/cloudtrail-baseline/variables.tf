variable "name_prefix" {
  type = string
}

variable "trail_name" {
  type    = string
  default = null
}

variable "is_multi_region_trail" {
  type    = bool
  default = true
}

variable "include_global_service_events" {
  type    = bool
  default = true
}

variable "enable_log_file_validation" {
  type    = bool
  default = true
}

variable "kms_key_id" {
  type    = string
  default = null
}

variable "force_destroy" {
  type    = bool
  default = false
}

variable "tags" {
  type    = map(string)
  default = {}
}
