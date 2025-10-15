-- Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
--
-- Saga State Storage - PostgreSQL Schema
-- This schema defines the database structure for persisting Saga instances,
-- step states, and event logs in PostgreSQL.
--
-- Design Principles:
-- 1. Support all fields required by StateStorage interface
-- 2. Enable efficient querying for active, timeout, and completed Sagas
-- 3. Preserve complete audit trail through event logging
-- 4. Use JSONB for flexible metadata and complex data structures
-- 5. Enforce referential integrity with foreign keys
-- 6. Optimize for both write and read operations with strategic indexes

-- ============================================================================
-- Table: saga_instances
-- Description: Stores Saga instance metadata and overall state
-- ============================================================================

CREATE TABLE IF NOT EXISTS saga_instances (
    -- Primary identification
    id VARCHAR(255) PRIMARY KEY,
    definition_id VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    description TEXT,
    
    -- State information
    state INTEGER NOT NULL,
    current_step INTEGER NOT NULL DEFAULT 0,
    total_steps INTEGER NOT NULL DEFAULT 0,
    
    -- Timing information
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    timed_out_at TIMESTAMP WITH TIME ZONE,
    
    -- Data fields (stored as JSONB for flexibility)
    initial_data JSONB,
    current_data JSONB,
    result_data JSONB,
    
    -- Error information (stored as JSONB)
    error JSONB,
    
    -- Configuration
    timeout_ms BIGINT,
    retry_policy JSONB,
    
    -- Metadata and tracing
    metadata JSONB,
    trace_id VARCHAR(255),
    span_id VARCHAR(255),
    
    -- Constraints
    CONSTRAINT chk_state CHECK (state >= 0 AND state <= 8),
    CONSTRAINT chk_current_step CHECK (current_step >= 0),
    CONSTRAINT chk_total_steps CHECK (total_steps >= 0)
);

-- Indexes for saga_instances
CREATE INDEX IF NOT EXISTS idx_saga_state ON saga_instances(state);
CREATE INDEX IF NOT EXISTS idx_saga_definition_id ON saga_instances(definition_id);
CREATE INDEX IF NOT EXISTS idx_saga_created_at ON saga_instances(created_at);
CREATE INDEX IF NOT EXISTS idx_saga_updated_at ON saga_instances(updated_at);
CREATE INDEX IF NOT EXISTS idx_saga_started_at ON saga_instances(started_at) WHERE started_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_saga_completed_at ON saga_instances(completed_at) WHERE completed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_saga_timed_out_at ON saga_instances(timed_out_at) WHERE timed_out_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_saga_trace_id ON saga_instances(trace_id) WHERE trace_id IS NOT NULL;

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_saga_state_created ON saga_instances(state, created_at);
CREATE INDEX IF NOT EXISTS idx_saga_state_updated ON saga_instances(state, updated_at);
CREATE INDEX IF NOT EXISTS idx_saga_definition_state ON saga_instances(definition_id, state);

-- GIN index for metadata searches
CREATE INDEX IF NOT EXISTS idx_saga_metadata ON saga_instances USING GIN(metadata);

-- Comment on table
COMMENT ON TABLE saga_instances IS 'Stores Saga instance metadata, state, and execution data';
COMMENT ON COLUMN saga_instances.id IS 'Unique identifier for the Saga instance (UUID)';
COMMENT ON COLUMN saga_instances.definition_id IS 'Identifier of the Saga definition this instance follows';
COMMENT ON COLUMN saga_instances.state IS 'Current state of the Saga (0=pending, 1=running, 2=step_completed, 3=completed, 4=compensating, 5=compensated, 6=failed, 7=cancelled, 8=timed_out)';
COMMENT ON COLUMN saga_instances.timeout_ms IS 'Timeout duration in milliseconds';
COMMENT ON COLUMN saga_instances.retry_policy IS 'JSON object containing retry configuration';
COMMENT ON COLUMN saga_instances.error IS 'JSON object containing error details if Saga failed';
COMMENT ON COLUMN saga_instances.trace_id IS 'Distributed tracing identifier for correlation';

-- ============================================================================
-- Table: saga_steps
-- Description: Stores the state of individual steps within a Saga
-- ============================================================================

CREATE TABLE IF NOT EXISTS saga_steps (
    -- Primary identification
    id VARCHAR(255) PRIMARY KEY,
    saga_id VARCHAR(255) NOT NULL,
    step_index INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    
    -- State information
    state INTEGER NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 1,
    
    -- Timing information
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    
    -- Data fields (stored as JSONB)
    input_data JSONB,
    output_data JSONB,
    
    -- Error information
    error JSONB,
    
    -- Compensation information
    compensation_state JSONB,
    
    -- Metadata
    metadata JSONB,
    
    -- Constraints
    CONSTRAINT chk_step_state CHECK (state >= 0 AND state <= 6),
    CONSTRAINT chk_step_attempts CHECK (attempts >= 0),
    CONSTRAINT chk_step_max_attempts CHECK (max_attempts >= 0),
    CONSTRAINT chk_step_index CHECK (step_index >= 0),
    
    -- Foreign key to saga_instances
    CONSTRAINT fk_saga_steps_saga FOREIGN KEY (saga_id) 
        REFERENCES saga_instances(id) ON DELETE CASCADE
);

