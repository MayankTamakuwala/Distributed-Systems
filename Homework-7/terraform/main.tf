terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

locals {
  azs = slice(data.aws_availability_zones.available.names, 0, 2)

  common_tags = {
    Project = "homework-7"
  }
}

# --- ECR ---
resource "aws_ecr_repository" "app" {
  name         = "${var.name_prefix}-app"
  force_delete = true

  tags = local.common_tags
}
# Networking

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-vpc"
  })
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-igw"
  })
}

resource "aws_subnet" "public" {
  for_each = {
    "0" = { az = local.azs[0], cidr = var.public_subnet_cidrs[0] }
    "1" = { az = local.azs[1], cidr = var.public_subnet_cidrs[1] }
  }

  vpc_id                  = aws_vpc.main.id
  availability_zone       = each.value.az
  cidr_block              = each.value.cidr
  map_public_ip_on_launch = true

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-public-${tonumber(each.key) + 1}"
  })
}

resource "aws_subnet" "private" {
  for_each = {
    "0" = { az = local.azs[0], cidr = var.private_subnet_cidrs[0] }
    "1" = { az = local.azs[1], cidr = var.private_subnet_cidrs[1] }
  }

  vpc_id            = aws_vpc.main.id
  availability_zone = each.value.az
  cidr_block        = each.value.cidr

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-private-${tonumber(each.key) + 1}"
  })
}

resource "aws_eip" "nat" {
  domain = "vpc"

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-nat-eip"
  })
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public["0"].id

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-nat"
  })

  depends_on = [aws_internet_gateway.main]
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-public-rt"
  })
}

resource "aws_route_table_association" "public" {
  for_each = aws_subnet.public

  subnet_id      = each.value.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }

  tags = merge(local.common_tags, {
    Name = "${var.name_prefix}-private-rt"
  })
}

resource "aws_route_table_association" "private" {
  for_each = aws_subnet.private

  subnet_id      = each.value.id
  route_table_id = aws_route_table.private.id
}
# Security Groups

resource "aws_security_group" "alb" {
  name        = "${var.name_prefix}-alb-sg"
  description = "ALB security group"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

resource "aws_security_group" "ecs" {
  name        = "${var.name_prefix}-ecs-sg"
  description = "ECS task security group"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}
# ALB

resource "aws_lb" "main" {
  name               = "hw7-order-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = [for subnet in aws_subnet.public : subnet.id]

  tags = local.common_tags
}

resource "aws_lb_target_group" "receiver" {
  name        = "hw7-order-receiver"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.main.id

  health_check {
    enabled             = true
    path                = "/health"
    healthy_threshold   = 2
    unhealthy_threshold = 2
    interval            = 30
    timeout             = 5
    matcher             = "200"
  }

  tags = local.common_tags
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.main.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.receiver.arn
  }
}
# SNS + SQS

resource "aws_sns_topic" "orders" {
  name = "hw7-orders"

  tags = local.common_tags
}

resource "aws_sqs_queue" "orders" {
  name                       = "hw7-orders"
  visibility_timeout_seconds = 60
  message_retention_seconds  = 345600

  tags = local.common_tags
}

data "aws_iam_policy_document" "sqs_queue_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["sns.amazonaws.com"]
    }

    actions   = ["sqs:SendMessage"]
    resources = [aws_sqs_queue.orders.arn]

    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = [aws_sns_topic.orders.arn]
    }
  }
}

resource "aws_sqs_queue_policy" "orders" {
  queue_url = aws_sqs_queue.orders.id
  policy    = data.aws_iam_policy_document.sqs_queue_policy.json
}

resource "aws_sns_topic_subscription" "orders_sqs" {
  topic_arn = aws_sns_topic.orders.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.orders.arn
}
# CloudWatch Log Groups

resource "aws_cloudwatch_log_group" "receiver" {
  name              = "/ecs/${var.name_prefix}-receiver"
  retention_in_days = 7

  tags = local.common_tags
}

resource "aws_cloudwatch_log_group" "processor" {
  name              = "/ecs/${var.name_prefix}-processor"
  retention_in_days = 7

  tags = local.common_tags
}
# IAM Roles

data "aws_iam_policy_document" "ecs_task_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "ecs_execution" {
  name               = "${var.name_prefix}-ecs-exec-role"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume_role.json

  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "ecs_execution" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# receiver task role - sns publish
resource "aws_iam_role" "receiver_task" {
  name               = "${var.name_prefix}-receiver-role"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume_role.json

  tags = local.common_tags
}

data "aws_iam_policy_document" "receiver_task" {
  statement {
    effect    = "Allow"
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.orders.arn]
  }
}

