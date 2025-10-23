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

// Package monitoring provides web dashboard capabilities for Saga monitoring and management.
// It includes the SagaDashboard core structure for real-time status display and operation intervention.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/innovationmech/swit/pkg/saga"
)

// DashboardConfig contains configuration for the Saga monitoring dashboard.
type DashboardConfig struct {
	// ServerConfig contains web server configuration.
	// If nil, default server configuration will be used.
	ServerConfig *ServerConfig

	// Coordinator is the Saga coordinator to monitor and control.
	// This is required and must be provided.
	Coordinator saga.SagaCoordinator

	// MetricsCollector collects and provides metrics data.
	// If nil, a default collector will be created.
	MetricsCollector MetricsCollector

	// HealthManager manages health checks for the system.
	// If nil, basic health checks will be used.
	HealthManager *SagaHealthManager

	// EnableAutoStart automatically starts the server when the dashboard is created.
	// Default is false.
	EnableAutoStart bool

	// ContextPropagation enables context propagation for all operations.
	// Default is true.
	EnableContextPropagation bool
}

// Validate validates the dashboard configuration.
func (c *DashboardConfig) Validate() error {
	if c.Coordinator == nil {
		return fmt.Errorf("coordinator is required")
	}

	// Validate server config if provided
	if c.ServerConfig != nil {
		if err := c.ServerConfig.Validate(); err != nil {
			return fmt.Errorf("invalid server configuration: %w", err)
		}
	}

	return nil
}

// DefaultDashboardConfig returns a default dashboard configuration.
// The coordinator must be provided separately as it's required.
func DefaultDashboardConfig() *DashboardConfig {
	return &DashboardConfig{
		ServerConfig:             DefaultServerConfig(),
		EnableAutoStart:          false,
		EnableContextPropagation: true,
	}
}

// SagaDashboard is the core structure for the Saga monitoring dashboard.
// It provides real-time status display, metrics collection, and operation intervention
// capabilities for Saga instances.
//
// Key responsibilities:
//  1. Web server lifecycle management (Start, Stop, Shutdown)
//  2. Integration with Saga coordinator for monitoring and control
//  3. Metrics collection and aggregation
//  4. Health check management
//  5. Context propagation for all operations
//  6. Thread-safe concurrent access
type SagaDashboard struct {
	config *DashboardConfig

	// Core dependencies
	coordinator      saga.SagaCoordinator
	metricsCollector MetricsCollector
	healthManager    *SagaHealthManager
	server           *MonitoringServer

	// State management
	mu      sync.RWMutex
	running atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewSagaDashboard creates a new Saga monitoring dashboard with the given configuration.
//
// Parameters:
//   - config: Dashboard configuration. If nil, default configuration is used.
//     Note: Coordinator must be provided in the config.
//
// Returns:
//   - A configured SagaDashboard ready to start.
//   - An error if the configuration is invalid or initialization fails.
//
// Example:
//
//	config := &monitoring.DashboardConfig{
//	    Coordinator: coordinator,
//	    ServerConfig: monitoring.DefaultServerConfig(),
//	}
//	dashboard, err := monitoring.NewSagaDashboard(config)
//	if err != nil {
//	    return err
//	}
//	if err := dashboard.Start(context.Background()); err != nil {
//	    return err
//	}
func NewSagaDashboard(config *DashboardConfig) (*SagaDashboard, error) {
	if config == nil {
		config = DefaultDashboardConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dashboard configuration: %w", err)
	}

	// Apply defaults for server config
	if config.ServerConfig == nil {
		config.ServerConfig = DefaultServerConfig()
	}

	// Create metrics collector if not provided
	if config.MetricsCollector == nil {
		collector, err := NewSagaMetricsCollector(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics collector: %w", err)
		}
		config.MetricsCollector = collector
	}

	// Create health manager if not provided
	if config.HealthManager == nil {
		config.HealthManager = NewSagaHealthManager(nil)

		// Register coordinator health checker
		coordChecker := NewCoordinatorHealthChecker(
			"saga-coordinator",
			config.Coordinator,
			1000, // Max 1000 active sagas before marking as degraded
		)
		if err := config.HealthManager.RegisterChecker(coordChecker); err != nil {
			return nil, fmt.Errorf("failed to register coordinator health checker: %w", err)
		}
	}

	// Link health manager to server config
	config.ServerConfig.HealthManager = config.HealthManager

	// Create monitoring server
	server, err := NewMonitoringServer(config.ServerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoring server: %w", err)
	}

	// Create and register Saga query API
	queryAPI := NewSagaQueryAPI(config.Coordinator)
	server.SetQueryAPI(queryAPI)

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	dashboard := &SagaDashboard{
		config:           config,
		coordinator:      config.Coordinator,
		metricsCollector: config.MetricsCollector,
		healthManager:    config.HealthManager,
		server:           server,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Auto-start if configured
	if config.EnableAutoStart {
		if err := dashboard.Start(ctx); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to auto-start dashboard: %w", err)
		}
	}

	return dashboard, nil
}

// Start starts the dashboard and its web server.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//     If context propagation is enabled, this context will be used for all operations.
//
// Returns:
//   - An error if the dashboard fails to start.
//
// Example:
//
//	if err := dashboard.Start(context.Background()); err != nil {
//	    log.Fatalf("Failed to start dashboard: %v", err)
//	}
func (d *SagaDashboard) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already running
	if d.running.Load() {
		return fmt.Errorf("dashboard is already running")
	}

	// Use provided context if context propagation is enabled
	if d.config.EnableContextPropagation && ctx != nil {
		// Create a new context that combines the dashboard context and provided context
		// This allows the dashboard to be cancelled from either source
		d.ctx, d.cancel = context.WithCancel(ctx)
	}

	// Mark health manager as ready
	d.healthManager.SetReady(true)

	// Start the server
	if err := d.server.Start(d.ctx); err != nil {
		d.healthManager.SetReady(false)
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Mark dashboard as running
	d.running.Store(true)

	return nil
}

// Stop stops the dashboard and its web server gracefully.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control during shutdown.
//
// Returns:
//   - An error if the dashboard fails to stop gracefully.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := dashboard.Stop(ctx); err != nil {
//	    log.Printf("Error during shutdown: %v", err)
//	}
func (d *SagaDashboard) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if running
	if !d.running.Load() {
		return nil // Already stopped
	}

	// Mark as not ready (but still alive for liveness checks)
	d.healthManager.SetReady(false)

	// Stop the server
	if err := d.server.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	// Mark dashboard as stopped
	d.running.Store(false)

	// Cancel the dashboard context
	if d.cancel != nil {
		d.cancel()
	}

	return nil
}

