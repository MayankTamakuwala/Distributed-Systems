# STEP I - MySQL Implementation Notes

## Database Schema Design

### Table Structure

I went with a two-table relational design since carts and items have a clear parent-child relationship:

1. **`shopping_carts`** - Parent table
   - `id` (INT, AUTO_INCREMENT, PK)
   - `customer_id` (INT, NOT NULL, indexed)
   - `created_at` (TIMESTAMP, auto-set)
   - `updated_at` (TIMESTAMP, auto-updated)

2. **`cart_items`** - Child table
   - `id` (INT, AUTO_INCREMENT, PK)
   - `cart_id` (INT, FK -> shopping_carts.id, CASCADE DELETE)
   - `product_id` (INT, NOT NULL)
   - `quantity` (INT, NOT NULL)
   - `added_at` (TIMESTAMP)
   - UNIQUE KEY `uk_cart_product(cart_id, product_id)`

### Why This Design

- **Two tables** keep things normalized. A single-table approach would mean denormalization and messier item updates.
- **Foreign key with CASCADE DELETE** makes sure we don't end up with orphaned cart_items if a cart gets deleted.
- **Unique key on (cart_id, product_id)** is really useful - it lets me do `INSERT ... ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)` which handles the "add same product again" case cleanly without needing application-level checks.
- **`idx_customer_id` index** supports customer purchase history queries efficiently.

### Connection Pooling Configuration

- `MaxOpenConns: 25` - should be enough for 100 concurrent sessions on db.t3.micro
- `MaxIdleConns: 10` - keeps some warm connections ready without wasting resources
- `ConnMaxLifetime: 5 minutes` - avoids stale connections to RDS

### Transaction Design

The `AddItem` operation uses a transaction with `SELECT ... FOR UPDATE` to:
1. Verify the cart exists (returns 404 if not)
2. Lock the cart row to prevent concurrent modification
3. Upsert the item
4. Touch `updated_at` timestamp
5. Commit atomically

This prevents race conditions when two requests try to modify the same cart simultaneously.

## Performance Test Results

Ran exactly 150 operations as required (50 create, 50 add items, 50 get cart):

| Metric | Value |
|--------|-------|
| Total Operations | 150 |
| Success Rate | 100% |
| Avg Response Time | 196.75ms |
| P50 Response Time | 192.23ms |
| P95 Response Time | 203.91ms |
| P99 Response Time | 482.64ms |

### Per-Operation Breakdown
| Operation | Avg (ms) | Success |
|-----------|----------|---------|
| CREATE_CART | 197.75 | 50/50 |
| ADD_ITEMS | 193.56 | 50/50 |
| GET_CART | 198.93 | 50/50 |

## CloudWatch Monitoring

### RDS CPU Utilization
![RDS CPU Utilization](hw8%20sql%20CPU%20Utilization.png)
CPU spiked to about 11% during the 150-operation test, then dropped right back to idle. The db.t3.micro instance handled it fine.

### RDS Database Connections
![RDS DB Connections](hw8%20sql%20DB%20Connections.png)
Connection pool stayed stable during testing. The MaxOpen/MaxIdle config seems to be working as expected - no connection exhaustion issues.

### ECS CPU/Memory Utilization
![ECS CPU/Memory](ECS%20CPU%3AMemory%20Utilization.png)
ECS task barely used any resources (~0.7% CPU spike), which makes sense since the Go app is super lightweight and most time is spent waiting on the database round-trip.

### ECS Service Health
![ECS Cluster](ECS%20Cluster.png)
Service running with 1 healthy task, ALB target passing health checks.

## Key Challenges

1. **Schema auto-creation**: I have the tables auto-create on startup via `CREATE TABLE IF NOT EXISTS` so I don't need to SSH into the RDS instance separately. Required some careful SQL syntax to get right.

2. **JOIN query for GetCart**: Initially I was thinking of doing two separate queries (one for the cart, one for items), but a single LEFT JOIN is cleaner and faster. Had to use `sql.NullInt64` for product_id and quantity since a cart with no items returns NULL for those columns in the JOIN result.

3. **Connection pooling tuning**: Had to find the right balance for the pool settings. Too many connections wastes resources on db.t3.micro, too few and requests queue up.

## Comparison with In-Memory Approach (Week 5)

Week 5 used an in-memory product API (no database at all) with Locust load testing at 50 concurrent users.

| Metric | Week 5 (In-Memory) | Week 8 (MySQL) | Difference |
|--------|-------------------|----------------|------------|
| Avg Response Time | 110.69ms | 196.75ms | +86.06ms (+78%) |
| P50 Response Time | 97ms | 192.23ms | +95.23ms (+98%) |
| P95 Response Time | 170ms | 203.91ms | +33.91ms (+20%) |
| P99 Response Time | 190ms | 482.64ms | +292.64ms (+154%) |
| Success Rate | 100% | 100% | Same |
| Throughput | ~116 RPS | Sequential test | N/A |

### Observations

- **Average latency went up about 78%** when adding MySQL. The ~86ms overhead is basically the network round-trip from ECS to RDS within the private subnet plus SQL query execution.
- **P99 jumped a lot** (190ms -> 483ms), probably because of occasional connection establishment overhead or transaction lock contention. This is the biggest cost of adding a database layer.
- **P95 was actually pretty close** (170ms -> 204ms), meaning under normal conditions MySQL only adds about 34ms. That's totally acceptable for most requests.
- **Success rate stayed at 100%** - MySQL integration didn't introduce any reliability issues.

### Trade-offs

The ~86ms average latency increase is the price you pay for getting:
- **Persistence** - data survives restarts and deployments
- **ACID transactions** - guaranteed consistency for cart operations
- **Concurrent access safety** - via row-level locking with `SELECT ... FOR UPDATE`
- **Query flexibility** - can do complex queries, aggregations, JOINs across related data
- **Data durability** - RDS automated backups and point-in-time recovery

For a shopping cart, this trade-off is clearly worth it. Users expect their cart to persist across sessions - losing cart data on a restart would be a terrible experience.
