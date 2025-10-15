-- Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
--
-- Saga State Storage - PostgreSQL Migration Management Script
-- This script provides a complete migration framework for Saga database schema
-- with version tracking, incremental upgrades, and rollback capabilities.
--
-- Usage:
--   1. Initial setup: Run this script to create migration infrastructure
--   2. Apply migrations: Execute apply_migration() for each version
--   3. Rollback: Execute rollback_migration() to undo specific version
--   4. Check status: Query saga_migrations table for migration history
--
-- Migration Versions:
--   V1 (Initial): Base schema with saga_instances, saga_steps, saga_events tables
--   V2: Add optimistic locking with version column
--
-- ============================================================================
-- Migration Management Infrastructure
-- ============================================================================

-- Table: saga_migrations
-- Description: Tracks all executed migrations with timestamps and status
CREATE TABLE IF NOT EXISTS saga_migrations (
    version INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    applied_by VARCHAR(255) DEFAULT CURRENT_USER,
    execution_time_ms INTEGER,
    checksum VARCHAR(64),
    status VARCHAR(20) NOT NULL DEFAULT 'applied',
    rollback_available BOOLEAN DEFAULT TRUE,
    rollback_script TEXT,
    metadata JSONB,
    
    CONSTRAINT chk_migration_version CHECK (version > 0),
    CONSTRAINT chk_migration_status CHECK (status IN ('applied', 'rolled_back', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_migrations_status ON saga_migrations(status);
CREATE INDEX IF NOT EXISTS idx_migrations_applied_at ON saga_migrations(applied_at DESC);

COMMENT ON TABLE saga_migrations IS 'Tracks database schema migration history for Saga storage';
COMMENT ON COLUMN saga_migrations.version IS 'Migration version number (sequential)';
COMMENT ON COLUMN saga_migrations.checksum IS 'SHA-256 checksum of migration script for verification';
COMMENT ON COLUMN saga_migrations.rollback_available IS 'Whether this migration can be rolled back';

-- ============================================================================
-- Migration Helper Functions
-- ============================================================================

-- Function: get_current_schema_version
-- Purpose: Returns the current schema version (highest applied migration)
CREATE OR REPLACE FUNCTION get_current_schema_version()
RETURNS INTEGER AS $$
DECLARE
    current_version INTEGER;
BEGIN
    SELECT COALESCE(MAX(version), 0)
    INTO current_version
    FROM saga_migrations
    WHERE status = 'applied';
    
    RETURN current_version;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_current_schema_version IS 'Returns the highest successfully applied migration version';

-- Function: is_migration_applied
-- Purpose: Check if a specific migration version has been applied
CREATE OR REPLACE FUNCTION is_migration_applied(migration_version INTEGER)
RETURNS BOOLEAN AS $$
DECLARE
    is_applied BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1
        FROM saga_migrations
        WHERE version = migration_version
        AND status = 'applied'
    ) INTO is_applied;
    
    RETURN is_applied;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION is_migration_applied IS 'Checks if a migration version has been successfully applied';

-- Function: record_migration
-- Purpose: Records a migration execution in the migrations table
CREATE OR REPLACE FUNCTION record_migration(
    p_version INTEGER,
    p_name VARCHAR(255),
    p_description TEXT,
    p_execution_time_ms INTEGER,
    p_checksum VARCHAR(64) DEFAULT NULL,
    p_rollback_available BOOLEAN DEFAULT TRUE,
    p_rollback_script TEXT DEFAULT NULL
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO saga_migrations (
        version, name, description, execution_time_ms,
        checksum, rollback_available, rollback_script, status
    ) VALUES (
        p_version, p_name, p_description, p_execution_time_ms,
        p_checksum, p_rollback_available, p_rollback_script, 'applied'
    )
    ON CONFLICT (version) DO UPDATE SET
        applied_at = NOW(),
        applied_by = CURRENT_USER,
        status = 'applied',
        execution_time_ms = p_execution_time_ms;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION record_migration IS 'Records a migration execution with metadata';

-- Function: record_migration_rollback
-- Purpose: Records a migration rollback
CREATE OR REPLACE FUNCTION record_migration_rollback(
    p_version INTEGER,
    p_execution_time_ms INTEGER
)
RETURNS VOID AS $$
BEGIN
    UPDATE saga_migrations
    SET status = 'rolled_back',
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'rolled_back_at', NOW(),
            'rolled_back_by', CURRENT_USER,
            'rollback_time_ms', p_execution_time_ms
        )
    WHERE version = p_version;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION record_migration_rollback IS 'Records a migration rollback event';

-- ============================================================================
-- Migration V1: Initial Schema
-- ============================================================================

-- Function: apply_migration_v1
-- Purpose: Creates the initial Saga storage schema
-- Idempotent: Safe to run multiple times (uses IF NOT EXISTS)
CREATE OR REPLACE FUNCTION apply_migration_v1()
RETURNS TEXT AS $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
    execution_ms INTEGER;
BEGIN
    start_time := clock_timestamp();
    
    -- Check if already applied
    IF is_migration_applied(1) THEN
        RETURN 'Migration V1 already applied';
    END IF;
    
    -- Create saga_instances table
    CREATE TABLE IF NOT EXISTS saga_instances (
        id VARCHAR(255) PRIMARY KEY,
        definition_id VARCHAR(255) NOT NULL,
        name VARCHAR(255),
        description TEXT,
        state INTEGER NOT NULL,
        current_step INTEGER NOT NULL DEFAULT 0,
        total_steps INTEGER NOT NULL DEFAULT 0,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
        started_at TIMESTAMP WITH TIME ZONE,
        completed_at TIMESTAMP WITH TIME ZONE,
        timed_out_at TIMESTAMP WITH TIME ZONE,
        initial_data JSONB,
        current_data JSONB,
        result_data JSONB,
        error JSONB,
        timeout_ms BIGINT,
        retry_policy JSONB,
        metadata JSONB,
        trace_id VARCHAR(255),
        span_id VARCHAR(255),
        version INTEGER NOT NULL DEFAULT 1,
        CONSTRAINT chk_state CHECK (state >= 0 AND state <= 8),
        CONSTRAINT chk_current_step CHECK (current_step >= 0),
        CONSTRAINT chk_total_steps CHECK (total_steps >= 0),
        CONSTRAINT chk_version CHECK (version > 0)
    );
    
    -- Create indexes for saga_instances
    CREATE INDEX IF NOT EXISTS idx_saga_state ON saga_instances(state);
    CREATE INDEX IF NOT EXISTS idx_saga_definition_id ON saga_instances(definition_id);
    CREATE INDEX IF NOT EXISTS idx_saga_created_at ON saga_instances(created_at);
    CREATE INDEX IF NOT EXISTS idx_saga_updated_at ON saga_instances(updated_at);
    CREATE INDEX IF NOT EXISTS idx_saga_started_at ON saga_instances(started_at) WHERE started_at IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_saga_completed_at ON saga_instances(completed_at) WHERE completed_at IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_saga_timed_out_at ON saga_instances(timed_out_at) WHERE timed_out_at IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_saga_trace_id ON saga_instances(trace_id) WHERE trace_id IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_saga_state_created ON saga_instances(state, created_at);
    CREATE INDEX IF NOT EXISTS idx_saga_state_updated ON saga_instances(state, updated_at);
    CREATE INDEX IF NOT EXISTS idx_saga_definition_state ON saga_instances(definition_id, state);
    CREATE INDEX IF NOT EXISTS idx_saga_metadata ON saga_instances USING GIN(metadata);
    CREATE INDEX IF NOT EXISTS idx_saga_filter_state_timeout ON saga_instances(state, created_at) WHERE state IN (1, 2, 4);
    CREATE INDEX IF NOT EXISTS idx_saga_terminal_cleanup ON saga_instances(updated_at) WHERE state IN (3, 5, 6, 7, 8);
    
    -- Create saga_steps table
    CREATE TABLE IF NOT EXISTS saga_steps (
        id VARCHAR(255) PRIMARY KEY,
        saga_id VARCHAR(255) NOT NULL,
        step_index INTEGER NOT NULL,
        name VARCHAR(255) NOT NULL,
        state INTEGER NOT NULL,
        attempts INTEGER NOT NULL DEFAULT 0,
        max_attempts INTEGER NOT NULL DEFAULT 1,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
        started_at TIMESTAMP WITH TIME ZONE,
        completed_at TIMESTAMP WITH TIME ZONE,
        last_attempt_at TIMESTAMP WITH TIME ZONE,
        input_data JSONB,
        output_data JSONB,
        error JSONB,
        compensation_state JSONB,
        metadata JSONB,
        CONSTRAINT chk_step_state CHECK (state >= 0 AND state <= 6),
        CONSTRAINT chk_step_attempts CHECK (attempts >= 0),
        CONSTRAINT chk_step_max_attempts CHECK (max_attempts >= 0),
        CONSTRAINT chk_step_index CHECK (step_index >= 0),
        CONSTRAINT fk_saga_steps_saga FOREIGN KEY (saga_id) 
            REFERENCES saga_instances(id) ON DELETE CASCADE
    );
    
    -- Create indexes for saga_steps
    CREATE INDEX IF NOT EXISTS idx_step_saga_id ON saga_steps(saga_id);
    CREATE INDEX IF NOT EXISTS idx_step_state ON saga_steps(state);
    CREATE INDEX IF NOT EXISTS idx_step_created_at ON saga_steps(created_at);
    CREATE INDEX IF NOT EXISTS idx_step_started_at ON saga_steps(started_at) WHERE started_at IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_step_completed_at ON saga_steps(completed_at) WHERE completed_at IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_step_saga_index ON saga_steps(saga_id, step_index);
    CREATE INDEX IF NOT EXISTS idx_step_saga_state ON saga_steps(saga_id, state);
    CREATE INDEX IF NOT EXISTS idx_step_state_attempts ON saga_steps(state, attempts);
    CREATE UNIQUE INDEX IF NOT EXISTS idx_step_unique_saga_index ON saga_steps(saga_id, step_index);
    
    -- Create saga_events table
    CREATE TABLE IF NOT EXISTS saga_events (
        id BIGSERIAL PRIMARY KEY,
        saga_id VARCHAR(255) NOT NULL,
        event_type VARCHAR(100) NOT NULL,
        step_id VARCHAR(255),
        step_index INTEGER,
        timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
        event_data JSONB,
        old_state INTEGER,
        new_state INTEGER,
        error JSONB,
        trace_id VARCHAR(255),
        span_id VARCHAR(255),
        metadata JSONB,
        CONSTRAINT fk_saga_events_saga FOREIGN KEY (saga_id) 
            REFERENCES saga_instances(id) ON DELETE CASCADE
    );
    
    -- Create indexes for saga_events
    CREATE INDEX IF NOT EXISTS idx_event_saga_id ON saga_events(saga_id);
    CREATE INDEX IF NOT EXISTS idx_event_type ON saga_events(event_type);
    CREATE INDEX IF NOT EXISTS idx_event_timestamp ON saga_events(timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_event_step_id ON saga_events(step_id) WHERE step_id IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_event_trace_id ON saga_events(trace_id) WHERE trace_id IS NOT NULL;
    CREATE INDEX IF NOT EXISTS idx_event_saga_timestamp ON saga_events(saga_id, timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_event_saga_type ON saga_events(saga_id, event_type);
    CREATE INDEX IF NOT EXISTS idx_event_type_timestamp ON saga_events(event_type, timestamp DESC);
    
    -- Create views
    CREATE OR REPLACE VIEW active_sagas AS
    SELECT 
        id, definition_id, name, state, current_step, total_steps,
        created_at, updated_at, started_at, trace_id, metadata
    FROM saga_instances
    WHERE state IN (1, 2, 4)
    ORDER BY created_at DESC;
    
    CREATE OR REPLACE VIEW failed_sagas AS
    SELECT 
        id, definition_id, name, state, current_step, total_steps,
        created_at, updated_at, completed_at, error, trace_id
    FROM saga_instances
    WHERE state IN (6, 7, 8)
    ORDER BY updated_at DESC;
    
    CREATE OR REPLACE VIEW completed_sagas AS
    SELECT 
        id, definition_id, name, current_step, total_steps,
        created_at, started_at, completed_at, result_data, trace_id
    FROM saga_instances
    WHERE state = 3
    ORDER BY completed_at DESC;
    
    CREATE OR REPLACE VIEW saga_statistics AS
    SELECT 
        definition_id,
        COUNT(*) as total_instances,
        COUNT(CASE WHEN state = 3 THEN 1 END) as completed_count,
        COUNT(CASE WHEN state IN (6, 7, 8) THEN 1 END) as failed_count,
        COUNT(CASE WHEN state IN (1, 2, 4) THEN 1 END) as active_count,
        AVG(CASE 
            WHEN completed_at IS NOT NULL AND started_at IS NOT NULL 
            THEN EXTRACT(EPOCH FROM (completed_at - started_at))
        END) as avg_duration_seconds,
        MAX(updated_at) as last_execution
    FROM saga_instances
    GROUP BY definition_id;
    
    -- Create utility functions
    CREATE OR REPLACE FUNCTION cleanup_expired_sagas(older_than TIMESTAMP WITH TIME ZONE)
    RETURNS INTEGER AS $cleanup$
    DECLARE
        deleted_count INTEGER;
    BEGIN
        DELETE FROM saga_instances
        WHERE updated_at < older_than
          AND state IN (3, 5, 6, 7, 8);
        
        GET DIAGNOSTICS deleted_count = ROW_COUNT;
        RETURN deleted_count;
    END;
    $cleanup$ LANGUAGE plpgsql;
    
    CREATE OR REPLACE FUNCTION update_saga_updated_at()
    RETURNS TRIGGER AS $trigger$
    BEGIN
        NEW.updated_at = NOW();
        RETURN NEW;
    END;
    $trigger$ LANGUAGE plpgsql;
    
    DROP TRIGGER IF EXISTS trigger_saga_updated_at ON saga_instances;
    CREATE TRIGGER trigger_saga_updated_at
        BEFORE UPDATE ON saga_instances
        FOR EACH ROW
        EXECUTE FUNCTION update_saga_updated_at();
    
    end_time := clock_timestamp();
    execution_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time))::INTEGER;
    
    -- Record migration
    PERFORM record_migration(
        1,
        'Initial Saga Schema',
        'Creates base schema with saga_instances, saga_steps, and saga_events tables',
        execution_ms,
        NULL,
        TRUE,
        NULL
    );
    
    RETURN format('Migration V1 applied successfully in %s ms', execution_ms);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION apply_migration_v1 IS 'Applies V1 migration: Initial Saga storage schema';

-- Function: rollback_migration_v1
-- Purpose: Rolls back V1 migration by dropping all Saga tables
-- WARNING: This will delete all Saga data!
CREATE OR REPLACE FUNCTION rollback_migration_v1()
RETURNS TEXT AS $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
    execution_ms INTEGER;
BEGIN
    start_time := clock_timestamp();
    
    -- Check if migration was applied
    IF NOT is_migration_applied(1) THEN
        RETURN 'Migration V1 not applied, nothing to rollback';
    END IF;
    
    -- Drop objects in reverse order
    DROP VIEW IF EXISTS saga_statistics CASCADE;
    DROP VIEW IF EXISTS completed_sagas CASCADE;
    DROP VIEW IF EXISTS failed_sagas CASCADE;
    DROP VIEW IF EXISTS active_sagas CASCADE;
    
    DROP TRIGGER IF EXISTS trigger_saga_updated_at ON saga_instances;
    DROP FUNCTION IF EXISTS update_saga_updated_at CASCADE;
    DROP FUNCTION IF EXISTS cleanup_expired_sagas CASCADE;
    
    DROP TABLE IF EXISTS saga_events CASCADE;
    DROP TABLE IF EXISTS saga_steps CASCADE;
    DROP TABLE IF EXISTS saga_instances CASCADE;
    
    end_time := clock_timestamp();
    execution_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time))::INTEGER;
    
    -- Record rollback
    PERFORM record_migration_rollback(1, execution_ms);
    
    RETURN format('Migration V1 rolled back successfully in %s ms', execution_ms);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION rollback_migration_v1 IS 'Rolls back V1 migration: Drops all Saga storage tables';

