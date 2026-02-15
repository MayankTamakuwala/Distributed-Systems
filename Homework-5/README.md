# CS6650 – Homework 5  
# Product API + Terraform + Load Testing

---

## Overview

This project implements the **Product API** portion of the provided OpenAPI specification.  
The system was:

- Implemented using **Go + Gin**
- Containerized using **Docker**
- Deployed to **AWS ECS/ECR** using **Terraform**
- Stress tested using **Locust**
- Evaluated under baseline, high load, and stress conditions

The deployed version was tested against an AWS ALB endpoint.

---

# Project Structure

```
.
├── main.go                  # Gin-based Product API server
├── go.mod
├── Dockerfile               # Multi-stage Docker build
├── .dockerignore
├── loadtest/
│   ├── locustfile.py
│   └── requirements.txt
├── screenshots/             # All Postman, Docker, Terraform, Locust screenshots
└── README.md
```

Infrastructure (Terraform) was deployed using a fork of:

```
https://github.com/RuidiH/CS6650_2b_demo
```

---

# Running Locally (Go)

## Requirements
- Go 1.22+
- Docker (optional)
- Python 3 (for Locust testing)

## Run the server locally

```bash
go mod tidy
go run .
```

Server runs at:

```
http://localhost:8080
```

---

# Running via Docker

## Build Docker image

```bash
docker build -t product-api:local .
```

## Run container

```bash
docker run --rm -p 8080:8080 product-api:local
```

Server will be available at:

```
http://localhost:8080
```

The Docker image uses a **multi-stage build** to reduce final image size.

---

# API Endpoints Implemented

### GET `/products/{productId}`

Returns product details.

- `200 OK` – Product found
- `404 Not Found` – Product does not exist
- `400 Bad Request` – Invalid productId

---

### POST `/products/{productId}/details`

Adds or updates product details.

- `204 No Content` – Success
- `400 Bad Request` – Validation failure
- `404 Not Found` – Product does not exist

---

# Postman Examples (All Status Codes)

All screenshots are located in:

```
/screenshots
```

---

## 200 OK

GET `http://localhost:8080/products/1`

![GET 200](screenshots/GET%20200.png)

---

## 204 No Content

POST `http://localhost:8080/products/1/details`

```json
{
  "product_id": 1,
  "sku": "SKU-123",
  "manufacturer": "Acme",
  "category_id": 10,
  "weight": 50,
  "some_other_id": 77
}
```

![POST 204](screenshots/POST%20204.png)

---

## 400 Bad Request

POST with invalid body:

```json
{
  "product_id": 0,
  "sku": "",
  "manufacturer": "",
  "category_id": 0,
  "weight": -1,
  "some_other_id": 0
}
```

![POST 400](screenshots/POST%20400.png)

---

## 404 Not Found

GET `http://localhost:8080/products/999`

![GET 404](screenshots/GET%20404.png)

---

# Deploying Infrastructure (Terraform + AWS)

Infrastructure was deployed using Terraform from a fork of:

```
CS6650_2b_demo
```

## Steps to deploy on a new machine

### Clone Terraform repository

```bash
git clone <your-forked-repo-url>
cd terraform
```

### Initialize Terraform

```bash
terraform init
```

### Review plan

```bash
terraform plan
```

### Apply infrastructure

```bash
terraform apply
```

This provisions:

- ECR repository
- ECS cluster
- ECS service
- Application Load Balancer
- CloudWatch logs

After deployment, Terraform outputs the ALB endpoint.

Example:

```
http://<ALB-DNS>:8080
```

You can then send requests using Postman or curl to the deployed endpoint.

---

# Load Testing (Part IV)

Load testing was performed using **Locust** against the deployed AWS endpoint.

## Run Locust locally

```bash
pip install -r loadtest/requirements.txt
locust -f loadtest/locustfile.py --host http://<ALB-DNS>:8080
```

Open:

```
http://localhost:8089
```

---

## Load Testing Scenarios

| Test | Users | Spawn Rate | Duration |
|------|--------|------------|-----------|
| Baseline | 50 | 5/sec | 2 mins |
| Higher Load | 200 | 20/sec | 2 mins |
| Stress Test | 1000 | 100/sec | 2 mins |

All tests were run against the AWS deployed API.

---

## Performance Summary

### Baseline (50 users)
- ~117 RPS
- ~110ms average latency
- 0 failures

### Higher Load (200 users)
- ~465 RPS
- ~113ms average latency
- 0 failures

### Stress Test (1000 users)
- ~2216 RPS
- ~131ms average latency
- 0 failures

### FastHttpUser (1000 users)
- ~2274 RPS
- ~121ms average latency
- 0 failures

The system scaled linearly with increasing concurrency and maintained zero failures under stress.

---

# Architectural Design Discussion

## Scalable Backend Design

The complete OpenAPI specification includes product, cart, warehouse, and payment services. A scalable design would use a **microservices architecture**, where each service runs independently with its own database. 

Key design decisions:

- Stateless services
- Horizontal scaling via ECS auto-scaling
- Load balancing using ALB
- Caching layer (Redis/CDN) for product reads
- Message queues for asynchronous order processing
- Separate databases per service for isolation

This ensures scalability, fault tolerance, and independent deployment of services.

---

## Terraform is Declarative

Terraform is declarative, meaning infrastructure is defined by describing the desired final state rather than step-by-step instructions. Terraform determines what actions are required to reach that state.

Declarative advantages:

- Reproducibility
- Idempotency
- Infrastructure version control
- Reduced manual error
- Easy environment recreation

This differs from imperative scripts, where commands must be manually sequenced.

---

# In-Memory Storage Choice

The Product API stores data using:

```go
map[int]Product
```

Advantages:

- O(1) average lookup
- Optimized for read-heavy workloads
- Minimal infrastructure overhead

In production systems, this would be replaced by a persistent database such as PostgreSQL or DynamoDB.

---

# .gitignore & Repository Hygiene

The repository excludes:

- `.terraform/`
- `*.tfstate`
- `.env`
- `*.tfvars`
- `*.pem`
- `*.log`
- binary files

This ensures no sensitive data or large files are committed.

---
