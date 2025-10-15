-- Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
--
-- Saga State Storage - PostgreSQL Migration Script
-- This migration adds optimistic locking support via version column

-- ============================================================================
-- Migration: Add version column for optimistic locking
-- Version: 2
-- Description: Adds version column to saga_instances table for concurrency control
-- ============================================================================

-- Add version column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'saga_instances' 
        AND column_name = 'version'
    ) THEN
        ALTER TABLE saga_instances 
        ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
        
        -- Add check constraint
        ALTER TABLE saga_instances 
        ADD CONSTRAINT chk_version CHECK (version > 0);
        
        RAISE NOTICE 'Added version column to saga_instances table';
    ELSE
        RAISE NOTICE 'Version column already exists in saga_instances table';
    END IF;
END $$;

-- Update trigger to increment version on update
CREATE OR REPLACE FUNCTION increment_saga_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.version = OLD.version + 1;
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if it exists
DROP TRIGGER IF EXISTS trigger_saga_version_increment ON saga_instances;

-- Create new trigger for version increment
CREATE TRIGGER trigger_saga_version_increment
    BEFORE UPDATE ON saga_instances
    FOR EACH ROW
    EXECUTE FUNCTION increment_saga_version();

COMMENT ON TRIGGER trigger_saga_version_increment ON saga_instances IS 
    'Automatically increments version and updates updated_at timestamp on saga_instances update';

-- Record migration
INSERT INTO saga_schema_version (version, description) 
VALUES (2, 'Add optimistic locking support with version column')
ON CONFLICT (version) DO NOTHING;

-- ============================================================================
-- End of Migration
-- ============================================================================

