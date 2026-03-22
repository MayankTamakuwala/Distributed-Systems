# STEP III - Database Comparison & Analysis

**Data Source**: All numbers come from `combined_results.json` (150 MySQL ops + 150 DynamoDB ops)

## Part 1: Performance Comparison Table

| Metric | MySQL | DynamoDB | Winner | Margin |
|--------|-------|----------|--------|--------|
| Avg Response Time (ms) | 196.75 | 198.33 | MySQL | 1.58ms |
| P50 Response Time (ms) | 192.23 | 196.80 | MySQL | 4.57ms |
| P95 Response Time (ms) | 203.91 | 209.67 | MySQL | 5.76ms |
| P99 Response Time (ms) | 482.64 | 224.16 | DynamoDB | 258.48ms |
| Success Rate (%) | 100% | 100% | Tie | 0% |
| Total Operations | 150 | 150 | | |

### Operation-Specific Breakdown

| Operation | MySQL Avg (ms) | DynamoDB Avg (ms) | Faster By |
|-----------|---------------|-------------------|-----------|
| CREATE_CART | 197.75 | 199.48 | MySQL by 1.73ms |
| ADD_ITEMS | 193.56 | 199.78 | MySQL by 6.22ms |
| GET_CART | 198.93 | 195.74 | DynamoDB by 3.19ms |

### Consistency Model Impact

**MySQL (ACID)**:
- Strong consistency on every read, guaranteed
- Transactions ensure atomic multi-table operations (cart + items)
- `SELECT ... FOR UPDATE` gives us row-level locking for concurrent modifications
- Never observed any stale reads during testing

**DynamoDB (Eventual Consistency)**:
- Eventually consistent reads by default
- During the 150-operation sequential test, I didn't observe any consistency issues at all
- For shopping carts, eventual consistency is fine - a brief delay in seeing a newly added item isn't going to break the user experience
- Strong consistency is available via `ConsistentRead: true` but costs 2x the read capacity

**Practical Impact**: In our sequential testing, eventual consistency never came up as an issue. In high-concurrency scenarios with lots of rapid read-after-write patterns, DynamoDB's eventual consistency could potentially cause brief stale reads, but for shopping carts that's an acceptable tradeoff.

## CloudWatch Evidence

### MySQL RDS Metrics
![RDS CPU Utilization](hw8%20sql%20CPU%20Utilization.png)
*RDS CPU spiked to ~11% during the MySQL test run.*

![RDS DB Connections](hw8%20sql%20DB%20Connections.png)
*Connection pool kept things stable - no connection exhaustion.*

### DynamoDB Metrics
![DynamoDB Capacity](hw8%20DynamoDB%20Consumer%20read%3Awrite%20capacity%20rate.png)
*Clear spike in consumed read/write capacity during the DynamoDB test.*

![DynamoDB Latency](hw8%20DynamoDB%20SuccessfulRequestLatency.png)
*DynamoDB server-side latency was only 5-7ms. The ~200ms end-to-end times are mostly network hops (client -> ALB -> ECS -> DynamoDB).*

### ECS Metrics
![ECS CPU/Memory](ECS%20CPU%3AMemory%20Utilization.png)
*ECS CPU/Memory stayed very low (~0.7%), confirming the app is I/O bound for both backends.*

![ECS Service](ECS%20Cluster.png)
*Service running healthy with ALB health checks passing.*

## Part 2: Resource Efficiency Analysis

### Connection Management
- **MySQL**: You have to set up connection pooling yourself (MaxOpenConns, MaxIdleConns, etc.). Each connection takes memory on both the client and RDS side. If you get the config wrong, you either waste resources or hit connection limits under load.
- **DynamoDB**: HTTP-based API, no persistent connections to manage. The AWS SDK handles everything internally. With PAY_PER_REQUEST, there's basically zero capacity planning needed.

### Scaling
- **MySQL (RDS)**: Vertical scaling - you upgrade the instance size, which means downtime. Can add read replicas for read-heavy workloads, but connection limits are still bound by instance class.
- **DynamoDB**: Scales horizontally and automatically. PAY_PER_REQUEST goes from 0 to millions of requests without you doing anything. Way less operational overhead.

### Operational Complexity
- **MySQL**: Schema migrations, backup configuration, connection pool tuning, monitoring connections and CPU utilization
- **DynamoDB**: No schema migrations (schemaless), AWS handles backups, auto-scaling is built in, just monitor consumed capacity

## Part 3: Real-World Scenario Recommendations

### Scenario A: Startup MVP
100 users/day, 1 developer, limited budget, quick launch

**Recommendation: DynamoDB**

Both databases performed almost identically in our tests (~197-198ms avg), so performance isn't the differentiator here. DynamoDB wins for an MVP because:
- PAY_PER_REQUEST means near-zero cost at 100 users/day
- No server management, no connection pool tuning to worry about
- Schemaless design lets you iterate fast without migrations
- Our test showed 100% success rate with basically zero configuration effort

