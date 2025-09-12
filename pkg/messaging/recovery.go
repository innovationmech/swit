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

package messaging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ResourceMetrics represents metrics for resource usage and cleanup
type ResourceMetrics struct {
	TotalResources      int64     `json:"total_resources"`
	ActiveResources     int64     `json:"active_resources"`
	CleanupCount        int64     `json:"cleanup_count"`
	RecoveryCount       int64     `json:"recovery_count"`
	LeakCount           int64     `json:"leak_count"`
	LastCleanupTime     time.Time `json:"last_cleanup_time"`
	LastRecoveryTime    time.Time `json:"last_recovery_time"`
	LastLeakDetection   time.Time `json:"last_leak_detection"`
	AverageCleanupTime  float64   `json:"average_cleanup_time"`
	AverageRecoveryTime float64   `json:"average_recovery_time"`
	MemoryUsed          int64     `json:"memory_used"`
}

// RecoveryManager orchestrates resource cleanup and error recovery for messaging components.
// It provides automatic resource cleanup, error recovery with configurable retry policies,
// resource leak detection, and failure isolation mechanisms.
type RecoveryManager interface {
	// Initialize sets up the recovery manager with the provided configuration
	Initialize(ctx context.Context, config *RecoveryConfig) error

	// RegisterResource registers a resource for automatic cleanup
	RegisterResource(name string, resource RecoverableResource) error

	// UnregisterResource removes a resource from recovery management
	UnregisterResource(name string) error

	// RegisterRecoveryHandler registers a handler for specific error types
	RegisterRecoveryHandler(errorType MessagingErrorType, handler RecoveryHandler) error

	// HandleError processes an error with appropriate recovery actions
	HandleError(ctx context.Context, err error, context *ErrorContext) (*RecoveryResult, error)

	// PerformCleanup performs immediate cleanup of all registered resources
	PerformCleanup(ctx context.Context) (*CleanupResult, error)

	// DetectLeaks scans for resource leaks and returns detection results
	DetectLeaks(ctx context.Context) (*LeakDetectionResult, error)

	// GetRecoveryMetrics returns current recovery metrics
	GetRecoveryMetrics() *RecoveryMetrics

	// Close shuts down the recovery manager and cleans up all resources
	Close(ctx context.Context) error
}

// RecoverableResource defines the interface for resources that can be automatically recovered
type RecoverableResource interface {
	// GetResourceName returns the unique name of this resource
	GetResourceName() string

	// GetResourceType returns the type of resource (connection, handler, etc.)
	GetResourceType() ResourceType

	// IsHealthy checks if the resource is in a healthy state
	IsHealthy(ctx context.Context) bool

	// Recover attempts to recover the resource from an unhealthy state
	Recover(ctx context.Context) error

	// Cleanup releases all resources held by this resource
	Cleanup(ctx context.Context) error

	// GetLastActivity returns when the resource was last active
	GetLastActivity() time.Time

	// GetResourceMetrics returns resource-specific metrics
	GetResourceMetrics() *ResourceMetrics
}

// RecoveryHandler defines the interface for handling specific error types
type RecoveryHandler interface {
	// CanHandle determines if this handler can process the given error
	CanHandle(err error) bool

	// HandleError processes the error and returns recovery actions
	HandleError(ctx context.Context, err error, context *ErrorContext) (*RecoveryAction, error)

	// GetHandlerName returns the name of this recovery handler
	GetHandlerName() string
}

// ResourceType categorizes different types of recoverable resources
type ResourceType string

const (
	ResourceTypeConnection  ResourceType = "CONNECTION"
	ResourceTypeSubscriber  ResourceType = "SUBSCRIBER"
	ResourceTypePublisher   ResourceType = "PUBLISHER"
	ResourceTypeHandler     ResourceType = "HANDLER"
	ResourceTypeTransaction ResourceType = "TRANSACTION"
	ResourceTypeMemory      ResourceType = "MEMORY"
	ResourceTypeFileHandle  ResourceType = "FILE_HANDLE"
)

