variable "service_name" {
  description = "Name of the ECS service. Note: this value will be also used to name resources"
  type        = "string"
}

variable "product_domain" {
  description = "The product domain that this service belongs to"
  type        = "string"
}

variable "environment" {
  description = "Environment where the service run"
  type        = "string"
  default     = "development"
}

variable "ecs_service" {
  description = "Name of the ECS Service"
  type        = "string"
}

variable "ecs_cluster" {
  description = "Name of the ECS Cluster where the service belong"
  type        = "string"
}

variable "ecs_taskdef" {
  description = "The family of the task definition used by the service"
  type        = "string"
}

variable "image_name" {
  description = "Docker image name used by the service"
  type        = "string"
}

variable "lambda_tags" {
  description = "Custom lambda tags"
  type        = "map"
  default     = {}
}

variable "log_tags" {
  description = "Custom log group tags"
  type        = "map"
  default     = {}
}
