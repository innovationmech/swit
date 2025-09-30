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

// Package main demonstrates cross-transport context propagation in SWIT framework.
// This example shows how to propagate request context information across HTTP, gRPC,
// and messaging transports to maintain correlation IDs, user context, and trace information.
//
// Note: This is example/demonstration code that intentionally prints context values
// (correlation IDs, user IDs, etc.) to show how context propagation works.
// In production code, ensure appropriate logging practices and data sensitivity handling.
//
// lgtm[go/clear-text-logging]
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/messaging"
	"google.golang.org/grpc/metadata"
)

func main() {
	fmt.Println("=== SWIT Context Propagation Example ===")
	fmt.Println()

	// Example 1: HTTP to Messaging
	example1HTTPToMessaging()

	// Example 2: gRPC to Messaging
	example2GRPCToMessaging()

	// Example 3: Messaging to HTTP
	example3MessagingToHTTP()

	// Example 4: Gin to Messaging
	example4GinToMessaging()

	// Example 5: Round-trip propagation
	example5RoundTripPropagation()
}

// example1HTTPToMessaging demonstrates propagating context from HTTP request to messaging.
func example1HTTPToMessaging() {
	fmt.Println("Example 1: HTTP to Messaging")
	fmt.Println("-----------------------------")

	// Simulate an incoming HTTP request with headers
	req, _ := http.NewRequest("GET", "/api/users", nil)
	req.Header.Set("X-Correlation-ID", "http-corr-12345")
	req.Header.Set("X-User-ID", "user-789")
	req.Header.Set("X-Tenant-ID", "tenant-001")

	// Create a message to publish
	message := &messaging.Message{
		ID:      "msg-001",
		Topic:   "user.events",
		Payload: []byte(`{"event": "user_created"}`),
	}

	// Propagate context from HTTP to message
	ctx := messaging.PropagateHTTPToMessage(context.Background(), req, message)

	// Display the results
	fmt.Printf("HTTP Headers:\n")
	fmt.Printf("  Correlation-ID: %s\n", req.Header.Get("X-Correlation-ID"))
	fmt.Printf("  User-ID: %s\n", req.Header.Get("X-User-ID"))
	fmt.Printf("  Tenant-ID: %s\n\n", req.Header.Get("X-Tenant-ID"))

	fmt.Printf("Message Headers:\n")
	fmt.Printf("  Correlation-ID: %s\n", message.CorrelationID)
	fmt.Printf("  User-ID: %s\n", message.Headers["user_id"])
	fmt.Printf("  Tenant-ID: %s\n\n", message.Headers["tenant_id"])

	// Verify context
	if correlationID := ctx.Value(messaging.ContextKeyCorrelationID); correlationID != nil {
		fmt.Printf("✓ Context propagated successfully: Correlation ID = %v\n", correlationID)
	}

	fmt.Println()
}

