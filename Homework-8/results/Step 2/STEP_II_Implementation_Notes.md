# STEP II - DynamoDB Implementation Notes

## DynamoDB Table Design

### Table Structure

I went with a single-table design where the cart items are embedded as a list attribute inside each cart record:

- **Table name**: `hw8-cart-shopping-carts`
- **Partition key**: `cart_id` (String)
- **Billing mode**: PAY_PER_REQUEST (on-demand, no capacity planning needed)
- **No sort key** - not needed since all our access is by cart_id

### Item Structure
```json
{
  "cart_id": "42",
  "customer_id": 42,
  "items": [
    {"product_id": 1, "quantity": 3},
    {"product_id": 2, "quantity": 1}
  ],
  "created_at": "2026-03-22T01:15:00Z",
  "updated_at": "2026-03-22T01:16:30Z"
}
```

### Why This Design

- **Single table with embedded items**: Shopping carts rarely exceed 50 items, which is well within DynamoDB's 400KB item limit. Embedding means a single `GetItem` call returns everything - no need for `Query` operations. Much simpler than the MySQL approach.
- **cart_id as partition key**: Each cart has a unique ID so this gives even distribution. No hot partitions under normal shopping load.
- **No sort key**: All our access is by cart_id directly. We don't need to query ranges of items within a cart since they're embedded.
- **PAY_PER_REQUEST billing**: No capacity planning for a homework workload. It just auto-scales.
- **No secondary indexes**: Our only access pattern is by cart_id. Customer lookup would need a GSI but the assignment doesn't require it.

### How It Compares to MySQL
| Aspect | MySQL | DynamoDB |
|--------|-------|----------|
| Item storage | Separate `cart_items` table with FK | Embedded list attribute |
| Cart retrieval | JOIN query across 2 tables | Single GetItem |
| Add item | Transaction with SELECT FOR UPDATE + upsert | GetItem + UpdateItem |
| Consistency | ACID (strong) | Eventually consistent by default |
| Schema | Fixed, requires migrations | Schemaless, flexible |

## Implementation Challenges

### Reserved Keyword Issue

This one caught me off guard. My initial UpdateExpression was:
```
SET items = :items, updated_at = :now
```
And DynamoDB threw:
```
ValidationException: Invalid UpdateExpression: Attribute name is a reserved keyword
```
Turns out `items` is a reserved word in DynamoDB. Had to use expression attribute names (`#items` mapped to `items`) to work around it. Not something that was obvious from the docs at first.

### Eventual Consistency

DynamoDB GetItem is eventually consistent by default. I was a bit worried about create-then-immediately-read scenarios, but during testing:
- No consistency issues showed up in sequential operations (create then immediately get)
- For a shopping cart use case, eventual consistency is acceptable - a brief delay in seeing a newly added item is tolerable
- You can always enable strong consistency with `ConsistentRead: true` if needed (costs 2x read capacity)

### Cart ID Generation

Unlike MySQL's AUTO_INCREMENT, DynamoDB doesn't have a built-in counter. I used an atomic counter in the Go app (`sync/atomic`). This works fine for a single ECS task, but for multi-instance deployments you'd want UUIDs or a DynamoDB atomic counter item instead.

## Performance Test Results

Same exact test as STEP I - 150 operations (50 create, 50 add, 50 get):

| Metric | Value |
|--------|-------|
| Total Operations | 150 |
| Success Rate | 100% |
| Avg Response Time | 198.33ms |
| P50 Response Time | 196.80ms |
| P95 Response Time | 209.67ms |
| P99 Response Time | 224.16ms |

### Per-Operation Breakdown
| Operation | Avg (ms) | Success |
|-----------|----------|---------|
| CREATE_CART | 199.48 | 50/50 |
| ADD_ITEMS | 199.78 | 50/50 |
| GET_CART | 195.74 | 50/50 |

## CloudWatch Monitoring

### DynamoDB Consumed Read/Write Capacity
![DynamoDB Capacity](hw8%20DynamoDB%20Consumer%20read%3Awrite%20capacity%20rate.png)
You can clearly see the spike in both ConsumedReadCapacityUnits and ConsumedWriteCapacityUnits right when the test ran (~21:15). No throttling events, which is good - PAY_PER_REQUEST handled the burst without issues.

### DynamoDB Successful Request Latency
![DynamoDB Latency](hw8%20DynamoDB%20SuccessfulRequestLatency.png)
Server-side latency for PutItem, GetItem, and UpdateItem was only about 5-7ms. So the ~195-200ms we see in the test results is mostly network latency (client -> ALB -> ECS -> DynamoDB and back). DynamoDB itself is really fast.

### ECS CPU/Memory Utilization
![ECS CPU/Memory](ECS%20CPU%3AMemory%20Utilization.png)
Same as with MySQL - minimal ECS resource usage. The app is I/O bound (waiting on DynamoDB), not CPU bound.

### ECS Service Health
![ECS Cluster](ECS%20Cluster.png)
Service running with 1 healthy task, ALB target passing health checks.

## Key Observations

- DynamoDB had **much more consistent tail latency** - P99 was 224ms vs MySQL's 482ms. That's a pretty big difference at the edges.
- Average latencies were almost identical between MySQL and DynamoDB (~197ms vs ~198ms), so the real difference only shows up in tail latency.
- The add_items operation does a read-modify-write pattern (GetItem + UpdateItem), which isn't the most efficient approach, but it performed fine for our workload. MySQL's single upsert statement is arguably cleaner here.
