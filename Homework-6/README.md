# Homework 6 – Scalability & Resilience Report

---

# 1. System Overview

This project implements a product search API deployed on AWS ECS (Fargate).

## Architecture Components

- AWS ECS (Fargate)
- Application Load Balancer (ALB)
- Target Group (IP-based)
- ECS Service Auto Scaling (Target Tracking)
- Amazon CloudWatch Metrics
- Locust for load testing

Each request searches through 100 products from an in-memory dataset of 100,000 items.

Initial configuration:
- 0.25 vCPU
- 512 MB memory
- 1 ECS task

---

# 2. Part I – Baseline Deployment

The API was deployed to ECS Fargate and verified using:

curl http://<public-ip>/health

Basic load testing confirmed the service responded successfully.

---

# 3. Part II – Stress Testing & Bottleneck Identification

## Stress Tests Conducted

Six progressive stress tests were executed using Locust:

| Test | Users | Req/s | Duration |
|------|-------|-------|----------|
| 1 | 500 | 50 | 3 min |
| 2 | 700 | 70 | 3 min |
| 3 | 1000 | 50 | 3 min |
| 4 | 1000 | 70 | 3 min |
| 5 | 1000 | 100 | 5 min |
| 6 | 3000 | 200 | 7 min |

---

## Observations

As load increased:

- CPU utilization steadily increased
- Memory utilization remained relatively low
- Throughput initially increased
- Eventually throughput plateaued
- Latency increased significantly

In Test 6:

- CPU peaked at ~99.87%
- Memory remained around ~33%
- Throughput plateaued around ~2,300 RPS
- Average latency increased to ~1200ms+
- p95 exceeded 2 seconds

---

## Bottleneck Analysis

CloudWatch metrics clearly showed:

- CPU reached near 100%
- Memory remained far below limits
- Increasing load did not increase throughput

Conclusion:

The system became **CPU-bound**, not memory-bound.

The performance limitation was due to compute saturation.

---

# 4. Part III – Horizontal Scaling

## Load Balancer Configuration

An Application Load Balancer (ALB) was created:

- Listener: HTTP:80
- Target Group: IP-based, port 8080
- Health check path: /health
- Health interval: 30 seconds

The ECS service was attached to the ALB and verified via:

curl http://<ALB-DNS>/health

---

## Auto Scaling Policy

Configured Target Tracking scaling policy:

- Metric: ECSServiceAverageCPUUtilization
- Target: 70%
- Minimum tasks: 2
- Maximum tasks: 4
- Scale-in cooldown: 300 seconds
- Scale-out cooldown: 300 seconds

---

## Scale-Out Test

The same heavy stress test (3000 users, 200 req/s) was re-run.

Results:

- CPU exceeded 70%
- ECS automatically increased desired task count
- Running tasks increased from 2 → 3
- CloudWatch showed CPU per task reduced
- Throughput increased compared to single-task deployment

This demonstrates successful horizontal scaling.

---

# 5. Resilience (Fault Tolerance) Test

## Test Procedure

During sustained heavy load:

1. One running ECS task was manually stopped.
2. Observed system behavior.

---

## Observations

Immediately after stopping the task:

- ECS showed 1 task STOPPED
- Desired count remained unchanged
- A new task entered PENDING state
- Replacement task transitioned to RUNNING
- Target group marked new target healthy
- Locust continued processing requests
- Failure rate remained near 0%

The system maintained availability throughout the failure event.

---

## Resilience Conclusion

The architecture demonstrates:

- Self-healing via ECS
- Automatic task replacement
- Load redistribution via ALB
- No downtime during container failure

This confirms high availability and fault tolerance.

---

# 6. Final Conclusions

This assignment demonstrated:

1. Vertical scaling limits (CPU-bound bottleneck)
2. CloudWatch metrics used for performance diagnosis
3. Horizontal scaling via ECS Auto Scaling
4. Load balancer traffic distribution
5. Automatic recovery from task failure

The system successfully scaled under load and maintained availability during simulated failure.

This validates the effectiveness of cloud-native horizontal scaling and container orchestration.

---

# 7. Key Learnings

- CPU saturation is identifiable via CloudWatch
- Throughput plateau indicates compute bottleneck
- Target tracking scaling policies maintain stability
- Cooldown periods prevent oscillation
- ECS provides automatic self-healing
- ALB ensures traffic continuity during failure

---

# Assignment Status

- Stress testing completed
- Bottleneck identified
- Auto scaling configured
- Scale-out observed
- Resilience test validated
- Evidence captured

Homework 6 completed successfully.