// RecoveryConfig contains configuration for the recovery manager
type RecoveryConfig struct {
	// Enabled controls whether recovery mechanisms are active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// AutoCleanupInterval specifies how often to perform automatic cleanup
	AutoCleanupInterval time.Duration `json:"auto_cleanup_interval" yaml:"auto_cleanup_interval"`

	// LeakDetectionInterval specifies how often to scan for resource leaks
	LeakDetectionInterval time.Duration `json:"leak_detection_interval" yaml:"leak_detection_interval"`

	// ResourceTimeout specifies how long a resource can be idle before being considered a leak
	ResourceTimeout time.Duration `json:"resource_timeout" yaml:"resource_timeout"`

	// MaxRecoveryAttempts specifies maximum recovery attempts per resource
	MaxRecoveryAttempts int `json:"max_recovery_attempts" yaml:"max_recovery_attempts"`

	// CircuitBreakerThreshold specifies consecutive failures before circuit breaker opens
	CircuitBreakerThreshold int `json:"circuit_breaker_threshold" yaml:"circuit_breaker_threshold"`

	// DeadLetterQueueEnabled enables dead letter queue for unrecoverable errors
	DeadLetterQueueEnabled bool `json:"dead_letter_queue_enabled" yaml:"dead_letter_queue_enabled"`

	// RecoveryLogEnabled enables detailed recovery logging
	RecoveryLogEnabled bool `json:"recovery_log_enabled" yaml:"recovery_log_enabled"`
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Enabled:                 true,
		AutoCleanupInterval:     5 * time.Minute,
		LeakDetectionInterval:   10 * time.Minute,
		ResourceTimeout:         30 * time.Minute,
		MaxRecoveryAttempts:     3,
		CircuitBreakerThreshold: 5,
		DeadLetterQueueEnabled:  true,
		RecoveryLogEnabled:      true,
	}
}

// RecoveryResult contains the result of a recovery operation
type RecoveryResult struct {
	// Success indicates whether recovery was successful
	Success bool `json:"success"`

	// ActionsTaken lists the recovery actions that were performed
	ActionsTaken []string `json:"actions_taken"`

	// ResourcesRecovered counts how many resources were recovered
	ResourcesRecovered int `json:"resources_recovered"`

	// Errors contains any errors that occurred during recovery
	Errors []error `json:"errors,omitempty"`

	// Duration indicates how long recovery took
	Duration time.Duration `json:"duration"`
}

// CleanupResult contains the result of a cleanup operation
type CleanupResult struct {
	// ResourcesCleaned counts how many resources were cleaned up
	ResourcesCleaned int `json:"resources_cleaned"`

	// MemoryReleased indicates how much memory was released (in bytes)
	MemoryReleased int64 `json:"memory_released"`

	// ConnectionsClosed counts how many connections were closed
	ConnectionsClosed int `json:"connections_closed"`

	// Errors contains any errors that occurred during cleanup
	Errors []error `json:"errors,omitempty"`

	// Duration indicates how long cleanup took
	Duration time.Duration `json:"duration"`
}

// LeakDetectionResult contains the result of a leak detection scan
type LeakDetectionResult struct {
	// LeaksDetected indicates whether any leaks were found
	LeaksDetected bool `json:"leaks_detected"`

	// ResourceLeaks details the specific resource leaks found
	ResourceLeaks []*ResourceLeak `json:"resource_leaks,omitempty"`

	// TotalEstimatedMemoryWaste estimates total memory wasted by leaks
	TotalEstimatedMemoryWaste int64 `json:"total_estimated_memory_waste"`

	// ScanDuration indicates how long the leak detection scan took
	ScanDuration time.Duration `json:"scan_duration"`
}

// ResourceLeak represents a detected resource leak
type ResourceLeak struct {
	// ResourceName is the name of the leaking resource
	ResourceName string `json:"resource_name"`

	// ResourceType is the type of the leaking resource
	ResourceType ResourceType `json:"resource_type"`

	// IdleDuration indicates how long the resource has been idle
	IdleDuration time.Duration `json:"idle_duration"`

	// EstimatedMemoryWaste estimates memory wasted by this leak
	EstimatedMemoryWaste int64 `json:"estimated_memory_waste"`

	// Severity indicates the severity of this leak
	LeakSeverity LeakSeverity `json:"leak_severity"`

	// LastActivity indicates when the resource was last active
	LastActivity time.Time `json:"last_activity"`
}

