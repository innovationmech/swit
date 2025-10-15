// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// getTestDSN returns a test database DSN from environment or default.
func getTestDSN() string {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/saga_test?sslmode=disable"
	}
	return dsn
}

// setupTestDB creates a test database connection and cleans up any existing schema.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	dsn := getTestDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil, nil
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("Skipping test: cannot ping test database: %v", err)
		return nil, nil
	}

	// Clean up function
	cleanup := func() {
		// Drop all tables in reverse order
		cleanupQueries := []string{
			"DROP VIEW IF EXISTS saga_statistics CASCADE",
			"DROP VIEW IF EXISTS completed_sagas CASCADE",
			"DROP VIEW IF EXISTS failed_sagas CASCADE",
			"DROP VIEW IF EXISTS active_sagas CASCADE",
			"DROP TABLE IF EXISTS saga_migrations CASCADE",
			"DROP TABLE IF EXISTS saga_events CASCADE",
			"DROP TABLE IF EXISTS saga_steps CASCADE",
			"DROP TABLE IF EXISTS saga_instances CASCADE",
			"DROP TABLE IF EXISTS saga_schema_version CASCADE",
			"DROP FUNCTION IF EXISTS get_current_schema_version CASCADE",
			"DROP FUNCTION IF EXISTS is_migration_applied CASCADE",
			"DROP FUNCTION IF EXISTS record_migration CASCADE",
			"DROP FUNCTION IF EXISTS record_migration_rollback CASCADE",
			"DROP FUNCTION IF EXISTS apply_migration_v1 CASCADE",
			"DROP FUNCTION IF EXISTS rollback_migration_v1 CASCADE",
			"DROP FUNCTION IF EXISTS apply_migration_v2 CASCADE",
			"DROP FUNCTION IF EXISTS rollback_migration_v2 CASCADE",
			"DROP FUNCTION IF EXISTS apply_all_migrations CASCADE",
			"DROP FUNCTION IF EXISTS get_migration_status CASCADE",
			"DROP FUNCTION IF EXISTS cleanup_expired_sagas CASCADE",
			"DROP FUNCTION IF EXISTS update_saga_updated_at CASCADE",
			"DROP FUNCTION IF EXISTS increment_saga_version CASCADE",
		}

		for _, query := range cleanupQueries {
			db.Exec(query)
		}

		db.Close()
	}

	return db, cleanup
}

func TestMigrator_Initialize(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Test initialization
	err := migrator.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify saga_migrations table exists
	var exists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'saga_migrations'
		)
	`).Scan(&exists)

	if err != nil {
		t.Fatalf("Failed to check saga_migrations table: %v", err)
	}

	if !exists {
		t.Error("saga_migrations table was not created")
	}
}

func TestMigrator_GetCurrentVersion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Get version (should be 0 before any migrations)
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	if version != 0 {
		t.Errorf("Expected version 0, got %d", version)
	}
}

func TestMigrator_ApplyMigrationV1(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Apply V1
	result, err := migrator.ApplyMigration(ctx, 1)
	if err != nil {
		t.Fatalf("ApplyMigration(1) failed: %v", err)
	}

	t.Logf("Migration V1 result: %s", result)

	// Check version
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	if version != 1 {
		t.Errorf("Expected version 1 after applying V1, got %d", version)
	}

	// Verify tables exist
	tables := []string{"saga_instances", "saga_steps", "saga_events"}
	for _, table := range tables {
		var exists bool
		err = db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_name = $1
			)
		`, table).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check %s table: %v", table, err)
		}

		if !exists {
			t.Errorf("Table %s was not created", table)
		}
	}

	// Test idempotency - applying again should not fail
	result, err = migrator.ApplyMigration(ctx, 1)
	if err != nil {
		t.Fatalf("ApplyMigration(1) second time failed: %v", err)
	}

	if result != "Migration V1 already applied" {
		t.Errorf("Expected 'already applied' message, got: %s", result)
	}
}