-- Indexes for saga_steps
CREATE INDEX IF NOT EXISTS idx_step_saga_id ON saga_steps(saga_id);
CREATE INDEX IF NOT EXISTS idx_step_state ON saga_steps(state);
CREATE INDEX IF NOT EXISTS idx_step_created_at ON saga_steps(created_at);
CREATE INDEX IF NOT EXISTS idx_step_started_at ON saga_steps(started_at) WHERE started_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_step_completed_at ON saga_steps(completed_at) WHERE completed_at IS NOT NULL;

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_step_saga_index ON saga_steps(saga_id, step_index);
CREATE INDEX IF NOT EXISTS idx_step_saga_state ON saga_steps(saga_id, state);
CREATE INDEX IF NOT EXISTS idx_step_state_attempts ON saga_steps(state, attempts);

-- Unique constraint to ensure one step per index per saga
CREATE UNIQUE INDEX IF NOT EXISTS idx_step_unique_saga_index ON saga_steps(saga_id, step_index);

-- Comment on table
COMMENT ON TABLE saga_steps IS 'Stores individual step states within Saga executions';
COMMENT ON COLUMN saga_steps.id IS 'Unique identifier for the step instance';
COMMENT ON COLUMN saga_steps.saga_id IS 'Reference to parent Saga instance';
COMMENT ON COLUMN saga_steps.step_index IS 'Order/position of step within the Saga (0-based)';
COMMENT ON COLUMN saga_steps.state IS 'Current state of the step (0=pending, 1=running, 2=completed, 3=failed, 4=compensating, 5=compensated, 6=skipped)';
COMMENT ON COLUMN saga_steps.compensation_state IS 'JSON object containing compensation execution state';

-- ============================================================================
-- Table: saga_events
-- Description: Event log for Saga lifecycle and state transitions
-- Purpose: Provides complete audit trail and debugging support
-- ============================================================================

CREATE TABLE IF NOT EXISTS saga_events (
    -- Primary identification
    id BIGSERIAL PRIMARY KEY,
    saga_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    
    -- Event context
    step_id VARCHAR(255),
    step_index INTEGER,
    
    -- Timing
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Event details
    event_data JSONB,
    
    -- State information
    old_state INTEGER,
    new_state INTEGER,
    
    -- Error information (if applicable)
    error JSONB,
    
    -- Tracing
    trace_id VARCHAR(255),
    span_id VARCHAR(255),
    
    -- Metadata
    metadata JSONB,
    
    -- Foreign key to saga_instances
    CONSTRAINT fk_saga_events_saga FOREIGN KEY (saga_id) 
        REFERENCES saga_instances(id) ON DELETE CASCADE
);