// LeakSeverity indicates the severity of a resource leak
type LeakSeverity string

const (
	LeakSeverityLow      LeakSeverity = "LOW"
	LeakSeverityMedium   LeakSeverity = "MEDIUM"
	LeakSeverityHigh     LeakSeverity = "HIGH"
	LeakSeverityCritical LeakSeverity = "CRITICAL"
)

// RecoveryAction represents an action taken during error recovery
type RecoveryAction struct {
	// ActionType is the type of recovery action performed
	ActionType RecoveryActionType `json:"action_type"`

	// ResourceName is the name of the resource this action applies to
	ResourceName string `json:"resource_name"`

	// Description describes what the action does
	Description string `json:"description"`

	// Success indicates whether the action was successful
	Success bool `json:"success"`

	// Error contains any error that occurred during the action
	Error error `json:"error,omitempty"`

	// Metadata contains additional action-specific data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RecoveryActionType categorizes different types of recovery actions
type RecoveryActionType string

const (
	RecoveryActionRetry       RecoveryActionType = "RETRY"
	RecoveryActionReconnect   RecoveryActionType = "RECONNECT"
	RecoveryActionRestart     RecoveryActionType = "RESTART"
	RecoveryActionIsolate     RecoveryActionType = "ISOLATE"
	RecoveryActionCleanup     RecoveryActionType = "CLEANUP"
	RecoveryActionDeadLetter  RecoveryActionType = "DEAD_LETTER"
	RecoveryActionCircuitOpen RecoveryActionType = "CIRCUIT_OPEN"
)

// RecoveryMetrics tracks recovery manager performance statistics
type RecoveryMetrics struct {
	// TotalRecoveryAttempts is the total number of recovery attempts
	TotalRecoveryAttempts uint64 `json:"total_recovery_attempts"`

	// SuccessfulRecoveries is the number of successful recoveries
	SuccessfulRecoveries uint64 `json:"successful_recoveries"`

	// FailedRecoveries is the number of failed recoveries
	FailedRecoveries uint64 `json:"failed_recoveries"`

	// TotalCleanupOperations is the total number of cleanup operations
	TotalCleanupOperations uint64 `json:"total_cleanup_operations"`

	// ResourcesRecovered is the total number of resources recovered
	ResourcesRecovered uint64 `json:"resources_recovered"`

	// LeaksDetected is the total number of leaks detected
	LeaksDetected uint64 `json:"leaks_detected"`

	// AverageRecoveryTime is the average time taken for recovery operations
	AverageRecoveryTime time.Duration `json:"average_recovery_time"`

	// CircuitBreakerTrips is the number of times circuit breakers have been triggered
	CircuitBreakerTrips uint64 `json:"circuit_breaker_trips"`

	// LastRecoveryTime is when the last recovery operation was performed
	LastRecoveryTime time.Time `json:"last_recovery_time"`
}

// recoveryManagerImpl implements the RecoveryManager interface
type recoveryManagerImpl struct {
	config *RecoveryConfig

	// Resource management
	resources   map[string]RecoverableResource
	resourceMux sync.RWMutex

	// Recovery handlers
	handlers   map[MessagingErrorType]RecoveryHandler
	handlerMux sync.RWMutex

	// Circuit breakers
	circuitBreakers map[string]*CircuitBreaker
	breakerMux      sync.RWMutex

	// Metrics
	metrics    *RecoveryMetrics
	metricsMux sync.RWMutex

	// Background processes
	cleanupTicker    *time.Ticker
	leakDetectTicker *time.Ticker
	processMux       sync.RWMutex

	// Lifecycle
	started int32
	closed  int32
	closeCh chan struct{}
}

// NewRecoveryManager creates a new recovery manager instance
func NewRecoveryManager() RecoveryManager {
	return &recoveryManagerImpl{
		resources:       make(map[string]RecoverableResource),
		handlers:        make(map[MessagingErrorType]RecoveryHandler),
		circuitBreakers: make(map[string]*CircuitBreaker),
		metrics:         &RecoveryMetrics{},
		closeCh:         make(chan struct{}),
	}
}

// Initialize implements RecoveryManager interface
func (rm *recoveryManagerImpl) Initialize(ctx context.Context, config *RecoveryConfig) error {
	if !atomic.CompareAndSwapInt32(&rm.started, 0, 1) {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidState).
			Message("recovery manager already initialized").
			Build()
	}

	if config == nil {
		config = DefaultRecoveryConfig()
	}

	if err := config.validate(); err != nil {
		return fmt.Errorf("invalid recovery configuration: %w", err)
	}

	rm.config = config

	// Start background processes if enabled
	if config.Enabled {
		if config.AutoCleanupInterval > 0 {
			rm.startAutoCleanupLoop()
		}
		if config.LeakDetectionInterval > 0 {
			rm.startLeakDetectionLoop()
		}
	}

	logger.Logger.Info("Recovery manager initialized successfully",
		zap.Bool("enabled", config.Enabled),
		zap.Duration("cleanup_interval", config.AutoCleanupInterval),
		zap.Duration("leak_detection_interval", config.LeakDetectionInterval))

	return nil
}

