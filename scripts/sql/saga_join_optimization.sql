-- Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
--
-- Saga Query Optimization - JOIN Queries and Aggregations
-- This script provides optimized views and queries for common JOIN patterns
-- and aggregations across Saga tables.

-- ============================================================================
-- Optimized Views for Common JOIN Patterns
-- ============================================================================

-- View: saga_instances_with_step_summary
-- Purpose: Provides Saga instances with aggregated step information
-- Optimization: Reduces need for separate queries to get step counts
CREATE OR REPLACE VIEW saga_instances_with_step_summary AS
SELECT 
    si.id,
    si.definition_id,
    si.name,
    si.state,
    si.current_step,
    si.total_steps,
    si.created_at,
    si.updated_at,
    si.completed_at,
    si.trace_id,
    COALESCE(step_stats.total_steps_actual, 0) as actual_step_count,
    COALESCE(step_stats.completed_steps, 0) as completed_step_count,
    COALESCE(step_stats.failed_steps, 0) as failed_step_count,
    COALESCE(step_stats.pending_steps, 0) as pending_step_count,
    CASE 
        WHEN step_stats.total_steps_actual > 0 
        THEN (step_stats.completed_steps::float / step_stats.total_steps_actual * 100)
        ELSE 0 
    END as completion_percentage
FROM saga_instances si
LEFT JOIN (
    SELECT 
        saga_id,
        COUNT(*) as total_steps_actual,
        COUNT(*) FILTER (WHERE state = 2) as completed_steps,
        COUNT(*) FILTER (WHERE state = 3) as failed_steps,
        COUNT(*) FILTER (WHERE state = 0) as pending_steps
    FROM saga_steps
    GROUP BY saga_id
) step_stats ON si.id = step_stats.saga_id;

COMMENT ON VIEW saga_instances_with_step_summary IS 'Provides Saga instances with aggregated step statistics';

-- Index to optimize the JOIN in the view
CREATE INDEX IF NOT EXISTS idx_step_saga_state_agg ON saga_steps(saga_id, state);

-- View: saga_instances_with_recent_events
-- Purpose: Provides Saga instances with their most recent events
-- Optimization: Uses LATERAL join for efficient "top N per group" query
CREATE OR REPLACE VIEW saga_instances_with_recent_events AS
SELECT 
    si.id,
    si.definition_id,
    si.name,
    si.state,
    si.created_at,
    si.updated_at,
    re.recent_events
FROM saga_instances si
LEFT JOIN LATERAL (
    SELECT json_agg(
        json_build_object(
            'event_type', event_type,
            'timestamp', timestamp,
            'step_index', step_index,
            'old_state', old_state,
            'new_state', new_state
        ) ORDER BY timestamp DESC
    ) as recent_events
    FROM (
        SELECT event_type, timestamp, step_index, old_state, new_state
        FROM saga_events
        WHERE saga_id = si.id
        ORDER BY timestamp DESC
        LIMIT 10
    ) last_events
) re ON TRUE
WHERE si.updated_at > NOW() - INTERVAL '7 days';

COMMENT ON VIEW saga_instances_with_recent_events IS 'Provides Saga instances with their 10 most recent events';

-- View: saga_step_failure_summary
-- Purpose: Aggregates step failure statistics for monitoring
-- Optimization: Pre-aggregated data for dashboard queries
CREATE OR REPLACE VIEW saga_step_failure_summary AS
SELECT 
    ss.name as step_name,
    si.definition_id,
    COUNT(*) as total_executions,
    COUNT(*) FILTER (WHERE ss.state = 3) as failed_count,
    COUNT(*) FILTER (WHERE ss.state = 2) as succeeded_count,
    ROUND(AVG(ss.attempts), 2) as avg_attempts,
    MAX(ss.attempts) as max_attempts,
    COUNT(*) FILTER (WHERE ss.state = 3)::float / NULLIF(COUNT(*), 0) * 100 as failure_rate_pct
