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

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	if logger.Logger == nil {
		logger.Logger = zap.NewNop()
	}
}

// Mock event handler registry for testing
type mockEventHandlerRegistry struct {
	handlers      map[string]server.EventHandler
	registerError error
	unregisterErr error
}

func newMockEventHandlerRegistry() *mockEventHandlerRegistry {
	return &mockEventHandlerRegistry{
		handlers: make(map[string]server.EventHandler),
	}
}

func (m *mockEventHandlerRegistry) RegisterEventHandler(handler server.EventHandler) error {
	if m.registerError != nil {
		return m.registerError
	}
	m.handlers[handler.GetHandlerID()] = handler
	return nil
}

func (m *mockEventHandlerRegistry) UnregisterEventHandler(handlerID string) error {
	if m.unregisterErr != nil {
		return m.unregisterErr
	}
	delete(m.handlers, handlerID)
	return nil
}

func (m *mockEventHandlerRegistry) GetRegisteredHandlers() []string {
	ids := make([]string, 0, len(m.handlers))
	for id := range m.handlers {
		ids = append(ids, id)
	}
	return ids
}

// TestDynamicHandlerRegistry_RegisterHandler tests handler registration
func TestDynamicHandlerRegistry_RegisterHandler(t *testing.T) {
	tests := []struct {
		name        string
		handler     server.EventHandler
		shouldError bool
	}{
		{
			name: "valid_handler",
			handler: &mockEventHandler{
				id:     "test-handler",
				topics: []string{"test-topic"},
			},
			shouldError: false,
		},
		{
			name:        "nil_handler",
			handler:     nil,
			shouldError: true,
		},
		{
			name: "empty_handler_id",
			handler: &mockEventHandler{
				id:     "",
				topics: []string{"test-topic"},
			},
			shouldError: true,
		},
		{
			name: "empty_topics",
			handler: &mockEventHandler{
				id:     "test-handler",
				topics: []string{},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRegistry := newMockEventHandlerRegistry()
			dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

			err := dynamicRegistry.RegisterHandler(context.Background(), tt.handler)

			if tt.shouldError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify handler was registered
			if dynamicRegistry.Count() != 1 {
				t.Errorf("expected 1 handler, got %d", dynamicRegistry.Count())
			}
		})
	}
}

