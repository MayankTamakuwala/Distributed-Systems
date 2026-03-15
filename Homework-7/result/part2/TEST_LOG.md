# Part II Test Log

## Setup
- Region: us-west-2
- VPC: 10.0.0.0/16, public subnets 10.0.1.0/24 + 10.0.2.0/24, private 10.0.10.0/24 + 10.0.11.0/24
- ECS: 256 CPU, 512MB, Fargate
- Payment delay: 3s, sync concurrency: 15 buffered channel slots
- ALB: hw7-order-alb-672983928.us-west-2.elb.amazonaws.com

## Phase 1 - Sync Normal (5 users, 30s, spawn 1/s)
- 33 requests, 0% failures
- Avg 3168ms, median 3200ms, P99 3400ms
- 1.25 req/s
- Works fine, no contention with 5 users and 15 slots

## Phase 2 - Sync Flash (20 users, 60s, spawn 10/s)
- 285 requests, 0% failures
- Avg 3720ms, P95 4900ms, P99 5500ms
- 4.85 req/s
- Max throughput with 15 slots = 15/3 = 5 orders/sec
- Latency spikes because requests queue behind buffered channel

## Phase 3 - Async Flash (20 users, 60s, 1 worker)
- 2727 requests, 0% failures
- Avg 132ms, median 110ms, P99 520ms
- 45.70 req/s
- 9.6x more orders than sync (2727 vs 285)

## Phase 4 - Queue math
- Acceptance: ~45.7/s, worker rate: 0.33/s
- Queue growth: ~45.4 msg/s
- Backlog after 60s test: ~2727 messages
- Time to clear with 1 worker: ~136 min
- CloudWatch confirmed spike to ~5260 messages

## Phase 5 - Worker scaling

| Workers | Processing rate | Reqs  | Avg    | P99   | Backlog clear |
|---------|----------------|-------|--------|-------|---------------|
| 1       | 0.33/s         | 2727  | 132ms  | 520ms | ~136 min      |
| 5       | 1.67/s         | 2880  | 111ms  | 180ms | ~29 min       |
| 20      | 6.67/s         | 2832  | 118ms  | 220ms | ~7 min        |
| 100     | 33.33/s        | 2829  | 113ms  | 190ms | ~1.4 min      |

Min workers to prevent buildup: 47 * 3 = 141 goroutines
