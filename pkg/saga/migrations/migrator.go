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

// Package migrations provides database migration management for Saga storage.
package migrations

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"
)

//go:embed saga_migrations.sql
var migrationScript string

// Migrator handles database schema migrations for Saga storage.
type Migrator struct {
	db *sql.DB
}

// MigrationStatus represents the status of a migration.
type MigrationStatus struct {
	Version           int       `json:"version"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Status            string    `json:"status"`
	AppliedAt         time.Time `json:"applied_at"`
	AppliedBy         string    `json:"applied_by"`
	ExecutionTimeMs   int       `json:"execution_time_ms"`
	RollbackAvailable bool      `json:"rollback_available"`
}

// NewMigrator creates a new database migrator.
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db: db,
	}
}

// Initialize sets up the migration infrastructure by executing the migration script.
// This creates the saga_migrations table and all migration functions.
func (m *Migrator) Initialize(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, migrationScript)
	if err != nil {
		return fmt.Errorf("failed to initialize migration infrastructure: %w", err)
	}
	return nil
}

// GetCurrentVersion returns the current schema version.
func (m *Migrator) GetCurrentVersion(ctx context.Context) (int, error) {
	var version int
	err := m.db.QueryRowContext(ctx, "SELECT get_current_schema_version()").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current schema version: %w", err)
	}
	return version, nil
}

// IsMigrationApplied checks if a specific migration version has been applied.
func (m *Migrator) IsMigrationApplied(ctx context.Context, version int) (bool, error) {
	var applied bool
	err := m.db.QueryRowContext(ctx, "SELECT is_migration_applied($1)", version).Scan(&applied)
	if err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	return applied, nil
}

// ApplyMigration applies a specific migration version.
func (m *Migrator) ApplyMigration(ctx context.Context, version int) (string, error) {
	var result string
	query := fmt.Sprintf("SELECT apply_migration_v%d()", version)
	err := m.db.QueryRowContext(ctx, query).Scan(&result)
	if err != nil {
		return "", fmt.Errorf("failed to apply migration V%d: %w", version, err)
	}
	return result, nil
}

// RollbackMigration rolls back a specific migration version.
func (m *Migrator) RollbackMigration(ctx context.Context, version int) (string, error) {
	var result string
	query := fmt.Sprintf("SELECT rollback_migration_v%d()", version)
	err := m.db.QueryRowContext(ctx, query).Scan(&result)
	if err != nil {
		return "", fmt.Errorf("failed to rollback migration V%d: %w", version, err)
	}
	return result, nil
}

// ApplyAllMigrations applies all pending migrations.
func (m *Migrator) ApplyAllMigrations(ctx context.Context) (string, error) {
	var result string
	err := m.db.QueryRowContext(ctx, "SELECT apply_all_migrations()").Scan(&result)
	if err != nil {
		return "", fmt.Errorf("failed to apply all migrations: %w", err)
	}
	return result, nil
}

// GetMigrationStatus returns the status of all migrations.
func (m *Migrator) GetMigrationStatus(ctx context.Context) ([]MigrationStatus, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT * FROM get_migration_status()")
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}
	defer rows.Close()

	var statuses []MigrationStatus
	for rows.Next() {
		var status MigrationStatus
		err := rows.Scan(
			&status.Version,
			&status.Name,
			&status.Description,
			&status.Status,
			&status.AppliedAt,
			&status.AppliedBy,
			&status.ExecutionTimeMs,
			&status.RollbackAvailable,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration status: %w", err)
		}
		statuses = append(statuses, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration status: %w", err)
	}

	return statuses, nil
}

// Migrate performs automatic migration to the latest version.
// This is the main entry point for automatic schema migration.
func (m *Migrator) Migrate(ctx context.Context) error {
	// First, ensure migration infrastructure is set up
	if err := m.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration system: %w", err)
	}

	// Apply all pending migrations
	result, err := m.ApplyAllMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Log result (in production, use proper logger)
	fmt.Println(result)

	return nil
}

// ValidateMigrations verifies that all required migrations have been applied.
// Returns an error if any required migration is missing.
func (m *Migrator) ValidateMigrations(ctx context.Context, requiredVersion int) error {
	currentVersion, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion < requiredVersion {
		return fmt.Errorf(
			"schema version mismatch: required version %d, current version %d. "+
				"Please run migrations to upgrade the schema",
			requiredVersion, currentVersion,
		)
	}

	return nil
}