// RegisterResource implements RecoveryManager interface
func (rm *recoveryManagerImpl) RegisterResource(name string, resource RecoverableResource) error {
	if name == "" {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Message("resource name cannot be empty").
			Build()
	}

	if resource == nil {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Message("resource cannot be nil").
			Build()
	}

	rm.resourceMux.Lock()
	defer rm.resourceMux.Unlock()

	if _, exists := rm.resources[name]; exists {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Messagef("resource '%s' already registered", name).
			Build()
	}

	// Create circuit breaker for this resource
	breaker, err := NewCircuitBreaker(&CircuitBreakerConfig{
		Name:             fmt.Sprintf("recovery-%s", name),
		FailureThreshold: uint64(rm.config.CircuitBreakerThreshold),
		SuccessThreshold: 5,
		Timeout:          5 * time.Minute,
		Interval:         time.Minute,
		MaxRequests:      10,
	})
	if err != nil {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Messagef("failed to create circuit breaker for resource '%s': %v", name, err).
			Build()
	}

	rm.circuitBreakers[name] = breaker
	rm.resources[name] = resource

	if rm.config.RecoveryLogEnabled {
		logger.Logger.Info("Resource registered for recovery",
			zap.String("resource_name", name),
			zap.String("resource_type", string(resource.GetResourceType())))
	}

	return nil
}

// UnregisterResource implements RecoveryManager interface
func (rm *recoveryManagerImpl) UnregisterResource(name string) error {
	rm.resourceMux.Lock()
	defer rm.resourceMux.Unlock()

	if _, exists := rm.resources[name]; !exists {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Messagef("resource '%s' not found", name).
			Build()
	}

	delete(rm.resources, name)
	delete(rm.circuitBreakers, name)

	if rm.config.RecoveryLogEnabled {
		logger.Logger.Info("Resource unregistered from recovery",
			zap.String("resource_name", name))
	}

	return nil
}

// RegisterRecoveryHandler implements RecoveryManager interface
func (rm *recoveryManagerImpl) RegisterRecoveryHandler(errorType MessagingErrorType, handler RecoveryHandler) error {
	if handler == nil {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Message("recovery handler cannot be nil").
			Build()
	}

	rm.handlerMux.Lock()
	defer rm.handlerMux.Unlock()

	rm.handlers[errorType] = handler

	if rm.config.RecoveryLogEnabled {
		logger.Logger.Info("Recovery handler registered",
			zap.String("error_type", string(errorType)),
			zap.String("handler_name", handler.GetHandlerName()))
	}

	return nil
}

