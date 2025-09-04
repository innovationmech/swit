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

package tracing

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// GormTracingConfig holds configuration for GORM tracing
type GormTracingConfig struct {
	RecordSQL      bool          // Whether to record SQL statements
	SanitizeSQL    bool          // Whether to sanitize SQL parameters
	SlowThreshold  time.Duration // Threshold for marking queries as slow
	SkipOperations []string      // Operations to skip (e.g., "SELECT", "INSERT")
}

// DefaultGormTracingConfig returns default GORM tracing configuration
func DefaultGormTracingConfig() *GormTracingConfig {
	return &GormTracingConfig{
		RecordSQL:      true,
		SanitizeSQL:    true,
		SlowThreshold:  200 * time.Millisecond,
		SkipOperations: []string{},
	}
}

// GormTracing provides GORM tracing functionality
type GormTracing struct {
	tm     TracingManager
	config *GormTracingConfig
}

// NewGormTracing creates a new GORM tracing instance
func NewGormTracing(tm TracingManager, config *GormTracingConfig) *GormTracing {
	if config == nil {
		config = DefaultGormTracingConfig()
	}
	return &GormTracing{
		tm:     tm,
		config: config,
	}
}

// Initialize implements gorm.Plugin interface
func (gt *GormTracing) Initialize(db *gorm.DB) error {
	// Register callbacks for different operations
	if err := gt.registerCallbacks(db); err != nil {
		return fmt.Errorf("failed to register tracing callbacks: %w", err)
	}
	return nil
}

// Name returns the plugin name
func (gt *GormTracing) Name() string {
	return "tracing"
}

// registerCallbacks registers tracing callbacks for all GORM operations
func (gt *GormTracing) registerCallbacks(db *gorm.DB) error {
	// Create callbacks
	createCallback := db.Callback().Create()
	if err := createCallback.Before("gorm:create").Register("tracing:before_create", gt.beforeCreate); err != nil {
		return err
	}
	if err := createCallback.After("gorm:create").Register("tracing:after_create", gt.afterCreate); err != nil {
		return err
	}

	// Query callbacks
	queryCallback := db.Callback().Query()
	if err := queryCallback.Before("gorm:query").Register("tracing:before_query", gt.beforeQuery); err != nil {
		return err
	}
	if err := queryCallback.After("gorm:query").Register("tracing:after_query", gt.afterQuery); err != nil {
		return err
	}

	// Update callbacks
	updateCallback := db.Callback().Update()
	if err := updateCallback.Before("gorm:update").Register("tracing:before_update", gt.beforeUpdate); err != nil {
		return err
	}
	if err := updateCallback.After("gorm:update").Register("tracing:after_update", gt.afterUpdate); err != nil {
		return err
	}

	// Delete callbacks
	deleteCallback := db.Callback().Delete()
	if err := deleteCallback.Before("gorm:delete").Register("tracing:before_delete", gt.beforeDelete); err != nil {
		return err
	}
	if err := deleteCallback.After("gorm:delete").Register("tracing:after_delete", gt.afterDelete); err != nil {
		return err
	}

	// Row callbacks
	rowCallback := db.Callback().Row()
	if err := rowCallback.Before("gorm:row").Register("tracing:before_row", gt.beforeRow); err != nil {
		return err
	}
	if err := rowCallback.After("gorm:row").Register("tracing:after_row", gt.afterRow); err != nil {
		return err
	}

	// Raw callbacks
	rawCallback := db.Callback().Raw()
	if err := rawCallback.Before("gorm:raw").Register("tracing:before_raw", gt.beforeRaw); err != nil {
		return err
	}
	if err := rawCallback.After("gorm:raw").Register("tracing:after_raw", gt.afterRaw); err != nil {
		return err
	}

	return nil
}

// beforeCreate is called before CREATE operations
func (gt *GormTracing) beforeCreate(db *gorm.DB) {
	gt.startSpan(db, "CREATE")
}

// afterCreate is called after CREATE operations
func (gt *GormTracing) afterCreate(db *gorm.DB) {
	gt.finishSpan(db)
}

// beforeQuery is called before SELECT operations
func (gt *GormTracing) beforeQuery(db *gorm.DB) {
	gt.startSpan(db, "SELECT")
}

// afterQuery is called after SELECT operations
func (gt *GormTracing) afterQuery(db *gorm.DB) {
	gt.finishSpan(db)
}

// beforeUpdate is called before UPDATE operations
func (gt *GormTracing) beforeUpdate(db *gorm.DB) {
	gt.startSpan(db, "UPDATE")
}

// afterUpdate is called after UPDATE operations
func (gt *GormTracing) afterUpdate(db *gorm.DB) {
	gt.finishSpan(db)
}

// beforeDelete is called before DELETE operations
func (gt *GormTracing) beforeDelete(db *gorm.DB) {
	gt.startSpan(db, "DELETE")
}

// afterDelete is called after DELETE operations
func (gt *GormTracing) afterDelete(db *gorm.DB) {
	gt.finishSpan(db)
}

// beforeRow is called before ROW operations
func (gt *GormTracing) beforeRow(db *gorm.DB) {
	gt.startSpan(db, "ROW")
}

