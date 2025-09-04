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
	"fmt"
	"sync"

	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/pkg/tracing"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// dbConn is the global database connection for the application.
	dbConn *gorm.DB
	// once is used to ensure that the database connection is only initialized once.
	once sync.Once
	// newDbConn is a factory function for creating a database connection.
	// It's a variable so it can be replaced by a mock in tests.
	newDbConn = func() (*gorm.DB, error) {
		cfg := config.GetConfig()
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
			cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	}
)

// GetDB returns the global database connection for the application.
func GetDB() *gorm.DB {
	once.Do(func() {
		db, err := newDbConn()
		if err != nil {
			panic("fail to connect database")
		}
		dbConn = db
	})
	return dbConn
}

// SetupTracing installs tracing on the database instance
func SetupTracing(db *gorm.DB, tracingManager tracing.TracingManager) error {
	if tracingManager == nil {
		return nil // Skip tracing setup if no manager provided
	}

	config := tracing.DefaultGormTracingConfig()
	return tracing.InstallGormTracing(db, tracingManager, config)
}
