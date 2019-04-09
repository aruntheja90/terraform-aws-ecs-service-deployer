data "aws_iam_policy_document" "lambda_policy" {
  statement {
    sid    = "AllowWriteLog"
    effect = "Allow"

    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = [
      "${aws_cloudwatch_log_group.log_group.arn}/*",
    ]
  }

  statement {
    sid    = "AllowUpdateService"
    effect = "Allow"

    actions = [
      "ecs:UpdateService",
      "ecs:RegisterTaskDefinition",
      "ecs:DescribeTaskDefinition",
      "iam:PassRole",               # allow passing execution role
    ]

    resources = ["*"]
  }
}
