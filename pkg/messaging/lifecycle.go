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
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ShutdownConfig contains configuration for graceful shutdown procedures
type ShutdownConfig struct {
	// Timeout is the maximum time to wait for shutdown completion
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// ForceTimeout is the maximum time before forcing termination
	ForceTimeout time.Duration `yaml:"force_timeout" json:"force_timeout"`

	// DrainTimeout is the maximum time to wait for in-flight message completion
	DrainTimeout time.Duration `yaml:"drain_timeout" json:"drain_timeout"`

	// ReportInterval is the interval for reporting shutdown status
	ReportInterval time.Duration `yaml:"report_interval" json:"report_interval"`

	// MaxInflightMessages is the maximum number of messages to track during shutdown
	MaxInflightMessages int `yaml:"max_inflight_messages" json:"max_inflight_messages"`
}

// DefaultShutdownConfig returns default configuration for messaging shutdown
func DefaultShutdownConfig() *ShutdownConfig {
	return &ShutdownConfig{
		Timeout:             30 * time.Second,
		ForceTimeout:        60 * time.Second,
		DrainTimeout:        20 * time.Second,
		ReportInterval:      5 * time.Second,
		MaxInflightMessages: 1000,
	}
}

// ShutdownStatus represents the current shutdown status
type ShutdownStatus struct {
	// Phase represents the current shutdown phase
	Phase ShutdownPhase `json:"phase"`

	// StartTime is when shutdown began
	StartTime time.Time `json:"start_time"`

	// CompletedSteps lists completed shutdown steps
	CompletedSteps []string `json:"completed_steps"`

	// PendingSteps lists remaining shutdown steps
	PendingSteps []string `json:"pending_steps"`

	// InflightMessages is the current count of in-flight messages
	InflightMessages int64 `json:"inflight_messages"`

	// BrokerStatuses maps broker names to their shutdown status
	BrokerStatuses map[string]BrokerShutdownStatus `json:"broker_statuses"`

	// HandlerStatuses maps handler IDs to their shutdown status
	HandlerStatuses map[string]HandlerShutdownStatus `json:"handler_statuses"`

	// Error holds any error that occurred during shutdown
	Error error `json:"error,omitempty"`
}

// ShutdownPhase represents different phases of graceful shutdown
type ShutdownPhase int

const (
	ShutdownPhaseIdle ShutdownPhase = iota
	ShutdownPhaseInitiated
	ShutdownPhaseStoppingNewMessages
	ShutdownPhaseDrainingInflight
	ShutdownPhaseClosingConnections
	ShutdownPhaseCleaningHandlers
	ShutdownPhaseReleasingResources
	ShutdownPhaseCompleted
	ShutdownPhaseForcedTermination
)

// String returns the string representation of the shutdown phase
func (p ShutdownPhase) String() string {
	switch p {
	case ShutdownPhaseIdle:
		return "idle"
	case ShutdownPhaseInitiated:
		return "initiated"
	case ShutdownPhaseStoppingNewMessages:
		return "stopping_new_messages"
	case ShutdownPhaseDrainingInflight:
		return "draining_inflight"
	case ShutdownPhaseClosingConnections:
		return "closing_connections"
	case ShutdownPhaseCleaningHandlers:
		return "cleaning_handlers"
	case ShutdownPhaseReleasingResources:
		return "releasing_resources"
	case ShutdownPhaseCompleted:
		return "completed"
	case ShutdownPhaseForcedTermination:
		return "forced_termination"
	default:
		return "unknown"
	}
}

// BrokerShutdownStatus represents the shutdown status of a specific broker
type BrokerShutdownStatus struct {
	Name              string    `json:"name"`
	ConnectionsClosed bool      `json:"connections_closed"`
	InflightMessages  int64     `json:"inflight_messages"`
	LastActivity      time.Time `json:"last_activity"`
	Error             error     `json:"error,omitempty"`
}

// HandlerShutdownStatus represents the shutdown status of a specific handler
type HandlerShutdownStatus struct {
	HandlerID        string    `json:"handler_id"`
	Stopped          bool      `json:"stopped"`
	InflightMessages int64     `json:"inflight_messages"`
	LastActivity     time.Time `json:"last_activity"`
	Error            error     `json:"error,omitempty"`
}

