output "kms_key_arn" {
  value = aws_kms_key.platform.arn
}

output "kms_key_id" {
  value = aws_kms_key.platform.key_id
}
