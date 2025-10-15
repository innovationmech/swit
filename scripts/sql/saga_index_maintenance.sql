-- Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
--
-- Saga Index Maintenance and Optimization Script
-- This script provides maintenance operations for PostgreSQL indexes
-- to ensure optimal query performance for Saga state storage.

-- ============================================================================
-- Index Statistics and Health Check
-- ============================================================================

-- View: index_usage_stats
-- Purpose: Monitor index usage and identify unused indexes
CREATE OR REPLACE VIEW saga_index_usage_stats AS
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan as scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size,
    CASE 
        WHEN idx_scan = 0 THEN 'UNUSED'
        WHEN idx_scan < 100 THEN 'LOW_USAGE'
        ELSE 'ACTIVE'
    END as usage_status
FROM pg_stat_user_indexes
WHERE schemaname = CURRENT_SCHEMA()
  AND tablename IN ('saga_instances', 'saga_steps', 'saga_events')
ORDER BY idx_scan DESC;

COMMENT ON VIEW saga_index_usage_stats IS 'Provides index usage statistics for Saga tables';

-- ============================================================================
-- Index Bloat Detection
-- ============================================================================

-- View: index_bloat_check
-- Purpose: Identify indexes that may need rebuilding due to bloat
CREATE OR REPLACE VIEW saga_index_bloat_check AS
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size,
    pg_relation_size(indexrelid) as index_bytes,
    CASE 
        WHEN pg_relation_size(indexrelid) > 100 * 1024 * 1024 THEN 'LARGE'
        WHEN pg_relation_size(indexrelid) > 10 * 1024 * 1024 THEN 'MEDIUM'
        ELSE 'SMALL'
    END as size_category
FROM pg_stat_user_indexes
WHERE schemaname = CURRENT_SCHEMA()
  AND tablename IN ('saga_instances', 'saga_steps', 'saga_events')
ORDER BY pg_relation_size(indexrelid) DESC;

COMMENT ON VIEW saga_index_bloat_check IS 'Helps identify potentially bloated indexes';

-- ============================================================================
-- Table and Index Size Analysis
-- ============================================================================

-- View: saga_storage_size
-- Purpose: Analyze storage usage across Saga tables
CREATE OR REPLACE VIEW saga_storage_size AS
SELECT
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - 
                   pg_relation_size(schemaname||'.'||tablename)) AS indexes_size,
    pg_total_relation_size(schemaname||'.'||tablename) as total_bytes,
    (pg_total_relation_size(schemaname||'.'||tablename) - 
     pg_relation_size(schemaname||'.'||tablename))::float / 
     NULLIF(pg_total_relation_size(schemaname||'.'||tablename), 0) * 100 AS index_ratio_pct
FROM pg_tables
WHERE schemaname = CURRENT_SCHEMA()
  AND tablename IN ('saga_instances', 'saga_steps', 'saga_events')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

COMMENT ON VIEW saga_storage_size IS 'Provides storage size analysis for Saga tables and indexes';

-- ============================================================================
-- Index Maintenance Functions
-- ============================================================================

-- Function: reindex_saga_tables
-- Purpose: Rebuild all indexes on Saga tables to eliminate bloat
CREATE OR REPLACE FUNCTION reindex_saga_tables()
RETURNS TABLE(
    table_name TEXT,
    status TEXT,
    duration INTERVAL
) AS $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
BEGIN
    -- Reindex saga_instances
    start_time := clock_timestamp();
    REINDEX TABLE saga_instances;
    end_time := clock_timestamp();
    table_name := 'saga_instances';
    status := 'SUCCESS';
    duration := end_time - start_time;
    RETURN NEXT;
    
    -- Reindex saga_steps
    start_time := clock_timestamp();
    REINDEX TABLE saga_steps;
    end_time := clock_timestamp();
    table_name := 'saga_steps';
    status := 'SUCCESS';
    duration := end_time - start_time;
    RETURN NEXT;
    
    -- Reindex saga_events
    start_time := clock_timestamp();
    REINDEX TABLE saga_events;
    end_time := clock_timestamp();
    table_name := 'saga_events';
    status := 'SUCCESS';
    duration := end_time - start_time;
    RETURN NEXT;
    