// GracefulShutdownManager manages the graceful shutdown process for messaging components
type GracefulShutdownManager struct {
	config      *ShutdownConfig
	coordinator MessagingCoordinator

	// Shutdown state tracking
	status         *ShutdownStatus
	statusMu       sync.RWMutex
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc

	// In-flight message tracking
	inflightCounter   int64
	inflightCounterMu sync.RWMutex

	// Broker and handler tracking
	brokers  map[string]MessageBroker
	handlers map[string]EventHandler

	// Completion channels
	shutdownComplete chan struct{}
	forceShutdown    chan struct{}
}

// NewGracefulShutdownManager creates a new graceful shutdown manager
func NewGracefulShutdownManager(coordinator MessagingCoordinator, config *ShutdownConfig) *GracefulShutdownManager {
	if config == nil {
		config = DefaultShutdownConfig()
	}

	return &GracefulShutdownManager{
		config:      config,
		coordinator: coordinator,
		status: &ShutdownStatus{
			Phase:           ShutdownPhaseIdle,
			BrokerStatuses:  make(map[string]BrokerShutdownStatus),
			HandlerStatuses: make(map[string]HandlerShutdownStatus),
			CompletedSteps:  make([]string, 0),
			PendingSteps:    make([]string, 0),
		},
		brokers:          make(map[string]MessageBroker),
		handlers:         make(map[string]EventHandler),
		shutdownComplete: make(chan struct{}),
		forceShutdown:    make(chan struct{}),
	}
}

// InitiateGracefulShutdown starts the graceful shutdown process
func (gsm *GracefulShutdownManager) InitiateGracefulShutdown(ctx context.Context) error {
	gsm.statusMu.Lock()
	defer gsm.statusMu.Unlock()

	if gsm.status.Phase != ShutdownPhaseIdle {
		return fmt.Errorf("shutdown already in progress, current phase: %s", gsm.status.Phase.String())
	}

	logger.Logger.Info("Initiating graceful shutdown of messaging system")

	// Initialize shutdown context
	gsm.shutdownCtx, gsm.shutdownCancel = context.WithTimeout(ctx, gsm.config.Timeout)
	gsm.status.Phase = ShutdownPhaseInitiated
	gsm.status.StartTime = time.Now()

	// Initialize pending steps
	gsm.status.PendingSteps = []string{
		"stop_accepting_new_messages",
		"drain_inflight_messages",
		"close_broker_connections",
		"cleanup_event_handlers",
		"release_coordinator_resources",
	}

	// Populate broker and handler maps from coordinator
	if err := gsm.populateComponentMaps(); err != nil {
		gsm.status.Error = err
		return fmt.Errorf("failed to populate component maps: %w", err)
	}

	// Start shutdown sequence in goroutine
	go gsm.executeShutdownSequence()

	// Start force shutdown timer
	go gsm.handleForceShutdown()

	// Start status reporting
	go gsm.reportShutdownStatus()

	return nil
}

// GetShutdownStatus returns the current shutdown status
func (gsm *GracefulShutdownManager) GetShutdownStatus() *ShutdownStatus {
	gsm.statusMu.RLock()
	defer gsm.statusMu.RUnlock()

	// Create a copy of the status to avoid concurrent access issues
	statusCopy := &ShutdownStatus{
		Phase:            gsm.status.Phase,
		StartTime:        gsm.status.StartTime,
		CompletedSteps:   make([]string, len(gsm.status.CompletedSteps)),
		PendingSteps:     make([]string, len(gsm.status.PendingSteps)),
		InflightMessages: gsm.getInflightMessageCount(),
		BrokerStatuses:   make(map[string]BrokerShutdownStatus),
		HandlerStatuses:  make(map[string]HandlerShutdownStatus),
		Error:            gsm.status.Error,
	}

	copy(statusCopy.CompletedSteps, gsm.status.CompletedSteps)
	copy(statusCopy.PendingSteps, gsm.status.PendingSteps)

	for k, v := range gsm.status.BrokerStatuses {
		statusCopy.BrokerStatuses[k] = v
	}

	for k, v := range gsm.status.HandlerStatuses {
		statusCopy.HandlerStatuses[k] = v
	}

	return statusCopy
}

