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

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"go.uber.org/zap"
)

// TransportManager defines the interface for transport management in lifecycle
type TransportManager interface {
	GetTransports() []transport.NetworkTransport
	InitializeAllServices(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(timeout time.Duration) error
}

// LifecyclePhase represents different phases of server lifecycle
type LifecyclePhase int

const (
	PhaseUninitialized LifecyclePhase = iota
	PhaseDependenciesInitialized
	PhaseTransportsInitialized
	PhaseServicesRegistered
	PhaseDiscoveryRegistered
	PhaseStarted
	PhaseStopping
	PhaseStopped
)

// String returns the string representation of the lifecycle phase
func (p LifecyclePhase) String() string {
	switch p {
	case PhaseUninitialized:
		return "uninitialized"
	case PhaseDependenciesInitialized:
		return "dependencies_initialized"
	case PhaseTransportsInitialized:
		return "transports_initialized"
	case PhaseServicesRegistered:
		return "services_registered"
	case PhaseDiscoveryRegistered:
		return "discovery_registered"
	case PhaseStarted:
		return "started"
	case PhaseStopping:
		return "stopping"
	case PhaseStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// LifecycleManager manages the server startup and shutdown sequences
type LifecycleManager struct {
	serviceName      string
	currentPhase     LifecyclePhase
	transportManager TransportManager
	serviceDiscovery *discovery.ServiceDiscovery
	dependencies     BusinessDependencyContainer
	config           *ServerConfig

	// Cleanup functions for partial initialization failures
	cleanupFunctions []func() error
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(serviceName string, config *ServerConfig) *LifecycleManager {
	return &LifecycleManager{
		serviceName:      serviceName,
		currentPhase:     PhaseUninitialized,
		config:           config,
		cleanupFunctions: make([]func() error, 0),
	}
}

// SetTransportManager sets the transport manager for lifecycle management
func (lm *LifecycleManager) SetTransportManager(tm TransportManager) {
	lm.transportManager = tm
}

// SetServiceDiscovery sets the service discovery client for lifecycle management
func (lm *LifecycleManager) SetServiceDiscovery(sd *discovery.ServiceDiscovery) {
	lm.serviceDiscovery = sd
}

// SetDependencies sets the dependency container for lifecycle management
func (lm *LifecycleManager) SetDependencies(deps BusinessDependencyContainer) {
	lm.dependencies = deps
}

// GetCurrentPhase returns the current lifecycle phase
func (lm *LifecycleManager) GetCurrentPhase() LifecyclePhase {
	return lm.currentPhase
}

// StartupSequence executes the complete server startup sequence
// Order: dependencies → transports → service registration → discovery registration
func (lm *LifecycleManager) StartupSequence(ctx context.Context) error {
	logger.Logger.Info("Starting server lifecycle sequence",
		zap.String("service", lm.serviceName),
		zap.String("phase", lm.currentPhase.String()))

	// Phase 1: Initialize dependencies
	if err := lm.initializeDependencies(ctx); err != nil {
		return lm.handleStartupError("dependencies initialization", err)
	}

	// Phase 2: Initialize transports
	if err := lm.initializeTransports(ctx); err != nil {
		return lm.handleStartupError("transports initialization", err)
	}

	// Phase 3: Register services
	if err := lm.registerServices(ctx); err != nil {
		return lm.handleStartupError("services registration", err)
	}

	// Phase 4: Start transports
	if err := lm.startTransports(ctx); err != nil {
		return lm.handleStartupError("transports startup", err)
	}

	// Phase 5: Register with service discovery
	if err := lm.registerWithDiscovery(ctx); err != nil {
		return lm.handleStartupError("discovery registration", err)
	}

	lm.currentPhase = PhaseStarted
	logger.Logger.Info("Server startup sequence completed successfully",
		zap.String("service", lm.serviceName),
		zap.String("phase", lm.currentPhase.String()))

	return nil
}

// ShutdownSequence executes the complete server shutdown sequence
// Order: discovery deregistration → transports shutdown → dependencies cleanup
func (lm *LifecycleManager) ShutdownSequence(ctx context.Context) error {
	if lm.currentPhase == PhaseStopped {
		return nil // Already stopped
	}

	lm.currentPhase = PhaseStopping
	logger.Logger.Info("Starting server shutdown sequence",
		zap.String("service", lm.serviceName),
		zap.String("phase", lm.currentPhase.String()))

	var shutdownErrors []error

	// Phase 1: Deregister from service discovery
	if err := lm.deregisterFromDiscovery(ctx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("discovery deregistration failed: %w", err))
	}

	// Phase 2: Stop transports
	if err := lm.stopTransports(ctx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("transports shutdown failed: %w", err))
	}

	// Phase 3: Cleanup dependencies
	if err := lm.cleanupDependencies(ctx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("dependencies cleanup failed: %w", err))
	}

	lm.currentPhase = PhaseStopped

	if len(shutdownErrors) > 0 {
		logger.Logger.Error("Server shutdown completed with errors",
			zap.String("service", lm.serviceName),
			zap.Int("error_count", len(shutdownErrors)))
		return fmt.Errorf("shutdown errors: %v", shutdownErrors)
	}

	logger.Logger.Info("Server shutdown sequence completed successfully",
		zap.String("service", lm.serviceName),
		zap.String("phase", lm.currentPhase.String()))

	return nil
}

// initializeDependencies initializes the dependency container
func (lm *LifecycleManager) initializeDependencies(ctx context.Context) error {
	logger.Logger.Info("Initializing dependencies", zap.String("service", lm.serviceName))

	// Dependencies are typically initialized externally and passed in
	// This phase is mainly for validation and setup
	if lm.dependencies != nil {
		// Add cleanup function for dependencies
		lm.addCleanupFunction(func() error {
			return lm.dependencies.Close()
		})
	}

	lm.currentPhase = PhaseDependenciesInitialized
	logger.Logger.Info("Dependencies initialized successfully", zap.String("service", lm.serviceName))
	return nil
}

// initializeTransports initializes the transport layer
func (lm *LifecycleManager) initializeTransports(ctx context.Context) error {
	logger.Logger.Info("Initializing transports", zap.String("service", lm.serviceName))

	if lm.transportManager == nil {
		return fmt.Errorf("transport manager not set")
	}

	// Transport initialization is handled by the base server
	// This phase validates that transports are ready
	transports := lm.transportManager.GetTransports()
	logger.Logger.Info("Transports ready for startup",
		zap.String("service", lm.serviceName),
		zap.Int("transport_count", len(transports)))

	lm.currentPhase = PhaseTransportsInitialized
	return nil
}

// registerServices registers all services with the transport layer
func (lm *LifecycleManager) registerServices(ctx context.Context) error {
	logger.Logger.Info("Registering services", zap.String("service", lm.serviceName))

	if lm.transportManager == nil {
		return fmt.Errorf("transport manager not set")
	}

	// Initialize all services
	if err := lm.transportManager.InitializeAllServices(ctx); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	lm.currentPhase = PhaseServicesRegistered
	logger.Logger.Info("Services registered successfully", zap.String("service", lm.serviceName))
	return nil
}

// startTransports starts all registered transports
func (lm *LifecycleManager) startTransports(ctx context.Context) error {
	logger.Logger.Info("Starting transports", zap.String("service", lm.serviceName))

	if lm.transportManager == nil {
		return fmt.Errorf("transport manager not set")
	}

	// Start all transports
	if err := lm.transportManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transports: %w", err)
	}

	// Add cleanup function for transports
	lm.addCleanupFunction(func() error {
		return lm.transportManager.Stop(lm.config.ShutdownTimeout)
	})

	logger.Logger.Info("Transports started successfully", zap.String("service", lm.serviceName))
	return nil
}

