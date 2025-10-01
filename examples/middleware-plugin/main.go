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
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

func main() {
	fmt.Println("=== Middleware Plugin System Example ===")
	fmt.Println()

	ctx := context.Background()

	// Create plugin registry and loader
	registry := messaging.NewPluginRegistry()
	loader := messaging.NewPluginLoader(registry)

	// Load audit plugin from factory (static loading)
	fmt.Println("1. Loading audit plugin...")
	auditConfig := map[string]interface{}{
		"audit_file": "/tmp/swit-audit.log",
	}

	if err := loader.LoadFromFactory(ctx, "audit", NewPlugin, auditConfig); err != nil {
		log.Fatalf("Failed to load audit plugin: %v", err)
	}
	fmt.Println("   ✓ Audit plugin loaded successfully")
	fmt.Println()

	// Start all plugins
	fmt.Println("2. Starting plugins...")
	if err := registry.StartAll(ctx); err != nil {
		log.Fatalf("Failed to start plugins: %v", err)
	}
	fmt.Println("   ✓ All plugins started")
	fmt.Println()

	// List loaded plugins
	fmt.Println("3. Loaded plugins:")
	for _, name := range registry.ListPlugins() {
		plugin, _ := registry.GetPlugin(name)
		meta := plugin.Metadata()
		fmt.Printf("   - %s (v%s): %s\n", meta.Name, meta.Version, meta.Description)
		fmt.Printf("     State: %s\n", plugin.State())
	}
	fmt.Println()

	// Create middleware from plugin
	fmt.Println("4. Creating middleware from plugin...")
	plugin, err := registry.GetPlugin("audit")
	if err != nil {
		log.Fatalf("Failed to get audit plugin: %v", err)
	}

	auditMiddleware, err := plugin.CreateMiddleware()
	if err != nil {
		log.Fatalf("Failed to create middleware: %v", err)
	}
	fmt.Println("   ✓ Middleware created")
	fmt.Println()

	// Create middleware chain
	fmt.Println("5. Building middleware chain...")
	chain := messaging.NewMiddlewareChain(auditMiddleware)

	// Create a simple message handler
	handler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		fmt.Printf("   Processing message: %s (topic: %s)\n", msg.ID, msg.Topic)
		time.Sleep(50 * time.Millisecond) // Simulate processing
		return nil
	})

	// Build handler with middleware chain
	wrappedHandler := chain.Build(handler)
	fmt.Println("   ✓ Middleware chain built")
	fmt.Println()

	// Process some test messages
	fmt.Println("6. Processing test messages...")
	messages := []*messaging.Message{
		{
			ID:            "msg-001",
			Topic:         "orders.created",
			Payload:       []byte(`{"order_id": "12345"}`),
			CorrelationID: "corr-001",
			Timestamp:     time.Now(),
			Headers: map[string]string{
				"user_id": "user-123",
				"source":  "web-app",
			},
		},
		{
			ID:            "msg-002",
			Topic:         "payments.processed",
			Payload:       []byte(`{"payment_id": "67890"}`),
			CorrelationID: "corr-002",
			Timestamp:     time.Now(),
			Headers: map[string]string{
				"user_id": "user-456",
				"source":  "mobile-app",
			},
		},
		{
			ID:            "msg-003",
			Topic:         "inventory.updated",
			Payload:       []byte(`{"item_id": "item-789"}`),
			CorrelationID: "corr-003",
			Timestamp:     time.Now(),
			Headers: map[string]string{
				"warehouse": "warehouse-1",
			},
		},
	}

	for _, msg := range messages {
		if err := wrappedHandler.Handle(ctx, msg); err != nil {
			log.Printf("   ✗ Error processing message %s: %v\n", msg.ID, err)
		}
	}
	fmt.Println("   ✓ All messages processed")
	fmt.Println()

	// Perform health check
	fmt.Println("7. Performing health checks...")
	healthResults := registry.HealthCheckAll(ctx)
	for name, err := range healthResults {
		if err != nil {
			fmt.Printf("   ✗ Plugin %s: unhealthy - %v\n", name, err)
		} else {
			fmt.Printf("   ✓ Plugin %s: healthy\n", name)
		}
	}
	fmt.Println()

	// Demonstrate plugin reload (for factory-based plugins)
	fmt.Println("8. Demonstrating plugin reload...")
	newConfig := map[string]interface{}{
		"audit_file": "/tmp/swit-audit-new.log",
	}

	if err := loader.Reload(ctx, "audit", newConfig); err != nil {
		log.Printf("   Note: Reload failed (expected for this example): %v\n", err)
	} else {
		fmt.Println("   ✓ Plugin reloaded successfully")
	}
	fmt.Println()

	// Stop plugins
	fmt.Println("9. Stopping plugins...")
	if err := registry.StopAll(ctx); err != nil {
		log.Printf("Failed to stop plugins: %v", err)
	}
	fmt.Println("   ✓ All plugins stopped")
	fmt.Println()

	// Shutdown plugins
	fmt.Println("10. Shutting down plugins...")
	if err := registry.ShutdownAll(ctx); err != nil {
		log.Printf("Failed to shutdown plugins: %v", err)
	}
	fmt.Println("    ✓ All plugins shut down")
	fmt.Println()

	fmt.Println("=== Example Complete ===")
	fmt.Println("\nCheck the audit log at: /tmp/swit-audit.log")
}