// HandleError implements RecoveryManager interface
func (rm *recoveryManagerImpl) HandleError(ctx context.Context, err error, errorCtx *ErrorContext) (*RecoveryResult, error) {
	startTime := time.Now()
	result := &RecoveryResult{
		ActionsTaken: make([]string, 0),
	}

	if atomic.LoadInt32(&rm.closed) == 1 {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("recovery manager is closed"))
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Classify the error to determine appropriate recovery strategy
	errorType := rm.classifyError(err)
	if errorType == "" {
		// Unknown error type, use default handling
		errorType = ErrorTypeInternal
	}

	// Find appropriate recovery handler
	handler := rm.getRecoveryHandler(errorType)
	if handler == nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("no recovery handler for error type: %s", errorType))
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Execute recovery action
	action, err := handler.HandleError(ctx, err, errorCtx)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("recovery handler failed: %w", err))
	} else {
		result.ActionsTaken = append(result.ActionsTaken, string(action.ActionType))
		result.Success = action.Success
		if !action.Success {
			result.Errors = append(result.Errors, action.Error)
		}
		result.ResourcesRecovered = rm.executeRecoveryAction(ctx, action)
	}

	// Update metrics
	rm.updateRecoveryMetrics(result, time.Since(startTime))

	if rm.config.RecoveryLogEnabled {
		logger.Logger.Info("Error recovery completed",
			zap.String("error_type", string(errorType)),
			zap.Bool("success", result.Success),
			zap.Strings("actions", result.ActionsTaken),
			zap.Duration("duration", result.Duration))
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// PerformCleanup implements RecoveryManager interface
func (rm *recoveryManagerImpl) PerformCleanup(ctx context.Context) (*CleanupResult, error) {
	startTime := time.Now()
	result := &CleanupResult{}

	rm.resourceMux.RLock()
	resources := make([]RecoverableResource, 0, len(rm.resources))
	for _, resource := range rm.resources {
		resources = append(resources, resource)
	}
	rm.resourceMux.RUnlock()

	var cleanupErrors []error

	for _, resource := range resources {
		select {
		case <-ctx.Done():
			break
		default:
			if err := resource.Cleanup(ctx); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to cleanup resource '%s': %w", resource.GetResourceName(), err))
				if rm.config.RecoveryLogEnabled {
					logger.Logger.Error("Resource cleanup failed",
						zap.String("resource_name", resource.GetResourceName()),
						zap.Error(err))
				}
			} else {
				result.ResourcesCleaned++

				// Track metrics
				metrics := resource.GetResourceMetrics()
				if metrics != nil {
					result.MemoryReleased += metrics.MemoryUsed
					if resource.GetResourceType() == ResourceTypeConnection {
						result.ConnectionsClosed++
					}
				}

				if rm.config.RecoveryLogEnabled {
					logger.Logger.Debug("Resource cleanup completed",
						zap.String("resource_name", resource.GetResourceName()),
						zap.String("resource_type", string(resource.GetResourceType())))
				}
			}
		}
	}

	result.Errors = cleanupErrors
	result.Duration = time.Since(startTime)

	// Update metrics
	rm.updateCleanupMetrics(result)

	if rm.config.RecoveryLogEnabled {
		logger.Logger.Info("Cleanup operation completed",
			zap.Int("resources_cleaned", result.ResourcesCleaned),
			zap.Int64("memory_released", result.MemoryReleased),
			zap.Int("connections_closed", result.ConnectionsClosed),
			zap.Duration("duration", result.Duration))
	}

	return result, nil
}

