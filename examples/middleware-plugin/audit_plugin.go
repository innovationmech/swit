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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// AuditPlugin is an example plugin that provides audit logging functionality.
type AuditPlugin struct {
	*messaging.BasePlugin
	auditFile *os.File
	logger    *log.Logger
	mutex     sync.Mutex
}

// NewPlugin is the exported factory function that plugin loader will call.
func NewPlugin() (messaging.Plugin, error) {
	metadata := messaging.PluginMetadata{
		Name:        "audit",
		Version:     "1.0.0",
		Description: "Audit logging middleware plugin for message processing",
		Author:      "Swit Team",
		Tags:        []string{"logging", "audit", "compliance"},
		CreatedAt:   time.Now(),
	}

	plugin := &AuditPlugin{
		BasePlugin: messaging.NewBasePlugin(metadata),
	}

	return plugin, nil
}

// Initialize prepares the audit plugin for operation.
func (ap *AuditPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	if err := ap.BasePlugin.Initialize(ctx, config); err != nil {
		return err
	}

	// Get audit file path from config
	auditPath := "/tmp/audit.log"
	if path, ok := config["audit_file"].(string); ok {
		auditPath = path
	}

	// Open or create the audit file
	file, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		ap.SetState(messaging.PluginStateFailed)
		return fmt.Errorf("failed to open audit file: %w", err)
	}

	ap.auditFile = file
	ap.logger = log.New(file, "[AUDIT] ", log.LstdFlags)

	return nil
}

// Start begins the plugin's operation.
func (ap *AuditPlugin) Start(ctx context.Context) error {
	if err := ap.BasePlugin.Start(ctx); err != nil {
		return err
	}

	ap.logger.Println("Audit plugin started")
	return nil
}

// Stop halts the plugin's operation.
func (ap *AuditPlugin) Stop(ctx context.Context) error {
	if err := ap.BasePlugin.Stop(ctx); err != nil {
		return err
	}

	ap.logger.Println("Audit plugin stopped")
	return nil
}

// Shutdown performs final cleanup.
func (ap *AuditPlugin) Shutdown(ctx context.Context) error {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	if ap.auditFile != nil {
		ap.logger.Println("Audit plugin shutting down")
		if err := ap.auditFile.Close(); err != nil {
			return fmt.Errorf("failed to close audit file: %w", err)
		}
		ap.auditFile = nil
	}

	return ap.BasePlugin.Shutdown(ctx)
}

// CreateMiddleware creates a middleware instance from this plugin.
func (ap *AuditPlugin) CreateMiddleware() (messaging.Middleware, error) {
	if ap.State() != messaging.PluginStateStarted {
		return nil, fmt.Errorf("plugin must be started before creating middleware")
	}

	return &AuditMiddleware{
		plugin: ap,
	}, nil
}

// HealthCheck performs a health check on the plugin.
func (ap *AuditPlugin) HealthCheck(ctx context.Context) error {
	if err := ap.BasePlugin.HealthCheck(ctx); err != nil {
		return err
	}

	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	if ap.auditFile == nil {
		return fmt.Errorf("audit file is not open")
	}

	return nil
}

// writeAuditLog writes an audit log entry.
func (ap *AuditPlugin) writeAuditLog(entry AuditEntry) error {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	ap.logger.Println(string(data))
	return nil
}

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp     time.Time         `json:"timestamp"`
	MessageID     string            `json:"message_id"`
	Topic         string            `json:"topic"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Action        string            `json:"action"`
	Success       bool              `json:"success"`
	Error         string            `json:"error,omitempty"`
	Duration      time.Duration     `json:"duration"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// AuditMiddleware is the middleware implementation that uses the audit plugin.
type AuditMiddleware struct {
	plugin *AuditPlugin
}

// Name returns the middleware name.
func (am *AuditMiddleware) Name() string {
	return "audit"
}

// Wrap wraps a handler with audit logging functionality.
func (am *AuditMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		start := time.Now()

		entry := AuditEntry{
			Timestamp:     start,
			MessageID:     message.ID,
			Topic:         message.Topic,
			CorrelationID: message.CorrelationID,
			Action:        "process_message",
			Metadata:      message.Headers,
		}

		err := next.Handle(ctx, message)

		entry.Duration = time.Since(start)
		entry.Success = (err == nil)
		if err != nil {
			entry.Error = err.Error()
		}

		// Write audit log asynchronously to avoid blocking
		go func() {
			if logErr := am.plugin.writeAuditLog(entry); logErr != nil {
				log.Printf("Failed to write audit log: %v", logErr)
			}
		}()

		return err
	})
}
