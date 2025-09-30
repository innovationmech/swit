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

package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
	"go.uber.org/zap"
)

// HealthChecker defines the interface for health check implementations
type HealthChecker interface {
	// Check performs a health check and returns the status
	Check(ctx context.Context) error
	// GetName returns the name of the health check
	GetName() string
}

// ReadinessChecker defines the interface for readiness checks
type ReadinessChecker interface {
	// CheckReadiness checks if the service is ready to accept traffic
	CheckReadiness(ctx context.Context) error
	// GetName returns the name of the readiness check
	GetName() string
}

// LivenessChecker defines the interface for liveness checks
type LivenessChecker interface {
	// CheckLiveness checks if the service is alive and should not be restarted
	CheckLiveness(ctx context.Context) error
	// GetName returns the name of the liveness check
	GetName() string
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthCheckConfig contains configuration for health checks
type HealthCheckConfig struct {
	// Timeout is the maximum duration for a health check
	Timeout time.Duration
	// Interval is the interval between periodic health checks
	Interval time.Duration
	// MaxConcurrent is the maximum number of concurrent health checks
	MaxConcurrent int
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Timeout:       5 * time.Second,
		Interval:      30 * time.Second,
		MaxConcurrent: 10,
	}
}

// HealthCheckAdapter provides a unified adapter for health checks
type HealthCheckAdapter struct {
	mu               sync.RWMutex
	healthCheckers   map[string]HealthChecker
	readinessChecker map[string]ReadinessChecker
	livenessCheckers map[string]LivenessChecker
	config           *HealthCheckConfig
	lastResults      map[string]*HealthCheckResult
}

// NewHealthCheckAdapter creates a new health check adapter
func NewHealthCheckAdapter(config *HealthCheckConfig) *HealthCheckAdapter {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}

	return &HealthCheckAdapter{
		healthCheckers:   make(map[string]HealthChecker),
		readinessChecker: make(map[string]ReadinessChecker),
		livenessCheckers: make(map[string]LivenessChecker),
		config:           config,
		lastResults:      make(map[string]*HealthCheckResult),
	}
}

// RegisterHealthChecker registers a health checker
func (a *HealthCheckAdapter) RegisterHealthChecker(checker HealthChecker) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := checker.GetName()
	if _, exists := a.healthCheckers[name]; exists {
		return fmt.Errorf("health checker %s already registered", name)
	}

	a.healthCheckers[name] = checker
	logger.Logger.Info("Registered health checker", zap.String("name", name))
	return nil
}

// RegisterReadinessChecker registers a readiness checker
func (a *HealthCheckAdapter) RegisterReadinessChecker(checker ReadinessChecker) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := checker.GetName()
	if _, exists := a.readinessChecker[name]; exists {
		return fmt.Errorf("readiness checker %s already registered", name)
	}

	a.readinessChecker[name] = checker
	logger.Logger.Info("Registered readiness checker", zap.String("name", name))
	return nil
}

// RegisterLivenessChecker registers a liveness checker
func (a *HealthCheckAdapter) RegisterLivenessChecker(checker LivenessChecker) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := checker.GetName()
	if _, exists := a.livenessCheckers[name]; exists {
		return fmt.Errorf("liveness checker %s already registered", name)
	}

	a.livenessCheckers[name] = checker
	logger.Logger.Info("Registered liveness checker", zap.String("name", name))
	return nil
}

// CheckHealth performs all registered health checks
func (a *HealthCheckAdapter) CheckHealth(ctx context.Context) (*types.HealthStatus, error) {
	a.mu.RLock()
	checkers := make([]HealthChecker, 0, len(a.healthCheckers))
	for _, checker := range a.healthCheckers {
		checkers = append(checkers, checker)
	}
	a.mu.RUnlock()

	results := a.runHealthChecks(ctx, checkers)
	return a.aggregateResults(results)
}

// CheckReadiness performs all registered readiness checks
func (a *HealthCheckAdapter) CheckReadiness(ctx context.Context) (*types.HealthStatus, error) {
	a.mu.RLock()
	checkers := make([]ReadinessChecker, 0, len(a.readinessChecker))
	for _, checker := range a.readinessChecker {
		checkers = append(checkers, checker)
	}
	a.mu.RUnlock()

	results := a.runReadinessChecks(ctx, checkers)
	return a.aggregateResults(results)
}

// CheckLiveness performs all registered liveness checks
func (a *HealthCheckAdapter) CheckLiveness(ctx context.Context) (*types.HealthStatus, error) {
	a.mu.RLock()
	checkers := make([]LivenessChecker, 0, len(a.livenessCheckers))
	for _, checker := range a.livenessCheckers {
		checkers = append(checkers, checker)
	}
	a.mu.RUnlock()

	results := a.runLivenessChecks(ctx, checkers)
	return a.aggregateResults(results)
}

// runHealthChecks executes health checks concurrently with timeout
func (a *HealthCheckAdapter) runHealthChecks(ctx context.Context, checkers []HealthChecker) []*HealthCheckResult {
	results := make([]*HealthCheckResult, 0, len(checkers))
	resultChan := make(chan *HealthCheckResult, len(checkers))

	sem := make(chan struct{}, a.config.MaxConcurrent)

	var wg sync.WaitGroup
	for _, checker := range checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := a.executeHealthCheck(ctx, c)
			resultChan <- result
		}(checker)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
		a.cacheResult(result)
	}

	return results
}