// DetectLeaks implements RecoveryManager interface
func (rm *recoveryManagerImpl) DetectLeaks(ctx context.Context) (*LeakDetectionResult, error) {
	startTime := time.Now()
	result := &LeakDetectionResult{}

	rm.resourceMux.RLock()
	resources := make([]RecoverableResource, 0, len(rm.resources))
	for _, resource := range rm.resources {
		resources = append(resources, resource)
	}
	rm.resourceMux.RUnlock()

	now := time.Now()
	var leaks []*ResourceLeak

	for _, resource := range resources {
		select {
		case <-ctx.Done():
			break
		default:
			// Check if resource is healthy
			healthy := resource.IsHealthy(ctx)
			lastActivity := resource.GetLastActivity()
			idleDuration := now.Sub(lastActivity)

			// Consider it a leak if it's unhealthy or idle for too long
			if !healthy || idleDuration > rm.config.ResourceTimeout {
				metrics := resource.GetResourceMetrics()
				memoryWaste := int64(0)
				if metrics != nil {
					memoryWaste = metrics.MemoryUsed
				}

				severity := rm.calculateLeakSeverity(idleDuration, memoryWaste)

				leak := &ResourceLeak{
					ResourceName:         resource.GetResourceName(),
					ResourceType:         resource.GetResourceType(),
					IdleDuration:         idleDuration,
					EstimatedMemoryWaste: memoryWaste,
					LeakSeverity:         severity,
					LastActivity:         lastActivity,
				}

				leaks = append(leaks, leak)
				result.TotalEstimatedMemoryWaste += memoryWaste
			}
		}
	}

	result.LeaksDetected = len(leaks) > 0
	result.ResourceLeaks = leaks
	result.ScanDuration = time.Since(startTime)

	// Update metrics
	rm.updateLeakDetectionMetrics(result)

	if rm.config.RecoveryLogEnabled {
		logger.Logger.Info("Leak detection completed",
			zap.Bool("leaks_detected", result.LeaksDetected),
			zap.Int("leak_count", len(leaks)),
			zap.Int64("memory_waste", result.TotalEstimatedMemoryWaste),
			zap.Duration("scan_duration", result.ScanDuration))
	}

	return result, nil
}

// GetRecoveryMetrics implements RecoveryManager interface
func (rm *recoveryManagerImpl) GetRecoveryMetrics() *RecoveryMetrics {
	rm.metricsMux.RLock()
	defer rm.metricsMux.RUnlock()

	// Return a copy to avoid external modification
	metricsCopy := *rm.metrics
	return &metricsCopy
}

// Close implements RecoveryManager interface
func (rm *recoveryManagerImpl) Close(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&rm.closed, 0, 1) {
		return nil // Already closed
	}

	// Stop background processes
	close(rm.closeCh)

	rm.processMux.Lock()
	if rm.cleanupTicker != nil {
		rm.cleanupTicker.Stop()
		rm.cleanupTicker = nil
	}
	if rm.leakDetectTicker != nil {
		rm.leakDetectTicker.Stop()
		rm.leakDetectTicker = nil
	}
	rm.processMux.Unlock()

	// Perform final cleanup
	if rm.config.Enabled {
		_, err := rm.PerformCleanup(ctx)
		if err != nil {
			logger.Logger.Error("Final cleanup failed during recovery manager shutdown", zap.Error(err))
		}
	}

	logger.Logger.Info("Recovery manager closed successfully")
	return nil
}

// Helper methods

func (rm *recoveryManagerImpl) classifyError(err error) MessagingErrorType {
	// Check if it's a structured messaging error
	var msgErr *BaseMessagingError
	if errors.As(err, &msgErr) {
		return msgErr.Type
	}

	// Basic classification for unstructured errors
	if strings.Contains(err.Error(), "connection") {
		return ErrorTypeConnection
	}
	if strings.Contains(err.Error(), "timeout") {
		return ErrorTypeResource
	}

	return ErrorTypeInternal
}

func (rm *recoveryManagerImpl) getRecoveryHandler(errorType MessagingErrorType) RecoveryHandler {
	rm.handlerMux.RLock()
	defer rm.handlerMux.RUnlock()

	return rm.handlers[errorType]
}

func (rm *recoveryManagerImpl) executeRecoveryAction(ctx context.Context, action *RecoveryAction) int {
	if action == nil {
		return 0
	}

	// Execute action based on type
	switch action.ActionType {
	case RecoveryActionCleanup:
		rm.resourceMux.RLock()
		if resource, exists := rm.resources[action.ResourceName]; exists {
			_ = resource.Cleanup(ctx)
			rm.resourceMux.RUnlock()
			return 1
		}
		rm.resourceMux.RUnlock()

	case RecoveryActionReconnect, RecoveryActionRestart:
		rm.resourceMux.RLock()
		if resource, exists := rm.resources[action.ResourceName]; exists {
			_ = resource.Recover(ctx)
			rm.resourceMux.RUnlock()
			return 1
		}
		rm.resourceMux.RUnlock()
	}

	return 0
}