// TestDynamicHandlerRegistry_DuplicateRegistration tests duplicate handler detection
func TestDynamicHandlerRegistry_DuplicateRegistration(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	handler := &mockEventHandler{
		id:     "test-handler",
		topics: []string{"test-topic"},
	}

	// First registration should succeed
	err := dynamicRegistry.RegisterHandler(context.Background(), handler)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration should fail
	err = dynamicRegistry.RegisterHandler(context.Background(), handler)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

// TestDynamicHandlerRegistry_UnregisterHandler tests handler unregistration
func TestDynamicHandlerRegistry_UnregisterHandler(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	handler := &mockEventHandler{
		id:     "test-handler",
		topics: []string{"test-topic"},
	}

	// Register handler
	err := dynamicRegistry.RegisterHandler(context.Background(), handler)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Unregister handler
	err = dynamicRegistry.UnregisterHandler(context.Background(), "test-handler")
	if err != nil {
		t.Errorf("unregistration failed: %v", err)
	}

	// Verify handler was removed
	if dynamicRegistry.Count() != 0 {
		t.Errorf("expected 0 handlers after unregistration, got %d", dynamicRegistry.Count())
	}

	// Try to unregister again (should fail)
	err = dynamicRegistry.UnregisterHandler(context.Background(), "test-handler")
	if err == nil {
		t.Error("expected error when unregistering non-existent handler")
	}
}

// TestDynamicHandlerRegistry_GetHandler tests handler retrieval
func TestDynamicHandlerRegistry_GetHandler(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	handler := &mockEventHandler{
		id:     "test-handler",
		topics: []string{"test-topic"},
	}

	// Register handler
	err := dynamicRegistry.RegisterHandler(context.Background(), handler)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Get handler
	retrieved, err := dynamicRegistry.GetHandler("test-handler")
	if err != nil {
		t.Errorf("failed to get handler: %v", err)
	}

	if retrieved.GetHandlerID() != "test-handler" {
		t.Errorf("expected handler ID 'test-handler', got '%s'", retrieved.GetHandlerID())
	}

	// Try to get non-existent handler
	_, err = dynamicRegistry.GetHandler("non-existent")
	if err == nil {
		t.Error("expected error when getting non-existent handler")
	}
}

// TestDynamicHandlerRegistry_GetAllHandlers tests listing all handlers
func TestDynamicHandlerRegistry_GetAllHandlers(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	handlers := []*mockEventHandler{
		{id: "handler1", topics: []string{"topic1"}},
		{id: "handler2", topics: []string{"topic2"}},
		{id: "handler3", topics: []string{"topic3"}},
	}

	// Register all handlers
	for _, h := range handlers {
		err := dynamicRegistry.RegisterHandler(context.Background(), h)
		if err != nil {
			t.Fatalf("failed to register handler: %v", err)
		}
	}

	// Get all handlers
	allIDs := dynamicRegistry.GetAllHandlers()
	if len(allIDs) != len(handlers) {
		t.Errorf("expected %d handlers, got %d", len(handlers), len(allIDs))
	}

	// Verify all handler IDs are present
	idMap := make(map[string]bool)
	for _, id := range allIDs {
		idMap[id] = true
	}

	for _, h := range handlers {
		if !idMap[h.id] {
			t.Errorf("handler %s not found in list", h.id)
		}
	}
}

// TestDynamicHandlerRegistry_Metadata tests metadata operations
func TestDynamicHandlerRegistry_Metadata(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	handler := &mockEventHandler{
		id:        "test-handler",
		topics:    []string{"test-topic"},
		brokerReq: "test-broker",
	}

	// Register handler
	err := dynamicRegistry.RegisterHandler(context.Background(), handler)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Get metadata
	metadata, err := dynamicRegistry.GetHandlerMetadata("test-handler")
	if err != nil {
		t.Errorf("failed to get metadata: %v", err)
	}

	if metadata.HandlerID != "test-handler" {
		t.Errorf("expected handler ID 'test-handler', got '%s'", metadata.HandlerID)
	}

	if len(metadata.Topics) != 1 || metadata.Topics[0] != "test-topic" {
		t.Errorf("unexpected topics: %v", metadata.Topics)
	}

	if metadata.BrokerReq != "test-broker" {
		t.Errorf("expected broker req 'test-broker', got '%s'", metadata.BrokerReq)
	}

	// Set custom metadata
	err = dynamicRegistry.SetHandlerMetadata("test-handler", "custom-key", "custom-value")
	if err != nil {
		t.Errorf("failed to set metadata: %v", err)
	}

	// Verify custom metadata
	metadata, _ = dynamicRegistry.GetHandlerMetadata("test-handler")
	if metadata.CustomData["custom-key"] != "custom-value" {
		t.Errorf("custom metadata not set correctly")
	}
}

// TestDynamicHandlerRegistry_StartShutdown tests lifecycle management
func TestDynamicHandlerRegistry_StartShutdown(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	handler := &mockEventHandler{
		id:     "test-handler",
		topics: []string{"test-topic"},
	}

	// Register handler
	err := dynamicRegistry.RegisterHandler(context.Background(), handler)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Initially not started
	if dynamicRegistry.IsStarted() {
		t.Error("registry should not be started initially")
	}

	// Start registry
	err = dynamicRegistry.Start(context.Background())
	if err != nil {
		t.Errorf("start failed: %v", err)
	}

	if !dynamicRegistry.IsStarted() {
		t.Error("registry should be started")
	}

	if !handler.initCalled {
		t.Error("handler Initialize should have been called")
	}

	// Try to start again (should fail)
	err = dynamicRegistry.Start(context.Background())
	if err == nil {
		t.Error("expected error when starting already started registry")
	}

	// Shutdown registry
	err = dynamicRegistry.Shutdown(context.Background())
	if err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	if dynamicRegistry.IsStarted() {
		t.Error("registry should not be started after shutdown")
	}

	if !handler.shutCalled {
		t.Error("handler Shutdown should have been called")
	}
}

// TestDynamicHandlerRegistry_RegisterAfterStart tests registration after start
func TestDynamicHandlerRegistry_RegisterAfterStart(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	// Start empty registry
	err := dynamicRegistry.Start(context.Background())
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	handler := &mockEventHandler{
		id:     "test-handler",
		topics: []string{"test-topic"},
	}

	// Register handler after start
	err = dynamicRegistry.RegisterHandler(context.Background(), handler)
	if err != nil {
		t.Errorf("registration after start failed: %v", err)
	}

	// Handler should be initialized immediately
	if !handler.initCalled {
		t.Error("handler should be initialized immediately when registered after start")
	}
}

// TestDynamicHandlerRegistry_RegisterHandlerWithDiscovery tests integrated discovery and registration
func TestDynamicHandlerRegistry_RegisterHandlerWithDiscovery(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)
	discovery := NewHandlerDiscovery()

	service := &serviceWithFieldHandlers{
		Handler1: &mockEventHandler{id: "handler1", topics: []string{"topic1"}},
		Handler2: &mockEventHandler{id: "handler2", topics: []string{"topic2"}},
	}

	handlerIDs, err := dynamicRegistry.RegisterHandlerWithDiscovery(
		context.Background(),
		discovery,
		service,
	)

	if err != nil {
		t.Fatalf("registration with discovery failed: %v", err)
	}

	if len(handlerIDs) != 2 {
		t.Errorf("expected 2 handler IDs, got %d", len(handlerIDs))
	}

	// Verify handlers are registered
	if dynamicRegistry.Count() != 2 {
		t.Errorf("expected 2 registered handlers, got %d", dynamicRegistry.Count())
	}
}