// registerWithDiscovery registers the service with service discovery
func (lm *LifecycleManager) registerWithDiscovery(ctx context.Context) error {
	if !lm.config.IsDiscoveryEnabled() {
		logger.Logger.Info("Service discovery disabled, skipping registration",
			zap.String("service", lm.serviceName))
		lm.currentPhase = PhaseDiscoveryRegistered
		return nil
	}

	logger.Logger.Info("Registering with service discovery", zap.String("service", lm.serviceName))

	if lm.serviceDiscovery == nil {
		return fmt.Errorf("service discovery not set")
	}

	// Service discovery registration is handled by the base server
	// This phase is mainly for tracking and cleanup setup
	lm.addCleanupFunction(func() error {
		return lm.deregisterFromDiscovery(context.Background())
	})

	lm.currentPhase = PhaseDiscoveryRegistered
	logger.Logger.Info("Service discovery registration completed", zap.String("service", lm.serviceName))
	return nil
}

// deregisterFromDiscovery deregisters the service from service discovery
func (lm *LifecycleManager) deregisterFromDiscovery(ctx context.Context) error {
	if !lm.config.IsDiscoveryEnabled() || lm.serviceDiscovery == nil {
		return nil
	}

	logger.Logger.Info("Deregistering from service discovery", zap.String("service", lm.serviceName))

	// Service discovery deregistration is handled by the base server
	// This is a placeholder for future enhancements

	logger.Logger.Info("Service discovery deregistration completed", zap.String("service", lm.serviceName))
	return nil
}

