locals {
  lambda_name        = "${var.service_name}-ecs-deploy"
  lambda_description = "Lambda Function to deploy ECS service ${var.service_name}"

  log_retention_days = 14

  global_tags = {
    Environment   = "${var.environment}"
    ProductDomain = "${var.product_domain}"
    Description   = "${local.lambda_description}"
    ManagedBy     = "Terraform"
  }

  lambda_tags = {
    Name = "${local.lambda_name}"
  }
}