// TestDynamicHandlerRegistry_RegisterHandlerWithDiscovery_NoHandlers tests error handling
func TestDynamicHandlerRegistry_RegisterHandlerWithDiscovery_NoHandlers(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)
	discovery := NewHandlerDiscovery()

	// Service with no handlers
	type emptyService struct{}
	service := &emptyService{}

	_, err := dynamicRegistry.RegisterHandlerWithDiscovery(
		context.Background(),
		discovery,
		service,
	)

	if err == nil {
		t.Error("expected error when no handlers are discovered")
	}
}

// TestDynamicHandlerRegistry_Concurrent tests concurrent operations
func TestDynamicHandlerRegistry_Concurrent(t *testing.T) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	// Concurrently register multiple handlers
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			handler := &mockEventHandler{
				id:     fmt.Sprintf("handler-%d", id),
				topics: []string{fmt.Sprintf("topic-%d", id)},
			}
			_ = dynamicRegistry.RegisterHandler(context.Background(), handler)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all handlers registered
	if dynamicRegistry.Count() != 10 {
		t.Errorf("expected 10 handlers, got %d", dynamicRegistry.Count())
	}
}

// Benchmark for handler registration
func BenchmarkDynamicHandlerRegistry_RegisterHandler(b *testing.B) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler := &mockEventHandler{
			id:     fmt.Sprintf("handler-%d", i),
			topics: []string{"test-topic"},
		}
		_ = dynamicRegistry.RegisterHandler(context.Background(), handler)
	}
}

// Benchmark for handler lookup
func BenchmarkDynamicHandlerRegistry_GetHandler(b *testing.B) {
	mockRegistry := newMockEventHandlerRegistry()
	dynamicRegistry := NewDynamicHandlerRegistry(mockRegistry)

	// Pre-register some handlers
	for i := 0; i < 100; i++ {
		handler := &mockEventHandler{
			id:     fmt.Sprintf("handler-%d", i),
			topics: []string{"test-topic"},
		}
		_ = dynamicRegistry.RegisterHandler(context.Background(), handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dynamicRegistry.GetHandler("handler-50")
	}
}

// Helper to avoid unused import error
func init() {
	_ = time.Now()
}