-- ============================================================================
-- Migration V2: Add Optimistic Locking
-- ============================================================================

-- Function: apply_migration_v2
-- Purpose: Adds version column for optimistic locking (if not already present)
CREATE OR REPLACE FUNCTION apply_migration_v2()
RETURNS TEXT AS $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
    execution_ms INTEGER;
    version_exists BOOLEAN;
BEGIN
    start_time := clock_timestamp();
    
    -- Check if already applied
    IF is_migration_applied(2) THEN
        RETURN 'Migration V2 already applied';
    END IF;
    
    -- Check if V1 is applied
    IF NOT is_migration_applied(1) THEN
        RETURN 'ERROR: Migration V1 must be applied before V2';
    END IF;
    
    -- Check if version column already exists
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'saga_instances' 
        AND column_name = 'version'
    ) INTO version_exists;
    
    IF NOT version_exists THEN
        -- Add version column
        ALTER TABLE saga_instances 
        ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
        
        -- Add check constraint
        ALTER TABLE saga_instances 
        ADD CONSTRAINT chk_version CHECK (version > 0);
        
        RAISE NOTICE 'Added version column to saga_instances table';
    ELSE
        RAISE NOTICE 'Version column already exists in saga_instances table';
    END IF;
    
    -- Create/replace version increment trigger
    CREATE OR REPLACE FUNCTION increment_saga_version()
    RETURNS TRIGGER AS $trigger$
    BEGIN
        NEW.version = OLD.version + 1;
        NEW.updated_at = NOW();
        RETURN NEW;
    END;
    $trigger$ LANGUAGE plpgsql;
    
    DROP TRIGGER IF EXISTS trigger_saga_version_increment ON saga_instances;
    
    CREATE TRIGGER trigger_saga_version_increment
        BEFORE UPDATE ON saga_instances
        FOR EACH ROW
        EXECUTE FUNCTION increment_saga_version();
    
    end_time := clock_timestamp();
    execution_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time))::INTEGER;
    
    -- Record migration
    PERFORM record_migration(
        2,
        'Add Optimistic Locking',
        'Adds version column to saga_instances for concurrency control',
        execution_ms,
        NULL,
        TRUE,
        NULL
    );
    
    RETURN format('Migration V2 applied successfully in %s ms', execution_ms);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION apply_migration_v2 IS 'Applies V2 migration: Adds optimistic locking support';