-- Indexes for saga_events
CREATE INDEX IF NOT EXISTS idx_event_saga_id ON saga_events(saga_id);
CREATE INDEX IF NOT EXISTS idx_event_type ON saga_events(event_type);
CREATE INDEX IF NOT EXISTS idx_event_timestamp ON saga_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_event_step_id ON saga_events(step_id) WHERE step_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_event_trace_id ON saga_events(trace_id) WHERE trace_id IS NOT NULL;

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_event_saga_timestamp ON saga_events(saga_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_event_saga_type ON saga_events(saga_id, event_type);
CREATE INDEX IF NOT EXISTS idx_event_type_timestamp ON saga_events(event_type, timestamp DESC);

-- Comment on table
COMMENT ON TABLE saga_events IS 'Audit log of all Saga lifecycle events and state transitions';
COMMENT ON COLUMN saga_events.id IS 'Auto-incrementing event sequence number';
COMMENT ON COLUMN saga_events.saga_id IS 'Reference to the Saga instance';
COMMENT ON COLUMN saga_events.event_type IS 'Type of event (e.g., saga.started, saga.step.completed)';
COMMENT ON COLUMN saga_events.event_data IS 'JSON object containing event-specific details';
COMMENT ON COLUMN saga_events.old_state IS 'State before the event (for state transitions)';
COMMENT ON COLUMN saga_events.new_state IS 'State after the event (for state transitions)';

-- ============================================================================
-- Performance Optimization Views
-- ============================================================================

-- View: active_sagas
-- Purpose: Efficiently query all active Saga instances
CREATE OR REPLACE VIEW active_sagas AS
SELECT 
    id, definition_id, name, state, current_step, total_steps,
    created_at, updated_at, started_at, trace_id, metadata
FROM saga_instances
WHERE state IN (1, 2, 4)  -- running, step_completed, compensating
ORDER BY created_at DESC;

COMMENT ON VIEW active_sagas IS 'Provides efficient access to all active (non-terminal) Saga instances';

-- View: failed_sagas
-- Purpose: Efficiently query all failed Saga instances
CREATE OR REPLACE VIEW failed_sagas AS
SELECT 
    id, definition_id, name, state, current_step, total_steps,
    created_at, updated_at, completed_at, error, trace_id
FROM saga_instances
WHERE state IN (6, 7, 8)  -- failed, cancelled, timed_out
ORDER BY updated_at DESC;

COMMENT ON VIEW failed_sagas IS 'Provides efficient access to all failed Saga instances';

-- View: completed_sagas
-- Purpose: Efficiently query successfully completed Saga instances
CREATE OR REPLACE VIEW completed_sagas AS
SELECT 
    id, definition_id, name, current_step, total_steps,
    created_at, started_at, completed_at, result_data, trace_id
FROM saga_instances
WHERE state = 3  -- completed
ORDER BY completed_at DESC;

COMMENT ON VIEW completed_sagas IS 'Provides efficient access to successfully completed Saga instances';

-- ============================================================================
-- Maintenance Functions
-- ============================================================================

-- Function: cleanup_expired_sagas
-- Purpose: Remove Saga instances older than a specified date
CREATE OR REPLACE FUNCTION cleanup_expired_sagas(older_than TIMESTAMP WITH TIME ZONE)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired saga instances (cascade will handle related records)
    DELETE FROM saga_instances
    WHERE updated_at < older_than
      AND state IN (3, 5, 6, 7, 8);  -- terminal states only
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_sagas IS 'Removes Saga instances in terminal states older than the specified timestamp';

-- Function: update_saga_updated_at
-- Purpose: Automatically update the updated_at timestamp on saga_instances
CREATE OR REPLACE FUNCTION update_saga_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at
CREATE TRIGGER trigger_saga_updated_at
    BEFORE UPDATE ON saga_instances
    FOR EACH ROW
    EXECUTE FUNCTION update_saga_updated_at();

COMMENT ON TRIGGER trigger_saga_updated_at ON saga_instances IS 'Automatically updates the updated_at timestamp on any saga_instances update';

-- ============================================================================
-- Statistics and Monitoring Views
-- ============================================================================

-- View: saga_statistics
-- Purpose: Provide aggregated statistics about Saga executions
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

COMMENT ON VIEW saga_statistics IS 'Provides aggregated statistics grouped by Saga definition';

-- ============================================================================
-- Indexes for Performance Tuning
-- ============================================================================

-- Additional indexes for filtering operations
CREATE INDEX IF NOT EXISTS idx_saga_filter_state_timeout ON saga_instances(state, created_at) 
    WHERE state IN (1, 2, 4);

CREATE INDEX IF NOT EXISTS idx_saga_terminal_cleanup ON saga_instances(updated_at) 
    WHERE state IN (3, 5, 6, 7, 8);

-- ============================================================================
-- Partitioning Recommendations (for high-volume deployments)
-- ============================================================================

-- For high-volume Saga execution environments, consider partitioning saga_events
-- by timestamp to improve query performance and enable efficient data archival.
--
-- Example partitioning strategy (uncomment and customize as needed):
--
-- ALTER TABLE saga_events RENAME TO saga_events_template;
-- CREATE TABLE saga_events (LIKE saga_events_template INCLUDING ALL) 
--     PARTITION BY RANGE (timestamp);
-- CREATE TABLE saga_events_2025_01 PARTITION OF saga_events
--     FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
-- -- Add more partitions as needed...

-- ============================================================================
-- Grant Permissions (customize based on your security requirements)
-- ============================================================================

-- Example permission grants (uncomment and customize as needed):
-- GRANT SELECT, INSERT, UPDATE, DELETE ON saga_instances TO saga_app_role;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON saga_steps TO saga_app_role;
-- GRANT SELECT, INSERT ON saga_events TO saga_app_role;
-- GRANT SELECT ON active_sagas, failed_sagas, completed_sagas, saga_statistics TO saga_app_role;
-- GRANT USAGE ON SEQUENCE saga_events_id_seq TO saga_app_role;

-- ============================================================================
-- Schema Version Information
-- ============================================================================

CREATE TABLE IF NOT EXISTS saga_schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    description TEXT
);

INSERT INTO saga_schema_version (version, description) 
VALUES (1, 'Initial Saga state storage schema with instances, steps, and events tables')
ON CONFLICT (version) DO NOTHING;

COMMENT ON TABLE saga_schema_version IS 'Tracks schema version for migration management';

