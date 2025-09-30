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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"go.uber.org/zap"
)

// DynamicHandlerRegistry provides runtime registration and management of event handlers.
// It supports dynamic addition and removal of handlers, allowing for flexible
// handler lifecycle management without service restart.
//
// Features:
// - Runtime handler registration and deregistration
// - Handler lifecycle management (initialize/shutdown)
// - Concurrent access support
// - Handler metadata tracking
// - Automatic cleanup on shutdown
type DynamicHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]*registeredHandler
	registry server.EventHandlerRegistry
	started  bool
}

// registeredHandler wraps an event handler with metadata.
type registeredHandler struct {
	handler      server.EventHandler
	registeredAt time.Time
	initialized  bool
	topics       []string
	brokerReq    string
	metadata     map[string]interface{}
}

// NewDynamicHandlerRegistry creates a new dynamic handler registry.
//
// Parameters:
//   - registry: The underlying event handler registry to use for actual registration
//
// Returns:
//   - *DynamicHandlerRegistry: New dynamic registry instance
func NewDynamicHandlerRegistry(registry server.EventHandlerRegistry) *DynamicHandlerRegistry {
	return &DynamicHandlerRegistry{
		handlers: make(map[string]*registeredHandler),
		registry: registry,
		started:  false,
	}
}

// RegisterHandler registers an event handler at runtime.
// If the registry is already started, the handler will be initialized immediately.
//
// Parameters:
//   - ctx: Context for initialization
//   - handler: Event handler to register
//
// Returns:
//   - error: Registration error if validation or initialization fails
//
// Example usage:
//
//	handler := &MyEventHandler{...}
//	if err := registry.RegisterHandler(ctx, handler); err != nil {
//	    return fmt.Errorf("failed to register handler: %w", err)
//	}
func (dhr *DynamicHandlerRegistry) RegisterHandler(ctx context.Context, handler server.EventHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	handlerID := handler.GetHandlerID()
	if handlerID == "" {
		return fmt.Errorf("handler ID cannot be empty")
	}

	topics := handler.GetTopics()
	if len(topics) == 0 {
		return fmt.Errorf("handler must specify at least one topic")
	}

	dhr.mu.Lock()
	defer dhr.mu.Unlock()

	// Check for duplicate registration
	if _, exists := dhr.handlers[handlerID]; exists {
		return fmt.Errorf("handler with ID '%s' is already registered", handlerID)
	}

	// Register with underlying registry
	if err := dhr.registry.RegisterEventHandler(handler); err != nil {
		return fmt.Errorf("failed to register handler with underlying registry: %w", err)
	}

	// Create handler registration record
	registered := &registeredHandler{
		handler:      handler,
		registeredAt: time.Now(),
		initialized:  false,
		topics:       topics,
		brokerReq:    handler.GetBrokerRequirement(),
		metadata:     make(map[string]interface{}),
	}

	// Initialize handler if registry is already started
	if dhr.started {
		if err := handler.Initialize(ctx); err != nil {
			// Rollback registration
			dhr.registry.UnregisterEventHandler(handlerID)
			return fmt.Errorf("failed to initialize handler: %w", err)
		}
		registered.initialized = true
	}

	dhr.handlers[handlerID] = registered

	logger.Logger.Info("Event handler registered dynamically",
		zap.String("handler_id", handlerID),
		zap.Strings("topics", topics),
		zap.Bool("initialized", registered.initialized))

	return nil
}

