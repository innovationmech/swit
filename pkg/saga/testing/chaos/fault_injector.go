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

// Package chaos provides chaos testing utilities for the Saga system.
// It includes fault injection mechanisms to test system reliability and fault tolerance.
package chaos

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

var (
	// ErrNetworkPartition simulates a network partition error.
	ErrNetworkPartition = errors.New("chaos: network partition")

	// ErrServiceCrash simulates a service crash error.
	ErrServiceCrash = errors.New("chaos: service crash")

	// ErrMessageLoss simulates a message loss error.
	ErrMessageLoss = errors.New("chaos: message loss")

	// ErrDatabaseFailure simulates a database failure error.
	ErrDatabaseFailure = errors.New("chaos: database failure")

	// ErrTimeout simulates a timeout error.
	ErrTimeout = errors.New("chaos: operation timeout")

	// ErrPartialFailure simulates a partial failure error.
	ErrPartialFailure = errors.New("chaos: partial failure")
)

// FaultType represents the type of fault to inject.
type FaultType string

const (
	// FaultTypeNetworkPartition causes network communication failures.
	FaultTypeNetworkPartition FaultType = "network_partition"

	// FaultTypeServiceCrash causes service crash simulations.
	FaultTypeServiceCrash FaultType = "service_crash"

	// FaultTypeMessageLoss causes message loss.
	FaultTypeMessageLoss FaultType = "message_loss"

	// FaultTypeTimeout causes timeout errors.
	FaultTypeTimeout FaultType = "timeout"

	// FaultTypeDatabaseFailure causes database operation failures.
	FaultTypeDatabaseFailure FaultType = "database_failure"

	// FaultTypePartialFailure causes partial failures in operations.
	FaultTypePartialFailure FaultType = "partial_failure"

	// FaultTypeDelay causes artificial delays.
	FaultTypeDelay FaultType = "delay"

	// FaultTypeRandomError causes random errors.
	FaultTypeRandomError FaultType = "random_error"
)

// FaultConfig defines the configuration for fault injection.
type FaultConfig struct {
	// Type specifies the type of fault to inject.
	Type FaultType

	// Probability is the chance (0.0-1.0) that the fault will be injected.
	Probability float64

	// Duration specifies how long the fault should be active.
	Duration time.Duration

	// Delay specifies the artificial delay to introduce (for FaultTypeDelay).
	Delay time.Duration

	// TargetStep specifies which step(s) should be affected (empty means all steps).
	TargetStep string

	// ErrorMessage is the custom error message to use.
	ErrorMessage string

	// Metadata contains additional fault-specific configuration.
	Metadata map[string]interface{}
}

// FaultInjector provides fault injection capabilities for chaos testing.
type FaultInjector struct {
	mu       sync.RWMutex
	faults   map[string]*FaultConfig
	enabled  bool
	rng      *rand.Rand
	stats    *InjectionStats
	logFunc  func(string, ...interface{})
}

// InjectionStats tracks fault injection statistics.
type InjectionStats struct {
	mu                    sync.Mutex
	TotalInjections       int64
	SuccessfulInjections  int64
	FailedInjections      int64
	InjectionsByType      map[FaultType]int64
	LastInjectionTime     time.Time
}

// NewFaultInjector creates a new fault injector instance.
func NewFaultInjector() *FaultInjector {
	return &FaultInjector{
		faults:  make(map[string]*FaultConfig),
		enabled: true,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stats: &InjectionStats{
			InjectionsByType: make(map[FaultType]int64),
		},
		logFunc: func(format string, args ...interface{}) {
			// Default: no-op logging
		},
	}
}

// SetLogFunc sets the logging function for fault injection events.
func (fi *FaultInjector) SetLogFunc(logFunc func(string, ...interface{})) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.logFunc = logFunc
}

// AddFault adds a fault configuration with the given ID.
func (fi *FaultInjector) AddFault(id string, config *FaultConfig) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.faults[id] = config
}

