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
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// PerformanceMetrics holds comprehensive performance metrics for a server
type PerformanceMetrics struct {
	// Timing metrics
	StartupTime  time.Duration `json:"startup_time"`
	ShutdownTime time.Duration `json:"shutdown_time"`
	Uptime       time.Duration `json:"uptime"`

	// Resource metrics
	MemoryUsage    uint64  `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`
	CPUUsage       float64 `json:"cpu_usage"`

	// Service metrics
	ServiceCount   int `json:"service_count"`
	TransportCount int `json:"transport_count"`

	// Operation counters
	StartCount   int64 `json:"start_count"`
	StopCount    int64 `json:"stop_count"`
	RequestCount int64 `json:"request_count"`
	ErrorCount   int64 `json:"error_count"`

	// Messaging metrics integration
	MessagingMetrics interface{} `json:"messaging_metrics,omitempty"`

	// Performance thresholds
	MaxStartupTime  time.Duration `json:"max_startup_time"`
	MaxShutdownTime time.Duration `json:"max_shutdown_time"`
	MaxMemoryUsage  uint64        `json:"max_memory_usage"`

	// Timestamp
	LastUpdated time.Time `json:"last_updated"`

	mu sync.RWMutex
}

// NewPerformanceMetrics creates a new performance metrics instance with default thresholds
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		MaxStartupTime:  500 * time.Millisecond,
		MaxShutdownTime: 200 * time.Millisecond,
		MaxMemoryUsage:  50 * 1024 * 1024, // 50MB
		LastUpdated:     time.Now(),
	}
}

// RecordStartupTime records the server startup time and increments start counter
func (m *PerformanceMetrics) RecordStartupTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartupTime = duration
	m.LastUpdated = time.Now()
	atomic.AddInt64(&m.StartCount, 1)
}

// RecordShutdownTime records the server shutdown time and increments stop counter
func (m *PerformanceMetrics) RecordShutdownTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShutdownTime = duration
	m.LastUpdated = time.Now()
	atomic.AddInt64(&m.StopCount, 1)
}

// RecordMemoryUsage records current memory usage and goroutine count
func (m *PerformanceMetrics) RecordMemoryUsage() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.MemoryUsage = memStats.Alloc
	m.GoroutineCount = runtime.NumGoroutine()
	m.LastUpdated = time.Now()
}

// RecordServiceMetrics records service and transport counts
func (m *PerformanceMetrics) RecordServiceMetrics(serviceCount, transportCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ServiceCount = serviceCount
	m.TransportCount = transportCount
	m.LastUpdated = time.Now()
}

// RecordUptime records the current uptime
func (m *PerformanceMetrics) RecordUptime(uptime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Uptime = uptime
	m.LastUpdated = time.Now()
}

// IncrementRequestCount atomically increments the request counter
func (m *PerformanceMetrics) IncrementRequestCount() {
	atomic.AddInt64(&m.RequestCount, 1)
}

// IncrementErrorCount atomically increments the error counter
func (m *PerformanceMetrics) IncrementErrorCount() {
	atomic.AddInt64(&m.ErrorCount, 1)
}

// PerformanceSnapshot represents a snapshot of performance metrics without mutex
type PerformanceSnapshot struct {
	// Timing metrics
	StartupTime  time.Duration `json:"startup_time"`
	ShutdownTime time.Duration `json:"shutdown_time"`
	Uptime       time.Duration `json:"uptime"`

	// Resource metrics
	MemoryUsage    uint64  `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`
	CPUUsage       float64 `json:"cpu_usage"`

	// Service metrics
	ServiceCount   int `json:"service_count"`
	TransportCount int `json:"transport_count"`

	// Operation counters
	StartCount   int64 `json:"start_count"`
	StopCount    int64 `json:"stop_count"`
	RequestCount int64 `json:"request_count"`
	ErrorCount   int64 `json:"error_count"`

	// Messaging metrics integration
	MessagingMetrics interface{} `json:"messaging_metrics,omitempty"`

	// Performance thresholds
	MaxStartupTime  time.Duration `json:"max_startup_time"`
	MaxShutdownTime time.Duration `json:"max_shutdown_time"`
	MaxMemoryUsage  uint64        `json:"max_memory_usage"`

	// Timestamp
	LastUpdated time.Time `json:"last_updated"`
}

// GetSnapshot returns a thread-safe snapshot of the current metrics
func (m *PerformanceMetrics) GetSnapshot() PerformanceSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return PerformanceSnapshot{
		StartupTime:      m.StartupTime,
		ShutdownTime:     m.ShutdownTime,
		Uptime:           m.Uptime,
		MemoryUsage:      m.MemoryUsage,
		GoroutineCount:   m.GoroutineCount,
		CPUUsage:         m.CPUUsage,
		ServiceCount:     m.ServiceCount,
		TransportCount:   m.TransportCount,
		StartCount:       atomic.LoadInt64(&m.StartCount),
		StopCount:        atomic.LoadInt64(&m.StopCount),
		RequestCount:     atomic.LoadInt64(&m.RequestCount),
		ErrorCount:       atomic.LoadInt64(&m.ErrorCount),
		MessagingMetrics: m.MessagingMetrics,
		MaxStartupTime:   m.MaxStartupTime,
		MaxShutdownTime:  m.MaxShutdownTime,
		MaxMemoryUsage:   m.MaxMemoryUsage,
		LastUpdated:      m.LastUpdated,
	}
}

