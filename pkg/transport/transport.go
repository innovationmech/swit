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

package transport

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/tracing"
	"github.com/innovationmech/swit/pkg/types"
	"go.uber.org/zap"
)

// MessagingCoordinator interface to avoid import cycle with messaging package
type MessagingCoordinator interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	RegisterBroker(name string, broker MessageBroker) error
	GetBroker(name string) (MessageBroker, error)
	GetRegisteredBrokers() []string
	RegisterEventHandler(handler EventHandler) error
	UnregisterEventHandler(handlerID string) error
	GetRegisteredHandlers() []string
	IsStarted() bool
	GetMetrics() any
	HealthCheck(ctx context.Context) (any, error)
}

// MessageBroker minimal interface to avoid import cycle
type MessageBroker interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
}

// EventHandler minimal interface to avoid import cycle
type EventHandler interface {
	GetHandlerID() string
}

// TransportError represents an error from a specific transport
type TransportError struct {
	TransportName string
	Err           error
}

func (te *TransportError) Error() string {
	return fmt.Sprintf("transport '%s': %v", te.TransportName, te.Err)
}

// MultiError represents multiple transport errors
type MultiError struct {
	Errors []TransportError
}

func (me *MultiError) Error() string {
	if len(me.Errors) == 0 {
		return "no errors"
	}
	if len(me.Errors) == 1 {
		return me.Errors[0].Error()
	}

	var errStrs []string
	for _, err := range me.Errors {
		errStrs = append(errStrs, err.Error())
	}
	return fmt.Sprintf("multiple transport errors: %s", strings.Join(errStrs, "; "))
}

func (me *MultiError) HasErrors() bool {
	return len(me.Errors) > 0
}

// Unwrap returns the first error if there's only one error, otherwise returns nil
func (me *MultiError) Unwrap() error {
	if len(me.Errors) == 1 {
		return me.Errors[0].Err
	}
	return nil
}

// GetErrorByTransport returns the error for a specific transport, if any
func (me *MultiError) GetErrorByTransport(transportName string) *TransportError {
	for _, err := range me.Errors {
		if err.TransportName == transportName {
			return &err
		}
	}
	return nil
}

// IsStopError checks if an error is a transport stop error
func IsStopError(err error) bool {
	if err == nil {
		return false
	}
	_, isMultiError := err.(*MultiError)
	_, isTransportError := err.(*TransportError)
	return isMultiError || isTransportError
}

// ExtractStopErrors extracts transport stop errors from an error
func ExtractStopErrors(err error) []TransportError {
	if err == nil {
		return nil
	}

	if multiErr, ok := err.(*MultiError); ok {
		return multiErr.Errors
	}

	if transportErr, ok := err.(*TransportError); ok {
		return []TransportError{*transportErr}
	}

	return nil
}

// NetworkTransport defines the interface for different transport mechanisms
type NetworkTransport interface {
	// Start starts the transport server
	Start(ctx context.Context) error
	// Stop gracefully stops the transport server
	Stop(ctx context.Context) error
	// GetName returns the transport name
	GetName() string
	// GetAddress returns the listening address
	GetAddress() string
}

// TransportCoordinator manages multiple transport instances and their service registries
type TransportCoordinator struct {
	transports       []NetworkTransport
	registryManager  *MultiTransportRegistry
	messagingCoord   MessagingCoordinator   // Messaging coordinator for unified messaging support
	tracingManager   tracing.TracingManager // Unified tracing manager for all transports
	messagingEnabled bool                   // Whether messaging is enabled for this coordinator
	mu               sync.RWMutex
}

// NewTransportCoordinator creates a new transport coordinator
func NewTransportCoordinator() *TransportCoordinator {
	return &TransportCoordinator{
		transports:       make([]NetworkTransport, 0),
		registryManager:  NewMultiTransportRegistry(),
		messagingCoord:   nil, // Will be set later to avoid import cycle
		messagingEnabled: false,
	}
}

