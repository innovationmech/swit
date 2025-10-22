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

package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/security"
)

// RunAuditExamples demonstrates how to use the Saga audit logging system
func RunAuditExamples() {
	fmt.Println("=== Saga Audit Logging Examples ===")

	// Example 1: Basic audit logging with memory storage
	basicAuditExample()

	// Example 2: File-based audit logging
	fileAuditExample()

	// Example 3: Audit logging with authentication context
	auditWithAuthExample()

	// Example 4: Querying audit logs
	queryAuditLogsExample()

	// Example 5: Saga state change audit hook
	sagaAuditHookExample()

	// Example 6: Custom audit entry enrichment
	auditEnrichmentExample()
}

func basicAuditExample() {
	fmt.Println("--- Example 1: Basic Audit Logging ---")

	// Create memory-based audit storage
	storage := security.NewMemoryAuditStorage()

	// Create audit logger
	auditLogger, err := security.NewAuditLogger(&security.AuditLoggerConfig{
		Storage: storage,
		Source:  "order-service",
	})
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	ctx := context.Background()

	// Log a saga start event
	err = auditLogger.LogAction(
		ctx,
		security.AuditLevelInfo,
		security.AuditActionSagaStarted,
		"saga",
		"order-saga-123",
		"Order saga started successfully",
		map[string]interface{}{
			"order_id":      "ORD-123",
			"customer_id":   "CUST-456",
			"total_amount":  99.99,
			"definition_id": "order-processing-saga",
		},
	)
	if err != nil {
		log.Printf("Failed to log audit entry: %v", err)
		return
	}

	// Log a saga completion event
	err = auditLogger.LogAction(
		ctx,
		security.AuditLevelInfo,
		security.AuditActionSagaCompleted,
		"saga",
		"order-saga-123",
		"Order saga completed successfully",
		map[string]interface{}{
			"order_id":       "ORD-123",
			"duration_ms":    1500,
			"steps_executed": 5,
		},
	)
	if err != nil {
		log.Printf("Failed to log audit entry: %v", err)
		return
	}

	fmt.Println("✓ Audit entries logged successfully")

	// Query all entries
	entries, err := storage.Query(ctx, nil)
	if err != nil {
		log.Printf("Failed to query audit entries: %v", err)
		return
	}

	fmt.Printf("✓ Total audit entries: %d\n", len(entries))
}

func fileAuditExample() {
	fmt.Println("--- Example 2: File-Based Audit Logging ---")

	// Create temporary directory for audit logs
	tmpDir := "/tmp/saga-audit-logs"
	auditFilePath := filepath.Join(tmpDir, "saga-audit.log")

	// Create file-based audit storage
	storage, err := security.NewFileAuditStorage(&security.FileAuditStorageConfig{
		FilePath:    auditFilePath,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxBackups:  5,
	})
	if err != nil {
		log.Fatalf("Failed to create file audit storage: %v", err)
	}

	// Create audit logger
	auditLogger, err := security.NewAuditLogger(&security.AuditLoggerConfig{
		Storage: storage,
		Source:  "payment-service",
	})
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	ctx := context.Background()

	// Log multiple events
	events := []struct {
		level  security.AuditLevel
		action security.AuditAction
		sagaID string
		msg    string
	}{
		{security.AuditLevelInfo, security.AuditActionSagaStarted, "payment-saga-001", "Payment processing started"},
		{security.AuditLevelInfo, security.AuditActionStepCompleted, "payment-saga-001", "Authorization step completed"},
		{security.AuditLevelInfo, security.AuditActionStepCompleted, "payment-saga-001", "Capture step completed"},
		{security.AuditLevelInfo, security.AuditActionSagaCompleted, "payment-saga-001", "Payment processing completed"},
	}

	for _, event := range events {
		err = auditLogger.LogAction(ctx, event.level, event.action, "saga", event.sagaID, event.msg, nil)
		if err != nil {
			log.Printf("Failed to log event: %v", err)
			continue
		}
	}

	fmt.Printf("✓ Audit entries written to file: %s\n", auditFilePath)
	fmt.Println("✓ Log rotation and archiving configured")
}

