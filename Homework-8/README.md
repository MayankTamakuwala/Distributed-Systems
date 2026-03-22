# Homework 8: Welcome to the Data Layer!

Shopping cart API with two database backends (MySQL and DynamoDB) deployed on AWS using ECS Fargate, with Terraform for infrastructure.

## Project Structure

```
Homework-8/
  main.go                  # API server (products + shopping carts)
  store/
    store.go               # CartStore interface
    mysql.go               # MySQL (RDS) backend
    dynamo.go              # DynamoDB backend
  schema.sql               # MySQL table definitions (reference)
  Dockerfile               # Multi-stage Go build for ECS
  go.mod / go.sum          # Go dependencies
  terraform/
    main.tf                # VPC, ALB, ECS, RDS, DynamoDB, IAM
    variables.tf           # Configurable vars (db_type, db_password, etc.)
    outputs.tf             # ALB DNS, RDS endpoint, DynamoDB table name
  perftest/
    perftest.py            # 150-operation performance test script
    requirements.txt       # Python dependencies (requests)
  results/
    Step 1/                # MySQL implementation deliverables
    Step 2/                # DynamoDB implementation deliverables
    Step 3/                # Comparison analysis deliverables
```

## API Endpoints

| Method | Path | Description | Status |
|--------|------|-------------|--------|
| GET | `/health` | Health check | 200 |
| GET | `/products/:productId` | Get product by ID | 200 / 404 |
| POST | `/products/:productId/details` | Update product details | 204 / 400 / 404 |
| POST | `/shopping-carts` | Create new cart | 201 / 400 |
| GET | `/shopping-carts/:id` | Get cart with items | 200 / 404 |
| POST | `/shopping-carts/:id/items` | Add item to cart | 204 / 400 / 404 |

## How to Run

### Backend Selection

Set `DB_TYPE` env var to switch between backends:

```bash
DB_TYPE=mysql MYSQL_DSN="user:pass@tcp(host:3306)/dbname?parseTime=true" go run .
DB_TYPE=dynamodb AWS_REGION=us-west-2 DYNAMO_TABLE=shopping_carts go run .
```

### Deploy to AWS

```bash
cd terraform
terraform init
terraform apply -var="container_image=<ecr-url>" -var="db_password=<password>"
```

### Run Performance Tests

```bash
pip install -r perftest/requirements.txt

# MySQL test
python perftest/perftest.py --base-url http://<alb-dns> --output results/Step\ 1/mysql_test_results.json

# DynamoDB test (redeploy with db_type=dynamodb first)
python perftest/perftest.py --base-url http://<alb-dns> --output results/Step\ 2/dynamodb_test_results.json
```

### Cleanup

```bash
cd terraform
terraform destroy -auto-approve -var="container_image=<ecr-url>" -var="db_password=<password>"
```

## Deliverables

### STEP I: MySQL

- Working shopping cart API with RDS MySQL backend
- Terraform config with RDS integration (db.t3.micro, private subnet, ECS-only access)
- Database schema (two tables: `shopping_carts` + `cart_items`)
- Performance test results (`mysql_test_results.json` - 150 ops, 100% success)
- CloudWatch screenshots (RDS CPU, DB connections, ECS metrics)
- Implementation notes with Week 5 vs Week 8 comparison

All files in `results/Step 1/`

### STEP II: DynamoDB

- Same shopping cart API with DynamoDB backend
- Single-table design with `cart_id` partition key, embedded items
- Performance test results (`dynamodb_test_results.json` - 150 ops, 100% success)
- CloudWatch screenshots (consumed capacity, request latency, ECS metrics)
- Implementation notes with eventual consistency observations

All files in `results/Step 2/`

### STEP III: Comparison Analysis

- `combined_results.json` merging both test datasets
- Performance comparison tables (Avg, P50, P95, P99, per-operation breakdown)
- Consistency model impact assessment
- Resource efficiency analysis
- Real-world scenario recommendations (Startup MVP, Growing Business, High-Traffic, Global Platform)
- Evidence-based architecture recommendations with polyglot strategy
- Learning reflection
- Verification screenshot

All files in `results/Step 3/`

## Key Results

| Metric | MySQL | DynamoDB |
|--------|-------|----------|
| Avg Response Time | 196.75ms | 198.33ms |
| P99 Response Time | 482.64ms | 224.16ms |
| Success Rate | 100% | 100% |

Both performed similarly on average, but DynamoDB had significantly better tail latency (P99). See `results/Step 3/STEP_III_Comparison_Report.md` for the full analysis.
