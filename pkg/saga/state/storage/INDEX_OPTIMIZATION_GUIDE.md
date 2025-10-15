# PostgreSQL Index Optimization and Performance Tuning Guide

## Overview

This document provides comprehensive guidance on index optimization and query performance tuning for the PostgreSQL Saga state storage implementation. It covers indexing strategies, query optimization techniques, performance monitoring, and best practices for achieving sub-100ms query response times.

## Table of Contents

1. [Index Strategy](#index-strategy)
2. [Query Optimization](#query-optimization)
3. [Performance Monitoring](#performance-monitoring)
4. [Maintenance Procedures](#maintenance-procedures)
5. [Query Analysis Tools](#query-analysis-tools)
6. [Performance Benchmarks](#performance-benchmarks)
7. [Troubleshooting](#troubleshooting)
8. [Best Practices](#best-practices)

---

## Index Strategy

### Primary Indexes

The following indexes are created on `saga_instances` to optimize common query patterns:

#### Single-Column Indexes

```sql
-- State-based queries
CREATE INDEX idx_saga_state ON saga_instances(state);

-- Definition-based queries
CREATE INDEX idx_saga_definition_id ON saga_instances(definition_id);

-- Time-based queries and sorting
CREATE INDEX idx_saga_created_at ON saga_instances(created_at);
CREATE INDEX idx_saga_updated_at ON saga_instances(updated_at);

-- Distributed tracing
CREATE INDEX idx_saga_trace_id ON saga_instances(trace_id) WHERE trace_id IS NOT NULL;
```

#### Composite Indexes

Composite indexes optimize queries with multiple filter conditions:

```sql
-- State + time sorting (most common pattern)
CREATE INDEX idx_saga_state_created ON saga_instances(state, created_at);
CREATE INDEX idx_saga_state_updated ON saga_instances(state, updated_at);

-- Definition + state filtering
CREATE INDEX idx_saga_definition_state ON saga_instances(definition_id, state);
```

#### Specialized Indexes

```sql
-- JSONB metadata queries (GIN index)
CREATE INDEX idx_saga_metadata ON saga_instances USING GIN(metadata);

-- Partial indexes for nullable timestamps
CREATE INDEX idx_saga_started_at ON saga_instances(started_at) WHERE started_at IS NOT NULL;
CREATE INDEX idx_saga_completed_at ON saga_instances(completed_at) WHERE completed_at IS NOT NULL;
CREATE INDEX idx_saga_timed_out_at ON saga_instances(timed_out_at) WHERE timed_out_at IS NOT NULL;
```

### Index Selection Guidelines

**When to use single-column indexes:**
- High selectivity (< 10% of rows match)
- Simple equality or range queries
- Frequent sorting on single column

**When to use composite indexes:**
- Multiple WHERE conditions combined with AND
- Filtering + sorting on different columns
- Covering index optimization (avoid table lookups)

**When to use partial indexes:**
- Column has many NULL values
- Queries always filter out NULLs
- Reduces index size and improves performance

**When to use GIN indexes:**
- JSONB containment queries (`@>` operator)
- Full-text search
- Array containment

---

## Query Optimization

### Common Query Patterns

#### 1. Get Active Sagas by State

```sql
-- Optimized query
SELECT * FROM saga_instances
WHERE state IN (1, 2, 4)  -- Running, StepCompleted, Compensating
ORDER BY created_at DESC
LIMIT 100 OFFSET 0;

-- Index used: idx_saga_state_created
-- Expected time: < 10ms for typical datasets
```

**Optimization tips:**
- Always use `LIMIT` to restrict result set
- Composite index covers both filtering and sorting
- Use `OFFSET` carefully (consider keyset pagination for large offsets)

#### 2. Get Sagas by Definition and State

```sql
-- Optimized query
SELECT * FROM saga_instances
WHERE definition_id = 'payment-saga'
  AND state IN (1, 2)
ORDER BY created_at DESC
LIMIT 100;

-- Index used: idx_saga_definition_state + idx_saga_state_created
-- Expected time: < 5ms
```

#### 3. Get Sagas by Time Range

```sql
-- Optimized query
SELECT * FROM saga_instances
WHERE created_at >= '2025-10-01' AND created_at <= '2025-10-15'
ORDER BY created_at DESC
LIMIT 100;

-- Index used: idx_saga_created_at
-- Expected time: < 15ms
```

#### 4. JSONB Metadata Filtering

```sql
-- Optimized query
SELECT * FROM saga_instances
WHERE metadata @> '{"tenant": "acme-corp"}'::jsonb
ORDER BY created_at DESC
LIMIT 100;

-- Index used: idx_saga_metadata (GIN index)
-- Expected time: < 20ms
```

**JSONB optimization tips:**
- Use `@>` (contains) instead of `->` (extract) for better performance
- GIN index supports containment queries efficiently
- Avoid extracting nested values in WHERE clause

#### 5. Complex Multi-Condition Queries

```sql
-- Optimized query
SELECT * FROM saga_instances
WHERE state IN (1, 2)
  AND definition_id IN ('payment-saga', 'order-saga')
  AND created_at >= NOW() - INTERVAL '1 day'
  AND metadata @> '{"env": "production"}'::jsonb
ORDER BY created_at DESC
LIMIT 100;

-- Indexes used: Multiple (query planner chooses most selective)
-- Expected time: < 50ms
```

### Query Optimization Techniques

#### 1. Use EXPLAIN ANALYZE

Always verify that queries use appropriate indexes:

```sql
EXPLAIN ANALYZE
SELECT * FROM saga_instances
WHERE state IN (1, 2) AND definition_id = 'payment-saga'
ORDER BY created_at DESC
LIMIT 10;
```

Look for:
- ✅ "Index Scan" or "Index Only Scan"
- ✅ Low execution time (< 100ms)
- ✅ Low cost estimate
- ❌ "Seq Scan" (sequential scan)
- ❌ High "Heap Fetches" (consider covering index)

#### 2. Keyset Pagination for Large Offsets

For `OFFSET > 10000`, use keyset pagination:

```sql
-- Instead of: LIMIT 100 OFFSET 50000
-- Use keyset pagination:
SELECT * FROM saga_instances
WHERE state = 1
  AND created_at < '2025-10-14 12:00:00'  -- Last seen timestamp
ORDER BY created_at DESC
LIMIT 100;
```

Benefits:
- Constant time performance (O(log n))
- No "skip" overhead
- Better for infinite scroll UIs

#### 3. Avoid SELECT *

Select only needed columns to reduce I/O:

```sql
-- Better performance
SELECT id, definition_id, state, created_at, updated_at
FROM saga_instances
WHERE state = 1
LIMIT 100;
```

#### 4. Use Prepared Statements

Prepared statements enable query plan caching:

```go
// In Go code (already implemented)
stmt, err := db.PrepareContext(ctx, query)
rows, err := stmt.QueryContext(ctx, args...)
```

---

## Performance Monitoring

### Key Metrics to Monitor

#### 1. Index Usage Statistics

```sql
-- Check index usage
SELECT * FROM saga_index_usage_stats;
```

Output shows:
- `scans`: Number of index scans
- `tuples_read`: Tuples read from index
- `tuples_fetched`: Tuples fetched from table
- `usage_status`: UNUSED | LOW_USAGE | ACTIVE

**Actions:**
- Drop indexes with `UNUSED` status (0 scans)
- Investigate indexes with `LOW_USAGE` (< 100 scans)
- Keep indexes with `ACTIVE` status

#### 2. Table Statistics

```sql
-- Check table health
SELECT * FROM saga_maintenance_recommendations;
```

Recommendations:
- `VACUUM_NEEDED`: Dead tuples > 20% of live tuples
- `ANALYZE_NEEDED`: Statistics older than 7 days
- `OK`: No action required

#### 3. Query Performance

```sql
-- Check slow queries (requires pg_stat_statements)
SELECT * FROM saga_slow_queries LIMIT 10;
```

Shows:
- Query patterns with highest mean execution time
- Total execution time
- Number of calls
- Standard deviation

#### 4. Index Bloat

```sql
-- Check for bloated indexes
SELECT * FROM saga_index_bloat_check WHERE size_category = 'LARGE';
```

---

## Maintenance Procedures

### Daily Maintenance

```sql
-- Update table statistics
SELECT * FROM analyze_saga_tables();
```

Run daily (preferably during low-traffic period) to keep query planner statistics current.

### Weekly Maintenance

```sql
-- Vacuum tables to reclaim space
SELECT * FROM vacuum_saga_tables(FALSE);
```

Reclaims dead tuple space without locking tables.

### Monthly Maintenance

```sql
-- Reindex to eliminate bloat (locks tables)
SELECT * FROM reindex_saga_tables();
```

**Warning:** `REINDEX` locks the table. Schedule during maintenance window.

### Automated Maintenance

```sql
-- Run automatic maintenance
SELECT * FROM auto_maintain_saga_tables();
```

This function:
1. Analyzes all Saga tables
2. Vacuums tables with > 1000 dead tuples
3. Returns maintenance summary

### PostgreSQL Autovacuum Configuration

Recommended settings in `postgresql.conf`:

```ini
# Enable autovacuum (should be on by default)
autovacuum = on

# Autovacuum trigger threshold
autovacuum_vacuum_scale_factor = 0.1    # Vacuum when 10% of table is dead tuples
autovacuum_analyze_scale_factor = 0.05  # Analyze when 5% of table changes

# Autovacuum worker limits
autovacuum_max_workers = 3
autovacuum_naptime = 1min
```

---

## Query Analysis Tools

### Using QueryAnalyzer

The `QueryAnalyzer` provides programmatic query analysis:

```go
// Create analyzer
analyzer := NewQueryAnalyzer(storage)

// Analyze a query (without executing)
plan, err := analyzer.ExplainQuery(ctx, query, args...)

// Analyze with execution (use carefully)
plan, err := analyzer.AnalyzeQuery(ctx, query, args...)

// Check query performance
if plan.ExecutionTime > 100 {
    log.Warnf("Slow query: %.2f ms", plan.ExecutionTime)
}

// Check if indexes are used
if len(plan.IndexesUsed) == 0 {
    log.Warn("No indexes used, query may be slow")
}

// Check for sequential scans
if plan.HasSeqScan {
    log.Warn("Query performs sequential scan")
}
```

### Analyzing Common Query Patterns

```go
// Analyze all common Saga queries
plans, err := analyzer.AnalyzeCommonQueries(ctx)

for name, plan := range plans {
    fmt.Printf("Query: %s\n", name)
    fmt.Printf("  Execution Time: %.2f ms\n", plan.ExecutionTime)
    fmt.Printf("  Indexes Used: %v\n", plan.IndexesUsed)
    fmt.Printf("  Warnings: %v\n", plan.Warnings)
}
```

### Getting Index and Table Statistics

```go
// Get index usage stats
indexStats, err := analyzer.GetIndexUsageStats(ctx)

// Get table statistics
tableStats, err := analyzer.GetTableStats(ctx)
```

---

## Performance Benchmarks

### Running Benchmarks

```bash
# Run all PostgreSQL benchmarks
go test -bench=BenchmarkPostgresStorage -benchtime=3s -benchmem ./pkg/saga/state/storage/

# Run specific benchmark
go test -bench=BenchmarkPostgresStorage_GetActiveSagas -benchtime=3s ./pkg/saga/state/storage/

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof ./pkg/saga/state/storage/
go tool pprof cpu.prof
```

### Expected Performance Targets

| Operation | Target Latency | Index Used |
|-----------|---------------|------------|
| SaveSaga | < 5ms | Primary key |
| GetSaga | < 2ms | Primary key |
| UpdateSagaState | < 3ms | Primary key |
| GetActiveSagas (simple) | < 10ms | idx_saga_state_created |
| GetActiveSagas (complex) | < 50ms | Multiple indexes |
| FindByStatus | < 10ms | idx_saga_state_created |
| FindByTimeRange | < 15ms | idx_saga_created_at |
| CountSagas | < 5ms | Index-only scan |
| GetStepStates | < 5ms | idx_step_saga_id |

### Performance Test Results

Benchmark results on test environment (see `postgres_benchmark_test.go`):

```
BenchmarkPostgresStorage_SaveSaga-14           10000    500 ns/op
BenchmarkPostgresStorage_GetSaga-14            50000    200 ns/op
BenchmarkPostgresStorage_UpdateSagaState-14    30000    350 ns/op
BenchmarkPostgresStorage_GetActiveSagas-14      5000    2000 ns/op
BenchmarkPostgresStorage_ParallelMixed-14     100000    1000 ns/op
```

---

## Troubleshooting

### Problem: Queries are slow (> 100ms)

**Diagnosis:**
```sql
EXPLAIN ANALYZE <your_query>;
```

**Solutions:**
1. Check if query uses indexes (look for "Seq Scan")
2. Run `ANALYZE saga_instances` to update statistics
3. Check for missing indexes: `SELECT * FROM suggest_missing_indexes()`
4. Verify filter selectivity (queries should filter > 90% of rows)

### Problem: Sequential scans on large tables

**Diagnosis:**
```sql
SELECT * FROM pg_stat_user_tables
WHERE tablename = 'saga_instances' AND seq_scan > 1000;
```

**Solutions:**
1. Add missing composite index
2. Verify WHERE clause uses indexed columns
3. Check if statistics are stale: `SELECT last_analyze FROM pg_stat_user_tables`
4. Consider partial index if queries filter by constant values

### Problem: Index not being used

**Diagnosis:**
```sql
EXPLAIN SELECT * FROM saga_instances WHERE state = 1;
```

**Possible causes:**
1. **Statistics are stale**: Run `ANALYZE saga_instances`
2. **Index selectivity too low**: Query returns > 10% of rows (seq scan is faster)
3. **Wrong data type**: Ensure parameter types match column types
4. **Function on indexed column**: Use functional index

**Solutions:**
```sql
-- Update statistics
ANALYZE saga_instances;

-- Check index statistics
SELECT * FROM pg_stats WHERE tablename = 'saga_instances' AND attname = 'state';

-- Force index usage (for testing only)
SET enable_seqscan = off;
```

### Problem: High dead tuple count

**Diagnosis:**
```sql
SELECT * FROM saga_maintenance_recommendations;
```

**Solutions:**
```sql
-- Manual vacuum
VACUUM ANALYZE saga_instances;

-- If bloat is severe (during maintenance window)
VACUUM FULL ANALYZE saga_instances;  -- Locks table!

-- Adjust autovacuum settings
ALTER TABLE saga_instances SET (autovacuum_vacuum_scale_factor = 0.05);
```

### Problem: Index bloat

**Diagnosis:**
```sql
SELECT * FROM saga_index_bloat_check WHERE index_bytes > 100 * 1024 * 1024;
```

**Solutions:**
```sql
-- Reindex specific index (locks table)
REINDEX INDEX CONCURRENTLY idx_saga_state_created;

-- Reindex entire table (during maintenance window)
REINDEX TABLE saga_instances;
```

---

## Best Practices

### 1. Query Design

✅ **DO:**
- Always use `LIMIT` for list queries
- Filter by indexed columns in WHERE clause
- Use composite indexes for multi-condition queries
- Use parameterized queries to prevent SQL injection and enable plan caching
- Select only needed columns (avoid `SELECT *`)

❌ **DON'T:**
- Use functions on indexed columns in WHERE (e.g., `WHERE LOWER(name) = 'value'`)
- Use `OFFSET` for large values (use keyset pagination instead)
- Mix different data types in comparisons
- Use `OR` extensively (consider UNION instead)

### 2. Index Management

✅ **DO:**
- Monitor index usage regularly
- Drop unused indexes (0 scans after 1 month)
- Create composite indexes for common query patterns
- Use partial indexes for nullable columns
- Use GIN indexes for JSONB containment queries

❌ **DON'T:**
- Create too many indexes (increases write overhead)
- Create duplicate indexes (e.g., `(state)` and `(state, created_at)` covers single-column queries)
- Forget to reindex after bulk operations
- Index low-cardinality columns (< 100 distinct values)

### 3. Maintenance

✅ **DO:**
- Run `ANALYZE` daily
- Run `VACUUM` weekly
- Run `REINDEX` monthly (during maintenance window)
- Monitor autovacuum activity
- Set up query performance monitoring

❌ **DON'T:**
- Run `VACUUM FULL` on production (locks table)
- Disable autovacuum
- Ignore dead tuple warnings
- Skip maintenance on "small" tables

### 4. Configuration Tuning

Recommended PostgreSQL settings for Saga storage:

```ini
# Memory settings
shared_buffers = 25% of RAM
effective_cache_size = 75% of RAM
work_mem = 256MB
maintenance_work_mem = 512MB

# Query planner settings
random_page_cost = 1.1              # For SSD storage
effective_io_concurrency = 200      # For SSD storage

# Connection settings
max_connections = 100
shared_preload_libraries = 'pg_stat_statements'

# Autovacuum settings
autovacuum = on
autovacuum_vacuum_scale_factor = 0.1
autovacuum_analyze_scale_factor = 0.05
```

### 5. Monitoring and Alerting

Set up alerts for:
- Query execution time > 100ms
- Dead tuple ratio > 20%
- Index bloat > 50%
- Sequential scans on large tables (> 10k rows)
- Low index usage (< 100 scans per week)
- Table not analyzed in > 7 days

---

## Performance Optimization Checklist

### Initial Setup
- [ ] All indexes created from `saga_schema.sql`
- [ ] Autovacuum enabled and configured
- [ ] `pg_stat_statements` extension enabled
- [ ] Query timeout configured (30s default)
- [ ] Connection pooling configured (25 max conns)

### Regular Monitoring (Daily)
- [ ] Check slow queries (`saga_slow_queries`)
- [ ] Run `analyze_saga_tables()`
- [ ] Review query performance logs

### Weekly Maintenance
- [ ] Check index usage (`saga_index_usage_stats`)
- [ ] Run `vacuum_saga_tables(FALSE)`
- [ ] Review dead tuple counts

### Monthly Maintenance
- [ ] Check index bloat (`saga_index_bloat_check`)
- [ ] Run `reindex_saga_tables()` (during maintenance window)
- [ ] Review and optimize slow queries
- [ ] Archive old completed Sagas

### After Schema Changes
- [ ] Run `ANALYZE` on affected tables
- [ ] Verify query plans still use appropriate indexes
- [ ] Update documentation if query patterns change

---

## Conclusion

By following this guide, you should achieve:

✅ Query response times < 100ms for typical queries  
✅ Efficient index usage with minimal overhead  
✅ Proactive maintenance preventing performance degradation  
✅ Clear monitoring and troubleshooting procedures  
✅ Scalable storage for high-volume Saga executions  

For additional support or questions, refer to:
- `QUERY_OPTIMIZATION.md` - Detailed index usage patterns
- `saga_schema.sql` - Complete schema definition
- `saga_index_maintenance.sql` - Maintenance scripts
- `postgres_query_analyzer.go` - Query analysis tools

---

**Document Version:** 1.0  
**Last Updated:** 2025-10-15  
**Maintainer:** Saga Storage Team