func auditWithAuthExample() {
	fmt.Println("--- Example 3: Audit Logging with Authentication Context ---")

	storage := security.NewMemoryAuditStorage()
	auditLogger, err := security.NewAuditLogger(&security.AuditLoggerConfig{
		Storage: storage,
		Source:  "inventory-service",
	})
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	// Create authentication context
	authCtx := &security.AuthContext{
		Credentials: &security.AuthCredentials{
			Type:   security.AuthTypeAPIKey,
			APIKey: "api-key-12345",
			UserID: "user-admin",
			Scopes: []string{"saga:execute", "saga:compensate"},
		},
		Timestamp:     time.Now(),
		SagaID:        "inventory-saga-456",
		CorrelationID: "corr-789",
		Source:        "api-gateway",
	}

	// Add auth context to request context
	ctx := security.ContextWithAuth(context.Background(), authCtx)

	// Log authentication event
	err = auditLogger.LogAuthEvent(
		ctx,
		security.AuditActionAuthSuccess,
		"user-admin",
		true,
		map[string]interface{}{
			"auth_type":  "api_key",
			"ip":         "192.168.1.100",
			"user_agent": "SagaClient/1.0",
		},
	)
	if err != nil {
		log.Printf("Failed to log auth event: %v", err)
		return
	}

	// Log saga operation with auth context
	err = auditLogger.LogAction(
		ctx,
		security.AuditLevelInfo,
		security.AuditActionSagaStarted,
		"saga",
		"inventory-saga-456",
		"Inventory update saga started",
		map[string]interface{}{
			"operation":  "restock",
			"item_count": 100,
		},
	)
	if err != nil {
		log.Printf("Failed to log audit entry: %v", err)
		return
	}

	fmt.Println("✓ Audit entries with authentication context logged")

	// Query entries and show user info
	entries, err := storage.Query(ctx, nil)
	if err != nil {
		log.Printf("Failed to query entries: %v", err)
		return
	}

	for _, entry := range entries {
		fmt.Printf("  - Action: %s, User: %s, TraceID: %s\n",
			entry.Action, entry.UserID, entry.TraceID)
	}
}