FROM saga_steps ss
JOIN saga_instances si ON ss.saga_id = si.id
WHERE ss.created_at > NOW() - INTERVAL '30 days'
GROUP BY ss.name, si.definition_id
HAVING COUNT(*) FILTER (WHERE ss.state = 3) > 0
ORDER BY failure_rate_pct DESC;

COMMENT ON VIEW saga_step_failure_summary IS 'Aggregates step failure statistics for the last 30 days';

-- ============================================================================
-- Optimized Queries for Common JOIN Patterns
-- ============================================================================

-- Query 1: Get Saga with all step details (use sparingly, prefer separate queries)
-- Optimization: Uses array aggregation to avoid N+1 queries
CREATE OR REPLACE FUNCTION get_saga_with_steps(p_saga_id VARCHAR)
RETURNS JSON AS $$
    SELECT json_build_object(
        'saga', row_to_json(si.*),
        'steps', COALESCE(
            (
                SELECT json_agg(row_to_json(ss.*) ORDER BY ss.step_index)
                FROM saga_steps ss
                WHERE ss.saga_id = p_saga_id
            ),
            '[]'::json
        ),
        'event_count', (
            SELECT COUNT(*) FROM saga_events WHERE saga_id = p_saga_id
        )
    )
    FROM saga_instances si
    WHERE si.id = p_saga_id;
$$ LANGUAGE SQL STABLE;

COMMENT ON FUNCTION get_saga_with_steps IS 'Efficiently retrieves a Saga with all its steps in a single query';

-- Query 2: Get Sagas with failed steps
-- Optimization: Uses EXISTS subquery which is more efficient than JOIN for filtering
CREATE OR REPLACE FUNCTION get_sagas_with_failed_steps(
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE(
    saga_id VARCHAR,
    definition_id VARCHAR,
    saga_state INTEGER,
    failed_step_count BIGINT,
    last_failed_step VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        si.id,
        si.definition_id,
        si.state,
        (SELECT COUNT(*) FROM saga_steps WHERE saga_id = si.id AND state = 3) as failed_step_count,
        (SELECT name FROM saga_steps WHERE saga_id = si.id AND state = 3 ORDER BY step_index DESC LIMIT 1) as last_failed_step
    FROM saga_instances si
    WHERE EXISTS (
        SELECT 1 FROM saga_steps ss
        WHERE ss.saga_id = si.id AND ss.state = 3
    )
    ORDER BY si.updated_at DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION get_sagas_with_failed_steps IS 'Efficiently retrieves Sagas that have failed steps';

-- Query 3: Get Saga execution timeline (Saga + Steps + Events)
-- Optimization: Uses CTEs for better readability and performance
CREATE OR REPLACE FUNCTION get_saga_execution_timeline(p_saga_id VARCHAR)
RETURNS JSON AS $$
WITH saga_info AS (
    SELECT * FROM saga_instances WHERE id = p_saga_id
),
step_timeline AS (
    SELECT 
        step_index,
        name,
        state,
        attempts,
        created_at,
        started_at,
        completed_at,
        EXTRACT(EPOCH FROM (completed_at - started_at)) as duration_seconds
    FROM saga_steps
    WHERE saga_id = p_saga_id
    ORDER BY step_index
),
event_timeline AS (
    SELECT 
        timestamp,
        event_type,
        step_index,
        old_state,
        new_state
    FROM saga_events
    WHERE saga_id = p_saga_id
    ORDER BY timestamp DESC
    LIMIT 50
)
SELECT json_build_object(
    'saga', (SELECT row_to_json(s) FROM saga_info s),
    'steps', (SELECT json_agg(row_to_json(st)) FROM step_timeline st),
    'events', (SELECT json_agg(row_to_json(et)) FROM event_timeline et)
);
$$ LANGUAGE SQL STABLE;

COMMENT ON FUNCTION get_saga_execution_timeline IS 'Retrieves complete execution timeline for a Saga';

-- ============================================================================
-- Performance-Optimized Aggregation Queries
-- ============================================================================

-- Query: Saga execution statistics by definition
-- Optimization: Uses window functions to avoid multiple aggregations
CREATE OR REPLACE VIEW saga_execution_statistics AS
SELECT 
    definition_id,
    state,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration_seconds,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - started_at))) as median_duration_seconds,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - started_at))) as p95_duration_seconds,
    MIN(created_at) as first_execution,
    MAX(created_at) as last_execution
