resource "aws_cloudwatch_log_group" "ai_requests" {
  name              = "/aws/labra/${var.name_prefix}/ai-requests"
  retention_in_days = var.log_retention_days

  tags = merge(var.tags, {
    Component = "ai-requests"
  })
}

resource "aws_ssm_parameter" "ai_feature_enabled" {
  name  = "/labra/${var.name_prefix}/ai/feature_enabled"
  type  = "String"
  value = var.feature_enabled ? "true" : "false"

  tags = merge(var.tags, {
    Component = "ai-feature-flag"
  })
}

resource "aws_ssm_parameter" "ai_kill_switch" {
  name  = "/labra/${var.name_prefix}/ai/kill_switch"
  type  = "String"
  value = var.kill_switch_enabled ? "true" : "false"

  tags = merge(var.tags, {
    Component = "ai-kill-switch"
  })
}

data "aws_iam_policy_document" "ai_runtime_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ai_runtime" {
  count = var.create_runtime_role ? 1 : 0

  name               = "${var.name_prefix}-ai-runtime-role"
  assume_role_policy = data.aws_iam_policy_document.ai_runtime_assume_role.json

  tags = merge(var.tags, {
    Component = "ai-runtime-role"
  })
}

data "aws_iam_policy_document" "ai_runtime_permissions" {
  statement {
    sid = "BedrockInvoke"
    actions = [
      "bedrock:InvokeModel",
      "bedrock:InvokeModelWithResponseStream"
    ]
    resources = var.allowed_model_arns
  }

  statement {
    sid = "AIRuntimeLogging"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = ["${aws_cloudwatch_log_group.ai_requests.arn}:*"]
  }

  statement {
    sid = "AIRuntimeReadFeatureFlags"
    actions = [
      "ssm:GetParameter",
      "ssm:GetParameters"
    ]
    resources = [
      aws_ssm_parameter.ai_feature_enabled.arn,
      aws_ssm_parameter.ai_kill_switch.arn
    ]
  }
}

resource "aws_iam_policy" "ai_runtime_permissions" {
  count = var.create_runtime_role ? 1 : 0

  name   = "${var.name_prefix}-ai-runtime-policy"
  policy = data.aws_iam_policy_document.ai_runtime_permissions.json

  tags = merge(var.tags, {
    Component = "ai-runtime-policy"
  })
}

resource "aws_iam_role_policy_attachment" "ai_runtime_permissions" {
  count = var.create_runtime_role ? 1 : 0

  role       = aws_iam_role.ai_runtime[0].name
  policy_arn = aws_iam_policy.ai_runtime_permissions[0].arn
}
