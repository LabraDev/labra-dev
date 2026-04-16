variable "name_prefix" {
  type = string
}

variable "subnet_id" {
  type = string
}

variable "security_group_ids" {
  type    = list(string)
  default = []
}

variable "instance_type" {
  type    = string
  default = "t3.micro"
}

variable "root_volume_size_gib" {
  type    = number
  default = 20
}

variable "key_name" {
  type    = string
  default = null
}

variable "create_instance_profile" {
  type    = bool
  default = true
}

variable "ssm_managed" {
  type    = bool
  default = true
}

variable "ami_ssm_parameter" {
  type    = string
  default = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
}

variable "bootstrap_sqlite" {
  type    = bool
  default = true
}

variable "tags" {
  type    = map(string)
  default = {}
}