// WaitForCompletion waits for the shutdown process to complete
func (gsm *GracefulShutdownManager) WaitForCompletion() error {
	select {
	case <-gsm.shutdownComplete:
		status := gsm.GetShutdownStatus()
		if status.Error != nil {
			return status.Error
		}
		return nil
	case <-gsm.forceShutdown:
		return fmt.Errorf("shutdown was forcefully terminated due to timeout")
	}
}

// executeShutdownSequence performs the actual shutdown sequence
func (gsm *GracefulShutdownManager) executeShutdownSequence() {
	defer close(gsm.shutdownComplete)

	logger.Logger.Info("Executing graceful shutdown sequence")

	// Step 1: Stop accepting new messages
	if err := gsm.stopAcceptingNewMessages(); err != nil {
		gsm.setShutdownError(fmt.Errorf("failed to stop accepting new messages: %w", err))
		return
	}

	// Step 2: Drain in-flight messages
	if err := gsm.drainInflightMessages(); err != nil {
		gsm.setShutdownError(fmt.Errorf("failed to drain in-flight messages: %w", err))
		return
	}

	// Step 3: Close broker connections gracefully
	if err := gsm.closeBrokerConnections(); err != nil {
		gsm.setShutdownError(fmt.Errorf("failed to close broker connections: %w", err))
		return
	}

	// Step 4: Clean up event handler registrations
	if err := gsm.cleanupEventHandlers(); err != nil {
		gsm.setShutdownError(fmt.Errorf("failed to cleanup event handlers: %w", err))
		return
	}

	// Step 5: Release coordinator resources
	if err := gsm.releaseCoordinatorResources(); err != nil {
		gsm.setShutdownError(fmt.Errorf("failed to release coordinator resources: %w", err))
		return
	}

	gsm.statusMu.Lock()
	gsm.status.Phase = ShutdownPhaseCompleted
	gsm.statusMu.Unlock()

	logger.Logger.Info("Graceful shutdown sequence completed successfully",
		zap.Duration("duration", time.Since(gsm.status.StartTime)),
		zap.Int("completed_steps", len(gsm.status.CompletedSteps)))
}

// stopAcceptingNewMessages stops accepting new messages for all brokers
func (gsm *GracefulShutdownManager) stopAcceptingNewMessages() error {
	gsm.statusMu.Lock()
	gsm.status.Phase = ShutdownPhaseStoppingNewMessages
	gsm.statusMu.Unlock()

	logger.Logger.Info("Stopping acceptance of new messages")

	// This is typically handled by pausing subscribers or marking them as stopping
	// The actual implementation depends on the broker type
	for name, broker := range gsm.brokers {
		logger.Logger.Debug("Stopping new message acceptance for broker", zap.String("broker", name))

		gsm.statusMu.Lock()
		status := gsm.status.BrokerStatuses[name]
		status.LastActivity = time.Now()
		gsm.status.BrokerStatuses[name] = status
		gsm.statusMu.Unlock()

		// For now, we'll just mark that we've processed this broker
		// In a real implementation, this would involve broker-specific operations
		_ = broker
	}

	gsm.markStepCompleted("stop_accepting_new_messages")
	return nil
}

// drainInflightMessages waits for in-flight messages to complete processing
func (gsm *GracefulShutdownManager) drainInflightMessages() error {
	gsm.statusMu.Lock()
	gsm.status.Phase = ShutdownPhaseDrainingInflight
	gsm.statusMu.Unlock()

	logger.Logger.Info("Draining in-flight messages",
		zap.Int64("current_inflight", gsm.getInflightMessageCount()))

	drainCtx, cancel := context.WithTimeout(gsm.shutdownCtx, gsm.config.DrainTimeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-drainCtx.Done():
			inflightCount := gsm.getInflightMessageCount()
			if inflightCount > 0 {
				logger.Logger.Warn("Drain timeout reached with messages still in-flight",
					zap.Int64("remaining_inflight", inflightCount))
				// Continue with shutdown even if there are still in-flight messages
			}
			gsm.markStepCompleted("drain_inflight_messages")
			return nil

		case <-ticker.C:
			inflightCount := gsm.getInflightMessageCount()
			if inflightCount == 0 {
				logger.Logger.Info("All in-flight messages drained successfully")
				gsm.markStepCompleted("drain_inflight_messages")
				return nil
			}

			logger.Logger.Debug("Waiting for in-flight messages to complete",
				zap.Int64("remaining", inflightCount))
		}
	}
}