// afterRow is called after ROW operations
func (gt *GormTracing) afterRow(db *gorm.DB) {
	gt.finishSpan(db)
}

// beforeRaw is called before RAW operations
func (gt *GormTracing) beforeRaw(db *gorm.DB) {
	gt.startSpan(db, "RAW")
}

// afterRaw is called after RAW operations
func (gt *GormTracing) afterRaw(db *gorm.DB) {
	gt.finishSpan(db)
}

// startSpan starts a new database operation span
func (gt *GormTracing) startSpan(db *gorm.DB, operation string) {
	// Skip if operation is in skip list
	for _, skip := range gt.config.SkipOperations {
		if strings.EqualFold(skip, operation) {
			return
		}
	}

	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Extract table name if available
	tableName := ""
	if db.Statement.Table != "" {
		tableName = db.Statement.Table
	} else if db.Statement.Schema != nil {
		tableName = db.Statement.Schema.Table
	}

	operationName := fmt.Sprintf("db:%s", strings.ToLower(operation))
	if tableName != "" {
		operationName = fmt.Sprintf("db:%s %s", strings.ToLower(operation), tableName)
	}

	// Start the span
	ctx, span := gt.tm.StartSpan(
		ctx,
		operationName,
		WithSpanKind(trace.SpanKindClient),
		WithAttributes(
			semconv.DBSystemMySQL, // Assuming MySQL, can be made configurable
			attribute.String("db.operation", operation),
		),
	)

	// Add table name if available
	if tableName != "" {
		span.SetAttribute("db.sql.table", tableName)
	}

	// Store span in context for later use
	db.Statement.Context = ctx
	db.Set("tracing:span", span)
	db.Set("tracing:start_time", time.Now())
}

// finishSpan finishes the database operation span
func (gt *GormTracing) finishSpan(db *gorm.DB) {
	spanValue, exists := db.Get("tracing:span")
	if !exists {
		return
	}

	span, ok := spanValue.(Span)
	if !ok {
		return
	}
	defer span.End()

	// Record execution time
	if startTime, exists := db.Get("tracing:start_time"); exists {
		if st, ok := startTime.(time.Time); ok {
			duration := time.Since(st)
			span.SetAttribute("db.duration_ms", float64(duration.Nanoseconds())/1e6)

			// Mark as slow if exceeds threshold
			if duration > gt.config.SlowThreshold {
				span.SetAttribute("db.slow_query", true)
				span.AddEvent("slow_query", trace.WithAttributes(
					attribute.String("db.slow_threshold", gt.config.SlowThreshold.String()),
				))
			}
		}
	}

	// Record SQL statement if configured
	if gt.config.RecordSQL && db.Statement.SQL.String() != "" {
		sql := db.Statement.SQL.String()
		if gt.config.SanitizeSQL {
			sql = gt.sanitizeSQL(sql)
		}
		span.SetAttribute("db.statement", sql)
	}

	// Record rows affected
	if db.Statement.RowsAffected >= 0 {
		span.SetAttribute("db.rows_affected", db.Statement.RowsAffected)
	}

	// Record database name if available
	if db.Migrator().CurrentDatabase() != "" {
		span.SetAttribute("db.name", db.Migrator().CurrentDatabase())
	}

	// Handle errors
	if db.Error != nil {
		span.RecordError(db.Error)
		span.SetStatus(codes.Error, db.Error.Error())

		// Add error event
		span.AddEvent("db.error", trace.WithAttributes(
			attribute.String("db.error.message", db.Error.Error()),
		))
	} else {
		span.SetStatus(codes.Ok, "")
	}

	// Add connection info if available
	if sqlDB, err := db.DB(); err == nil {
		stats := sqlDB.Stats()
		span.SetAttribute("db.connections.open", stats.OpenConnections)
		span.SetAttribute("db.connections.idle", stats.Idle)
		span.SetAttribute("db.connections.in_use", stats.InUse)
		span.SetAttribute("db.connections.wait_count", stats.WaitCount)
		span.SetAttribute("db.connections.wait_duration_ms", float64(stats.WaitDuration.Nanoseconds())/1e6)
	}
}

// sanitizeSQL removes or replaces sensitive information in SQL statements
func (gt *GormTracing) sanitizeSQL(sql string) string {
	// Replace parameter placeholders with ? to avoid logging sensitive data
	paramRegex := regexp.MustCompile(`\$\d+|\?`)
	sanitized := paramRegex.ReplaceAllString(sql, "?")

	// Remove extra whitespace
	spaceRegex := regexp.MustCompile(`\s+`)
	sanitized = spaceRegex.ReplaceAllString(sanitized, " ")

	return strings.TrimSpace(sanitized)
}

// InstallGormTracing installs tracing on a GORM database instance
func InstallGormTracing(db *gorm.DB, tm TracingManager, config *GormTracingConfig) error {
	gormTracing := NewGormTracing(tm, config)
	return db.Use(gormTracing)
}

// GormDBWithTracing wraps a GORM DB instance with tracing context
func GormDBWithTracing(db *gorm.DB, ctx context.Context) *gorm.DB {
	return db.WithContext(ctx)
}