-- Function: rollback_migration_v2
-- Purpose: Rolls back V2 migration by removing version column and trigger
CREATE OR REPLACE FUNCTION rollback_migration_v2()
RETURNS TEXT AS $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
    execution_ms INTEGER;
BEGIN
    start_time := clock_timestamp();
    
    -- Check if migration was applied
    IF NOT is_migration_applied(2) THEN
        RETURN 'Migration V2 not applied, nothing to rollback';
    END IF;
    
    -- Drop trigger
    DROP TRIGGER IF EXISTS trigger_saga_version_increment ON saga_instances;
    DROP FUNCTION IF EXISTS increment_saga_version CASCADE;
    
    -- Remove version column (if it exists)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'saga_instances' 
        AND column_name = 'version'
    ) THEN
        ALTER TABLE saga_instances DROP COLUMN version;
        RAISE NOTICE 'Removed version column from saga_instances table';
    END IF;
    
    end_time := clock_timestamp();
    execution_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time))::INTEGER;
    
    -- Record rollback
    PERFORM record_migration_rollback(2, execution_ms);
    
    RETURN format('Migration V2 rolled back successfully in %s ms', execution_ms);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION rollback_migration_v2 IS 'Rolls back V2 migration: Removes optimistic locking support';

-- ============================================================================
-- Convenience Functions for Migration Management
-- ============================================================================