EXCEPTION
    WHEN OTHERS THEN
        table_name := SQLERRM;
        status := 'ERROR';
        duration := interval '0 seconds';
        RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION reindex_saga_tables IS 'Rebuilds all indexes on Saga tables (can be slow on large tables)';

-- Function: analyze_saga_tables
-- Purpose: Update table statistics for optimal query planning
CREATE OR REPLACE FUNCTION analyze_saga_tables()
RETURNS TABLE(
    table_name TEXT,
    rows_affected BIGINT,
    status TEXT
) AS $$
BEGIN
    -- Analyze saga_instances
    ANALYZE saga_instances;
    table_name := 'saga_instances';
    SELECT reltuples::BIGINT INTO rows_affected FROM pg_class WHERE relname = 'saga_instances';
    status := 'SUCCESS';
    RETURN NEXT;
    
    -- Analyze saga_steps
    ANALYZE saga_steps;
    table_name := 'saga_steps';
    SELECT reltuples::BIGINT INTO rows_affected FROM pg_class WHERE relname = 'saga_steps';
    status := 'SUCCESS';
    RETURN NEXT;
    
    -- Analyze saga_events
    ANALYZE saga_events;
    table_name := 'saga_events';
    SELECT reltuples::BIGINT INTO rows_affected FROM pg_class WHERE relname = 'saga_events';
    status := 'SUCCESS';
    RETURN NEXT;
    
EXCEPTION
    WHEN OTHERS THEN
        table_name := SQLERRM;
        rows_affected := 0;
        status := 'ERROR';
        RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION analyze_saga_tables IS 'Updates statistics for Saga tables to improve query planning';

-- Function: vacuum_saga_tables
-- Purpose: Reclaim storage and update statistics
CREATE OR REPLACE FUNCTION vacuum_saga_tables(full_vacuum BOOLEAN DEFAULT FALSE)
RETURNS TABLE(
    table_name TEXT,
    dead_tuples_before BIGINT,
    status TEXT
) AS $$
DECLARE
    dead_tuples BIGINT;
BEGIN
    -- Get dead tuples count before vacuum
    SELECT n_dead_tup INTO dead_tuples 
    FROM pg_stat_user_tables 
    WHERE relname = 'saga_instances';
    
    IF full_vacuum THEN
        VACUUM FULL ANALYZE saga_instances;
    ELSE
        VACUUM ANALYZE saga_instances;
    END IF;
    
    table_name := 'saga_instances';
    dead_tuples_before := dead_tuples;
    status := 'SUCCESS';
    RETURN NEXT;
    
    -- saga_steps
    SELECT n_dead_tup INTO dead_tuples 
    FROM pg_stat_user_tables 
    WHERE relname = 'saga_steps';
    
    IF full_vacuum THEN
        VACUUM FULL ANALYZE saga_steps;
    ELSE
        VACUUM ANALYZE saga_steps;
    END IF;
    
    table_name := 'saga_steps';
    dead_tuples_before := dead_tuples;
    status := 'SUCCESS';
    RETURN NEXT;
    
    -- saga_events
    SELECT n_dead_tup INTO dead_tuples 
    FROM pg_stat_user_tables 
    WHERE relname = 'saga_events';
    
    IF full_vacuum THEN
        VACUUM FULL ANALYZE saga_events;
    ELSE
        VACUUM ANALYZE saga_events;
    END IF;
    
    table_name := 'saga_events';
    dead_tuples_before := dead_tuples;
    status := 'SUCCESS';
    RETURN NEXT;
    
EXCEPTION
    WHEN OTHERS THEN
        table_name := SQLERRM;
        dead_tuples_before := 0;
        status := 'ERROR';
        RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION vacuum_saga_tables IS 'Performs VACUUM on Saga tables to reclaim storage';

-- ============================================================================
-- Query Performance Analysis
-- ============================================================================

