data "aws_ssm_parameter" "ami" {
  name = var.ami_ssm_parameter
}

data "aws_iam_policy_document" "ec2_assume_role" {
  count = var.create_instance_profile ? 1 : 0

  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "metadata_host" {
  count = var.create_instance_profile ? 1 : 0

  name               = "${var.name_prefix}-metadata-host-role"
  assume_role_policy = data.aws_iam_policy_document.ec2_assume_role[0].json

  tags = merge(var.tags, {
    Component = "metadata-host-role"
  })
}

resource "aws_iam_role_policy_attachment" "metadata_host_ssm" {
  count = var.create_instance_profile && var.ssm_managed ? 1 : 0

  role       = aws_iam_role.metadata_host[0].name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "metadata_host" {
  count = var.create_instance_profile ? 1 : 0

  name = "${var.name_prefix}-metadata-host-profile"
  role = aws_iam_role.metadata_host[0].name

  tags = merge(var.tags, {
    Component = "metadata-host-profile"
  })
}

locals {
  bootstrap_user_data = var.bootstrap_sqlite ? join("\n", [
    "#!/bin/bash",
    "set -euo pipefail",
    "",
    "dnf install -y sqlite",
    "mkdir -p /opt/labra/metadata",
    "touch /opt/labra/metadata/labra.db",
    "chown ec2-user:ec2-user /opt/labra/metadata/labra.db",
  ]) : null
}

resource "aws_instance" "metadata_host" {
  ami                    = data.aws_ssm_parameter.ami.value
  instance_type          = var.instance_type
  subnet_id              = var.subnet_id
  vpc_security_group_ids = var.security_group_ids
  key_name               = var.key_name

  iam_instance_profile = var.create_instance_profile ? aws_iam_instance_profile.metadata_host[0].name : null
  user_data            = local.bootstrap_user_data

  metadata_options {
    http_endpoint = "enabled"
    http_tokens   = "required"
  }

  root_block_device {
    encrypted   = true
    volume_size = var.root_volume_size_gib
    volume_type = "gp3"
  }

  tags = merge(var.tags, {
    Name      = "${var.name_prefix}-metadata-host"
    Component = "metadata-host"
  })
}