// closeBrokerConnections gracefully closes all broker connections
func (gsm *GracefulShutdownManager) closeBrokerConnections() error {
	gsm.statusMu.Lock()
	gsm.status.Phase = ShutdownPhaseClosingConnections
	gsm.statusMu.Unlock()

	logger.Logger.Info("Closing broker connections gracefully",
		zap.Int("broker_count", len(gsm.brokers)))

	var closeErrors []error
	for name, broker := range gsm.brokers {
		logger.Logger.Debug("Closing broker connection", zap.String("broker", name))

		if err := broker.Disconnect(gsm.shutdownCtx); err != nil {
			logger.Logger.Error("Failed to disconnect broker gracefully",
				zap.String("broker", name),
				zap.Error(err))
			closeErrors = append(closeErrors, fmt.Errorf("broker %s: %w", name, err))

			gsm.statusMu.Lock()
			status := gsm.status.BrokerStatuses[name]
			status.Error = err
			gsm.status.BrokerStatuses[name] = status
			gsm.statusMu.Unlock()
		} else {
			logger.Logger.Debug("Broker disconnected successfully", zap.String("broker", name))

			gsm.statusMu.Lock()
			status := gsm.status.BrokerStatuses[name]
			status.ConnectionsClosed = true
			status.LastActivity = time.Now()
			gsm.status.BrokerStatuses[name] = status
			gsm.statusMu.Unlock()
		}
	}

	if len(closeErrors) > 0 {
		return fmt.Errorf("errors closing broker connections: %v", closeErrors)
	}

	gsm.markStepCompleted("close_broker_connections")
	return nil
}

// cleanupEventHandlers cleans up event handler registrations
func (gsm *GracefulShutdownManager) cleanupEventHandlers() error {
	gsm.statusMu.Lock()
	gsm.status.Phase = ShutdownPhaseCleaningHandlers
	handlerIDs := make([]string, 0, len(gsm.status.HandlerStatuses))
	for handlerID := range gsm.status.HandlerStatuses {
		handlerIDs = append(handlerIDs, handlerID)
	}
	gsm.statusMu.Unlock()

	logger.Logger.Info("Cleaning up event handler registrations",
		zap.Int("handler_count", len(handlerIDs)))

	// Since we don't have direct access to handler instances in the graceful shutdown manager,
	// we mark handlers as stopped and let the coordinator handle the actual shutdown.
	// This is a coordination step to track the cleanup process.
	for _, handlerID := range handlerIDs {
		logger.Logger.Debug("Marking event handler for cleanup", zap.String("handler_id", handlerID))

		gsm.statusMu.Lock()
		status := gsm.status.HandlerStatuses[handlerID]
		status.Stopped = true
		status.LastActivity = time.Now()
		gsm.status.HandlerStatuses[handlerID] = status
		gsm.statusMu.Unlock()

		logger.Logger.Debug("Event handler marked for cleanup", zap.String("handler_id", handlerID))
	}

	gsm.markStepCompleted("cleanup_event_handlers")
	return nil
}

// releaseCoordinatorResources releases coordinator resources
func (gsm *GracefulShutdownManager) releaseCoordinatorResources() error {
	gsm.statusMu.Lock()
	gsm.status.Phase = ShutdownPhaseReleasingResources
	gsm.statusMu.Unlock()

	logger.Logger.Info("Releasing coordinator resources")

	if err := gsm.coordinator.Stop(gsm.shutdownCtx); err != nil {
		return fmt.Errorf("failed to stop messaging coordinator: %w", err)
	}

	gsm.markStepCompleted("release_coordinator_resources")
	return nil
}

