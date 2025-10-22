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

// Package monitoring provides health check capabilities for the Saga system.
// It includes health checkers for coordinators, storage, messaging, and provides
// liveness and readiness probes compatible with Kubernetes and other orchestrators.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// HealthStatus represents the health status of a component.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is healthy.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy indicates the component is unhealthy.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusDegraded indicates the component is degraded but operational.
	HealthStatusDegraded HealthStatus = "degraded"
)

// ComponentHealth represents the health of a single component.
type ComponentHealth struct {
	// Name is the component name.
	Name string `json:"name"`
	// Status is the health status.
	Status HealthStatus `json:"status"`
	// Message provides additional information about the health status.
	Message string `json:"message,omitempty"`
	// Error contains error details if the component is unhealthy.
	Error string `json:"error,omitempty"`
	// CheckDuration is the time taken to perform the health check.
	CheckDuration time.Duration `json:"check_duration"`
	// Timestamp is when the health check was performed.
	Timestamp time.Time `json:"timestamp"`
}

// HealthReport contains the overall health status of the Saga system.
type HealthReport struct {
	// Status is the overall health status.
	Status HealthStatus `json:"status"`
	// Components contains health information for each component.
	Components map[string]*ComponentHealth `json:"components"`
	// Timestamp is when the health report was generated.
	Timestamp time.Time `json:"timestamp"`
	// TotalCheckDuration is the total time taken to perform all checks.
	TotalCheckDuration time.Duration `json:"total_check_duration"`
}

// HealthChecker defines the interface for health check implementations.
type HealthChecker interface {
	// Check performs a health check and returns the component health.
	Check(ctx context.Context) *ComponentHealth
	// GetName returns the name of the component being checked.
	GetName() string
}

// CoordinatorHealthChecker checks the health of a Saga coordinator.
type CoordinatorHealthChecker struct {
	name        string
	coordinator saga.SagaCoordinator
	// maxActiveSagas is the threshold for degraded status.
	maxActiveSagas int64
}

// NewCoordinatorHealthChecker creates a new coordinator health checker.
//
// Parameters:
//   - name: The name of the coordinator (e.g., "orchestrator", "choreography").
//   - coordinator: The coordinator to check.
//   - maxActiveSagas: The maximum number of active Sagas before marking as degraded (0 = no limit).
//
// Returns:
//   - A configured CoordinatorHealthChecker.
func NewCoordinatorHealthChecker(name string, coordinator saga.SagaCoordinator, maxActiveSagas int64) *CoordinatorHealthChecker {
	return &CoordinatorHealthChecker{
		name:           name,
		coordinator:    coordinator,
		maxActiveSagas: maxActiveSagas,
	}
}

// Check performs the coordinator health check.
func (c *CoordinatorHealthChecker) Check(ctx context.Context) *ComponentHealth {
	start := time.Now()
	health := &ComponentHealth{
		Name:      c.name,
		Timestamp: start,
	}

	// Check if coordinator responds
	err := c.coordinator.HealthCheck(ctx)
	health.CheckDuration = time.Since(start)

	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.Error = err.Error()
		health.Message = "coordinator health check failed"
		return health
	}

	// Get metrics to check for degraded state
	metrics := c.coordinator.GetMetrics()
	if metrics == nil {
		health.Status = HealthStatusHealthy
		health.Message = "coordinator is healthy"
		return health
	}

	// Check if we're over the threshold
	if c.maxActiveSagas > 0 && metrics.ActiveSagas > c.maxActiveSagas {
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("active sagas (%d) exceeds threshold (%d)", metrics.ActiveSagas, c.maxActiveSagas)
		return health
	}

	health.Status = HealthStatusHealthy
	health.Message = fmt.Sprintf("coordinator healthy with %d active sagas", metrics.ActiveSagas)
	return health
}

// GetName returns the coordinator name.
func (c *CoordinatorHealthChecker) GetName() string {
	return c.name
}

// StorageHealthChecker checks the health of a Saga state storage.
type StorageHealthChecker struct {
	name    string
	storage saga.StateStorage
	timeout time.Duration
}

// NewStorageHealthChecker creates a new storage health checker.
//
// Parameters:
//   - name: The name of the storage (e.g., "postgres", "redis", "memory").
//   - storage: The storage to check.
//   - timeout: The timeout for health check operations (default: 5s).
//
// Returns:
//   - A configured StorageHealthChecker.
func NewStorageHealthChecker(name string, storage saga.StateStorage, timeout time.Duration) *StorageHealthChecker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &StorageHealthChecker{
		name:    name,
		storage: storage,
		timeout: timeout,
	}
}

