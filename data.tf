data "aws_iam_policy_document" "cloudwatchlogs_policy" {
  statement = {
    effect = "Allow"

    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = [
      "${aws_cloudwatch_log_group.log_group.arn}/*",
    ]
  }
}
