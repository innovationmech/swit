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

// Package main provides a CLI tool for managing Saga database migrations.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/innovationmech/swit/pkg/saga/migrations"
)

const (
	defaultDSN     = "postgres://postgres:postgres@localhost:5432/saga?sslmode=disable"
	defaultTimeout = 30 * time.Second
)

func main() {
	var (
		dsn        string
		action     string
		version    int
		showHelp   bool
		timeoutSec int
	)

	flag.StringVar(&dsn, "dsn", "", "PostgreSQL connection string (e.g., postgres://user:pass@host:port/dbname)")
	flag.StringVar(&action, "action", "migrate", "Action to perform: migrate, status, init, apply, rollback, validate")
	flag.IntVar(&version, "version", 0, "Version number for apply/rollback actions")
	flag.IntVar(&timeoutSec, "timeout", 30, "Operation timeout in seconds")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Get DSN from flag or environment
	if dsn == "" {
		dsn = os.Getenv("SAGA_DSN")
		if dsn == "" {
			dsn = os.Getenv("DATABASE_URL")
			if dsn == "" {
				fmt.Fprintf(os.Stderr, "Error: DSN not provided. Use -dsn flag or SAGA_DSN/DATABASE_URL env var\n")
				fmt.Fprintf(os.Stderr, "Example: %s\n", defaultDSN)
				os.Exit(1)
			}
		}
	}

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database connection: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected to database: %s\n", sanitizeDSN(dsn))

	// Create migrator
	migrator := migrations.NewMigrator(db)

	// Execute action
	switch strings.ToLower(action) {
	case "init", "initialize":
		if err := runInit(ctx, migrator); err != nil {
			fmt.Fprintf(os.Stderr, "Initialization failed: %v\n", err)
			os.Exit(1)
		}

	case "migrate":
		if err := runMigrate(ctx, migrator); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}

	case "status":
		if err := runStatus(ctx, migrator); err != nil {
			fmt.Fprintf(os.Stderr, "Status check failed: %v\n", err)
			os.Exit(1)
		}

	case "apply":
		if version <= 0 {
			fmt.Fprintf(os.Stderr, "Error: -version flag is required for apply action\n")
			os.Exit(1)
		}
		if err := runApply(ctx, migrator, version); err != nil {
			fmt.Fprintf(os.Stderr, "Apply failed: %v\n", err)
			os.Exit(1)
		}

	case "rollback":
		if version <= 0 {
			fmt.Fprintf(os.Stderr, "Error: -version flag is required for rollback action\n")
			os.Exit(1)
		}
		if err := runRollback(ctx, migrator, version); err != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
			os.Exit(1)
		}

	case "validate":
		requiredVersion := version
		if requiredVersion <= 0 {
			requiredVersion = 2 // Default to latest version
		}
		if err := runValidate(ctx, migrator, requiredVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", action)
		fmt.Fprintf(os.Stderr, "Valid actions: init, migrate, status, apply, rollback, validate\n")
		os.Exit(1)
	}

	fmt.Println("\n✓ Operation completed successfully")
}

func runInit(ctx context.Context, migrator *migrations.Migrator) error {
	fmt.Println("Initializing migration system...")
	if err := migrator.Initialize(ctx); err != nil {
		return err
	}
	fmt.Println("Migration system initialized")
	return nil
}

func runMigrate(ctx context.Context, migrator *migrations.Migrator) error {
	fmt.Println("Running database migrations...")

	// Show current version
	currentVersion, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}
	fmt.Printf("Current schema version: %d\n", currentVersion)

	// Apply all migrations
	result, err := migrator.ApplyAllMigrations(ctx)
	if err != nil {
		return err
	}

	fmt.Println("\nMigration result:")
	fmt.Println(result)

	// Show new version
	newVersion, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get new version: %w", err)
	}
	fmt.Printf("\nNew schema version: %d\n", newVersion)

	return nil
}

func runStatus(ctx context.Context, migrator *migrations.Migrator) error {
	fmt.Println("Checking migration status...\n")

	// Get current version
	currentVersion, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	fmt.Printf("Current Schema Version: %d\n\n", currentVersion)

	// Get migration statuses
	statuses, err := migrator.GetMigrationStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	if len(statuses) == 0 {
		fmt.Println("No migrations have been applied yet.")
		fmt.Println("Run with -action=migrate to apply all migrations.")
		return nil
	}

	fmt.Println("Migration History:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-8s %-30s %-12s %-20s %s\n", "Version", "Name", "Status", "Applied At", "Time")
	fmt.Println(strings.Repeat("-", 80))

	for _, status := range statuses {
		appliedAt := status.AppliedAt.Format("2006-01-02 15:04:05")
		fmt.Printf("%-8d %-30s %-12s %-20s %dms\n",
			status.Version,
			truncate(status.Name, 30),
			status.Status,
			appliedAt,
			status.ExecutionTimeMs,
		)
	}
	fmt.Println(strings.Repeat("-", 80))

	return nil
}

