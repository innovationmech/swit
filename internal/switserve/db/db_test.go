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

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestGetDB tests the GetDB function to ensure it returns a singleton database connection.
func TestGetDB(t *testing.T) {
	// IMPORTANT: We cannot safely reset sync.Once due to race conditions
	// Instead, we use a test-specific approach by mocking the factory function

	// Store original values
	originalNewDbConn := newDbConn
	originalDBConn := dbConn

	// Reset only dbConn, let sync.Once handle its own state
	dbConn = nil

	// Replace the newDbConn factory with a mock
	newDbConn = func() (*gorm.DB, error) {
		return &gorm.DB{}, nil
	}

	// Ensure we restore the original values
	defer func() {
		newDbConn = originalNewDbConn
		dbConn = originalDBConn
	}()

	// Call GetDB multiple times
	db1 := GetDB()
	db2 := GetDB()

	// Assert that the DB connection is not nil
	assert.NotNil(t, db1, "GetDB() should not return a nil connection")

	// Assert that both calls return the same instance
	assert.Same(t, db1, db2, "GetDB() should return the same instance on subsequent calls")
}