// Check performs the storage health check.
func (s *StorageHealthChecker) Check(ctx context.Context) *ComponentHealth {
	start := time.Now()
	health := &ComponentHealth{
		Name:      s.name,
		Timestamp: start,
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Try to list active Sagas as a health check
	// This tests both connectivity and basic query functionality
	_, err := s.storage.GetActiveSagas(checkCtx, nil)
	health.CheckDuration = time.Since(start)

	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.Error = err.Error()
		health.Message = "storage health check failed"
		return health
	}

	health.Status = HealthStatusHealthy
	health.Message = "storage is healthy"

	// Check if response time indicates degraded performance
	if health.CheckDuration > s.timeout/2 {
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("storage response time degraded (%v)", health.CheckDuration)
	}

	return health
}

// GetName returns the storage name.
func (s *StorageHealthChecker) GetName() string {
	return s.name
}

// MessagingHealthChecker checks the health of a Saga event publisher (messaging system).
type MessagingHealthChecker struct {
	name      string
	publisher saga.EventPublisher
	// isConnected is a function that checks if the messaging system is connected.
	// This is optional and allows for custom connectivity checks.
	isConnected func(ctx context.Context) error
}

// NewMessagingHealthChecker creates a new messaging health checker.
//
// Parameters:
//   - name: The name of the messaging system (e.g., "nats", "rabbitmq").
//   - publisher: The event publisher to check.
//   - isConnected: Optional function to check connectivity (can be nil).
//
// Returns:
//   - A configured MessagingHealthChecker.
func NewMessagingHealthChecker(name string, publisher saga.EventPublisher, isConnected func(ctx context.Context) error) *MessagingHealthChecker {
	return &MessagingHealthChecker{
		name:        name,
		publisher:   publisher,
		isConnected: isConnected,
	}
}

// Check performs the messaging health check.
func (m *MessagingHealthChecker) Check(ctx context.Context) *ComponentHealth {
	start := time.Now()
	health := &ComponentHealth{
		Name:      m.name,
		Timestamp: start,
	}

	// If custom connectivity check is provided, use it
	if m.isConnected != nil {
		err := m.isConnected(ctx)
		health.CheckDuration = time.Since(start)

		if err != nil {
			health.Status = HealthStatusUnhealthy
			health.Error = err.Error()
			health.Message = "messaging system not connected"
			return health
		}
	}

	health.CheckDuration = time.Since(start)
	health.Status = HealthStatusHealthy
	health.Message = "messaging system is healthy"
	return health
}

// GetName returns the messaging system name.
func (m *MessagingHealthChecker) GetName() string {
	return m.name
}

// SagaHealthManager manages health checks for the entire Saga system.
// It aggregates multiple health checkers and provides overall system health status.
type SagaHealthManager struct {
	mu             sync.RWMutex
	checkers       map[string]HealthChecker
	lastReport     *HealthReport
	lastReportTime time.Time
	cacheDuration  time.Duration
	checkTimeout   time.Duration
	ready          atomic.Bool
	alive          atomic.Bool
}

// HealthManagerConfig contains configuration for the health manager.
type HealthManagerConfig struct {
	// CacheDuration is how long to cache health check results (default: 5s).
	CacheDuration time.Duration
	// CheckTimeout is the timeout for individual health checks (default: 10s).
	CheckTimeout time.Duration
	// InitiallyReady indicates if the system should start in ready state (default: false).
	InitiallyReady bool
	// InitiallyAlive indicates if the system should start in alive state (default: true).
	InitiallyAlive bool
}

// DefaultHealthManagerConfig returns a default configuration.
func DefaultHealthManagerConfig() *HealthManagerConfig {
	return &HealthManagerConfig{
		CacheDuration:  5 * time.Second,
		CheckTimeout:   10 * time.Second,
		InitiallyReady: false,
		InitiallyAlive: true,
	}
}

// NewSagaHealthManager creates a new Saga health manager.
//
// Parameters:
//   - config: Configuration for the health manager (nil uses defaults).
//
// Returns:
//   - A configured SagaHealthManager.
func NewSagaHealthManager(config *HealthManagerConfig) *SagaHealthManager {
	if config == nil {
		config = DefaultHealthManagerConfig()
	}

	manager := &SagaHealthManager{
		checkers:      make(map[string]HealthChecker),
		cacheDuration: config.CacheDuration,
		checkTimeout:  config.CheckTimeout,
	}

	manager.ready.Store(config.InitiallyReady)
	manager.alive.Store(config.InitiallyAlive)

	return manager
}

// RegisterChecker registers a health checker.
//
// Parameters:
//   - checker: The health checker to register.
//
// Returns:
//   - An error if a checker with the same name is already registered.
func (h *SagaHealthManager) RegisterChecker(checker HealthChecker) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := checker.GetName()
	if _, exists := h.checkers[name]; exists {
		return fmt.Errorf("health checker %s already registered", name)
	}

	h.checkers[name] = checker
	return nil
}