// RemoveFault removes a fault configuration by ID.
func (fi *FaultInjector) RemoveFault(id string) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	delete(fi.faults, id)
}

// ClearFaults removes all fault configurations.
func (fi *FaultInjector) ClearFaults() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.faults = make(map[string]*FaultConfig)
}

// Enable enables fault injection.
func (fi *FaultInjector) Enable() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.enabled = true
}

// Disable disables fault injection.
func (fi *FaultInjector) Disable() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.enabled = false
}

// IsEnabled returns whether fault injection is enabled.
func (fi *FaultInjector) IsEnabled() bool {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	return fi.enabled
}

// GetStats returns the current injection statistics.
func (fi *FaultInjector) GetStats() *InjectionStats {
	fi.stats.mu.Lock()
	defer fi.stats.mu.Unlock()
	
	// Return a copy
	stats := &InjectionStats{
		TotalInjections:      fi.stats.TotalInjections,
		SuccessfulInjections: fi.stats.SuccessfulInjections,
		FailedInjections:     fi.stats.FailedInjections,
		InjectionsByType:     make(map[FaultType]int64),
		LastInjectionTime:    fi.stats.LastInjectionTime,
	}
	for k, v := range fi.stats.InjectionsByType {
		stats.InjectionsByType[k] = v
	}
	return stats
}

// ResetStats resets the injection statistics.
func (fi *FaultInjector) ResetStats() {
	fi.stats.mu.Lock()
	defer fi.stats.mu.Unlock()
	fi.stats.TotalInjections = 0
	fi.stats.SuccessfulInjections = 0
	fi.stats.FailedInjections = 0
	fi.stats.InjectionsByType = make(map[FaultType]int64)
}

// InjectIntoStep injects faults when executing a Saga step.
func (fi *FaultInjector) InjectIntoStep(ctx context.Context, stepName string, execute func(context.Context) (interface{}, error)) (interface{}, error) {
	if !fi.IsEnabled() {
		return execute(ctx)
	}

	fi.mu.RLock()
	faultConfigs := fi.getMatchingFaults(stepName)
	fi.mu.RUnlock()

	// Apply all matching faults
	for _, config := range faultConfigs {
		if fi.shouldInjectFault(config) {
			if err := fi.applyFault(ctx, config); err != nil {
				fi.recordInjection(config.Type, true)
				fi.logFunc("Injected fault: %s for step: %s", config.Type, stepName)
				return nil, err
			}
		}
	}

	// Execute the actual operation
	result, err := execute(ctx)
	return result, err
}

// InjectIntoStateStorage injects faults into state storage operations.
func (fi *FaultInjector) InjectIntoStateStorage(ctx context.Context, operation string, execute func(context.Context) error) error {
	if !fi.IsEnabled() {
		return execute(ctx)
	}

	fi.mu.RLock()
	faultConfigs := fi.getMatchingFaults(operation)
	fi.mu.RUnlock()

	// Apply all matching faults
	for _, config := range faultConfigs {
		if fi.shouldInjectFault(config) {
			if err := fi.applyFault(ctx, config); err != nil {
				fi.recordInjection(config.Type, true)
				fi.logFunc("Injected storage fault: %s for operation: %s", config.Type, operation)
				return err
			}
		}
	}

	// Execute the actual operation
	return execute(ctx)
}

// InjectIntoEventPublisher injects faults into event publishing.
func (fi *FaultInjector) InjectIntoEventPublisher(ctx context.Context, eventType saga.SagaEventType, publish func(context.Context) error) error {
	if !fi.IsEnabled() {
		return publish(ctx)
	}

	fi.mu.RLock()
	faultConfigs := fi.getMatchingFaults(string(eventType))
	fi.mu.RUnlock()

	// Apply all matching faults
	for _, config := range faultConfigs {
		if fi.shouldInjectFault(config) {
			if err := fi.applyFault(ctx, config); err != nil {
				fi.recordInjection(config.Type, true)
				fi.logFunc("Injected event publisher fault: %s for event: %s", config.Type, eventType)
				return err
			}
		}
	}

	// Execute the actual operation
	return publish(ctx)
}

