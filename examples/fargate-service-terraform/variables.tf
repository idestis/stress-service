variable "aws_region" {
  default     = "us-east-1"
  description = "The AWS region in which all resources will be created."
  type        = string
}

variable "cluster_name" {
  default     = "stress-demo"
  description = "The AWS ECS cluster name to be created."
  type        = string
}

variable "service_name" {
  default     = "stress-service"
  description = "The AWS region in which all resources will be created."
  type        = string
}