// NewTransportCoordinatorWithMessaging creates a new transport coordinator with messaging enabled
func NewTransportCoordinatorWithMessaging() *TransportCoordinator {
	tc := NewTransportCoordinator()
	tc.EnableMessaging()
	return tc
}

// SetMessagingCoordinator sets the messaging coordinator to avoid import cycles
func (m *TransportCoordinator) SetMessagingCoordinator(coordinator MessagingCoordinator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messagingCoord = coordinator
}

// Register adds transport to the coordinator
func (m *TransportCoordinator) Register(transport NetworkTransport) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transports = append(m.transports, transport)

	// Inject tracing manager if available
	if m.tracingManager != nil {
		m.injectTracingManagerToTransport(transport)
	}

	logger.Logger.Info("Transport registered",
		zap.String("transport", transport.GetName()))
}

// SetTracingManager sets the tracing manager and distributes it to all transports
func (m *TransportCoordinator) SetTracingManager(tracingManager tracing.TracingManager) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tracingManager = tracingManager

	// Inject tracing manager into all existing transports
	for _, transport := range m.transports {
		m.injectTracingManagerToTransport(transport)
	}

	logger.Logger.Info("Tracing manager set for transport coordinator")
}

// GetTracingManager returns the current tracing manager
func (m *TransportCoordinator) GetTracingManager() tracing.TracingManager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tracingManager
}

// injectTracingManagerToTransport injects the tracing manager into a transport
func (m *TransportCoordinator) injectTracingManagerToTransport(transport NetworkTransport) {
	// Check if transport is HTTP type and inject tracing manager
	if httpTransport, ok := transport.(*HTTPNetworkService); ok {
		if httpTransport.config != nil {
			httpTransport.config.TracingManager = m.tracingManager
		}
		logger.Logger.Debug("Tracing manager injected into HTTP transport")
	}

	// Check if transport is gRPC type and inject tracing manager
	if grpcTransport, ok := transport.(*GRPCNetworkService); ok {
		if grpcTransport.config != nil {
			grpcTransport.config.TracingManager = m.tracingManager
		}
		logger.Logger.Debug("Tracing manager injected into gRPC transport")
	}
}

// Start starts all registered transports and messaging coordinator if enabled
func (m *TransportCoordinator) Start(ctx context.Context) error {
	m.mu.RLock()
	transports := make([]NetworkTransport, len(m.transports))
	copy(transports, m.transports)
	messagingEnabled := m.messagingEnabled
	messagingCoord := m.messagingCoord
	m.mu.RUnlock()

	if messagingEnabled && messagingCoord == nil {
		return fmt.Errorf("messaging is enabled but coordinator is not set")
	}

	// Start messaging coordinator first if enabled
	if messagingEnabled && messagingCoord != nil {
		logger.Logger.Info("Starting messaging coordinator")
		if err := messagingCoord.Start(ctx); err != nil {
			return fmt.Errorf("failed to start messaging coordinator: %w", err)
		}
		logger.Logger.Info("Messaging coordinator started successfully")
	}

	// Start all network transports
	for _, transport := range transports {
		logger.Logger.Info("Starting transport",
			zap.String("transport", transport.GetName()))
		if err := transport.Start(ctx); err != nil {
			// If messaging was started, stop it on transport failure
			if messagingEnabled && messagingCoord != nil {
				if stopErr := messagingCoord.Stop(ctx); stopErr != nil {
					logger.Logger.Error("Failed to stop messaging coordinator after transport start failure", zap.Error(stopErr))
				}
			}
			return fmt.Errorf("failed to start %s transport: %w", transport.GetName(), err)
		}
	}

	logger.Logger.Info("All transports started successfully")
	return nil
}