// populateComponentMaps populates broker and handler maps from the coordinator
func (gsm *GracefulShutdownManager) populateComponentMaps() error {
	// Get brokers from coordinator
	brokerNames := gsm.coordinator.GetRegisteredBrokers()
	for _, name := range brokerNames {
		broker, err := gsm.coordinator.GetBroker(name)
		if err != nil {
			return fmt.Errorf("failed to get broker %s: %w", name, err)
		}
		gsm.brokers[name] = broker

		// Initialize broker status
		gsm.status.BrokerStatuses[name] = BrokerShutdownStatus{
			Name:         name,
			LastActivity: time.Now(),
		}
	}

	// Get handlers from coordinator
	handlerIDs := gsm.coordinator.GetRegisteredHandlers()
	for _, handlerID := range handlerIDs {
		// Track handlers by ID only since we can't access the actual handler instances
		// The coordinator's Stop method will handle the actual handler shutdown
		gsm.status.HandlerStatuses[handlerID] = HandlerShutdownStatus{
			HandlerID:    handlerID,
			LastActivity: time.Now(),
		}
	}

	return nil
}

// markStepCompleted marks a shutdown step as completed
func (gsm *GracefulShutdownManager) markStepCompleted(step string) {
	gsm.statusMu.Lock()
	defer gsm.statusMu.Unlock()

	// Add to completed steps
	gsm.status.CompletedSteps = append(gsm.status.CompletedSteps, step)

	// Remove from pending steps
	for i, pendingStep := range gsm.status.PendingSteps {
		if pendingStep == step {
			gsm.status.PendingSteps = append(gsm.status.PendingSteps[:i], gsm.status.PendingSteps[i+1:]...)
			break
		}
	}

	logger.Logger.Debug("Shutdown step completed", zap.String("step", step))
}

// setShutdownError sets an error in the shutdown status
func (gsm *GracefulShutdownManager) setShutdownError(err error) {
	gsm.statusMu.Lock()
	defer gsm.statusMu.Unlock()

	gsm.status.Error = err
	logger.Logger.Error("Shutdown error occurred", zap.Error(err))
}

// getInflightMessageCount returns the current count of in-flight messages
func (gsm *GracefulShutdownManager) getInflightMessageCount() int64 {
	gsm.inflightCounterMu.RLock()
	defer gsm.inflightCounterMu.RUnlock()
	return gsm.inflightCounter
}

// handleForceShutdown handles forced shutdown after timeout
func (gsm *GracefulShutdownManager) handleForceShutdown() {
	forceTimer := time.NewTimer(gsm.config.ForceTimeout)
	defer forceTimer.Stop()

	select {
	case <-gsm.shutdownComplete:
		// Normal shutdown completed
		return
	case <-forceTimer.C:
		logger.Logger.Warn("Force shutdown triggered due to timeout",
			zap.Duration("timeout", gsm.config.ForceTimeout))

		gsm.statusMu.Lock()
		gsm.status.Phase = ShutdownPhaseForcedTermination
		// Record an error to ensure WaitForCompletion surfaces forced termination
		if gsm.status.Error == nil {
			gsm.status.Error = fmt.Errorf("shutdown was forcefully terminated due to timeout")
		}
		gsm.statusMu.Unlock()

		// Cancel the shutdown context to force termination
		if gsm.shutdownCancel != nil {
			gsm.shutdownCancel()
		}

		close(gsm.forceShutdown)
	}
}

// reportShutdownStatus periodically reports shutdown status
func (gsm *GracefulShutdownManager) reportShutdownStatus() {
	ticker := time.NewTicker(gsm.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-gsm.shutdownComplete:
			return
		case <-gsm.forceShutdown:
			return
		case <-ticker.C:
			status := gsm.GetShutdownStatus()
			logger.Logger.Info("Shutdown status report",
				zap.String("phase", status.Phase.String()),
				zap.Duration("elapsed", time.Since(status.StartTime)),
				zap.Int("completed_steps", len(status.CompletedSteps)),
				zap.Int("pending_steps", len(status.PendingSteps)),
				zap.Int64("inflight_messages", status.InflightMessages))
		}
	}
}

// IncrementInflightMessages increments the in-flight message counter
func (gsm *GracefulShutdownManager) IncrementInflightMessages() {
	gsm.inflightCounterMu.Lock()
	defer gsm.inflightCounterMu.Unlock()
	gsm.inflightCounter++
}

// DecrementInflightMessages decrements the in-flight message counter
func (gsm *GracefulShutdownManager) DecrementInflightMessages() {
	gsm.inflightCounterMu.Lock()
	defer gsm.inflightCounterMu.Unlock()
	if gsm.inflightCounter > 0 {
		gsm.inflightCounter--
	}
}