// Shutdown performs a complete shutdown of the dashboard and all its components.
// This is more thorough than Stop() and should be called when the dashboard
// will not be restarted.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control during shutdown.
//
// Returns:
//   - An error if shutdown fails.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := dashboard.Shutdown(ctx); err != nil {
//	    log.Printf("Error during shutdown: %v", err)
//	}
func (d *SagaDashboard) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if running
	if !d.running.Load() {
		// Even if not running, still perform cleanup
		if d.cancel != nil {
			d.cancel()
		}
		return nil
	}

	// Mark as not alive (indicates complete shutdown)
	d.healthManager.SetAlive(false)
	d.healthManager.SetReady(false)

	// Stop the server
	if err := d.server.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop server during shutdown: %w", err)
	}

	// Mark dashboard as stopped
	d.running.Store(false)

	// Cancel the dashboard context
	if d.cancel != nil {
		d.cancel()
	}

	return nil
}

// IsRunning returns true if the dashboard is currently running.
func (d *SagaDashboard) IsRunning() bool {
	return d.running.Load()
}

// GetCoordinator returns the Saga coordinator being monitored.
func (d *SagaDashboard) GetCoordinator() saga.SagaCoordinator {
	return d.coordinator
}

// GetMetricsCollector returns the metrics collector.
func (d *SagaDashboard) GetMetricsCollector() MetricsCollector {
	return d.metricsCollector
}

// GetHealthManager returns the health manager.
func (d *SagaDashboard) GetHealthManager() *SagaHealthManager {
	return d.healthManager
}

// GetServer returns the underlying monitoring server.
// This allows advanced users to register custom routes and middleware.
func (d *SagaDashboard) GetServer() *MonitoringServer {
	return d.server
}

// GetConfig returns the dashboard configuration.
func (d *SagaDashboard) GetConfig() *DashboardConfig {
	return d.config
}

// GetAddress returns the actual listening address after the server has started.
// Returns empty string if the server hasn't started yet.
func (d *SagaDashboard) GetAddress() string {
	return d.server.GetAddress()
}

// GetContext returns the dashboard context.
// This context is cancelled when the dashboard is stopped or shut down.
func (d *SagaDashboard) GetContext() context.Context {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ctx
}
