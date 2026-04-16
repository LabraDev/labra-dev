variable "name_prefix" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}

variable "job_visibility_timeout_seconds" {
  type    = number
  default = 120
}

variable "job_message_retention_seconds" {
  type    = number
  default = 1209600
}

variable "job_max_receive_count" {
  type    = number
  default = 5
}

variable "webhook_visibility_timeout_seconds" {
  type    = number
  default = 120
}

variable "webhook_message_retention_seconds" {
  type    = number
  default = 345600
}

variable "webhook_max_receive_count" {
  type    = number
  default = 3
}

variable "dead_letter_message_retention_seconds" {
  type    = number
  default = 1209600
}

variable "enable_alarms" {
  type    = bool
  default = true
}

variable "alarm_visible_messages_threshold" {
  type    = number
  default = 10
}

variable "alarm_period_seconds" {
  type    = number
  default = 300
}

variable "alarm_evaluation_periods" {
  type    = number
  default = 1
}
