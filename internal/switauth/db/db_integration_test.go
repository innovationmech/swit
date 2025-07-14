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

//go:build integration
// +build integration

package db

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/innovationmech/swit/internal/switauth/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Reset the singleton state for testing
func resetSingleton() {
	dbConn = nil
	once = sync.Once{}
}

// TestDB is a test database implementation
func setupTestDB(t *testing.T) func() {
	// Create a temporary SQLite database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Store the original database connection
	originalDB := dbConn
	originalOnce := once

	// Replace with our test database
	dbConn = db
	once = sync.Once{}

	// Return cleanup function
	return func() {
		// Restore original state
		dbConn = originalDB
		once = originalOnce

		// Close test database
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

func TestGetDB_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Only run this test on CI or when explicitly requested
	if os.Getenv("RUN_DB_INTEGRATION_TESTS") != "true" {
		t.Skip("skipping integration test; set RUN_DB_INTEGRATION_TESTS=true to run")
	}

	resetSingleton()

	cleanup := setupTestDB(t)
	defer cleanup()

	// Test that we can get a database instance
	db := GetDB()
	if db == nil {
		t.Error("Expected non-nil database instance")
	}

	// Test that it's the same instance on subsequent calls
	db2 := GetDB()
	if db != db2 {
		t.Error("Expected singleton pattern - should return same instance")
	}
}

func TestGetDB_SingletonPatternIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	resetSingleton()

	cleanup := setupTestDB(t)
	defer cleanup()

	// Test the singleton pattern with actual database
	var dbs []*gorm.DB

	// Call GetDB multiple times from different goroutines
	var wg sync.WaitGroup
	results := make(chan *gorm.DB, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- GetDB()
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	for db := range results {
		dbs = append(dbs, db)
	}

	// All instances should be the same
	if len(dbs) != 10 {
		t.Errorf("Expected 10 database instances, got %d", len(dbs))
	}

	first := dbs[0]
	for i, db := range dbs {
		if db != first {
			t.Errorf("Database instance %d is different from first instance", i)
		}
	}
}

func TestDatabaseConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	resetSingleton()

	cleanup := setupTestDB(t)
	defer cleanup()

	// Test that the database connection is actually usable
	db := GetDB()

	// Try to execute a simple query
	result := db.Exec("SELECT 1")
	if result.Error != nil {
		t.Errorf("Failed to execute simple query: %v", result.Error)
	}

	// Test that we can create a table and query it
	err := db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)").Error
	if err != nil {
		t.Errorf("Failed to create test table: %v", err)
	}

	err = db.Exec("INSERT INTO test_table (id, name) VALUES (?, ?)", 1, "test").Error
	if err != nil {
		t.Errorf("Failed to insert test data: %v", err)
	}

	var name string
	err = db.Raw("SELECT name FROM test_table WHERE id = ?", 1).Scan(&name).Error
	if err != nil {
		t.Errorf("Failed to query test data: %v", err)
	}

	if name != "test" {
		t.Errorf("Expected name 'test', got '%s'", name)
	}
}
