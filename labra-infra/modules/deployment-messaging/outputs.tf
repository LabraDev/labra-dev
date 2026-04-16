output "deploy_jobs_queue_name" {
  value = aws_sqs_queue.deploy_jobs.name
}

output "deploy_jobs_queue_url" {
  value = aws_sqs_queue.deploy_jobs.url
}

output "deploy_jobs_queue_arn" {
  value = aws_sqs_queue.deploy_jobs.arn
}

output "deploy_jobs_dlq_name" {
  value = aws_sqs_queue.deploy_jobs_dlq.name
}

output "deploy_jobs_dlq_url" {
  value = aws_sqs_queue.deploy_jobs_dlq.url
}

output "deploy_jobs_dlq_arn" {
  value = aws_sqs_queue.deploy_jobs_dlq.arn
}

output "webhook_events_queue_name" {
  value = aws_sqs_queue.webhook_events.name
}

output "webhook_events_queue_url" {
  value = aws_sqs_queue.webhook_events.url
}

output "webhook_events_queue_arn" {
  value = aws_sqs_queue.webhook_events.arn
}

output "webhook_events_dlq_name" {
  value = aws_sqs_queue.webhook_events_dlq.name
}

output "webhook_events_dlq_url" {
  value = aws_sqs_queue.webhook_events_dlq.url
}

output "webhook_events_dlq_arn" {
  value = aws_sqs_queue.webhook_events_dlq.arn
}

output "alarm_names" {
  value = compact([
    try(aws_cloudwatch_metric_alarm.deploy_jobs_backlog[0].alarm_name, null),
    try(aws_cloudwatch_metric_alarm.webhook_events_backlog[0].alarm_name, null)
  ])
}