func queryAuditLogsExample() {
	fmt.Println("--- Example 4: Querying Audit Logs ---")

	storage := security.NewMemoryAuditStorage()
	auditLogger, err := security.NewAuditLogger(&security.AuditLoggerConfig{
		Storage: storage,
		Source:  "shipping-service",
	})
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	ctx := context.Background()

	// Log various events
	testData := []struct {
		level  security.AuditLevel
		action security.AuditAction
		userID string
	}{
		{security.AuditLevelInfo, security.AuditActionSagaStarted, "user-1"},
		{security.AuditLevelInfo, security.AuditActionSagaCompleted, "user-1"},
		{security.AuditLevelWarning, security.AuditActionStepFailed, "user-2"},
		{security.AuditLevelError, security.AuditActionSagaFailed, "user-2"},
		{security.AuditLevelInfo, security.AuditActionSagaStarted, "user-3"},
	}

	for i, data := range testData {
		err = auditLogger.LogAction(
			ctx,
			data.level,
			data.action,
			"saga",
			fmt.Sprintf("saga-%d", i),
			"Test event",
			map[string]interface{}{"test": true},
		)
		if err != nil {
			log.Printf("Failed to log entry: %v", err)
		}
	}

	fmt.Println("Query Examples:")

	// Query 1: All error level entries
	fmt.Println("\n1. Query error level entries:")
	entries, err := auditLogger.Query(ctx, &security.AuditFilter{
		Levels: []security.AuditLevel{security.AuditLevelError},
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("   Found %d error entries\n", len(entries))
	}

	// Query 2: Filter by action
	fmt.Println("\n2. Query saga start events:")
	entries, err = auditLogger.Query(ctx, &security.AuditFilter{
		Actions: []security.AuditAction{security.AuditActionSagaStarted},
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("   Found %d saga start events\n", len(entries))
	}

	// Query 3: Filter by user
	fmt.Println("\n3. Query entries for user-1:")
	entries, err = auditLogger.Query(ctx, &security.AuditFilter{
		UserID: "user-1",
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("   Found %d entries for user-1\n", len(entries))
	}

	// Query 4: Time range query
	fmt.Println("\n4. Query recent entries (last hour):")
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	entries, err = auditLogger.Query(ctx, &security.AuditFilter{
		StartTime: &oneHourAgo,
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("   Found %d recent entries\n", len(entries))
	}

	// Query 5: Pagination
	fmt.Println("\n5. Query with pagination (limit 2, offset 1):")
	entries, err = auditLogger.Query(ctx, &security.AuditFilter{
		Limit:  2,
		Offset: 1,
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("   Retrieved %d entries\n", len(entries))
	}

	// Count total entries
	count, err := auditLogger.Count(ctx, nil)
	if err != nil {
		log.Printf("Count failed: %v", err)
	} else {
		fmt.Printf("\n✓ Total audit entries: %d\n", count)
	}
}

func sagaAuditHookExample() {
	fmt.Println("--- Example 5: Saga State Change Audit Hook ---")

	storage := security.NewMemoryAuditStorage()
	auditLogger, err := security.NewAuditLogger(&security.AuditLoggerConfig{
		Storage: storage,
		Source:  "saga-coordinator",
	})
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	// Create saga audit hook
	auditHook := security.NewSagaAuditHook(auditLogger)

	// Simulate saga state changes
	ctx := context.Background()
	sagaID := "order-saga-999"

	// State change 1: Pending -> Running
	auditHook.OnStateChange(ctx, sagaID, saga.StatePending, saga.StateRunning, map[string]interface{}{
		"trigger":    "api_request",
		"request_id": "req-123",
	})

	// State change 2: Running -> StepCompleted
	auditHook.OnStateChange(ctx, sagaID, saga.StateRunning, saga.StateStepCompleted, map[string]interface{}{
		"step_id":   "step-1",
		"step_name": "validate_order",
	})

	// State change 3: StepCompleted -> Completed
	auditHook.OnStateChange(ctx, sagaID, saga.StateStepCompleted, saga.StateCompleted, map[string]interface{}{
		"duration_ms": 2500,
		"steps_count": 3,
	})

	fmt.Println("✓ Saga state changes captured in audit log")

	// Query state change events
	entries, err := auditLogger.Query(ctx, &security.AuditFilter{
		Actions:    []security.AuditAction{security.AuditActionStateChanged},
		ResourceID: sagaID,
	})
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Printf("✓ Recorded %d state transitions:\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  - %s → %s at %s\n",
			entry.OldState, entry.NewState, entry.Timestamp.Format("15:04:05"))
	}
}

func auditEnrichmentExample() {
	fmt.Println("--- Example 6: Custom Audit Entry Enrichment ---")

	storage := security.NewMemoryAuditStorage()

	// Create enricher function to add custom metadata
	enricher := func(entry *security.AuditEntry) {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]string)
		}
		// Add environment info
		entry.Metadata["environment"] = "production"
		entry.Metadata["region"] = "us-east-1"
		entry.Metadata["version"] = "1.2.3"
		entry.Metadata["hostname"] = "saga-node-01"
	}

	auditLogger, err := security.NewAuditLogger(&security.AuditLoggerConfig{
		Storage:  storage,
		Source:   "notification-service",
		Enricher: enricher,
	})
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	ctx := context.Background()

	// Log an event
	err = auditLogger.LogAction(
		ctx,
		security.AuditLevelInfo,
		security.AuditActionSagaStarted,
		"saga",
		"notification-saga-111",
		"Notification saga started",
		map[string]interface{}{
			"notification_type": "email",
			"recipients":        5,
		},
	)
	if err != nil {
		log.Printf("Failed to log entry: %v", err)
		return
	}

	fmt.Println("✓ Audit entry enriched with custom metadata")

	// Retrieve and display enriched entry
	entries, err := storage.Query(ctx, nil)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	if len(entries) > 0 {
		entry := entries[0]
		fmt.Println("✓ Enriched metadata:")
		for key, value := range entry.Metadata {
			fmt.Printf("  - %s: %s\n", key, value)
		}
	}
}

// Additional Example: Audit log cleanup

func auditCleanupExample() {
	fmt.Println("--- Example 7: Audit Log Cleanup ---")

	storage := security.NewMemoryAuditStorage()
	auditLogger, err := security.NewAuditLogger(&security.AuditLoggerConfig{
		Storage: storage,
		Source:  "cleanup-service",
	})
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	ctx := context.Background()

	// Log some entries
	for i := 0; i < 10; i++ {
		err = auditLogger.LogAction(ctx,
			security.AuditLevelInfo,
			security.AuditActionSagaStarted,
			"saga",
			fmt.Sprintf("saga-%d", i),
			"Test entry",
			nil,
		)
		if err != nil {
			log.Printf("Failed to log entry: %v", err)
		}
	}

	// Count before cleanup
	countBefore, _ := auditLogger.Count(ctx, nil)
	fmt.Printf("Entries before cleanup: %d\n", countBefore)

	// Cleanup entries older than 30 days
	err = storage.Cleanup(ctx, 30*24*time.Hour)
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
		return
	}

	// Count after cleanup
	countAfter, _ := auditLogger.Count(ctx, nil)
	fmt.Printf("Entries after cleanup: %d\n", countAfter)
	fmt.Println("✓ Audit log cleanup completed")
}
