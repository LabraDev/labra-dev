output "trail_arn" {
  value = aws_cloudtrail.trail.arn
}

output "trail_home_region" {
  value = aws_cloudtrail.trail.home_region
}

output "trail_bucket_name" {
  value = aws_s3_bucket.trail.id
}
