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

package messaging

import (
	"errors"
	"fmt"
	"sync"
)

// EventHandlerRegistry provides a bridge between the messaging layer and the server framework
// for registering and managing event handlers. It implements the server framework's
// EventHandlerRegistry interface while working with the messaging layer's EventHandler interface.
type EventHandlerRegistry struct {
	coordinator MessagingCoordinator
	mu          sync.RWMutex
}

// NewEventHandlerRegistry creates a new event handler registry with the provided coordinator
func NewEventHandlerRegistry(coordinator MessagingCoordinator) *EventHandlerRegistry {
	return &EventHandlerRegistry{
		coordinator: coordinator,
	}
}

// RegisterEventHandler registers an event handler for message processing
// This method bridges the server framework's registration patterns with
// the messaging layer's event handler system.
func (r *EventHandlerRegistry) RegisterEventHandler(handler EventHandler) error {
	if handler == nil {
		return fmt.Errorf("event handler cannot be nil")
	}

	handlerID := handler.GetHandlerID()
	if handlerID == "" {
		return fmt.Errorf("event handler must have a non-empty ID")
	}

	topics := handler.GetTopics()
	if len(topics) == 0 {
		return fmt.Errorf("event handler must specify at least one topic")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Register with the messaging coordinator
	if err := r.coordinator.RegisterEventHandler(handler); err != nil {
		return fmt.Errorf("failed to register event handler '%s': %w", handlerID, err)
	}

	return nil
}

// UnregisterEventHandler removes an event handler by ID
// This method provides cleanup functionality when handlers are no longer needed.
func (r *EventHandlerRegistry) UnregisterEventHandler(handlerID string) error {
	if handlerID == "" {
		return fmt.Errorf("handler ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Unregister from the messaging coordinator
	if err := r.coordinator.UnregisterEventHandler(handlerID); err != nil {
		return fmt.Errorf("failed to unregister event handler '%s': %w", handlerID, err)
	}

	return nil
}

// GetRegisteredHandlers lists all registered event handler IDs
// This method enables discovery and monitoring of registered handlers.
func (r *EventHandlerRegistry) GetRegisteredHandlers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.coordinator.GetRegisteredHandlers()
}

// GetCoordinator returns the underlying messaging coordinator
// This provides access to additional messaging functionality when needed.
func (r *EventHandlerRegistry) GetCoordinator() MessagingCoordinator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.coordinator
}

// ValidateHandler validates an event handler before registration
// This method performs comprehensive validation to ensure handlers
// meet the requirements for successful registration and operation.
func (r *EventHandlerRegistry) ValidateHandler(handler EventHandler) error {
	if handler == nil {
		return fmt.Errorf("event handler cannot be nil")
	}

	handlerID := handler.GetHandlerID()
	if handlerID == "" {
		return fmt.Errorf("event handler must have a non-empty ID")
	}

	// Validate handler ID format (alphanumeric, hyphens, underscores only)
	for _, ch := range handlerID {
		if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '-' && ch != '_' {
			return fmt.Errorf("handler ID can only contain letters, numbers, hyphens, and underscores")
		}
	}

	topics := handler.GetTopics()
	if len(topics) == 0 {
		return fmt.Errorf("event handler must specify at least one topic")
	}

	// Validate topic names
	for _, topic := range topics {
		if topic == "" {
			return fmt.Errorf("topic names cannot be empty")
		}
	}

	// Check if handler ID is already registered
	registeredHandlers := r.GetRegisteredHandlers()
	for _, registered := range registeredHandlers {
		if registered == handlerID {
			return fmt.Errorf("handler with ID '%s' is already registered", handlerID)
		}
	}

	return nil
}

// RegisterEventHandlerWithValidation registers an event handler after validation
// This is a convenience method that combines validation and registration.
func (r *EventHandlerRegistry) RegisterEventHandlerWithValidation(handler EventHandler) error {
	if err := r.ValidateHandler(handler); err != nil {
		return fmt.Errorf("handler validation failed: %w", err)
	}

	return r.RegisterEventHandler(handler)
}

// GetHandlerCount returns the number of registered event handlers
func (r *EventHandlerRegistry) GetHandlerCount() int {
	handlers := r.GetRegisteredHandlers()
	return len(handlers)
}

// Clear removes all registered event handlers
// This is primarily useful for testing and cleanup scenarios.
func (r *EventHandlerRegistry) Clear() error {
	handlers := r.GetRegisteredHandlers()

	var errs []error
	for _, handlerID := range handlers {
		if err := r.UnregisterEventHandler(handlerID); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
