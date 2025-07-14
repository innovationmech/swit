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
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/innovationmech/swit/internal/switserve/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Reset the singleton state for testing
func resetSingleton() {
	dbConn = nil
	once = sync.Once{}
	newDbConn = func() (*gorm.DB, error) {
		cfg := config.GetConfig()
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
			cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
}

// setupMockDB creates a mock database for testing
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	// Create a GORM database with the mock
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create GORM database: %v", err)
	}

	// Reset singleton
	resetSingleton()

	// Replace newDbConn with our mock
	originalNewDbConn := newDbConn
	newDbConn = func() (*gorm.DB, error) {
		return gormDB, nil
	}

	// Return cleanup function
	return gormDB, mock, func() {
		newDbConn = originalNewDbConn
		resetSingleton()
		db.Close()
	}
}

func TestGetDB_Success(t *testing.T) {
	resetSingleton()

	// Setup mock database
	_, mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Expect the database ping to succeed
	mock.ExpectPing()

	// Test the function
	db := GetDB()
	if db == nil {
		t.Error("Expected non-nil database instance")
	}

	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetDB_SingletonPattern(t *testing.T) {
	resetSingleton()

	// Setup mock database
	_, mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Expect the database ping to succeed (only once due to singleton)
	mock.ExpectPing()

	// Call GetDB multiple times
	db1 := GetDB()
	db2 := GetDB()
	db3 := GetDB()

	// Verify they all return the same instance
	if db1 == nil || db2 == nil || db3 == nil {
		t.Error("Expected non-nil database instances")
	}
	if db1 != db2 || db2 != db3 {
		t.Error("Expected singleton pattern - all calls should return the same instance")
	}

	// Verify the mock was called exactly once
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetDB_Failure(t *testing.T) {
	resetSingleton()

	// Replace newDbConn to return error
	originalNewDbConn := newDbConn
	newDbConn = func() (*gorm.DB, error) {
		return nil, errors.New("mock connection error")
	}
	defer func() { newDbConn = originalNewDbConn }()

	// Test that the function panics on connection failure
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on database connection failure")
		} else if r != "fail to connect database" {
			t.Errorf("Expected panic message 'fail to connect database', got: %v", r)
		}
	}()

	GetDB()
}

func TestGetDB_ConcurrentAccess(t *testing.T) {
	resetSingleton()

	// Setup mock database
	_, mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Expect the database ping to succeed (only once due to singleton)
	mock.ExpectPing()

	// Test concurrent access
	const numGoroutines = 10
	results := make(chan *gorm.DB, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
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
	var firstDB *gorm.DB
	count := 0
	for db := range results {
		if firstDB == nil {
			firstDB = db
		}
		if db != firstDB {
			t.Error("Expected all goroutines to get the same database instance")
		}
		count++
	}

	if count != numGoroutines {
		t.Errorf("Expected %d results, got %d", numGoroutines, count)
	}

	// Verify the mock was called exactly once
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetDB_DatabaseOperations(t *testing.T) {
	resetSingleton()

	// Setup mock database
	_, mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Set up expectations for table creation and queries
	mock.ExpectExec("CREATE TABLE users").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT name FROM users WHERE id = ?").WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("test"))

	// Get the database instance
	db := GetDB()
	if db == nil {
		t.Error("Expected non-nil database instance")
	}

	// Test that we can use the database for operations
	err := db.Exec("CREATE TABLE users (id INT, name VARCHAR(255))").Error
	if err != nil {
		t.Errorf("Failed to create table: %v", err)
	}

	var name string
	err = db.Raw("SELECT name FROM users WHERE id = ?", 1).Scan(&name).Error
	if err != nil {
		t.Errorf("Failed to query data: %v", err)
	}

	if name != "test" {
		t.Errorf("Expected name 'test', got '%s'", name)
	}

	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetDB_RaceCondition(t *testing.T) {
	resetSingleton()

	// Setup mock database
	_, mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Expect the database ping to succeed (only once due to singleton)
	mock.ExpectPing()

	// Test race condition with multiple goroutines
	var wg sync.WaitGroup
	var instances []*gorm.DB
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db := GetDB()

			mu.Lock()
			instances = append(instances, db)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Verify all instances are the same
	if len(instances) != 100 {
		t.Errorf("Expected 100 instances, got %d", len(instances))
	}

	first := instances[0]
	for i, db := range instances {
		if db != first {
			t.Errorf("Instance %d is different from first instance", i)
		}
	}

	// Verify the mock was called exactly once
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