// WrapStep wraps a Saga step with fault injection capability.
func (fi *FaultInjector) WrapStep(step saga.SagaStep) saga.SagaStep {
	return &chaosStep{
		wrapped:  step,
		injector: fi,
	}
}

// getMatchingFaults returns all fault configurations that match the target.
func (fi *FaultInjector) getMatchingFaults(target string) []*FaultConfig {
	var configs []*FaultConfig
	for _, config := range fi.faults {
		if config.TargetStep == "" || config.TargetStep == target {
			configs = append(configs, config)
		}
	}
	return configs
}

// shouldInjectFault determines if a fault should be injected based on probability.
func (fi *FaultInjector) shouldInjectFault(config *FaultConfig) bool {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	return fi.rng.Float64() < config.Probability
}

// applyFault applies the specified fault.
func (fi *FaultInjector) applyFault(ctx context.Context, config *FaultConfig) error {
	switch config.Type {
	case FaultTypeNetworkPartition:
		return fi.applyNetworkPartition(config)
	case FaultTypeServiceCrash:
		return fi.applyServiceCrash(config)
	case FaultTypeMessageLoss:
		return fi.applyMessageLoss(config)
	case FaultTypeTimeout:
		return fi.applyTimeout(ctx, config)
	case FaultTypeDatabaseFailure:
		return fi.applyDatabaseFailure(config)
	case FaultTypePartialFailure:
		return fi.applyPartialFailure(config)
	case FaultTypeDelay:
		return fi.applyDelay(ctx, config)
	case FaultTypeRandomError:
		return fi.applyRandomError(config)
	default:
		return nil
	}
}

// applyNetworkPartition simulates a network partition.
func (fi *FaultInjector) applyNetworkPartition(config *FaultConfig) error {
	if config.ErrorMessage != "" {
		return errors.New(config.ErrorMessage)
	}
	return ErrNetworkPartition
}

// applyServiceCrash simulates a service crash.
func (fi *FaultInjector) applyServiceCrash(config *FaultConfig) error {
	if config.ErrorMessage != "" {
		return errors.New(config.ErrorMessage)
	}
	return ErrServiceCrash
}

// applyMessageLoss simulates message loss.
func (fi *FaultInjector) applyMessageLoss(config *FaultConfig) error {
	if config.ErrorMessage != "" {
		return errors.New(config.ErrorMessage)
	}
	return ErrMessageLoss
}

// applyTimeout simulates a timeout by blocking until context is cancelled or duration expires.
func (fi *FaultInjector) applyTimeout(ctx context.Context, config *FaultConfig) error {
	delay := config.Delay
	if delay == 0 {
		delay = 5 * time.Second // Default timeout delay
	}

	select {
	case <-time.After(delay):
		if config.ErrorMessage != "" {
			return errors.New(config.ErrorMessage)
		}
		return ErrTimeout
	case <-ctx.Done():
		return ctx.Err()
	}
}

// applyDatabaseFailure simulates a database failure.
func (fi *FaultInjector) applyDatabaseFailure(config *FaultConfig) error {
	if config.ErrorMessage != "" {
		return errors.New(config.ErrorMessage)
	}
	return ErrDatabaseFailure
}

// applyPartialFailure simulates a partial failure.
func (fi *FaultInjector) applyPartialFailure(config *FaultConfig) error {
	if config.ErrorMessage != "" {
		return errors.New(config.ErrorMessage)
	}
	return ErrPartialFailure
}

// applyDelay introduces an artificial delay.
func (fi *FaultInjector) applyDelay(ctx context.Context, config *FaultConfig) error {
	delay := config.Delay
	if delay == 0 {
		delay = 1 * time.Second // Default delay
	}

	select {
	case <-time.After(delay):
		return nil // Delay applied successfully, continue execution
	case <-ctx.Done():
		return ctx.Err()
	}
}

