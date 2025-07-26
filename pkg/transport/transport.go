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
	"github.com/innovationmech/swit/pkg/types"
	"go.uber.org/zap"
)

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

// Transport defines the interface for different transport mechanisms
type Transport interface {
	// Start starts the transport server
	Start(ctx context.Context) error
	// Stop gracefully stops the transport server
	Stop(ctx context.Context) error
	// GetName returns the transport name
	GetName() string
	// GetAddress returns the listening address
	GetAddress() string
}

// Manager manages multiple transport instances and their service registries
type Manager struct {
	transports      []Transport
	registryManager *ServiceRegistryManager
	mu              sync.RWMutex
}

// NewManager creates a new transport manager
func NewManager() *Manager {
	return &Manager{
		transports:      make([]Transport, 0),
		registryManager: NewServiceRegistryManager(),
	}
}

// Register adds transport to the manager
func (m *Manager) Register(transport Transport) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transports = append(m.transports, transport)
	logger.Logger.Info("Transport registered",
		zap.String("transport", transport.GetName()))
}

// Start starts all registered transports
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	transports := make([]Transport, len(m.transports))
	copy(transports, m.transports)
	m.mu.RUnlock()

	for _, transport := range transports {
		logger.Logger.Info("Starting transport",
			zap.String("transport", transport.GetName()))
		if err := transport.Start(ctx); err != nil {
			return fmt.Errorf("failed to start %s transport: %w", transport.GetName(), err)
		}
	}

	logger.Logger.Info("All transports started successfully")
	return nil
}

// Stop gracefully stops all registered transports
// Returns a MultiError containing all errors encountered during shutdown
func (m *Manager) Stop(timeout time.Duration) error {
	m.mu.RLock()
	transports := make([]Transport, len(m.transports))
	copy(transports, m.transports)
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var stopErrors []TransportError

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

	if len(stopErrors) > 0 {
		multiErr := &MultiError{Errors: stopErrors}
		logger.Logger.Error("Transport shutdown completed with errors",
			zap.Int("failed_count", len(stopErrors)),
			zap.Int("total_count", len(transports)),
			zap.Error(multiErr))
		return multiErr
	}

	logger.Logger.Info("All transports stopped successfully")
	return nil
}

// GetTransports returns a list of all registered transports
func (m *Manager) GetTransports() []Transport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	transports := make([]Transport, len(m.transports))
	copy(transports, m.transports)
	return transports
}

// GetServiceRegistryManager returns the service registry manager
func (m *Manager) GetServiceRegistryManager() *ServiceRegistryManager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registryManager
}

// RegisterHTTPHandler registers a service handler with HTTP transport
func (m *Manager) RegisterHTTPHandler(handler HandlerRegister) error {
	return m.registryManager.RegisterHTTPHandler(handler)
}

// RegisterGRPCHandler registers a service handler with gRPC transport
func (m *Manager) RegisterGRPCHandler(handler HandlerRegister) error {
	return m.registryManager.RegisterGRPCHandler(handler)
}

// RegisterHandler registers a service handler with a specific transport
func (m *Manager) RegisterHandler(transportName string, handler HandlerRegister) error {
	return m.registryManager.RegisterHandler(transportName, handler)
}

// InitializeAllServices initializes all services across all transports
func (m *Manager) InitializeAllServices(ctx context.Context) error {
	return m.registryManager.InitializeAll(ctx)
}

// RegisterAllHTTPRoutes registers HTTP routes for all services across all transports
func (m *Manager) RegisterAllHTTPRoutes(router *gin.Engine) error {
	return m.registryManager.RegisterAllHTTP(router)
}

// RegisterAllGRPCServices registers gRPC services for all services across all transports
func (m *Manager) RegisterAllGRPCServices(server *grpc.Server) error {
	return m.registryManager.RegisterAllGRPC(server)
}

// CheckAllServicesHealth performs health checks on all services across all transports
func (m *Manager) CheckAllServicesHealth(ctx context.Context) map[string]map[string]*types.HealthStatus {
	return m.registryManager.CheckAllHealth(ctx)
}

// ShutdownAllServices gracefully shuts down all services across all transports
func (m *Manager) ShutdownAllServices(ctx context.Context) error {
	return m.registryManager.ShutdownAll(ctx)
}

// GetAllServiceMetadata returns metadata for all services across all transports
func (m *Manager) GetAllServiceMetadata() map[string][]*HandlerMetadata {
	return m.registryManager.GetAllServiceMetadata()
}

// GetTotalServiceCount returns the total number of services across all transports
func (m *Manager) GetTotalServiceCount() int {
	return m.registryManager.GetTotalServiceCount()
}
