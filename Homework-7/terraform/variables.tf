variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-west-2"
}

variable "container_image" {
  description = "Container image URI for the Go application"
  type        = string
}

variable "sync_payment_concurrency" {
  description = "Maximum concurrent sync payment handlers"
  type        = number
  default     = 15
}

variable "worker_count" {
  description = "Number of worker goroutines in the processor service"
  type        = number
  default     = 5
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for public subnets (ALB)"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
}

variable "private_subnet_cidrs" {
  description = "CIDR blocks for private subnets (ECS)"
  type        = list(string)
  default     = ["10.0.10.0/24", "10.0.11.0/24"]
}

variable "task_cpu" {
  description = "CPU units for ECS tasks"
  type        = string
  default     = "256"
}

variable "task_memory" {
  description = "Memory (MB) for ECS tasks"
  type        = string
  default     = "512"
}

variable "name_prefix" {
  description = "Prefix for all resource names"
  type        = string
  default     = "hw7-order-platform"
}

# keep this OFF during part 2 locust testing, otherwise SNS will fan out
# to lambda and you'll get thousands of concurrent invocations
variable "enable_lambda" {
  description = "Deploy Lambda and subscribe it to SNS. Only enable for Part III."
  type        = bool
  default     = false
}