// stopTransports stops all transports
func (lm *LifecycleManager) stopTransports(ctx context.Context) error {
	if lm.transportManager == nil {
		return nil
	}

	logger.Logger.Info("Stopping transports", zap.String("service", lm.serviceName))

	if err := lm.transportManager.Stop(lm.config.ShutdownTimeout); err != nil {
		return fmt.Errorf("failed to stop transports: %w", err)
	}

	logger.Logger.Info("Transports stopped successfully", zap.String("service", lm.serviceName))
	return nil
}

// cleanupDependencies cleans up the dependency container
func (lm *LifecycleManager) cleanupDependencies(ctx context.Context) error {
	if lm.dependencies == nil {
		return nil
	}

	logger.Logger.Info("Cleaning up dependencies", zap.String("service", lm.serviceName))

	if err := lm.dependencies.Close(); err != nil {
		return fmt.Errorf("failed to close dependencies: %w", err)
	}

	logger.Logger.Info("Dependencies cleaned up successfully", zap.String("service", lm.serviceName))
	return nil
}

// handleStartupError handles errors during startup and performs cleanup
func (lm *LifecycleManager) handleStartupError(phase string, err error) error {
	logger.Logger.Error("Startup error occurred, performing cleanup",
		zap.String("service", lm.serviceName),
		zap.String("phase", phase),
		zap.String("current_phase", lm.currentPhase.String()),
		zap.Error(err))

	// Perform cleanup of partially initialized resources
	if cleanupErr := lm.performCleanup(); cleanupErr != nil {
		logger.Logger.Error("Cleanup failed after startup error",
			zap.String("service", lm.serviceName),
			zap.Error(cleanupErr))
		return fmt.Errorf("startup failed in %s: %w (cleanup also failed: %v)", phase, err, cleanupErr)
	}

	return fmt.Errorf("startup failed in %s: %w", phase, err)
}

// addCleanupFunction adds a cleanup function to be called on failure or shutdown
func (lm *LifecycleManager) addCleanupFunction(fn func() error) {
	lm.cleanupFunctions = append(lm.cleanupFunctions, fn)
}

// performCleanup executes all registered cleanup functions in reverse order
func (lm *LifecycleManager) performCleanup() error {
	var cleanupErrors []error

	// Execute cleanup functions in reverse order (LIFO)
	for i := len(lm.cleanupFunctions) - 1; i >= 0; i-- {
		if err := lm.cleanupFunctions[i](); err != nil {
			cleanupErrors = append(cleanupErrors, err)
			logger.Logger.Error("Cleanup function failed",
				zap.String("service", lm.serviceName),
				zap.Int("function_index", i),
				zap.Error(err))
		}
	}

	// Clear cleanup functions after execution
	lm.cleanupFunctions = lm.cleanupFunctions[:0]

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup errors: %v", cleanupErrors)
	}

	return nil
}

// IsStarted returns true if the server is in started state
func (lm *LifecycleManager) IsStarted() bool {
	return lm.currentPhase == PhaseStarted
}

// IsStopped returns true if the server is in stopped state
func (lm *LifecycleManager) IsStopped() bool {
	return lm.currentPhase == PhaseStopped
}

// IsStopping returns true if the server is currently stopping
func (lm *LifecycleManager) IsStopping() bool {
	return lm.currentPhase == PhaseStopping
}