FROM saga_instances
WHERE completed_at IS NOT NULL AND started_at IS NOT NULL
GROUP BY definition_id, state;

COMMENT ON VIEW saga_execution_statistics IS 'Provides aggregated execution statistics by Saga definition and state';

-- Query: Saga throughput by time bucket
-- Optimization: Uses time-series bucketing for efficient aggregation
CREATE OR REPLACE FUNCTION get_saga_throughput(
    p_definition_id VARCHAR DEFAULT NULL,
    p_bucket_interval INTERVAL DEFAULT '1 hour',
    p_start_time TIMESTAMP DEFAULT NOW() - INTERVAL '24 hours',
    p_end_time TIMESTAMP DEFAULT NOW()
)
RETURNS TABLE(
    time_bucket TIMESTAMP,
    started_count BIGINT,
    completed_count BIGINT,
    failed_count BIGINT,
    avg_duration_seconds NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        date_trunc('hour', created_at) as time_bucket,
        COUNT(*) FILTER (WHERE state IN (1, 2, 4)) as started_count,
        COUNT(*) FILTER (WHERE state = 3) as completed_count,
        COUNT(*) FILTER (WHERE state IN (6, 7, 8)) as failed_count,
        AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) FILTER (WHERE completed_at IS NOT NULL) as avg_duration_seconds
    FROM saga_instances
    WHERE created_at >= p_start_time 
      AND created_at <= p_end_time
      AND (p_definition_id IS NULL OR definition_id = p_definition_id)
    GROUP BY time_bucket
    ORDER BY time_bucket;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION get_saga_throughput IS 'Calculates Saga throughput metrics over time';

-- ============================================================================
-- Materialized Views for Heavy Aggregations (Optional)
-- ============================================================================

-- Materialized View: Saga daily summary
-- Purpose: Pre-compute daily statistics for dashboard queries
-- Refresh: Once per day
CREATE MATERIALIZED VIEW IF NOT EXISTS saga_daily_summary AS
SELECT 
    date_trunc('day', created_at) as date,
    definition_id,
    COUNT(*) as total_sagas,
    COUNT(*) FILTER (WHERE state = 3) as completed_sagas,
    COUNT(*) FILTER (WHERE state IN (6, 7, 8)) as failed_sagas,
    AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) FILTER (WHERE completed_at IS NOT NULL) as avg_duration_seconds,
    SUM(total_steps) as total_steps_executed
FROM saga_instances
GROUP BY date, definition_id;

-- Index on materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_saga_daily_summary_date_def 
    ON saga_daily_summary(date, definition_id);

COMMENT ON MATERIALIZED VIEW saga_daily_summary IS 'Daily aggregated Saga execution statistics';

-- Function to refresh materialized view
CREATE OR REPLACE FUNCTION refresh_saga_daily_summary()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY saga_daily_summary;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_saga_daily_summary IS 'Refreshes the saga_daily_summary materialized view';

-- ============================================================================
-- Query Performance Tips for JOINs
-- ============================================================================