func runApply(ctx context.Context, migrator *migrations.Migrator, version int) error {
	fmt.Printf("Applying migration V%d...\n", version)

	// Check if already applied
	applied, err := migrator.IsMigrationApplied(ctx, version)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if applied {
		fmt.Printf("Migration V%d is already applied\n", version)
		return nil
	}

	// Apply migration
	result, err := migrator.ApplyMigration(ctx, version)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func runRollback(ctx context.Context, migrator *migrations.Migrator, version int) error {
	fmt.Printf("⚠️  WARNING: Rolling back migration V%d\n", version)
	fmt.Println("This may result in data loss!")
	fmt.Print("Are you sure? Type 'yes' to continue: ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if strings.ToLower(confirmation) != "yes" {
		fmt.Println("Rollback cancelled")
		return nil
	}

	fmt.Printf("Rolling back migration V%d...\n", version)

	result, err := migrator.RollbackMigration(ctx, version)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func runValidate(ctx context.Context, migrator *migrations.Migrator, requiredVersion int) error {
	fmt.Printf("Validating schema version (required: %d)...\n", requiredVersion)

	currentVersion, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	fmt.Printf("Current version: %d\n", currentVersion)

	if err := migrator.ValidateMigrations(ctx, requiredVersion); err != nil {
		return err
	}

	fmt.Printf("✓ Schema version is valid (>= %d)\n", requiredVersion)
	return nil
}

func printHelp() {
	fmt.Println("Saga Database Migration Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  saga-migrate [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -dsn string")
	fmt.Println("        PostgreSQL connection string (required)")
	fmt.Println("        Can also be set via SAGA_DSN or DATABASE_URL env var")
	fmt.Println("        Example: postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	fmt.Println()
	fmt.Println("  -action string")
	fmt.Println("        Action to perform (default: migrate)")
	fmt.Println("        Available actions:")
	fmt.Println("          init      - Initialize migration system")
	fmt.Println("          migrate   - Apply all pending migrations")
	fmt.Println("          status    - Show migration status")
	fmt.Println("          apply     - Apply a specific migration (requires -version)")
	fmt.Println("          rollback  - Rollback a migration (requires -version)")
	fmt.Println("          validate  - Validate schema version (optional -version)")
	fmt.Println()
	fmt.Println("  -version int")
	fmt.Println("        Migration version number (for apply/rollback/validate actions)")
	fmt.Println()
	fmt.Println("  -timeout int")
	fmt.Println("        Operation timeout in seconds (default: 30)")
	fmt.Println()
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Initialize and run all migrations")
	fmt.Println("  saga-migrate -dsn 'postgres://localhost/saga' -action migrate")
	fmt.Println()
	fmt.Println("  # Check status")
	fmt.Println("  saga-migrate -dsn 'postgres://localhost/saga' -action status")
	fmt.Println()
	fmt.Println("  # Apply specific migration")
	fmt.Println("  saga-migrate -dsn 'postgres://localhost/saga' -action apply -version 2")
	fmt.Println()
	fmt.Println("  # Rollback migration")
	fmt.Println("  saga-migrate -dsn 'postgres://localhost/saga' -action rollback -version 2")
	fmt.Println()
	fmt.Println("  # Validate version")
	fmt.Println("  saga-migrate -dsn 'postgres://localhost/saga' -action validate -version 2")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  SAGA_DSN         - PostgreSQL connection string")
	fmt.Println("  DATABASE_URL     - Alternative connection string (fallback)")
	fmt.Println()
}

// sanitizeDSN removes sensitive information from DSN for display
func sanitizeDSN(dsn string) string {
	// Simple sanitization - hide password
	parts := strings.Split(dsn, "@")
	if len(parts) != 2 {
		return dsn
	}

	userParts := strings.Split(parts[0], "://")
	if len(userParts) != 2 {
		return dsn
	}

	protocol := userParts[0]
	credentials := userParts[1]

	credParts := strings.Split(credentials, ":")
	if len(credParts) != 2 {
		return dsn
	}

	return fmt.Sprintf("%s://%s:***@%s", protocol, credParts[0], parts[1])
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
