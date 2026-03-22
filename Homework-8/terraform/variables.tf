variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-west-2"
}

variable "container_image" {
  description = "Container image URI for the Go application"
  type        = string
}

variable "db_type" {
  description = "Database backend: mysql or dynamodb"
  type        = string
  default     = "mysql"
}

variable "db_username" {
  description = "RDS MySQL master username"
  type        = string
  default     = "admin"
}

variable "db_password" {
  description = "RDS MySQL master password"
  type        = string
  sensitive   = true
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
  description = "CIDR blocks for private subnets (ECS + RDS)"
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
  default     = "hw8-cart"
}