-- View: slow_queries_saga
-- Purpose: Identify slow queries on Saga tables (requires pg_stat_statements)
CREATE OR REPLACE VIEW saga_slow_queries AS
SELECT
    substring(query, 1, 100) as query_snippet,
    calls,
    total_exec_time / 1000.0 as total_time_sec,
    mean_exec_time as mean_time_ms,
    max_exec_time as max_time_ms,
    stddev_exec_time as stddev_time_ms,
    rows
FROM pg_stat_statements
WHERE query LIKE '%saga_instances%'
   OR query LIKE '%saga_steps%'
   OR query LIKE '%saga_events%'
ORDER BY mean_exec_time DESC
LIMIT 20;

COMMENT ON VIEW saga_slow_queries IS 'Shows slowest queries on Saga tables (requires pg_stat_statements extension)';

-- ============================================================================
-- Index Recommendation Functions
-- ============================================================================

-- Function: suggest_missing_indexes
-- Purpose: Analyze query patterns and suggest potentially missing indexes
CREATE OR REPLACE FUNCTION suggest_missing_indexes()
RETURNS TABLE(
    table_name TEXT,
    suggested_index TEXT,
    reason TEXT
) AS $$
BEGIN
    -- Check for sequential scans on large tables
    FOR table_name, suggested_index, reason IN
        SELECT 
            t.relname::TEXT as table_name,
            'CREATE INDEX idx_' || t.relname || '_missing ON ' || t.relname || ' (state, created_at DESC)' as suggested_index,
            'High sequential scan count: ' || s.seq_scan || ', might benefit from composite index' as reason
        FROM pg_stat_user_tables s
        JOIN pg_class t ON s.relid = t.oid
        WHERE s.schemaname = CURRENT_SCHEMA()
          AND t.relname IN ('saga_instances', 'saga_steps', 'saga_events')
          AND s.seq_scan > 1000
          AND s.idx_scan < s.seq_scan
    LOOP
        RETURN NEXT;
    END LOOP;
    
    RETURN;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION suggest_missing_indexes IS 'Suggests potentially missing indexes based on query patterns';

-- ============================================================================
-- Maintenance Schedule Recommendations
-- ============================================================================

-- View: maintenance_recommendations
-- Purpose: Provide maintenance recommendations based on table statistics
CREATE OR REPLACE VIEW saga_maintenance_recommendations AS
SELECT
    tablename,
    n_live_tup as live_tuples,
    n_dead_tup as dead_tuples,
    CASE 
        WHEN n_dead_tup > n_live_tup * 0.2 THEN 'VACUUM_NEEDED'
        WHEN n_dead_tup > 10000 THEN 'VACUUM_RECOMMENDED'
        ELSE 'OK'
    END as vacuum_status,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze,
    CASE 
        WHEN last_analyze IS NULL OR last_analyze < NOW() - INTERVAL '7 days' THEN 'ANALYZE_NEEDED'
        WHEN last_analyze < NOW() - INTERVAL '1 day' THEN 'ANALYZE_RECOMMENDED'
        ELSE 'OK'
    END as analyze_status
FROM pg_stat_user_tables
WHERE schemaname = CURRENT_SCHEMA()
  AND tablename IN ('saga_instances', 'saga_steps', 'saga_events');

COMMENT ON VIEW saga_maintenance_recommendations IS 'Provides maintenance recommendations based on table statistics';

-- ============================================================================
-- Usage Examples
-- ============================================================================

-- Example 1: Check index usage
-- SELECT * FROM saga_index_usage_stats;

-- Example 2: Check for bloated indexes
-- SELECT * FROM saga_index_bloat_check WHERE size_category = 'LARGE';

-- Example 3: Analyze storage usage
-- SELECT * FROM saga_storage_size;

-- Example 4: Update table statistics
-- SELECT * FROM analyze_saga_tables();

-- Example 5: Vacuum tables (light)
-- SELECT * FROM vacuum_saga_tables(FALSE);

-- Example 6: Vacuum tables (full - locks table)
-- SELECT * FROM vacuum_saga_tables(TRUE);

-- Example 7: Reindex all Saga tables (locks table)
-- SELECT * FROM reindex_saga_tables();

-- Example 8: Check maintenance recommendations
-- SELECT * FROM saga_maintenance_recommendations;