/*
1. Use EXPLAIN ANALYZE to verify JOIN strategies:
   
   EXPLAIN (ANALYZE, BUFFERS) 
   SELECT * FROM saga_instances_with_step_summary 
   WHERE state = 1 LIMIT 10;

2. Prefer EXISTS over JOIN for filtering:
   
   -- Good (uses semi-join)
   SELECT * FROM saga_instances si
   WHERE EXISTS (SELECT 1 FROM saga_steps ss WHERE ss.saga_id = si.id AND ss.state = 3);
   
   -- Less efficient (can produce duplicates)
   SELECT DISTINCT si.* FROM saga_instances si
   JOIN saga_steps ss ON si.id = ss.saga_id
   WHERE ss.state = 3;

3. Use LATERAL JOINs for top-N-per-group queries:
   
   SELECT si.*, recent_events.*
   FROM saga_instances si
   CROSS JOIN LATERAL (
       SELECT * FROM saga_events
       WHERE saga_id = si.id
       ORDER BY timestamp DESC
       LIMIT 5
   ) recent_events;

4. Avoid N+1 query pattern:
   
   -- Bad: Requires N+1 queries (1 for sagas, N for steps)
   SELECT * FROM saga_instances;  -- Then for each: SELECT * FROM saga_steps WHERE saga_id = ?
   
   -- Good: Use array aggregation
   SELECT si.id, array_agg(ss.name ORDER BY ss.step_index) as step_names
   FROM saga_instances si
   LEFT JOIN saga_steps ss ON si.id = ss.saga_id
   GROUP BY si.id;

5. Use appropriate JOIN types:
   
   - INNER JOIN: When you need both sides to match
   - LEFT JOIN: When you need all left-side records
   - EXISTS: When you just need to check existence
   - NOT EXISTS: More efficient than LEFT JOIN ... WHERE NULL for anti-joins

6. Index foreign keys:
   
   -- Already done in schema
   CREATE INDEX idx_step_saga_id ON saga_steps(saga_id);
   CREATE INDEX idx_event_saga_id ON saga_events(saga_id);

7. Consider denormalization for frequently accessed data:
   
   -- Store aggregated counts in saga_instances instead of JOINing
   ALTER TABLE saga_instances ADD COLUMN failed_step_count INTEGER DEFAULT 0;
   
   -- Update with trigger or in application code

8. Use CTEs for complex queries:
   
   WITH active_sagas AS (
       SELECT * FROM saga_instances WHERE state IN (1, 2, 4)
   ),
   failed_steps AS (
       SELECT saga_id, COUNT(*) as count 
       FROM saga_steps 
       WHERE state = 3 
       GROUP BY saga_id
   )
   SELECT a.*, COALESCE(f.count, 0) as failed_steps
   FROM active_sagas a
   LEFT JOIN failed_steps f ON a.id = f.saga_id;
*/

-- ============================================================================
-- Monitoring JOIN Performance
-- ============================================================================

-- View: Check for missing indexes on foreign keys
CREATE OR REPLACE VIEW saga_missing_fk_indexes AS
SELECT
    conrelid::regclass AS table_name,
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE contype = 'f'
  AND conrelid IN ('saga_steps'::regclass, 'saga_events'::regclass)
  AND NOT EXISTS (
      SELECT 1
      FROM pg_index i
      WHERE i.indrelid = conrelid
        AND array_to_string(conkey, ',') = array_to_string(i.indkey, ',')
  );

COMMENT ON VIEW saga_missing_fk_indexes IS 'Identifies foreign keys without supporting indexes';

-- ============================================================================
-- Usage Examples
-- ============================================================================

-- Example 1: Get Saga with all steps efficiently
-- SELECT * FROM get_saga_with_steps('saga-123');

-- Example 2: Get Sagas with failed steps
-- SELECT * FROM get_sagas_with_failed_steps(100, 0);

-- Example 3: Get execution timeline
-- SELECT * FROM get_saga_execution_timeline('saga-123');

-- Example 4: Get throughput statistics
-- SELECT * FROM get_saga_throughput('payment-saga', '1 hour'::interval);

-- Example 5: Query aggregated step summary
-- SELECT * FROM saga_instances_with_step_summary WHERE state = 1 LIMIT 100;

-- Example 6: Refresh daily summary (run via cron)
-- SELECT refresh_saga_daily_summary();

-- Example 7: Check for missing indexes
-- SELECT * FROM saga_missing_fk_indexes;

