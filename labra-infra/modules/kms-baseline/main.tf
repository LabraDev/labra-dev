resource "aws_kms_key" "platform" {
  description             = var.description
  deletion_window_in_days = var.deletion_window_in_days
  enable_key_rotation     = var.enable_key_rotation
  tags                    = var.tags
}

resource "aws_kms_alias" "platform" {
  name          = "alias/${var.name_prefix}-platform"
  target_key_id = aws_kms_key.platform.key_id
}