func TestMigrator_ApplyMigrationV2(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Apply V1 first
	_, err := migrator.ApplyMigration(ctx, 1)
	if err != nil {
		t.Fatalf("ApplyMigration(1) failed: %v", err)
	}

	// Apply V2
	result, err := migrator.ApplyMigration(ctx, 2)
	if err != nil {
		t.Fatalf("ApplyMigration(2) failed: %v", err)
	}

	t.Logf("Migration V2 result: %s", result)

	// Check version
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	if version != 2 {
		t.Errorf("Expected version 2 after applying V2, got %d", version)
	}

	// Verify version column exists
	var hasColumn bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.columns 
			WHERE table_name = 'saga_instances'
			AND column_name = 'version'
		)
	`).Scan(&hasColumn)

	if err != nil {
		t.Fatalf("Failed to check version column: %v", err)
	}

	if !hasColumn {
		t.Error("Version column was not added to saga_instances")
	}
}

func TestMigrator_ApplyAllMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Apply all migrations
	result, err := migrator.ApplyAllMigrations(ctx)
	if err != nil {
		t.Fatalf("ApplyAllMigrations failed: %v", err)
	}

	t.Logf("Apply all migrations result: %s", result)

	// Check final version
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	expectedVersion := 2 // Current latest version
	if version != expectedVersion {
		t.Errorf("Expected version %d after applying all migrations, got %d", expectedVersion, version)
	}
}

func TestMigrator_GetMigrationStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize and apply migrations
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err := migrator.ApplyAllMigrations(ctx)
	if err != nil {
		t.Fatalf("ApplyAllMigrations failed: %v", err)
	}

	// Get status
	statuses, err := migrator.GetMigrationStatus(ctx)
	if err != nil {
		t.Fatalf("GetMigrationStatus failed: %v", err)
	}

	if len(statuses) != 2 {
		t.Errorf("Expected 2 migration statuses, got %d", len(statuses))
	}

	// Verify status details
	for _, status := range statuses {
		t.Logf("Migration V%d: %s - %s (status: %s, time: %dms)",
			status.Version,
			status.Name,
			status.Description,
			status.Status,
			status.ExecutionTimeMs,
		)

		if status.Status != "applied" {
			t.Errorf("Migration V%d status is not 'applied': %s", status.Version, status.Status)
		}

		if status.AppliedAt.IsZero() {
			t.Errorf("Migration V%d has zero AppliedAt timestamp", status.Version)
		}
	}
}

func TestMigrator_RollbackMigration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize and apply all migrations
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err := migrator.ApplyAllMigrations(ctx)
	if err != nil {
		t.Fatalf("ApplyAllMigrations failed: %v", err)
	}

	// Rollback V2
	result, err := migrator.RollbackMigration(ctx, 2)
	if err != nil {
		t.Fatalf("RollbackMigration(2) failed: %v", err)
	}

	t.Logf("Rollback V2 result: %s", result)

	// Check version (should be 1 after rolling back V2)
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	if version != 1 {
		t.Errorf("Expected version 1 after rolling back V2, got %d", version)
	}

	// Verify version column is removed
	var hasColumn bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.columns 
			WHERE table_name = 'saga_instances'
			AND column_name = 'version'
		)
	`).Scan(&hasColumn)

	if err != nil {
		t.Fatalf("Failed to check version column: %v", err)
	}

	if hasColumn {
		t.Error("Version column should have been removed after rollback")
	}
}

func TestMigrator_Migrate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Use high-level Migrate function
	err := migrator.Migrate(ctx)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify everything is set up
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}

	expectedVersion := 2
	if version != expectedVersion {
		t.Errorf("Expected version %d after Migrate, got %d", expectedVersion, version)
	}

	// Test that running Migrate again is safe (idempotent)
	err = migrator.Migrate(ctx)
	if err != nil {
		t.Fatalf("Second Migrate failed: %v", err)
	}
}

func TestMigrator_ValidateMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize and apply V1 only
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	_, err := migrator.ApplyMigration(ctx, 1)
	if err != nil {
		t.Fatalf("ApplyMigration(1) failed: %v", err)
	}

	// Validate with required version 1 (should pass)
	err = migrator.ValidateMigrations(ctx, 1)
	if err != nil {
		t.Errorf("ValidateMigrations(1) should pass, got error: %v", err)
	}

	// Validate with required version 2 (should fail)
	err = migrator.ValidateMigrations(ctx, 2)
	if err == nil {
		t.Error("ValidateMigrations(2) should fail when only V1 is applied")
	} else {
		t.Logf("Expected validation error: %v", err)
	}

	// Apply V2 and validate again
	_, err = migrator.ApplyMigration(ctx, 2)
	if err != nil {
		t.Fatalf("ApplyMigration(2) failed: %v", err)
	}

	err = migrator.ValidateMigrations(ctx, 2)
	if err != nil {
		t.Errorf("ValidateMigrations(2) should pass after applying V2, got error: %v", err)
	}
}

// TestMigrator_IsMigrationApplied tests the IsMigrationApplied method.
func TestMigrator_IsMigrationApplied(t *testing.T) {
	db, cleanup := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanup()

	migrator := NewMigrator(db)
	ctx := context.Background()

	// Initialize
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Check V1 before applying (should be false)
	applied, err := migrator.IsMigrationApplied(ctx, 1)
	if err != nil {
		t.Fatalf("IsMigrationApplied(1) failed: %v", err)
	}
	if applied {
		t.Error("V1 should not be applied yet")
	}

	// Apply V1
	_, err = migrator.ApplyMigration(ctx, 1)
	if err != nil {
		t.Fatalf("ApplyMigration(1) failed: %v", err)
	}

	// Check V1 after applying (should be true)
	applied, err = migrator.IsMigrationApplied(ctx, 1)
	if err != nil {
		t.Fatalf("IsMigrationApplied(1) failed: %v", err)
	}
	if !applied {
		t.Error("V1 should be applied")
	}

	// Check V2 (should still be false)
	applied, err = migrator.IsMigrationApplied(ctx, 2)
	if err != nil {
		t.Fatalf("IsMigrationApplied(2) failed: %v", err)
	}
	if applied {
		t.Error("V2 should not be applied yet")
	}
}

// Example usage of the migrator
func ExampleMigrator() {
	// Connect to database
	db, err := sql.Open("postgres", "postgres://user:pass@localhost/dbname?sslmode=disable")
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer db.Close()

	// Create migrator
	migrator := NewMigrator(db)
	ctx := context.Background()

	// Run migrations
	if err := migrator.Migrate(ctx); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		return
	}

	// Check current version
	version, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		fmt.Printf("Failed to get version: %v\n", err)
		return
	}

	fmt.Printf("Database schema is at version %d\n", version)

	// Get migration status
	statuses, err := migrator.GetMigrationStatus(ctx)
	if err != nil {
		fmt.Printf("Failed to get status: %v\n", err)
		return
	}

	for _, status := range statuses {
		fmt.Printf("V%d: %s (%s)\n", status.Version, status.Name, status.Status)
	}
}