// CheckThresholds checks if any performance thresholds are exceeded
func (m *PerformanceMetrics) CheckThresholds() []string {
	snapshot := m.GetSnapshot()
	var violations []string

	if snapshot.StartupTime > snapshot.MaxStartupTime {
		violations = append(violations, fmt.Sprintf("startup time %v exceeds threshold %v",
			snapshot.StartupTime, snapshot.MaxStartupTime))
	}

	if snapshot.ShutdownTime > snapshot.MaxShutdownTime {
		violations = append(violations, fmt.Sprintf("shutdown time %v exceeds threshold %v",
			snapshot.ShutdownTime, snapshot.MaxShutdownTime))
	}

	if snapshot.MemoryUsage > snapshot.MaxMemoryUsage {
		violations = append(violations, fmt.Sprintf("memory usage %d bytes exceeds threshold %d bytes",
			snapshot.MemoryUsage, snapshot.MaxMemoryUsage))
	}

	return violations
}

// String returns a string representation of the metrics
func (m *PerformanceMetrics) String() string {
	snapshot := m.GetSnapshot()
	return fmt.Sprintf(
		"StartupTime: %v, ShutdownTime: %v, Uptime: %v, MemoryUsage: %d bytes, Goroutines: %d, Services: %d, Transports: %d, Requests: %d, Errors: %d",
		snapshot.StartupTime,
		snapshot.ShutdownTime,
		snapshot.Uptime,
		snapshot.MemoryUsage,
		snapshot.GoroutineCount,
		snapshot.ServiceCount,
		snapshot.TransportCount,
		snapshot.RequestCount,
		snapshot.ErrorCount,
	)
}

// PerformanceMonitor provides monitoring hooks and profiling capabilities
type PerformanceMonitor struct {
	metrics *PerformanceMetrics
	hooks   []PerformanceHook
	enabled bool
	mu      sync.RWMutex
}

// PerformanceHook defines a callback function for performance events
type PerformanceHook func(event string, metrics *PerformanceMetrics)

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: NewPerformanceMetrics(),
		hooks:   make([]PerformanceHook, 0),
		enabled: true,
	}
}

// AddHook adds a performance monitoring hook
func (pm *PerformanceMonitor) AddHook(hook PerformanceHook) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.hooks = append(pm.hooks, hook)
}

// RemoveAllHooks removes all performance monitoring hooks
func (pm *PerformanceMonitor) RemoveAllHooks() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.hooks = pm.hooks[:0]
}

// Enable enables performance monitoring
func (pm *PerformanceMonitor) Enable() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.enabled = true
}

// Disable disables performance monitoring
func (pm *PerformanceMonitor) Disable() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.enabled = false
}

// IsEnabled returns whether performance monitoring is enabled
func (pm *PerformanceMonitor) IsEnabled() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.enabled
}

// GetMetrics returns the current performance metrics
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	return pm.metrics
}

// RecordEvent records a performance event and triggers hooks
func (pm *PerformanceMonitor) RecordEvent(event string) {
	if !pm.IsEnabled() {
		return
	}

	pm.mu.RLock()
	hooks := make([]PerformanceHook, len(pm.hooks))
	copy(hooks, pm.hooks)
	pm.mu.RUnlock()

	// Trigger all hooks
	for _, hook := range hooks {
		go func(h PerformanceHook) {
			defer func() {
				if r := recover(); r != nil {
					logger.Logger.Error("Performance hook panicked",
						zap.String("event", event),
						zap.Any("panic", r))
				}
			}()
			h(event, pm.metrics)
		}(hook)
	}
}

// ProfileOperation profiles a function execution and records metrics
func (pm *PerformanceMonitor) ProfileOperation(name string, operation func() error) error {
	if !pm.IsEnabled() {
		return operation()
	}

	start := time.Now()
	err := operation()
	duration := time.Since(start)

	if err != nil {
		pm.metrics.IncrementErrorCount()
		logger.Logger.Warn("Profiled operation failed",
			zap.String("operation", name),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		logger.Logger.Debug("Profiled operation completed",
			zap.String("operation", name),
			zap.Duration("duration", duration))
	}

	pm.RecordEvent(fmt.Sprintf("operation_%s", name))
	return err
}

// StartPeriodicCollection starts periodic collection of performance metrics
func (pm *PerformanceMonitor) StartPeriodicCollection(ctx context.Context, interval time.Duration) {
	if !pm.IsEnabled() {
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pm.metrics.RecordMemoryUsage()
				pm.RecordEvent("periodic_collection")
			}
		}
	}()
}

