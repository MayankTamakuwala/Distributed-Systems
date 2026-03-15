output "alb_dns_name" {
  description = "DNS name of the application load balancer"
  value       = aws_lb.main.dns_name
}

output "sns_topic_arn" {
  description = "SNS topic ARN for async orders"
  value       = aws_sns_topic.orders.arn
}

output "sqs_queue_url" {
  description = "SQS queue URL for the processor"
  value       = aws_sqs_queue.orders.id
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "ecr_repository_url" {
  description = "ECR repository URL for the app image"
  value       = aws_ecr_repository.app.repository_url
}

output "lambda_function_name" {
  description = "Lambda function name for Part III (only when enable_lambda=true)"
  value       = var.enable_lambda ? aws_lambda_function.processor[0].function_name : "not deployed"
}
