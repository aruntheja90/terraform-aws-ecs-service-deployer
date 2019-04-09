module "lambda_role" {
  source = "github.com/traveloka/terraform-aws-iam-role//modules/lambda?ref=v0.4.4"

  product_domain   = "${var.product_domain}"
  descriptive_name = "${local.lambda_name}"
}

resource "aws_iam_role_policy" "lambda_policy" {
  role   = "${module.lambda_role.role_name}"
  policy = "${data.aws_iam_policy_document.lambda_policy.json}"
}

module "lambda_name" {
  source = "github.com/traveloka/terraform-aws-resource-naming?ref=v0.7.1"

  name_prefix   = "${local.lambda_name}"
  resource_type = "lambda_function"
}

data "archive_file" "bundle" {
  type        = "zip"
  source_file = "${path.module}/deploy.py"
  output_path = ".terraform/tmp/lambda.zip"
}

resource "aws_lambda_function" "lambda" {
  function_name = "${module.lambda_name.name}"
  description   = "${local.lambda_description}"

  role        = "${module.lambda_role.role_arn}"
  runtime     = "python3.6"
  handler     = "deploy.handler"
  memory_size = "128"
  timeout     = "60"

  filename         = "${data.archive_file.bundle.output_path}"
  source_code_hash = "${data.archive_file.bundle.output_base64sha256}"

  tags = "${merge(local.global_tags, local.lambda_tags, var.lambda_tags)}"

  environment = {
    variables = {
      ECS_CLUSTER = "${var.ecs_cluster}"
      ECS_SERVICE = "${var.ecs_service}"
      ECS_TAKDEF  = "${var.ecs_taskdef}"
      IMAGE_NAME  = "${var.image_name}"
    }
  }
}

resource "aws_cloudwatch_log_group" "log_group" {
  name              = "/aws/lambda/${module.lambda_name.name}"
  retention_in_days = "${local.log_retention_days}"
  tags              = "${merge(local.global_tags, var.log_tags)}"
}