resource "aws_iam_role_policy" "receiver_task" {
  name   = "sns-publish"
  role   = aws_iam_role.receiver_task.id
  policy = data.aws_iam_policy_document.receiver_task.json
}

# processor task role - sqs consume
resource "aws_iam_role" "processor_task" {
  name               = "${var.name_prefix}-processor-role"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume_role.json

  tags = local.common_tags
}

data "aws_iam_policy_document" "processor_task" {
  statement {
    effect = "Allow"
    actions = [
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:GetQueueUrl",
      "sqs:ReceiveMessage"
    ]
    resources = [aws_sqs_queue.orders.arn]
  }
}

resource "aws_iam_role_policy" "processor_task" {
  name   = "sqs-consume"
  role   = aws_iam_role.processor_task.id
  policy = data.aws_iam_policy_document.processor_task.json
}
# ECS Cluster

resource "aws_ecs_cluster" "main" {
  name = "${var.name_prefix}-cluster"

  tags = local.common_tags
}
# ECS Task Definitions

resource "aws_ecs_task_definition" "receiver" {
  family                   = "${var.name_prefix}-receiver"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.task_cpu
  memory                   = var.task_memory
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.receiver_task.arn

  container_definitions = jsonencode([
    {
      name      = "order-receiver"
      image     = var.container_image
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "APP_MODE", value = "receiver" },
        { name = "PORT", value = "8080" },
        { name = "AWS_REGION", value = var.aws_region },
        { name = "SNS_TOPIC_ARN", value = aws_sns_topic.orders.arn },
        { name = "SYNC_PAYMENT_CONCURRENCY", value = tostring(var.sync_payment_concurrency) }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.receiver.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ecs"
        }
      }
    }
  ])

  tags = local.common_tags
}

resource "aws_ecs_task_definition" "processor" {
  family                   = "${var.name_prefix}-processor"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.task_cpu
  memory                   = var.task_memory
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.processor_task.arn

  container_definitions = jsonencode([
    {
      name      = "order-processor"
      image     = var.container_image
      essential = true
      environment = [
        { name = "APP_MODE", value = "processor" },
        { name = "AWS_REGION", value = var.aws_region },
        { name = "SQS_QUEUE_URL", value = aws_sqs_queue.orders.id },
        { name = "WORKER_COUNT", value = tostring(var.worker_count) }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.processor.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ecs"
        }
      }
    }
  ])

  tags = local.common_tags
}
# ECS Services

resource "aws_ecs_service" "receiver" {
  name            = "order-receiver"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.receiver.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = [for subnet in aws_subnet.private : subnet.id]
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.receiver.arn
    container_name   = "order-receiver"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.http]

  tags = local.common_tags
}

resource "aws_ecs_service" "processor" {
  name            = "order-processor"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.processor.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = [for subnet in aws_subnet.private : subnet.id]
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = false
  }

  tags = local.common_tags
}
# Part III - Lambda (order processor via SNS, no SQS)
# ONLY deployed when enable_lambda = true
# Keep OFF during Part II Locust testing to avoid account deactivation!

resource "aws_cloudwatch_log_group" "lambda" {
  count             = var.enable_lambda ? 1 : 0
  name              = "/aws/lambda/${var.name_prefix}-processor"
  retention_in_days = 7

  tags = local.common_tags
}

data "aws_iam_policy_document" "lambda_assume_role" {
  count = var.enable_lambda ? 1 : 0

  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "lambda" {
  count              = var.enable_lambda ? 1 : 0
  name               = "${var.name_prefix}-lambda-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role[0].json

  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  count      = var.enable_lambda ? 1 : 0
  role       = aws_iam_role.lambda[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_function" "processor" {
  count         = var.enable_lambda ? 1 : 0
  function_name = "${var.name_prefix}-processor"
  role          = aws_iam_role.lambda[0].arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  architectures = ["x86_64"]
  memory_size   = 512
  timeout       = 30
  filename      = "${path.module}/../lambda/bootstrap.zip"

  source_code_hash = filebase64sha256("${path.module}/../lambda/bootstrap.zip")

  tags = local.common_tags

  depends_on = [aws_cloudwatch_log_group.lambda]
}

resource "aws_sns_topic_subscription" "orders_lambda" {
  count     = var.enable_lambda ? 1 : 0
  topic_arn = aws_sns_topic.orders.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.processor[0].arn
}

resource "aws_lambda_permission" "sns" {
  count         = var.enable_lambda ? 1 : 0
  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.processor[0].function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.orders.arn
}