### Scenario B: Growing Business
10K users/day, 5 developers, moderate budget, feature expansion

**Recommendation: MySQL**

At this scale, the growing business probably needs:
- Complex queries across related data (orders, products, customers) - MySQL's JOINs are really good for this
- ACID transactions for order processing and inventory management
- A well-defined schema helps a team of 5 developers stay on the same page
- Our test showed MySQL had slightly better average performance (196.75ms vs 198.33ms)

### Scenario C: High-Traffic Events
50K normal, 1M spike users, revenue-critical

**Recommendation: DynamoDB**

This is where DynamoDB really shines based on our data:
- P99 latency was dramatically better (224ms vs 482ms), meaning more predictable performance even under pressure
- Auto-scaling handles the 50K -> 1M spike without any manual intervention
- No risk of connection pool exhaustion during traffic spikes
- PAY_PER_REQUEST absorbs burst traffic automatically

### Scenario D: Global Platform
Millions of users, multi-region, 24/7 availability

**Recommendation: DynamoDB (with MySQL for some services)**

- DynamoDB Global Tables give you multi-region replication out of the box
- The more consistent tail latency (P99: 224ms vs 482ms) matters a lot at global scale
- No connection limit bottlenecks to worry about
- For services that need complex queries (reporting, analytics), MySQL/Aurora is still valuable as a complement

## Part 4: Evidence-Based Architecture Recommendations

### 1. Shopping Cart Winner: DynamoDB

For the shopping cart specifically, I'd go with DynamoDB:
- **P99 latency**: 224ms vs 482ms - DynamoDB is about 2x more predictable at the tail
- **Simpler operations**: Cart access is basically just key-value (get by ID, put by ID), no JOINs needed
- **Natural fit**: Shopping carts are session-based with high write volume and simple access patterns, which is exactly what DynamoDB is built for
- **Auto-scaling**: Cart traffic tends to be spiky (flash sales, holidays) and DynamoDB handles that natively

### 2. Supporting Evidence
- Response time advantage: DynamoDB P99 is 258ms faster
- Implementation complexity: DynamoDB required less code overall - no connection pooling setup, no transaction management, no schema creation logic
- Operational overhead: Basically zero for DynamoDB vs ongoing connection monitoring + schema migrations for MySQL

### 3. When to Choose MySQL Instead
- When you need **complex queries** (JOINs across carts, orders, customers, products)
- When you need **strong consistency guarantees** for financial transactions
- When your team is **more experienced with SQL** and wants to move fast
- When you need **ad-hoc reporting** capabilities

### 4. Polyglot Strategy for Complete E-commerce System

If I were building a full e-commerce platform, I'd use both:

| Service | Database | Rationale |
|---------|----------|-----------|
| Shopping Carts | DynamoDB | Simple access patterns, high write volume, spiky traffic |
| User Sessions | DynamoDB | Key-value access, TTL for auto-expiration, high throughput |
| Product Catalog | MySQL | Complex queries, category relationships, full-text search |
| Order History | MySQL | ACID transactions for financial data, complex reporting, JOINs with products |

## Part 5: Learning Reflection

### What Surprised Me

- **Performance was almost identical on average**: MySQL and DynamoDB had nearly the same average response times (~197ms vs ~198ms). I expected DynamoDB to be noticeably faster, but the real difference was in **tail latency** (P99), not averages.
- **Reserved keywords in DynamoDB**: The `items` attribute name being reserved caught me off guard and caused a runtime error. Had to use expression attribute names to work around it - not something you'd easily anticipate.
- **RDS provisioning time**: RDS MySQL took about 5 minutes to provision vs DynamoDB's near-instant table creation. Not a big deal for production, but it does affect developer workflow and CI/CD pipelines.

### What Failed Initially

- **DynamoDB `items` attribute**: The UpdateExpression failed because `items` is a reserved word. Fixed it by using `#items` as an expression attribute name. Took some debugging to figure out.
- **ECS deployment sequence**: The initial Terraform apply used an nginx placeholder image before our Go app was pushed to ECR. Health checks kept failing until we pushed the correct image and forced a new deployment.

### Key Insights

- **Choose MySQL when**: You need complex queries, strong consistency, relational integrity, or your team knows SQL well
- **Choose DynamoDB when**: You have simple access patterns, need auto-scaling, want zero operational overhead, or need predictable tail latency
- **The real differentiator** isn't average performance - it's the operational model. DynamoDB is "set and forget" while MySQL requires ongoing management (connections, scaling, backups, migrations)
- **Hands-on implementation** showed me that both databases work great for shopping carts at moderate scale. The choice depends more on your team's expertise and the broader system architecture than on raw performance numbers.
