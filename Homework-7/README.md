# HW7 - Order Processing Platform

Sync vs async order processing with Go, ECS Fargate, SNS/SQS, and Lambda.

## How it works

- `POST /orders/sync` - processes payment synchronously (3s delay via buffered channel)
- `POST /orders/async` - publishes to SNS, returns 202 immediately
- Processor service polls SQS and processes orders in the background

Single Go binary, two modes via `APP_MODE` env var (`receiver` or `processor`).

## Infra

Everything is in `terraform/`. Creates VPC, ALB, ECS cluster, SNS topic, SQS queue, and optionally a Lambda function (controlled by `enable_lambda` variable).

```bash
cd terraform
terraform init
terraform apply -var="container_image=<ECR_IMAGE_URI>"
```

Lambda is toggled separately to avoid accidental fan-out during load testing:
```bash
terraform apply -var="container_image=<ECR_IMAGE_URI>" -var="enable_lambda=true"
```

## Load testing

Locust tests are in `loadtest/`. Two user classes: `SyncUser` and `AsyncUser`.

```bash
cd loadtest
python3 -m venv venv && source venv/bin/activate && pip install locust

# sync normal
locust -f locustfile.py SyncUser --host http://<ALB_DNS> --users 5 --spawn-rate 1 --run-time 30s --headless

# async flash sale
locust -f locustfile.py AsyncUser --host http://<ALB_DNS> --users 20 --spawn-rate 10 --run-time 60s --headless
```

## Lambda (Part III)

Lambda function is in `lambda/`. Subscribes directly to SNS, no SQS needed. Test with curl only (not locust).

## Results

Test results and reports are in `result/`.