// runReadinessChecks executes readiness checks concurrently with timeout
func (a *HealthCheckAdapter) runReadinessChecks(ctx context.Context, checkers []ReadinessChecker) []*HealthCheckResult {
	results := make([]*HealthCheckResult, 0, len(checkers))
	resultChan := make(chan *HealthCheckResult, len(checkers))

	sem := make(chan struct{}, a.config.MaxConcurrent)

	var wg sync.WaitGroup
	for _, checker := range checkers {
		wg.Add(1)
		go func(c ReadinessChecker) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := a.executeReadinessCheck(ctx, c)
			resultChan <- result
		}(checker)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
		a.cacheResult(result)
	}

	return results
}

// runLivenessChecks executes liveness checks concurrently with timeout
func (a *HealthCheckAdapter) runLivenessChecks(ctx context.Context, checkers []LivenessChecker) []*HealthCheckResult {
	results := make([]*HealthCheckResult, 0, len(checkers))
	resultChan := make(chan *HealthCheckResult, len(checkers))

	sem := make(chan struct{}, a.config.MaxConcurrent)

	var wg sync.WaitGroup
	for _, checker := range checkers {
		wg.Add(1)
		go func(c LivenessChecker) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := a.executeLivenessCheck(ctx, c)
			resultChan <- result
		}(checker)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		results = append(results, result)
		a.cacheResult(result)
	}

	return results
}

// executeHealthCheck executes a single health check with timeout
func (a *HealthCheckAdapter) executeHealthCheck(ctx context.Context, checker HealthChecker) *HealthCheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	result := &HealthCheckResult{
		Name:      checker.GetName(),
		Timestamp: time.Now(),
	}

	start := time.Now()
	err := checker.Check(checkCtx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = types.HealthStatusUnhealthy
		result.Error = err.Error()
		logger.Logger.Debug("Health check failed",
			zap.String("name", result.Name),
			zap.Error(err),
			zap.Duration("duration", result.Duration))
	} else {
		result.Status = types.HealthStatusHealthy
		logger.Logger.Debug("Health check passed",
			zap.String("name", result.Name),
			zap.Duration("duration", result.Duration))
	}

	return result
}

// executeReadinessCheck executes a single readiness check with timeout
func (a *HealthCheckAdapter) executeReadinessCheck(ctx context.Context, checker ReadinessChecker) *HealthCheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	result := &HealthCheckResult{
		Name:      checker.GetName(),
		Timestamp: time.Now(),
	}

	start := time.Now()
	err := checker.CheckReadiness(checkCtx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = types.HealthStatusUnhealthy
		result.Error = err.Error()
		result.Message = "service not ready"
		logger.Logger.Debug("Readiness check failed",
			zap.String("name", result.Name),
			zap.Error(err),
			zap.Duration("duration", result.Duration))
	} else {
		result.Status = types.HealthStatusHealthy
		result.Message = "service ready"
		logger.Logger.Debug("Readiness check passed",
			zap.String("name", result.Name),
			zap.Duration("duration", result.Duration))
	}

	return result
}

// executeLivenessCheck executes a single liveness check with timeout
func (a *HealthCheckAdapter) executeLivenessCheck(ctx context.Context, checker LivenessChecker) *HealthCheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	result := &HealthCheckResult{
		Name:      checker.GetName(),
		Timestamp: time.Now(),
	}

	start := time.Now()
	err := checker.CheckLiveness(checkCtx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = types.HealthStatusUnhealthy
		result.Error = err.Error()
		result.Message = "service not alive"
		logger.Logger.Debug("Liveness check failed",
			zap.String("name", result.Name),
			zap.Error(err),
			zap.Duration("duration", result.Duration))
	} else {
		result.Status = types.HealthStatusHealthy
		result.Message = "service alive"
		logger.Logger.Debug("Liveness check passed",
			zap.String("name", result.Name),
			zap.Duration("duration", result.Duration))
	}

	return result
}

// aggregateResults aggregates check results into overall health status
func (a *HealthCheckAdapter) aggregateResults(results []*HealthCheckResult) (*types.HealthStatus, error) {
	healthStatus := types.NewHealthStatus(types.HealthStatusHealthy, "v1", 0)

	var failed int
	for _, result := range results {
		depStatus := types.DependencyStatus{
			Status:    result.Status,
			Timestamp: result.Timestamp,
			Latency:   result.Duration,
		}

		if result.Error != "" {
			depStatus.Error = result.Error
			failed++
		}

		healthStatus.AddDependency(result.Name, depStatus)
	}

	// If any check failed, mark overall status as unhealthy
	if failed > 0 {
		healthStatus.Status = types.HealthStatusUnhealthy
	}

	return healthStatus, nil
}

// cacheResult caches the result of a health check
func (a *HealthCheckAdapter) cacheResult(result *HealthCheckResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastResults[result.Name] = result
}

// GetLastResult returns the last cached result for a checker
func (a *HealthCheckAdapter) GetLastResult(name string) (*HealthCheckResult, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result, exists := a.lastResults[name]
	return result, exists
}

// GetAllLastResults returns all cached results
func (a *HealthCheckAdapter) GetAllLastResults() map[string]*HealthCheckResult {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := make(map[string]*HealthCheckResult, len(a.lastResults))
	for k, v := range a.lastResults {
		results[k] = v
	}
	return results
}