// applyRandomError generates a random error.
func (fi *FaultInjector) applyRandomError(config *FaultConfig) error {
	errorMessages := []string{
		"random network error",
		"random service unavailable",
		"random internal error",
		"random resource exhausted",
		"random unknown error",
	}

	fi.mu.Lock()
	idx := fi.rng.Intn(len(errorMessages))
	fi.mu.Unlock()

	if config.ErrorMessage != "" {
		return errors.New(config.ErrorMessage)
	}
	return errors.New(errorMessages[idx])
}

// recordInjection records an injection event in statistics.
func (fi *FaultInjector) recordInjection(faultType FaultType, success bool) {
	fi.stats.mu.Lock()
	defer fi.stats.mu.Unlock()

	fi.stats.TotalInjections++
	if success {
		fi.stats.SuccessfulInjections++
	} else {
		fi.stats.FailedInjections++
	}
	fi.stats.InjectionsByType[faultType]++
	fi.stats.LastInjectionTime = time.Now()
}

// chaosStep wraps a Saga step with fault injection.
type chaosStep struct {
	wrapped  saga.SagaStep
	injector *FaultInjector
}

// GetID returns the step ID.
func (s *chaosStep) GetID() string {
	return s.wrapped.GetID()
}

// GetName returns the step name.
func (s *chaosStep) GetName() string {
	return s.wrapped.GetName()
}

// GetDescription returns the step description.
func (s *chaosStep) GetDescription() string {
	return s.wrapped.GetDescription()
}

// Execute executes the step with fault injection.
func (s *chaosStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	return s.injector.InjectIntoStep(ctx, s.GetName(), func(ctx context.Context) (interface{}, error) {
		return s.wrapped.Execute(ctx, data)
	})
}

// Compensate executes compensation with fault injection.
func (s *chaosStep) Compensate(ctx context.Context, data interface{}) error {
	_, err := s.injector.InjectIntoStep(ctx, s.GetName()+"-compensate", func(ctx context.Context) (interface{}, error) {
		return nil, s.wrapped.Compensate(ctx, data)
	})
	return err
}

// GetTimeout returns the step timeout.
func (s *chaosStep) GetTimeout() time.Duration {
	return s.wrapped.GetTimeout()
}

// GetRetryPolicy returns the step retry policy.
func (s *chaosStep) GetRetryPolicy() saga.RetryPolicy {
	return s.wrapped.GetRetryPolicy()
}

// IsRetryable determines if an error is retryable.
func (s *chaosStep) IsRetryable(err error) bool {
	return s.wrapped.IsRetryable(err)
}

// GetMetadata returns the step metadata.
func (s *chaosStep) GetMetadata() map[string]interface{} {
	return s.wrapped.GetMetadata()
}

// ChaosStateStorage wraps a StateStorage with fault injection.
type ChaosStateStorage struct {
	wrapped  saga.StateStorage
	injector *FaultInjector
}

// NewChaosStateStorage creates a new chaos state storage wrapper.
func NewChaosStateStorage(wrapped saga.StateStorage, injector *FaultInjector) *ChaosStateStorage {
	return &ChaosStateStorage{
		wrapped:  wrapped,
		injector: injector,
	}
}

// SaveSaga saves a Saga instance with fault injection.
func (s *ChaosStateStorage) SaveSaga(ctx context.Context, sagaInstance saga.SagaInstance) error {
	return s.injector.InjectIntoStateStorage(ctx, "SaveSaga", func(ctx context.Context) error {
		return s.wrapped.SaveSaga(ctx, sagaInstance)
	})
}