// UnregisterChecker unregisters a health checker by name.
//
// Parameters:
//   - name: The name of the checker to unregister.
func (h *SagaHealthManager) UnregisterChecker(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checkers, name)
}

// CheckHealth performs all health checks and returns an aggregated report.
// Results are cached for the configured cache duration.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//
// Returns:
//   - A health report containing the status of all components.
//   - An error if the health check fails.
func (h *SagaHealthManager) CheckHealth(ctx context.Context) (*HealthReport, error) {
	h.mu.RLock()

	// Return cached result if still valid
	if h.lastReport != nil && time.Since(h.lastReportTime) < h.cacheDuration {
		report := h.lastReport
		h.mu.RUnlock()
		return report, nil
	}

	// Get snapshot of checkers
	checkers := make([]HealthChecker, 0, len(h.checkers))
	for _, checker := range h.checkers {
		checkers = append(checkers, checker)
	}
	h.mu.RUnlock()

	// Perform all health checks
	start := time.Now()
	components := h.runHealthChecks(ctx, checkers)

	// Aggregate results
	report := &HealthReport{
		Status:             HealthStatusHealthy,
		Components:         components,
		Timestamp:          time.Now(),
		TotalCheckDuration: time.Since(start),
	}

	// Determine overall status
	for _, comp := range components {
		if comp.Status == HealthStatusUnhealthy {
			report.Status = HealthStatusUnhealthy
			break
		}
		if comp.Status == HealthStatusDegraded && report.Status == HealthStatusHealthy {
			report.Status = HealthStatusDegraded
		}
	}

	// Cache result
	h.mu.Lock()
	h.lastReport = report
	h.lastReportTime = time.Now()
	h.mu.Unlock()

	return report, nil
}

// runHealthChecks executes all health checks concurrently.
func (h *SagaHealthManager) runHealthChecks(ctx context.Context, checkers []HealthChecker) map[string]*ComponentHealth {
	results := make(map[string]*ComponentHealth)
	resultChan := make(chan *ComponentHealth, len(checkers))

	var wg sync.WaitGroup
	for _, checker := range checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()

			// Create context with timeout
			checkCtx, cancel := context.WithTimeout(ctx, h.checkTimeout)
			defer cancel()

			result := c.Check(checkCtx)
			resultChan <- result
		}(checker)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results[result.Name] = result
	}

	return results
}

// CheckReadiness performs readiness checks.
// A service is ready when all critical components are healthy.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//
// Returns:
//   - true if the system is ready, false otherwise.
//   - An error if the readiness check fails.
func (h *SagaHealthManager) CheckReadiness(ctx context.Context) (bool, error) {
	// If manually marked as not ready, return immediately
	if !h.ready.Load() {
		return false, fmt.Errorf("system marked as not ready")
	}

	// Perform health checks
	report, err := h.CheckHealth(ctx)
	if err != nil {
		return false, fmt.Errorf("health check failed: %w", err)
	}

	// Ready if status is healthy or degraded (but not unhealthy)
	return report.Status != HealthStatusUnhealthy, nil
}

// CheckLiveness performs liveness checks.
// A service is alive if it can respond to requests (not deadlocked or crashed).
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//
// Returns:
//   - true if the system is alive, false otherwise.
//   - An error if the liveness check fails.
func (h *SagaHealthManager) CheckLiveness(ctx context.Context) (bool, error) {
	// If manually marked as not alive, return immediately
	if !h.alive.Load() {
		return false, fmt.Errorf("system marked as not alive")
	}

	// Liveness is primarily about the application responding
	// We perform a lightweight check
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		// Application is responsive
		return true, nil
	}
}

// SetReady manually sets the readiness state.
// This is useful during startup/shutdown operations.
//
// Parameters:
//   - ready: The desired readiness state.
func (h *SagaHealthManager) SetReady(ready bool) {
	h.ready.Store(ready)
}

// SetAlive manually sets the liveness state.
// This is useful for controlled shutdown or detecting fatal errors.
//
// Parameters:
//   - alive: The desired liveness state.
func (h *SagaHealthManager) SetAlive(alive bool) {
	h.alive.Store(alive)
}

// IsReady returns the current readiness state without performing checks.
func (h *SagaHealthManager) IsReady() bool {
	return h.ready.Load()
}

// IsAlive returns the current liveness state without performing checks.
func (h *SagaHealthManager) IsAlive() bool {
	return h.alive.Load()
}

// GetLastReport returns the last cached health report.
// Returns nil if no report has been generated yet.
func (h *SagaHealthManager) GetLastReport() *HealthReport {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastReport
}

// ClearCache clears the cached health report, forcing a fresh check on the next request.
func (h *SagaHealthManager) ClearCache() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastReport = nil
	h.lastReportTime = time.Time{}
}