// example2GRPCToMessaging demonstrates propagating context from gRPC to messaging.
func example2GRPCToMessaging() {
	fmt.Println("Example 2: gRPC to Messaging")
	fmt.Println("----------------------------")

	// Simulate incoming gRPC context with metadata
	md := metadata.New(map[string]string{
		"x-correlation-id": "grpc-corr-67890",
		"x-user-id":        "user-456",
		"x-tenant-id":      "tenant-002",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Create a message to publish
	message := &messaging.Message{
		ID:      "msg-002",
		Topic:   "order.events",
		Payload: []byte(`{"event": "order_placed"}`),
	}

	// Propagate context from gRPC to message
	ctx = messaging.PropagateGRPCToMessage(ctx, message)

	// Display the results
	fmt.Printf("gRPC Metadata:\n")
	fmt.Printf("  Correlation-ID: %s\n", md.Get("x-correlation-id")[0])
	fmt.Printf("  User-ID: %s\n", md.Get("x-user-id")[0])
	fmt.Printf("  Tenant-ID: %s\n\n", md.Get("x-tenant-id")[0])

	fmt.Printf("Message Headers:\n")
	fmt.Printf("  Correlation-ID: %s\n", message.CorrelationID)
	fmt.Printf("  User-ID: %s\n", message.Headers["user_id"])
	fmt.Printf("  Tenant-ID: %s\n\n", message.Headers["tenant_id"])

	// Verify context
	if correlationID := ctx.Value(messaging.ContextKeyCorrelationID); correlationID != nil {
		fmt.Printf("✓ Context propagated successfully: Correlation ID = %v\n", correlationID)
	}

	fmt.Println()
}

// example3MessagingToHTTP demonstrates propagating context from messaging to HTTP.
func example3MessagingToHTTP() {
	fmt.Println("Example 3: Messaging to HTTP")
	fmt.Println("----------------------------")

	// Simulate an incoming message
	message := &messaging.Message{
		ID:            "msg-003",
		Topic:         "webhook.trigger",
		CorrelationID: "msg-corr-abc123",
		Headers: map[string]string{
			"user_id":   "user-123",
			"tenant_id": "tenant-003",
		},
		Payload: []byte(`{"event": "webhook_triggered"}`),
	}

	// Create HTTP headers for outgoing webhook call
	header := http.Header{}
	ctx := messaging.PropagateMessageToHTTP(context.Background(), message, header)

	// Display the results
	fmt.Printf("Message Headers:\n")
	fmt.Printf("  Correlation-ID: %s\n", message.CorrelationID)
	fmt.Printf("  User-ID: %s\n", message.Headers["user_id"])
	fmt.Printf("  Tenant-ID: %s\n\n", message.Headers["tenant_id"])

	fmt.Printf("HTTP Headers:\n")
	fmt.Printf("  X-Correlation-ID: %s\n", header.Get("X-Correlation-ID"))
	fmt.Printf("  X-User-ID: %s\n", header.Get("X-User-ID"))
	fmt.Printf("  X-Tenant-ID: %s\n\n", header.Get("X-Tenant-ID"))

	// Verify context
	if correlationID := ctx.Value(messaging.ContextKeyCorrelationID); correlationID != nil {
		fmt.Printf("✓ Context propagated successfully: Correlation ID = %v\n", correlationID)
	}

	fmt.Println()
}

// example4GinToMessaging demonstrates propagating context from Gin context to messaging.
func example4GinToMessaging() {
	fmt.Println("Example 4: Gin to Messaging")
	fmt.Println("---------------------------")

	// Create a test Gin context
	gin.SetMode(gin.TestMode)
	w := &mockResponseWriter{}
	c, _ := gin.CreateTestContext(w)

	// Simulate incoming request with headers and gin context values
	req, _ := http.NewRequest("POST", "/api/orders", nil)
	req.Header.Set("X-Correlation-ID", "gin-corr-xyz789")
	c.Request = req
	c.Set("user_id", "user-999")
	c.Set("tenant_id", "tenant-004")

	// Create a message to publish
	message := &messaging.Message{
		ID:      "msg-004",
		Topic:   "order.created",
		Payload: []byte(`{"order_id": "order-123"}`),
	}

	// Propagate context from Gin to message
	ctx := messaging.PropagateGinToMessage(c, message)

	// Display the results
	fmt.Printf("Gin Context:\n")
	fmt.Printf("  Header Correlation-ID: %s\n", c.Request.Header.Get("X-Correlation-ID"))
	fmt.Printf("  Context User-ID: %v\n", c.GetString("user_id"))
	fmt.Printf("  Context Tenant-ID: %v\n\n", c.GetString("tenant_id"))

	fmt.Printf("Message Headers:\n")
	fmt.Printf("  Correlation-ID: %s\n", message.CorrelationID)
	fmt.Printf("  User-ID: %s\n", message.Headers["user_id"])
	fmt.Printf("  Tenant-ID: %s\n\n", message.Headers["tenant_id"])

	// Verify context
	if correlationID := ctx.Value(messaging.ContextKeyCorrelationID); correlationID != nil {
		fmt.Printf("✓ Context propagated successfully: Correlation ID = %v\n", correlationID)
	}

	fmt.Println()
}

// example5RoundTripPropagation demonstrates context propagation through multiple transports.
func example5RoundTripPropagation() {
	fmt.Println("Example 5: Round-trip Context Propagation")
	fmt.Println("------------------------------------------")

	fmt.Println("Step 1: HTTP Request arrives")
	// Step 1: HTTP Request arrives with correlation ID
	req, _ := http.NewRequest("GET", "/api/process", nil)
	req.Header.Set("X-Correlation-ID", "round-trip-001")
	req.Header.Set("X-User-ID", "user-roundtrip")
	fmt.Printf("  HTTP Correlation-ID: %s\n", req.Header.Get("X-Correlation-ID"))

	fmt.Println("\nStep 2: HTTP -> Message (publish to queue)")
	// Step 2: Service publishes message
	message1 := &messaging.Message{
		ID:      "msg-rt-001",
		Topic:   "process.request",
		Payload: []byte(`{"action": "process"}`),
	}
	ctx1 := messaging.PropagateHTTPToMessage(context.Background(), req, message1)
	fmt.Printf("  Message Correlation-ID: %s\n", message1.CorrelationID)

	// Simulate message being processed by another service
	time.Sleep(10 * time.Millisecond)

	fmt.Println("\nStep 3: Message -> gRPC (call downstream service)")
	// Step 3: Consumer calls gRPC service
	ctx2 := messaging.GlobalContextPropagator.ExtractFromMessage(ctx1, message1)
	_, grpcMD := messaging.GlobalContextPropagator.InjectToGRPC(ctx2)
	fmt.Printf("  gRPC Metadata Correlation-ID: %s\n", grpcMD.Get("x-correlation-id")[0])

	// Simulate gRPC service processing
	time.Sleep(10 * time.Millisecond)

	fmt.Println("\nStep 4: gRPC -> Message (publish result)")
	// Step 4: gRPC service publishes result
	message2 := &messaging.Message{
		ID:      "msg-rt-002",
		Topic:   "process.result",
		Payload: []byte(`{"status": "completed"}`),
	}
	ctx4 := metadata.NewIncomingContext(context.Background(), grpcMD)
	ctx5 := messaging.PropagateGRPCToMessage(ctx4, message2)
	fmt.Printf("  Message Correlation-ID: %s\n", message2.CorrelationID)

	fmt.Println("\nStep 5: Message -> HTTP (send webhook)")
	// Step 5: Send webhook with results
	webhookHeader := http.Header{}
	ctx6 := messaging.PropagateMessageToHTTP(ctx5, message2, webhookHeader)
	fmt.Printf("  Webhook Header Correlation-ID: %s\n", webhookHeader.Get("X-Correlation-ID"))

	// Verify correlation ID is preserved throughout the journey
	originalCorrelationID := req.Header.Get("X-Correlation-ID")
	finalCorrelationID := webhookHeader.Get("X-Correlation-ID")

	fmt.Printf("\n✓ Correlation ID preserved: %s -> %s\n", originalCorrelationID, finalCorrelationID)
	if originalCorrelationID == finalCorrelationID {
		fmt.Println("✓ Round-trip propagation successful!")
	} else {
		log.Printf("✗ Correlation ID mismatch!")
	}

	// Verify user context is also preserved
	if userID := ctx6.Value(messaging.ContextKeyUserID); userID == "user-roundtrip" {
		fmt.Printf("✓ User context preserved: %v\n", userID)
	}

	fmt.Println()
}

// mockResponseWriter is a simple mock implementation for testing
type mockResponseWriter struct {
	headers http.Header
	status  int
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = http.Header{}
	}
	return m.headers
}

func (m *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}