// UnregisterHandler removes an event handler at runtime.
// The handler will be shut down gracefully before removal.
//
// Parameters:
//   - ctx: Context for shutdown
//   - handlerID: ID of the handler to unregister
//
// Returns:
//   - error: Unregistration error if handler not found or shutdown fails
func (dhr *DynamicHandlerRegistry) UnregisterHandler(ctx context.Context, handlerID string) error {
	if handlerID == "" {
		return fmt.Errorf("handler ID cannot be empty")
	}

	dhr.mu.Lock()
	defer dhr.mu.Unlock()

	registered, exists := dhr.handlers[handlerID]
	if !exists {
		return fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	// Shutdown handler if initialized
	if registered.initialized {
		if err := registered.handler.Shutdown(ctx); err != nil {
			logger.Logger.Warn("Failed to shutdown handler during unregistration",
				zap.String("handler_id", handlerID),
				zap.Error(err))
			// Continue with unregistration despite shutdown error
		}
	}

	// Unregister from underlying registry
	if err := dhr.registry.UnregisterEventHandler(handlerID); err != nil {
		return fmt.Errorf("failed to unregister handler from underlying registry: %w", err)
	}

	delete(dhr.handlers, handlerID)

	logger.Logger.Info("Event handler unregistered dynamically",
		zap.String("handler_id", handlerID))

	return nil
}

// GetHandler retrieves a registered handler by ID.
//
// Parameters:
//   - handlerID: ID of the handler to retrieve
//
// Returns:
//   - server.EventHandler: The handler instance
//   - error: Error if handler not found
func (dhr *DynamicHandlerRegistry) GetHandler(handlerID string) (server.EventHandler, error) {
	dhr.mu.RLock()
	defer dhr.mu.RUnlock()

	registered, exists := dhr.handlers[handlerID]
	if !exists {
		return nil, fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	return registered.handler, nil
}

// GetAllHandlers returns all registered handler IDs.
//
// Returns:
//   - []string: List of all registered handler IDs
func (dhr *DynamicHandlerRegistry) GetAllHandlers() []string {
	dhr.mu.RLock()
	defer dhr.mu.RUnlock()

	handlerIDs := make([]string, 0, len(dhr.handlers))
	for id := range dhr.handlers {
		handlerIDs = append(handlerIDs, id)
	}

	return handlerIDs
}

// GetHandlerMetadata retrieves metadata about a registered handler.
//
// Parameters:
//   - handlerID: ID of the handler
//
// Returns:
//   - *HandlerMetadata: Handler metadata
//   - error: Error if handler not found
func (dhr *DynamicHandlerRegistry) GetHandlerMetadata(handlerID string) (*HandlerMetadata, error) {
	dhr.mu.RLock()
	defer dhr.mu.RUnlock()

	registered, exists := dhr.handlers[handlerID]
	if !exists {
		return nil, fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	return &HandlerMetadata{
		HandlerID:    handlerID,
		Topics:       registered.topics,
		BrokerReq:    registered.brokerReq,
		RegisteredAt: registered.registeredAt,
		Initialized:  registered.initialized,
		CustomData:   registered.metadata,
	}, nil
}

// SetHandlerMetadata sets custom metadata for a handler.
//
// Parameters:
//   - handlerID: ID of the handler
//   - key: Metadata key
//   - value: Metadata value
//
// Returns:
//   - error: Error if handler not found
func (dhr *DynamicHandlerRegistry) SetHandlerMetadata(handlerID string, key string, value interface{}) error {
	dhr.mu.Lock()
	defer dhr.mu.Unlock()

	registered, exists := dhr.handlers[handlerID]
	if !exists {
		return fmt.Errorf("handler with ID '%s' is not registered", handlerID)
	}

	registered.metadata[key] = value
	return nil
}

// Start initializes all registered handlers.
// This should be called when the messaging system starts up.
//
// Parameters:
//   - ctx: Context for initialization
//
// Returns:
//   - error: Initialization error
func (dhr *DynamicHandlerRegistry) Start(ctx context.Context) error {
	dhr.mu.Lock()
	defer dhr.mu.Unlock()

	if dhr.started {
		return fmt.Errorf("registry is already started")
	}

	initErrors := make([]error, 0)

	for handlerID, registered := range dhr.handlers {
		if !registered.initialized {
			if err := registered.handler.Initialize(ctx); err != nil {
				logger.Logger.Error("Failed to initialize handler during start",
					zap.String("handler_id", handlerID),
					zap.Error(err))
				initErrors = append(initErrors, fmt.Errorf("handler %s: %w", handlerID, err))
				continue
			}
			registered.initialized = true
			logger.Logger.Debug("Handler initialized",
				zap.String("handler_id", handlerID))
		}
	}

	if len(initErrors) > 0 {
		return fmt.Errorf("failed to initialize %d handlers: %v", len(initErrors), initErrors)
	}

	dhr.started = true

	logger.Logger.Info("Dynamic handler registry started",
		zap.Int("handler_count", len(dhr.handlers)))

	return nil
}

// Shutdown gracefully shuts down all registered handlers.
// This should be called when the messaging system shuts down.
//
// Parameters:
//   - ctx: Context for shutdown
//
// Returns:
//   - error: Shutdown error
func (dhr *DynamicHandlerRegistry) Shutdown(ctx context.Context) error {
	dhr.mu.Lock()
	defer dhr.mu.Unlock()

	if !dhr.started {
		return nil
	}

	shutdownErrors := make([]error, 0)

	for handlerID, registered := range dhr.handlers {
		if registered.initialized {
			if err := registered.handler.Shutdown(ctx); err != nil {
				logger.Logger.Error("Failed to shutdown handler",
					zap.String("handler_id", handlerID),
					zap.Error(err))
				shutdownErrors = append(shutdownErrors, fmt.Errorf("handler %s: %w", handlerID, err))
				continue
			}
			registered.initialized = false
			logger.Logger.Debug("Handler shutdown",
				zap.String("handler_id", handlerID))
		}
	}

	dhr.started = false

	if len(shutdownErrors) > 0 {
		return fmt.Errorf("failed to shutdown %d handlers: %v", len(shutdownErrors), shutdownErrors)
	}

	logger.Logger.Info("Dynamic handler registry shutdown",
		zap.Int("handler_count", len(dhr.handlers)))

	return nil
}

// IsStarted returns whether the registry has been started.
func (dhr *DynamicHandlerRegistry) IsStarted() bool {
	dhr.mu.RLock()
	defer dhr.mu.RUnlock()
	return dhr.started
}

// Count returns the number of registered handlers.
func (dhr *DynamicHandlerRegistry) Count() int {
	dhr.mu.RLock()
	defer dhr.mu.RUnlock()
	return len(dhr.handlers)
}

// HandlerMetadata contains metadata about a registered handler.
type HandlerMetadata struct {
	HandlerID    string
	Topics       []string
	BrokerReq    string
	RegisteredAt time.Time
	Initialized  bool
	CustomData   map[string]interface{}
}

// RegisterHandlerWithDiscovery combines discovery and registration.
// It discovers handlers from a service and registers them all.
//
// Parameters:
//   - ctx: Context for initialization
//   - discovery: Handler discovery instance
//   - service: Service instance to scan
//
// Returns:
//   - []string: List of registered handler IDs
//   - error: Registration error
//
// Example:
//
//	discovery := NewHandlerDiscovery()
//	handlerIDs, err := registry.RegisterHandlerWithDiscovery(ctx, discovery, myService)
//	if err != nil {
//	    return fmt.Errorf("failed to register discovered handlers: %w", err)
//	}
func (dhr *DynamicHandlerRegistry) RegisterHandlerWithDiscovery(
	ctx context.Context,
	discovery *HandlerDiscovery,
	service interface{},
) ([]string, error) {
	// Discover handlers
	handlers, err := discovery.DiscoverHandlers(service)
	if err != nil {
		return nil, fmt.Errorf("failed to discover handlers: %w", err)
	}

	if len(handlers) == 0 {
		return nil, fmt.Errorf("no handlers discovered from service")
	}

	// Register all discovered handlers
	registeredIDs := make([]string, 0, len(handlers))
	registrationErrors := make([]error, 0)

	for _, handler := range handlers {
		if err := dhr.RegisterHandler(ctx, handler); err != nil {
			logger.Logger.Warn("Failed to register discovered handler",
				zap.String("handler_id", handler.GetHandlerID()),
				zap.Error(err))
			registrationErrors = append(registrationErrors, err)
			continue
		}
		registeredIDs = append(registeredIDs, handler.GetHandlerID())
	}

	if len(registrationErrors) > 0 && len(registeredIDs) == 0 {
		return nil, fmt.Errorf("failed to register all discovered handlers: %d errors", len(registrationErrors))
	}

	logger.Logger.Info("Registered discovered handlers",
		zap.Int("discovered_count", len(handlers)),
		zap.Int("registered_count", len(registeredIDs)),
		zap.Int("error_count", len(registrationErrors)))

	return registeredIDs, nil
}
