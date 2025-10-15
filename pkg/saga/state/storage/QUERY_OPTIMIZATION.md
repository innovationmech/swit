# PostgreSQL Query Optimization and Index Usage

This document describes the query optimization strategies and index usage for the PostgreSQL Saga storage implementation.

## Index Strategy

### Primary Indexes

The following indexes are created on the `saga_instances` table to optimize common query patterns:

1. **`idx_saga_state`** - Single-column index on `state`
   - Used for filtering by Saga state (running, completed, failed, etc.)
   - Example query: `WHERE state IN (1, 2, 4)`

2. **`idx_saga_definition_id`** - Single-column index on `definition_id`
   - Used for filtering by Saga definition
   - Example query: `WHERE definition_id IN ('payment-saga', 'order-saga')`

3. **`idx_saga_created_at`** - Single-column index on `created_at`
   - Used for time-based queries and sorting
   - Example query: `WHERE created_at >= $1 AND created_at <= $2`

4. **`idx_saga_updated_at`** - Single-column index on `updated_at`
   - Used for sorting by last update time
   - Example query: `ORDER BY updated_at DESC`

### Composite Indexes

Composite indexes optimize queries with multiple filter conditions:

1. **`idx_saga_state_created`** - Composite index on `(state, created_at)`
   - Optimizes queries filtering by state and sorting by creation time
   - Example query: `WHERE state IN (1, 2) ORDER BY created_at DESC`

2. **`idx_saga_state_updated`** - Composite index on `(state, updated_at)`
   - Optimizes queries filtering by state and sorting by update time
   - Example query: `WHERE state = 1 ORDER BY updated_at DESC`

3. **`idx_saga_definition_state`** - Composite index on `(definition_id, state)`
   - Optimizes queries filtering by both definition and state
   - Example query: `WHERE definition_id = 'payment-saga' AND state IN (1, 2)`

### Specialized Indexes

1. **`idx_saga_metadata`** - GIN index on `metadata` (JSONB)
   - Enables efficient JSONB containment queries
   - Example query: `WHERE metadata @> '{"tenant": "test-tenant"}'::jsonb`

2. **Partial Indexes** (defined with WHERE clauses):
   - `idx_saga_started_at` - Only indexes non-NULL `started_at` values
   - `idx_saga_completed_at` - Only indexes non-NULL `completed_at` values
   - `idx_saga_timed_out_at` - Only indexes non-NULL `timed_out_at` values
   - These reduce index size and improve performance for nullable timestamp columns

## Query Patterns and Index Usage

### 1. GetActiveSagas with State Filter

```sql
SELECT * FROM saga_instances
WHERE state IN ($1, $2, $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5
```

**Index Used**: `idx_saga_state_created` (composite index)
- PostgreSQL uses the composite index for both filtering and sorting
- Very efficient for paginated queries

### 2. GetActiveSagas with Multiple Filters

```sql
SELECT * FROM saga_instances
WHERE state IN ($1, $2)
  AND definition_id IN ($3, $4)
  AND created_at >= $5
  AND metadata @> $6::jsonb
ORDER BY created_at DESC
LIMIT $7 OFFSET $8
```

**Indexes Used**:
- `idx_saga_state_created` for state filtering and sorting
- `idx_saga_definition_id` for definition filtering (if selective)
- `idx_saga_metadata` for JSONB containment check
- PostgreSQL query planner chooses the most selective index

### 3. GetTimeoutSagas

```sql
SELECT * FROM saga_instances
WHERE (
    timed_out_at IS NOT NULL AND timed_out_at < $1
) OR (
    state IN (1, 2, 4)
    AND started_at IS NOT NULL
    AND timeout_ms > 0
    AND (started_at + (timeout_ms || ' milliseconds')::INTERVAL) < $1
)
ORDER BY created_at ASC
```

**Indexes Used**:
- `idx_saga_timed_out_at` (partial index) for explicit timeout checks
- `idx_saga_state_created` for active state filtering

### 4. FindByStatus

```sql
SELECT * FROM saga_instances
WHERE state IN ($1, $2, $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5
```

**Index Used**: `idx_saga_state_created` (composite index)
- Optimized for common "list by status" queries