// SetMessagingMetrics sets the messaging metrics for integration with server performance monitoring.
func (pm *PerformanceMonitor) SetMessagingMetrics(messagingMetrics interface{}) {
	pm.metrics.MessagingMetrics = messagingMetrics
}

// GetMessagingMetrics returns the current messaging metrics.
func (pm *PerformanceMonitor) GetMessagingMetrics() interface{} {
	return pm.metrics.MessagingMetrics
}

// RecordMessagingEvent records a messaging-specific performance event.
func (pm *PerformanceMonitor) RecordMessagingEvent(event string, messagingMetrics interface{}) {
	pm.SetMessagingMetrics(messagingMetrics)
	pm.RecordEvent("messaging_" + event)
}

// PerformanceProfiler provides advanced profiling capabilities
type PerformanceProfiler struct {
	monitor *PerformanceMonitor
}

// NewPerformanceProfiler creates a new performance profiler
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		monitor: NewPerformanceMonitor(),
	}
}

// ProfileServerStartup profiles server startup performance
func (p *PerformanceProfiler) ProfileServerStartup(server *BusinessServerImpl) (time.Duration, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := server.Start(ctx)
	duration := time.Since(start)

	p.monitor.metrics.RecordStartupTime(duration)
	p.monitor.metrics.RecordMemoryUsage()

	if err == nil {
		transports := server.GetTransports()
		p.monitor.metrics.RecordServiceMetrics(0, len(transports))
		p.monitor.RecordEvent("server_startup_success")
	} else {
		p.monitor.metrics.IncrementErrorCount()
		p.monitor.RecordEvent("server_startup_failure")
	}

	return duration, err
}

// ProfileServerShutdown profiles server shutdown performance
func (p *PerformanceProfiler) ProfileServerShutdown(server *BusinessServerImpl) (time.Duration, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := server.Stop(ctx)
	duration := time.Since(start)

	p.monitor.metrics.RecordShutdownTime(duration)
	p.monitor.metrics.RecordMemoryUsage()

	if err == nil {
		p.monitor.RecordEvent("server_shutdown_success")
	} else {
		p.monitor.metrics.IncrementErrorCount()
		p.monitor.RecordEvent("server_shutdown_failure")
	}

	return duration, err
}

// GetMonitor returns the underlying performance monitor
func (p *PerformanceProfiler) GetMonitor() *PerformanceMonitor {
	return p.monitor
}

// GetMetrics returns the current performance metrics
func (p *PerformanceProfiler) GetMetrics() *PerformanceMetrics {
	return p.monitor.metrics
}

// Default performance hooks

// PerformanceLoggingHook creates a hook that logs performance events
func PerformanceLoggingHook(event string, metrics *PerformanceMetrics) {
	logger.Logger.Info("Performance event",
		zap.String("event", event),
		zap.String("metrics", metrics.String()))
}

// PerformanceThresholdViolationHook creates a hook that logs threshold violations
func PerformanceThresholdViolationHook(event string, metrics *PerformanceMetrics) {
	violations := metrics.CheckThresholds()
	if len(violations) > 0 {
		logger.Logger.Warn("Performance threshold violations detected",
			zap.String("event", event),
			zap.Strings("violations", violations))
	}
}

// PerformanceMetricsCollectionHook creates a hook that triggers metrics collection
func PerformanceMetricsCollectionHook(event string, metrics *PerformanceMetrics) {
	metrics.RecordMemoryUsage()
}

// MessagingPerformanceIntegrationHook creates a hook that integrates messaging metrics
// with server performance monitoring
func MessagingPerformanceIntegrationHook(event string, metrics *PerformanceMetrics) {
	// Check if this is a messaging event
	if len(event) > 10 && event[:10] == "messaging_" {
		// Update memory usage when messaging events occur
		metrics.RecordMemoryUsage()

		// If messaging metrics are available, we could perform additional integration
		if metrics.MessagingMetrics != nil {
			// In a full implementation, you might extract specific metrics
			// from the messaging metrics and integrate them with server metrics
		}
	}
}

// MessagingThresholdViolationHook creates a hook that monitors messaging-specific threshold violations
func MessagingThresholdViolationHook(event string, metrics *PerformanceMetrics) {
	// This hook can be extended to check messaging-specific thresholds
	// and trigger alerts when messaging performance degrades
	if len(event) > 10 && event[:10] == "messaging_" {
		// Check for messaging-specific threshold violations
		// This would integrate with the MessagingPerformanceMonitor
		// to check messaging-specific thresholds
	}
}

// Backward compatibility aliases
type ServerPerformanceMetrics = PerformanceMetrics

// NewServerPerformanceMetrics creates a new performance metrics instance (backward compatibility)
func NewServerPerformanceMetrics() *ServerPerformanceMetrics {
	return NewPerformanceMetrics()
}