-- Example 9: Get missing index suggestions
-- SELECT * FROM suggest_missing_indexes();

-- Example 10: Identify slow queries (requires pg_stat_statements)
-- SELECT * FROM saga_slow_queries;

-- ============================================================================
-- Automated Maintenance Setup (Optional)
-- ============================================================================

-- Function: auto_maintain_saga_tables
-- Purpose: Automatic maintenance function to be called periodically
CREATE OR REPLACE FUNCTION auto_maintain_saga_tables()
RETURNS TABLE(
    operation TEXT,
    table_name TEXT,
    status TEXT,
    details TEXT
) AS $$
DECLARE
    rec RECORD;
BEGIN
    -- Analyze all tables
    FOR rec IN SELECT * FROM analyze_saga_tables() LOOP
        operation := 'ANALYZE';
        table_name := rec.table_name;
        status := rec.status;
        details := 'Rows: ' || rec.rows_affected;
        RETURN NEXT;
    END LOOP;
    
    -- Vacuum tables if needed
    FOR rec IN 
        SELECT t.tablename, t.n_dead_tup
        FROM pg_stat_user_tables t
        WHERE t.schemaname = CURRENT_SCHEMA()
          AND t.tablename IN ('saga_instances', 'saga_steps', 'saga_events')
          AND t.n_dead_tup > 1000
    LOOP
        EXECUTE 'VACUUM ANALYZE ' || rec.tablename;
        operation := 'VACUUM';
        table_name := rec.tablename;
        status := 'SUCCESS';
        details := 'Dead tuples: ' || rec.n_dead_tup;
        RETURN NEXT;
    END LOOP;
    
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION auto_maintain_saga_tables IS 'Performs automatic maintenance tasks on Saga tables';

-- ============================================================================
-- Monitoring and Alerting Queries
-- ============================================================================

-- Query: Check for tables needing immediate maintenance
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_stat_user_tables
        WHERE schemaname = CURRENT_SCHEMA()
          AND tablename IN ('saga_instances', 'saga_steps', 'saga_events')
          AND n_dead_tup > n_live_tup * 0.5
    ) THEN
        RAISE WARNING 'Some Saga tables have high dead tuple ratio and need VACUUM';
    END IF;
END $$;

-- ============================================================================
-- Performance Tuning Parameters (Reference)
-- ============================================================================

-- These parameters affect index and query performance
-- (Set in postgresql.conf or via ALTER SYSTEM)

/*
-- Work memory for sorts and hash joins
SET work_mem = '256MB';

-- Maintenance work memory for VACUUM, CREATE INDEX
SET maintenance_work_mem = '512MB';

-- Effective cache size (hint to query planner)
SET effective_cache_size = '4GB';

-- Random page cost (lower for SSD)
SET random_page_cost = 1.1;

-- Autovacuum settings
SET autovacuum = on;
SET autovacuum_vacuum_scale_factor = 0.1;
SET autovacuum_analyze_scale_factor = 0.05;
*/

-- ============================================================================
-- Index Maintenance Best Practices
-- ============================================================================

/*
1. Regular Maintenance Schedule:
   - Run ANALYZE daily: SELECT * FROM analyze_saga_tables();
   - Run VACUUM weekly: SELECT * FROM vacuum_saga_tables(FALSE);
   - Run REINDEX monthly (during maintenance window)

2. Monitoring:
   - Check index usage weekly: SELECT * FROM saga_index_usage_stats;
   - Check for bloat monthly: SELECT * FROM saga_index_bloat_check;
   - Review slow queries daily: SELECT * FROM saga_slow_queries;

3. Optimization:
   - Drop unused indexes (idx_scan = 0 for > 1 month)
   - Rebuild large bloated indexes
   - Update PostgreSQL statistics after bulk operations

4. Performance Testing:
   - Use EXPLAIN ANALYZE to verify index usage
   - Benchmark queries before and after index changes
   - Monitor query execution time trends

5. Production Considerations:
   - Run REINDEX during low-traffic periods (it locks the table)
   - Use VACUUM instead of VACUUM FULL unless absolutely needed
   - Enable pg_stat_statements for query analysis
*/