// Stop gracefully stops all registered transports and messaging coordinator
// Returns a MultiError containing all errors encountered during shutdown
func (m *TransportCoordinator) Stop(timeout time.Duration) error {
	m.mu.RLock()
	transports := make([]NetworkTransport, len(m.transports))
	copy(transports, m.transports)
	messagingEnabled := m.messagingEnabled
	messagingCoord := m.messagingCoord
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var stopErrors []TransportError

	// Stop network transports first
	for _, transport := range transports {
		logger.Logger.Info("Stopping transport",
			zap.String("transport", transport.GetName()))
		if err := transport.Stop(ctx); err != nil {
			stopError := TransportError{
				TransportName: transport.GetName(),
				Err:           err,
			}
			stopErrors = append(stopErrors, stopError)
			logger.Logger.Error("Failed to stop transport",
				zap.String("transport", transport.GetName()),
				zap.Error(err))
			// Continue stopping other transports even if one fails
		}
	}

	// Stop messaging coordinator last if enabled
	if messagingEnabled && messagingCoord != nil {
		logger.Logger.Info("Stopping messaging coordinator")
		if err := messagingCoord.Stop(ctx); err != nil {
			stopError := TransportError{
				TransportName: "messaging",
				Err:           err,
			}
			stopErrors = append(stopErrors, stopError)
			logger.Logger.Error("Failed to stop messaging coordinator", zap.Error(err))
		} else {
			logger.Logger.Info("Messaging coordinator stopped successfully")
		}
	}

	if len(stopErrors) > 0 {
		multiErr := &MultiError{Errors: stopErrors}
		totalCount := len(transports)
		if messagingEnabled {
			totalCount++
		}
		logger.Logger.Error("Transport shutdown completed with errors",
			zap.Int("failed_count", len(stopErrors)),
			zap.Int("total_count", totalCount),
			zap.Error(multiErr))
		return multiErr
	}

	logger.Logger.Info("All transports and messaging stopped successfully")
	return nil
}

// GetTransports returns a list of all registered transports
func (m *TransportCoordinator) GetTransports() []NetworkTransport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	transports := make([]NetworkTransport, len(m.transports))
	copy(transports, m.transports)
	return transports
}

// GetMultiTransportRegistry returns the multi transport registry
func (m *TransportCoordinator) GetMultiTransportRegistry() *MultiTransportRegistry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registryManager
}

// RegisterHTTPService registers a service handler with HTTP transport
func (m *TransportCoordinator) RegisterHTTPService(handler TransportServiceHandler) error {
	return m.registryManager.RegisterHTTPService(handler)
}

// RegisterGRPCService registers a service handler with gRPC transport
func (m *TransportCoordinator) RegisterGRPCService(handler TransportServiceHandler) error {
	return m.registryManager.RegisterGRPCService(handler)
}

// RegisterMessagingService registers a service handler with messaging transport
func (m *TransportCoordinator) RegisterMessagingService(handler TransportServiceHandler) error {
	return m.registryManager.RegisterMessagingService(handler)
}

// RegisterHandler registers a service handler with a specific transport
func (m *TransportCoordinator) RegisterHandler(transportName string, handler TransportServiceHandler) error {
	return m.registryManager.RegisterHandler(transportName, handler)
}

// InitializeTransportServices initializes all services across all transports
func (m *TransportCoordinator) InitializeTransportServices(ctx context.Context) error {
	return m.registryManager.InitializeTransportServices(ctx)
}

// BindAllHTTPEndpoints registers HTTP routes for all services across all transports
func (m *TransportCoordinator) BindAllHTTPEndpoints(router *gin.Engine) error {
	return m.registryManager.BindAllHTTPEndpoints(router)
}

// BindAllGRPCServices registers gRPC services for all services across all transports
func (m *TransportCoordinator) BindAllGRPCServices(server *grpc.Server) error {
	return m.registryManager.BindAllGRPCServices(server)
}

// CheckAllServicesHealth performs health checks on all services across all transports
func (m *TransportCoordinator) CheckAllServicesHealth(ctx context.Context) map[string]map[string]*types.HealthStatus {
	return m.registryManager.CheckAllHealth(ctx)
}

