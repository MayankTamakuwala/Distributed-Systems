# Part III Test Log

## Lambda setup
- Function: hw7-order-platform-processor
- Runtime: provided.al2 (Go), x86_64, 512MB, 30s timeout
- Trigger: SNS topic hw7-orders (direct, no SQS)
- Same 3s payment delay

## Test
Sent 10 orders via curl with 2s spacing. All returned 202 Accepted, all processed by Lambda.

## Cold starts
- 2 out of 10 invocations had cold starts (first request + second concurrent instance)
- Init duration: 70-75ms
- Cold billed: ~3075ms vs warm billed: ~3004ms
- Overhead: ~2.5% on 3s processing, basically negligible
- Memory: 19-20MB out of 512MB

Cold starts happen on first invocation and after ~5min idle.

## Cost
- ECS (part 2): 2 tasks always running = ~$17/month
- Lambda for 10K orders/month: $0 (free tier covers it)
- Free tier covers up to ~267K orders/month
- Lambda beats ECS cost until ~1.7M orders/month

## Trade-offs
- Lambda: zero ops overhead, auto-scaling, $0 cost, but SNS only retries 2x (no SQS durability)
- ECS+SQS: more control, message retention up to 4 days, DLQ support, but costs $17/month and needs manual scaling

For a startup under 267K orders/month, Lambda makes sense. At higher volume or if order durability is critical, use SNS -> SQS -> Lambda hybrid.