-- Function: apply_all_migrations
-- Purpose: Applies all pending migrations in order
CREATE OR REPLACE FUNCTION apply_all_migrations()
RETURNS TEXT AS $$
DECLARE
    result TEXT;
    total_result TEXT := '';
    current_version INTEGER;
BEGIN
    current_version := get_current_schema_version();
    
    -- Apply V1 if needed
    IF current_version < 1 THEN
        result := apply_migration_v1();
        total_result := total_result || result || E'\n';
        RAISE NOTICE '%', result;
    END IF;
    
    -- Apply V2 if needed
    IF current_version < 2 THEN
        result := apply_migration_v2();
        total_result := total_result || result || E'\n';
        RAISE NOTICE '%', result;
    END IF;
    
    IF total_result = '' THEN
        total_result := 'All migrations already applied. Current version: ' || get_current_schema_version();
    ELSE
        total_result := total_result || 'All migrations completed. Current version: ' || get_current_schema_version();
    END IF;
    
    RETURN total_result;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION apply_all_migrations IS 'Applies all pending migrations to bring schema to latest version';

-- Function: get_migration_status
-- Purpose: Returns detailed migration status information
CREATE OR REPLACE FUNCTION get_migration_status()
RETURNS TABLE (
    version INTEGER,
    name VARCHAR(255),
    description TEXT,
    status VARCHAR(20),
    applied_at TIMESTAMP WITH TIME ZONE,
    applied_by VARCHAR(255),
    execution_time_ms INTEGER,
    rollback_available BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.version,
        m.name,
        m.description,
        m.status,
        m.applied_at,
        m.applied_by,
        m.execution_time_ms,
        m.rollback_available
    FROM saga_migrations m
    ORDER BY m.version;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_migration_status IS 'Returns detailed status of all migrations';

-- ============================================================================
-- Initial Execution and Verification
-- ============================================================================

-- Display current status
DO $$
DECLARE
    current_version INTEGER;
BEGIN
    current_version := get_current_schema_version();
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Saga Migration System Initialized';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Current Schema Version: %', current_version;
    RAISE NOTICE '';
    RAISE NOTICE 'Available Migrations:';
    RAISE NOTICE '  V1: Initial Saga Schema';
    RAISE NOTICE '  V2: Add Optimistic Locking';
    RAISE NOTICE '';
    RAISE NOTICE 'To apply migrations:';
    RAISE NOTICE '  SELECT apply_all_migrations();';
    RAISE NOTICE '';
    RAISE NOTICE 'To check status:';
    RAISE NOTICE '  SELECT * FROM get_migration_status();';
    RAISE NOTICE '';
    RAISE NOTICE 'To rollback a migration:';
    RAISE NOTICE '  SELECT rollback_migration_v2();';
    RAISE NOTICE '  SELECT rollback_migration_v1();';
    RAISE NOTICE '========================================';
END;
$$;

