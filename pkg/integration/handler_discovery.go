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
	"fmt"
	"reflect"
	"sync"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"go.uber.org/zap"
)

// HandlerDiscovery provides reflection-based discovery of event handlers.
// It can automatically detect and extract event handlers from struct instances
// at runtime, enabling dynamic handler registration without explicit configuration.
type HandlerDiscovery struct {
	mu sync.RWMutex
	// discovered handlers cache
	discoveredHandlers map[reflect.Type][]server.EventHandler
}

// NewHandlerDiscovery creates a new handler discovery instance.
func NewHandlerDiscovery() *HandlerDiscovery {
	return &HandlerDiscovery{
		discoveredHandlers: make(map[reflect.Type][]server.EventHandler),
	}
}

// DiscoverHandlers discovers all event handlers in the provided service instance.
// It uses reflection to find fields or methods that implement the EventHandler interface.
//
// Discovery rules:
// - Searches for exported struct fields that implement EventHandler
// - Searches for methods with EventHandler interface signature
// - Caches discovered handlers to avoid repeated reflection
//
// Parameters:
//   - service: The service instance to scan for handlers
//
// Returns:
//   - []server.EventHandler: List of discovered event handlers
//   - error: Discovery error if reflection fails
//
// Example usage:
//
//	discovery := NewHandlerDiscovery()
//	handlers, err := discovery.DiscoverHandlers(myService)
//	if err != nil {
//	    return err
//	}
//	for _, handler := range handlers {
//	    registry.RegisterEventHandler(handler)
//	}
func (hd *HandlerDiscovery) DiscoverHandlers(service interface{}) ([]server.EventHandler, error) {
	if service == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}

	serviceType := reflect.TypeOf(service)
	serviceValue := reflect.ValueOf(service)

	// Dereference pointer if necessary
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
		serviceValue = serviceValue.Elem()
	}

	// Check cache first
	hd.mu.RLock()
	if cached, exists := hd.discoveredHandlers[serviceType]; exists {
		hd.mu.RUnlock()
		logger.Logger.Debug("Using cached handler discovery",
			zap.String("service_type", serviceType.String()),
			zap.Int("handler_count", len(cached)))
		return cached, nil
	}
	hd.mu.RUnlock()

	handlers := make([]server.EventHandler, 0)

	// Discover handlers from struct fields
	fieldHandlers, err := hd.discoverFieldHandlers(serviceValue, serviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to discover field handlers: %w", err)
	}
	handlers = append(handlers, fieldHandlers...)

	// Note: Method-based handler discovery is intentionally not included
	// as it requires invoking methods which may have side effects or dependencies.
	// Field-based discovery is the recommended approach for handler registration.

	// Cache the results
	hd.mu.Lock()
	hd.discoveredHandlers[serviceType] = handlers
	hd.mu.Unlock()

	logger.Logger.Info("Discovered event handlers",
		zap.String("service_type", serviceType.String()),
		zap.Int("handler_count", len(handlers)))

	return handlers, nil
}

// discoverFieldHandlers discovers handlers from exported struct fields.
func (hd *HandlerDiscovery) discoverFieldHandlers(serviceValue reflect.Value, serviceType reflect.Type) ([]server.EventHandler, error) {
	if serviceType.Kind() != reflect.Struct {
		return nil, nil
	}

	handlers := make([]server.EventHandler, 0)
	eventHandlerType := reflect.TypeOf((*server.EventHandler)(nil)).Elem()

	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		fieldValue := serviceValue.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip nil values (only check if kind supports IsNil)
		if fieldValue.Kind() == reflect.Ptr || fieldValue.Kind() == reflect.Interface {
			if fieldValue.IsNil() {
				continue
			}
		}

		// Check if field implements EventHandler interface
		if fieldValue.Type().Implements(eventHandlerType) {
			handler, ok := fieldValue.Interface().(server.EventHandler)
			if ok && handler != nil {
				handlers = append(handlers, handler)
				logger.Logger.Debug("Discovered field handler",
					zap.String("field_name", field.Name),
					zap.String("handler_id", handler.GetHandlerID()))
			}
		}

		// Check pointer to field implements EventHandler interface
		if fieldValue.CanAddr() && fieldValue.Addr().Type().Implements(eventHandlerType) {
			handler, ok := fieldValue.Addr().Interface().(server.EventHandler)
			if ok && handler != nil {
				handlers = append(handlers, handler)
				logger.Logger.Debug("Discovered field handler (by pointer)",
					zap.String("field_name", field.Name),
					zap.String("handler_id", handler.GetHandlerID()))
			}
		}
	}

	return handlers, nil
}

// Note: Method-based handler discovery was intentionally removed to avoid
// side effects from method invocation during discovery. Use field-based
// discovery for event handlers.

// ClearCache clears the discovery cache.
// This is useful when service instances are recreated or handlers are modified.
func (hd *HandlerDiscovery) ClearCache() {
	hd.mu.Lock()
	defer hd.mu.Unlock()
	hd.discoveredHandlers = make(map[reflect.Type][]server.EventHandler)
	logger.Logger.Debug("Cleared handler discovery cache")
}

// ClearCacheForType clears the discovery cache for a specific service type.
func (hd *HandlerDiscovery) ClearCacheForType(service interface{}) {
	if service == nil {
		return
	}

	serviceType := reflect.TypeOf(service)
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
	}

	hd.mu.Lock()
	defer hd.mu.Unlock()
	delete(hd.discoveredHandlers, serviceType)
	logger.Logger.Debug("Cleared handler discovery cache for type",
		zap.String("service_type", serviceType.String()))
}

// DiscoverFromPackage discovers all handlers from a package by scanning
// all exported types in the provided instance slice.
//
// Parameters:
//   - services: Slice of service instances to scan
//
// Returns:
//   - []server.EventHandler: All discovered handlers
//   - error: Discovery error
//
// Example:
//
//	services := []interface{}{
//	    &OrderService{},
//	    &PaymentService{},
//	    &NotificationService{},
//	}
//	handlers, err := discovery.DiscoverFromPackage(services)
func (hd *HandlerDiscovery) DiscoverFromPackage(services []interface{}) ([]server.EventHandler, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("services list cannot be empty")
	}

	allHandlers := make([]server.EventHandler, 0)
	discoveryErrors := make([]error, 0)

	for _, service := range services {
		// Skip nil services
		if service == nil {
			discoveryErrors = append(discoveryErrors, fmt.Errorf("nil service in services list"))
			continue
		}

		handlers, err := hd.DiscoverHandlers(service)
		if err != nil {
			serviceType := "unknown"
			if reflect.TypeOf(service) != nil {
				serviceType = reflect.TypeOf(service).String()
			}
			logger.Logger.Warn("Failed to discover handlers from service",
				zap.String("service_type", serviceType),
				zap.Error(err))
			discoveryErrors = append(discoveryErrors, err)
			continue
		}
		allHandlers = append(allHandlers, handlers...)
	}

	if len(discoveryErrors) > 0 && len(allHandlers) == 0 {
		return nil, fmt.Errorf("failed to discover handlers from all services: %d errors", len(discoveryErrors))
	}

	logger.Logger.Info("Package-level handler discovery completed",
		zap.Int("service_count", len(services)),
		zap.Int("handler_count", len(allHandlers)),
		zap.Int("error_count", len(discoveryErrors)))

	return allHandlers, nil
}