// GetSaga retrieves a Saga instance with fault injection.
func (s *ChaosStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	var result saga.SagaInstance
	var err error
	
	storageErr := s.injector.InjectIntoStateStorage(ctx, "GetSaga", func(ctx context.Context) error {
		result, err = s.wrapped.GetSaga(ctx, sagaID)
		return err
	})
	
	if storageErr != nil {
		return nil, storageErr
	}
	return result, err
}

// UpdateSagaState updates Saga state with fault injection.
func (s *ChaosStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	return s.injector.InjectIntoStateStorage(ctx, "UpdateSagaState", func(ctx context.Context) error {
		return s.wrapped.UpdateSagaState(ctx, sagaID, state, metadata)
	})
}

// DeleteSaga deletes a Saga instance with fault injection.
func (s *ChaosStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	return s.injector.InjectIntoStateStorage(ctx, "DeleteSaga", func(ctx context.Context) error {
		return s.wrapped.DeleteSaga(ctx, sagaID)
	})
}

// GetActiveSagas retrieves active Sagas with fault injection.
func (s *ChaosStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	var result []saga.SagaInstance
	var err error
	
	storageErr := s.injector.InjectIntoStateStorage(ctx, "GetActiveSagas", func(ctx context.Context) error {
		result, err = s.wrapped.GetActiveSagas(ctx, filter)
		return err
	})
	
	if storageErr != nil {
		return nil, storageErr
	}
	return result, err
}

// GetTimeoutSagas retrieves timeout Sagas with fault injection.
func (s *ChaosStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	var result []saga.SagaInstance
	var err error
	
	storageErr := s.injector.InjectIntoStateStorage(ctx, "GetTimeoutSagas", func(ctx context.Context) error {
		result, err = s.wrapped.GetTimeoutSagas(ctx, before)
		return err
	})
	
	if storageErr != nil {
		return nil, storageErr
	}
	return result, err
}

// SaveStepState saves step state with fault injection.
func (s *ChaosStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	return s.injector.InjectIntoStateStorage(ctx, "SaveStepState", func(ctx context.Context) error {
		return s.wrapped.SaveStepState(ctx, sagaID, step)
	})
}

// GetStepStates retrieves step states with fault injection.
func (s *ChaosStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	var result []*saga.StepState
	var err error
	
	storageErr := s.injector.InjectIntoStateStorage(ctx, "GetStepStates", func(ctx context.Context) error {
		result, err = s.wrapped.GetStepStates(ctx, sagaID)
		return err
	})
	
	if storageErr != nil {
		return nil, storageErr
	}
	return result, err
}

// CleanupExpiredSagas cleans up expired Sagas with fault injection.
func (s *ChaosStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	return s.injector.InjectIntoStateStorage(ctx, "CleanupExpiredSagas", func(ctx context.Context) error {
		return s.wrapped.CleanupExpiredSagas(ctx, olderThan)
	})
}

// ChaosEventPublisher wraps an EventPublisher with fault injection.
type ChaosEventPublisher struct {
	wrapped  saga.EventPublisher
	injector *FaultInjector
}

// NewChaosEventPublisher creates a new chaos event publisher wrapper.
func NewChaosEventPublisher(wrapped saga.EventPublisher, injector *FaultInjector) *ChaosEventPublisher {
	return &ChaosEventPublisher{
		wrapped:  wrapped,
		injector: injector,
	}
}

// PublishEvent publishes an event with fault injection.
func (p *ChaosEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	return p.injector.InjectIntoEventPublisher(ctx, event.Type, func(ctx context.Context) error {
		return p.wrapped.PublishEvent(ctx, event)
	})
}

// Subscribe subscribes to events (no fault injection).
func (p *ChaosEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return p.wrapped.Subscribe(filter, handler)
}

// Unsubscribe unsubscribes from events (no fault injection).
func (p *ChaosEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return p.wrapped.Unsubscribe(subscription)
}

// Close closes the publisher.
func (p *ChaosEventPublisher) Close() error {
	return p.wrapped.Close()
}