### 5. FindByTimeRange

```sql
SELECT * FROM saga_instances
WHERE created_at >= $1 AND created_at <= $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
```

**Index Used**: `idx_saga_created_at`
- Efficient range scan on timestamp column

### 6. CountSagas

```sql
SELECT COUNT(*) FROM saga_instances
WHERE state IN ($1, $2) AND definition_id = $3
```

**Indexes Used**:
- `idx_saga_definition_state` (composite index) provides covering index benefit
- PostgreSQL can perform index-only scan for count queries

## Query Optimization Features

### 1. Parameterized Queries

All queries use parameterized statements (`$1`, `$2`, etc.) to:
- Prevent SQL injection attacks
- Enable prepared statement caching
- Allow PostgreSQL to reuse query plans

### 2. Sort Field Whitelisting

The `sanitizeSortField()` method uses a whitelist to prevent SQL injection:
- Only predefined fields are allowed for sorting
- Invalid fields fallback to `created_at` by default
- Prevents malicious ORDER BY clauses

### 3. Pagination

The implementation uses `LIMIT` and `OFFSET` for pagination:
```sql
LIMIT $n OFFSET $m
```

**Note**: For very large offsets (>10,000), consider using keyset pagination:
```sql
WHERE created_at > $last_seen_timestamp
ORDER BY created_at ASC
LIMIT $limit
```

### 4. JSONB Containment Queries

Metadata filtering uses the `@>` containment operator:
```sql
WHERE metadata @> '{"key": "value"}'::jsonb
```

This is much faster than extracting and comparing individual fields.

## Performance Characteristics

### Query Complexity

| Query Type | Time Complexity | Space Complexity | Index Used |
|------------|----------------|------------------|------------|
| GetActiveSagas (by state) | O(log n + k) | O(k) | idx_saga_state_created |
| GetActiveSagas (complex filter) | O(log n + k) | O(k) | Multiple indexes |
| GetTimeoutSagas | O(log n + k) | O(k) | idx_saga_timed_out_at |
| FindByStatus | O(log n + k) | O(k) | idx_saga_state_created |
| FindByTimeRange | O(log n + k) | O(k) | idx_saga_created_at |
| CountSagas | O(log n) | O(1) | Index-only scan |

Where:
- `n` = total number of Saga instances
- `k` = number of matching results

### Best Practices

1. **Use Filters**: Always provide filters when possible to reduce result set size
2. **Limit Results**: Always use `Limit` for paginated queries
3. **Index Hints**: PostgreSQL automatically chooses the best index based on statistics
4. **Analyze Queries**: Use `EXPLAIN ANALYZE` to verify index usage:

```sql
EXPLAIN ANALYZE
SELECT * FROM saga_instances
WHERE state IN (1, 2) AND definition_id = 'payment-saga'
ORDER BY created_at DESC
LIMIT 10;
```

5. **Maintain Statistics**: Run `ANALYZE saga_instances` periodically to update table statistics

## Monitoring and Tuning

### Check Index Usage

```sql
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename = 'saga_instances'
ORDER BY idx_scan DESC;
```

### Check Unused Indexes

```sql
SELECT schemaname, tablename, indexname
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND tablename = 'saga_instances';
```

### Check Table Size and Index Size

```sql
SELECT
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS indexes_size
FROM pg_tables
WHERE tablename = 'saga_instances';
```

## Recommendations for High-Volume Deployments

1. **Partitioning**: Consider partitioning the `saga_instances` table by `created_at` for time-series data
2. **Archival**: Use the `cleanup_expired_sagas()` function to archive old completed Sagas
3. **Connection Pooling**: Configure appropriate connection pool size (default: 25)
4. **Query Timeout**: Set appropriate query timeouts (default: 30s)
5. **Read Replicas**: Use read replicas for reporting queries
6. **Materialized Views**: Consider materialized views for complex aggregations

## Conclusion

The PostgreSQL storage implementation is optimized for:
- Fast filtering by state, definition, and time range
- Efficient sorting and pagination
- Flexible metadata queries using JSONB
- SQL injection prevention through parameterization and whitelisting
- Scalability through proper indexing strategies

All queries are designed to leverage existing indexes for optimal performance.

