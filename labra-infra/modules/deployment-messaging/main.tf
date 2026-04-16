resource "aws_sqs_queue" "deploy_jobs_dlq" {
  name                      = "${var.name_prefix}-deploy-jobs-dlq"
  message_retention_seconds = var.dead_letter_message_retention_seconds

  tags = merge(var.tags, {
    Component = "deploy-jobs-dlq"
  })
}

resource "aws_sqs_queue" "deploy_jobs" {
  name                       = "${var.name_prefix}-deploy-jobs"
  visibility_timeout_seconds = var.job_visibility_timeout_seconds
  message_retention_seconds  = var.job_message_retention_seconds
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.deploy_jobs_dlq.arn
    maxReceiveCount     = var.job_max_receive_count
  })

  tags = merge(var.tags, {
    Component = "deploy-jobs"
  })
}

resource "aws_sqs_queue" "webhook_events_dlq" {
  name                      = "${var.name_prefix}-webhook-events-dlq"
  message_retention_seconds = var.dead_letter_message_retention_seconds

  tags = merge(var.tags, {
    Component = "webhook-events-dlq"
  })
}

resource "aws_sqs_queue" "webhook_events" {
  name                       = "${var.name_prefix}-webhook-events"
  visibility_timeout_seconds = var.webhook_visibility_timeout_seconds
  message_retention_seconds  = var.webhook_message_retention_seconds
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.webhook_events_dlq.arn
    maxReceiveCount     = var.webhook_max_receive_count
  })

  tags = merge(var.tags, {
    Component = "webhook-events"
  })
}

resource "aws_cloudwatch_metric_alarm" "deploy_jobs_backlog" {
  count = var.enable_alarms ? 1 : 0

  alarm_name          = "${var.name_prefix}-deploy-jobs-backlog"
  alarm_description   = "Deployment queue backlog is above threshold"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  threshold           = var.alarm_visible_messages_threshold
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = var.alarm_period_seconds
  statistic           = "Average"

  dimensions = {
    QueueName = aws_sqs_queue.deploy_jobs.name
  }

  tags = merge(var.tags, {
    Component = "deploy-jobs-alarm"
  })
}

resource "aws_cloudwatch_metric_alarm" "webhook_events_backlog" {
  count = var.enable_alarms ? 1 : 0

  alarm_name          = "${var.name_prefix}-webhook-events-backlog"
  alarm_description   = "Webhook queue backlog is above threshold"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  threshold           = var.alarm_visible_messages_threshold
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = var.alarm_period_seconds
  statistic           = "Average"

  dimensions = {
    QueueName = aws_sqs_queue.webhook_events.name
  }

  tags = merge(var.tags, {
    Component = "webhook-events-alarm"
  })
}