// ShutdownAllServices gracefully shuts down all services across all transports
func (m *TransportCoordinator) ShutdownAllServices(ctx context.Context) error {
	return m.registryManager.ShutdownAll(ctx)
}

// GetAllServiceMetadata returns metadata for all services across all transports
func (m *TransportCoordinator) GetAllServiceMetadata() map[string][]*HandlerMetadata {
	return m.registryManager.GetAllServiceMetadata()
}

// GetTotalServiceCount returns the total number of services across all transports
func (m *TransportCoordinator) GetTotalServiceCount() int {
	return m.registryManager.GetTotalServiceCount()
}

// === Messaging Integration Methods ===

// EnableMessaging enables messaging support for this transport coordinator
func (m *TransportCoordinator) EnableMessaging() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messagingEnabled = true
}

// DisableMessaging disables messaging support for this transport coordinator
func (m *TransportCoordinator) DisableMessaging() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messagingEnabled = false
}

// IsMessagingEnabled returns whether messaging is enabled
func (m *TransportCoordinator) IsMessagingEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messagingEnabled
}

// GetMessagingCoordinator returns the messaging coordinator instance
func (m *TransportCoordinator) GetMessagingCoordinator() MessagingCoordinator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messagingCoord
}

// RegisterMessageBroker registers a message broker with the messaging coordinator
func (m *TransportCoordinator) RegisterMessageBroker(name string, broker MessageBroker) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.messagingEnabled {
		return fmt.Errorf("messaging is not enabled for this transport coordinator")
	}

	if m.messagingCoord == nil {
		return fmt.Errorf("messaging coordinator is not initialized")
	}

	if broker == nil {
		return fmt.Errorf("broker cannot be nil")
	}

	return m.messagingCoord.RegisterBroker(name, broker)
}

// RegisterEventHandler registers an event handler with the messaging coordinator
func (m *TransportCoordinator) RegisterEventHandler(handler EventHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.messagingEnabled {
		return fmt.Errorf("messaging is not enabled for this transport coordinator")
	}

	if m.messagingCoord == nil {
		return fmt.Errorf("messaging coordinator is not initialized")
	}

	return m.messagingCoord.RegisterEventHandler(handler)
}

// UnregisterEventHandler removes an event handler from the messaging coordinator
func (m *TransportCoordinator) UnregisterEventHandler(handlerID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.messagingEnabled {
		return fmt.Errorf("messaging is not enabled for this transport coordinator")
	}

	if m.messagingCoord == nil {
		return fmt.Errorf("messaging coordinator is not initialized")
	}

	return m.messagingCoord.UnregisterEventHandler(handlerID)
}

// GetMessagingMetrics returns messaging coordinator metrics if messaging is enabled
func (m *TransportCoordinator) GetMessagingMetrics() any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.messagingEnabled || m.messagingCoord == nil {
		return nil
	}

	return m.messagingCoord.GetMetrics()
}

// CheckMessagingHealth performs health check on messaging coordinator if enabled
func (m *TransportCoordinator) CheckMessagingHealth(ctx context.Context) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.messagingEnabled {
		return nil, fmt.Errorf("messaging is not enabled")
	}

	if m.messagingCoord == nil {
		return nil, fmt.Errorf("messaging coordinator is not initialized")
	}

	return m.messagingCoord.HealthCheck(ctx)
}

// GetRegisteredBrokers returns list of registered message brokers if messaging is enabled
func (m *TransportCoordinator) GetRegisteredBrokers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.messagingEnabled || m.messagingCoord == nil {
		return nil
	}

	return m.messagingCoord.GetRegisteredBrokers()
}

// GetRegisteredEventHandlers returns list of registered event handlers if messaging is enabled
func (m *TransportCoordinator) GetRegisteredEventHandlers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.messagingEnabled || m.messagingCoord == nil {
		return nil
	}

	return m.messagingCoord.GetRegisteredHandlers()
}
