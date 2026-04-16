variable "name_prefix" {
  type = string
}

variable "repositories" {
  type = list(string)
  default = [
    "control-api",
    "deploy-orchestrator",
    "webhook-ingestor",
    "deploy-runner",
    "ai-runtime"
  ]
}

variable "scan_on_push" {
  type    = bool
  default = true
}

variable "mutable_tags" {
  type    = bool
  default = false
}

variable "max_images_per_repo" {
  type    = number
  default = 200

  validation {
    condition     = var.max_images_per_repo > 0
    error_message = "max_images_per_repo must be greater than 0."
  }
}

variable "tags" {
  type    = map(string)
  default = {}
}