func (rm *recoveryManagerImpl) calculateLeakSeverity(idleDuration time.Duration, memoryWaste int64) LeakSeverity {
	if memoryWaste > 100*1024*1024 { // > 100MB
		return LeakSeverityCritical
	}
	if memoryWaste > 10*1024*1024 || idleDuration > time.Hour { // > 10MB or > 1 hour
		return LeakSeverityHigh
	}
	if memoryWaste > 1024*1024 || idleDuration > 30*time.Minute { // > 1MB or > 30 minutes
		return LeakSeverityMedium
	}
	return LeakSeverityLow
}

func (rm *recoveryManagerImpl) updateRecoveryMetrics(result *RecoveryResult, duration time.Duration) {
	rm.metricsMux.Lock()
	defer rm.metricsMux.Unlock()

	rm.metrics.TotalRecoveryAttempts++
	if result.Success {
		rm.metrics.SuccessfulRecoveries++
	} else {
		rm.metrics.FailedRecoveries++
	}

	// Update average recovery time
	if rm.metrics.AverageRecoveryTime == 0 {
		rm.metrics.AverageRecoveryTime = duration
	} else {
		rm.metrics.AverageRecoveryTime = (rm.metrics.AverageRecoveryTime + duration) / 2
	}

	rm.metrics.ResourcesRecovered += uint64(result.ResourcesRecovered)
	rm.metrics.LastRecoveryTime = time.Now()
}

func (rm *recoveryManagerImpl) updateCleanupMetrics(result *CleanupResult) {
	rm.metricsMux.Lock()
	defer rm.metricsMux.Unlock()

	rm.metrics.TotalCleanupOperations++
	rm.metrics.ResourcesRecovered += uint64(result.ResourcesCleaned)
}

func (rm *recoveryManagerImpl) updateLeakDetectionMetrics(result *LeakDetectionResult) {
	rm.metricsMux.Lock()
	defer rm.metricsMux.Unlock()

	rm.metrics.LeaksDetected += uint64(len(result.ResourceLeaks))
}

func (rm *recoveryManagerImpl) startAutoCleanupLoop() {
	ticker := time.NewTicker(rm.config.AutoCleanupInterval)

	rm.processMux.Lock()
	rm.cleanupTicker = ticker
	rm.processMux.Unlock()

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				_, err := rm.PerformCleanup(ctx)
				cancel()
				if err != nil {
					logger.Logger.Error("Auto cleanup failed", zap.Error(err))
				}
			case <-rm.closeCh:
				return
			}
		}
	}()
}

func (rm *recoveryManagerImpl) startLeakDetectionLoop() {
	ticker := time.NewTicker(rm.config.LeakDetectionInterval)

	rm.processMux.Lock()
	rm.leakDetectTicker = ticker
	rm.processMux.Unlock()

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				result, err := rm.DetectLeaks(ctx)
				cancel()
				if err != nil {
					logger.Logger.Error("Leak detection failed", zap.Error(err))
				} else if result.LeaksDetected {
					logger.Logger.Warn("Resource leaks detected",
						zap.Int("leak_count", len(result.ResourceLeaks)),
						zap.Int64("memory_waste", result.TotalEstimatedMemoryWaste))
				}
			case <-rm.closeCh:
				return
			}
		}
	}()
}

// Validation methods

func (c *RecoveryConfig) validate() error {
	if c.AutoCleanupInterval < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("auto_cleanup_interval cannot be negative").
			Build()
	}

	if c.LeakDetectionInterval < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("leak_detection_interval cannot be negative").
			Build()
	}

	if c.ResourceTimeout < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("resource_timeout cannot be negative").
			Build()
	}

	if c.MaxRecoveryAttempts < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("max_recovery_attempts cannot be negative").
			Build()
	}

	if c.CircuitBreakerThreshold < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("circuit_breaker_threshold cannot be negative").
			Build()
	}

	return nil
}
