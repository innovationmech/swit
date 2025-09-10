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
	"sync/atomic"
	"time"
)

// SubscriberState represents the current state of a subscriber.
type SubscriberState int

const (
	// SubscriberStateUnknown indicates the subscriber state is unknown
	SubscriberStateUnknown SubscriberState = iota

	// SubscriberStateConnecting indicates the subscriber is connecting
	SubscriberStateConnecting

	// SubscriberStateSubscribed indicates the subscriber is actively consuming messages
	SubscriberStateSubscribed

	// SubscriberStatePaused indicates the subscriber is paused
	SubscriberStatePaused

	// SubscriberStateUnsubscribing indicates the subscriber is unsubscribing
	SubscriberStateUnsubscribing

	// SubscriberStateClosed indicates the subscriber is closed
	SubscriberStateClosed
)

// String returns the string representation of SubscriberState.
func (s SubscriberState) String() string {
	switch s {
	case SubscriberStateUnknown:
		return "unknown"
	case SubscriberStateConnecting:
		return "connecting"
	case SubscriberStateSubscribed:
		return "subscribed"
	case SubscriberStatePaused:
		return "paused"
	case SubscriberStateUnsubscribing:
		return "unsubscribing"
	case SubscriberStateClosed:
		return "closed"
	default:
		return "invalid"
	}
}

// SubscriberStateMetrics provides additional state-specific metrics for subscriber monitoring.
type SubscriberStateMetrics struct {
	// ActiveHandlers is the number of currently active handlers
	ActiveHandlers int32 `json:"active_handlers"`

	// State is the current subscriber state
	State SubscriberState `json:"state"`

	// LastMessageTimestamp is when the last message was received
	LastMessageTimestamp time.Time `json:"last_message_timestamp"`

	// ConnectedSince tracks when the subscriber connected
	ConnectedSince time.Time `json:"connected_since"`

	// ProcessingTime tracks the total time spent processing messages
	ProcessingTime time.Duration `json:"processing_time"`

	// ConsumerGroupMembers is the number of members in consumer group
	ConsumerGroupMembers int `json:"consumer_group_members"`
}

// HandlerRegistration represents a registered message handler with its configuration.
type HandlerRegistration struct {
	// Handler is the message handler instance
	Handler MessageHandler `json:"-"`

	// Topic is the topic this handler is registered for
	Topic string `json:"topic"`

	// HandlerID is a unique identifier for this handler
	HandlerID string `json:"handler_id"`

	// Middleware is the combined middleware chain (global + handler-specific)
	Middleware []Middleware `json:"-"`

	// HandlerSpecificMiddleware stores middleware specific to this handler
	HandlerSpecificMiddleware []Middleware `json:"-"`

	// Config contains handler-specific configuration
	Config HandlerConfig `json:"config"`

	// RegisteredAt tracks when this handler was registered
	RegisteredAt time.Time `json:"registered_at"`
}

// HandlerConfig defines configuration for individual handlers.
type HandlerConfig struct {
	// MaxConcurrency for this specific handler
	MaxConcurrency int `json:"max_concurrency" default:"1"`

	// Timeout for individual message processing
	Timeout time.Duration `json:"timeout" default:"30s"`

	// RetryPolicy for this handler
	RetryPolicy *RetryConfig `json:"retry_policy,omitempty"`

	// DeadLetterPolicy for this handler
	DeadLetterPolicy *DeadLetterConfig `json:"dead_letter_policy,omitempty"`

	// AckMode for this handler
	AckMode AckMode `json:"ack_mode" default:"auto"`
}

// SubscriberManagerConfig defines configuration for the subscription manager.
type SubscriberManagerConfig struct {
	// GlobalMiddleware applied to all handlers
	GlobalMiddleware []Middleware `json:"-"`

	// MaxHandlers is the maximum number of handlers allowed
	MaxHandlers int `json:"max_handlers" default:"100"`

	// HealthCheckInterval for subscriber health monitoring
	HealthCheckInterval time.Duration `json:"health_check_interval" default:"30s"`

	// MetricsInterval for metrics collection
	MetricsInterval time.Duration `json:"metrics_interval" default:"10s"`

	// StopTimeout for graceful shutdown
	StopTimeout time.Duration `json:"stop_timeout" default:"30s"`
}

// ConsumerGroupStatus provides information about consumer group membership.
type ConsumerGroupStatus struct {
	// GroupID is the consumer group identifier
	GroupID string `json:"group_id"`

	// MemberID is this consumer's member ID
	MemberID string `json:"member_id"`

	// Generation is the current group generation
	Generation int32 `json:"generation"`

	// Members lists all members in the group
	Members []ConsumerGroupMember `json:"members"`

	// Coordinator is the group coordinator information
	Coordinator string `json:"coordinator"`

	// State is the group state
	State string `json:"state"`

	// Protocol is the partition assignment protocol
	Protocol string `json:"protocol"`

	// LastRebalance tracks when the last rebalance occurred
	LastRebalance time.Time `json:"last_rebalance"`
}

// ConsumerGroupMember represents a member of the consumer group.
type ConsumerGroupMember struct {
	// MemberID is the unique member identifier
	MemberID string `json:"member_id"`

	// ClientID is the client identifier
	ClientID string `json:"client_id"`

	// Host is the member's host address
	Host string `json:"host"`

	// AssignedPartitions lists partitions assigned to this member
	AssignedPartitions []PartitionAssignment `json:"assigned_partitions"`
}

// PartitionAssignment represents a topic partition assignment.
type PartitionAssignment struct {
	// Topic is the topic name
	Topic string `json:"topic"`

	// Partition is the partition number
	Partition int32 `json:"partition"`

	// CurrentOffset is the current committed offset
	CurrentOffset int64 `json:"current_offset"`

	// HighWaterMark is the partition's high water mark
	HighWaterMark int64 `json:"high_water_mark"`

	// Lag is the current lag for this partition
	Lag int64 `json:"lag"`
}

// AcknowledgmentHandle provides acknowledgment control for messages.
type AcknowledgmentHandle interface {
	// Ack acknowledges successful processing of the message
	Ack() error

	// Nack negatively acknowledges the message, indicating processing failure
	Nack(retry bool) error

	// IsAcked returns whether the message has been acknowledged
	IsAcked() bool

	// GetMessage returns the associated message
	GetMessage() *Message
}

// SubscriptionHandle provides control over an active subscription.
type SubscriptionHandle interface {
	// GetSubscriptionID returns the unique subscription identifier
	GetSubscriptionID() string

	// GetTopics returns the topics this subscription is consuming from
	GetTopics() []string

	// GetState returns the current subscription state
	GetState() SubscriberState

	// GetMetrics returns current subscription metrics
	GetMetrics() *SubscriberMetrics

	// GetConsumerGroupStatus returns consumer group information
	GetConsumerGroupStatus(ctx context.Context) (*ConsumerGroupStatus, error)

	// UpdateSubscription modifies the subscription (topics, handlers, etc.)
	UpdateSubscription(ctx context.Context, config *SubscriberConfig) error
}

// BaseSubscriber provides a basic implementation of EventSubscriber with common functionality.
// Broker-specific implementations can embed this to inherit default behavior.
type BaseSubscriber struct {
	// config holds the subscriber configuration
	config SubscriberConfig

	// handlers stores registered handlers by topic
	handlers      map[string][]*HandlerRegistration
	handlersMutex sync.RWMutex

	// state tracks the current subscriber state
	state      SubscriberState
	stateMutex sync.RWMutex

	// metrics tracks subscriber performance metrics
	metrics      *SubscriberMetrics
	metricsMutex sync.RWMutex

	// stateMetrics tracks state-specific metrics
	stateMetrics      *SubscriberStateMetrics
	stateMetricsMutex sync.RWMutex

	// activeHandlers tracks the number of currently processing handlers
	activeHandlers int32

	// stopCh signals subscriber shutdown
	stopCh chan struct{}

	// wg tracks active processing goroutines
	wg sync.WaitGroup

	// middleware holds global middleware chain
	globalMiddleware []Middleware
}

// NewBaseSubscriber creates a new base subscriber with the given configuration.
func NewBaseSubscriber(config SubscriberConfig) *BaseSubscriber {
	return &BaseSubscriber{
		config:           config,
		handlers:         make(map[string][]*HandlerRegistration),
		state:            SubscriberStateUnknown,
		metrics:          &SubscriberMetrics{},
		stateMetrics:     &SubscriberStateMetrics{},
		stopCh:           make(chan struct{}),
		globalMiddleware: make([]Middleware, 0),
	}
}

// RegisterHandler registers a message handler for a specific topic.
func (s *BaseSubscriber) RegisterHandler(topic string, handler MessageHandler, middleware ...Middleware) error {
	s.handlersMutex.Lock()
	defer s.handlersMutex.Unlock()

	if handler == nil {
		return NewConfigError("handler cannot be nil", nil)
	}

	if topic == "" {
		return NewConfigError("topic cannot be empty", nil)
	}

	handlerID := fmt.Sprintf("%s-%d", topic, time.Now().UnixNano())

	registration := &HandlerRegistration{
		Handler:                   handler,
		Topic:                     topic,
		HandlerID:                 handlerID,
		HandlerSpecificMiddleware: append([]Middleware(nil), middleware...), // Copy to prevent external modifications
		Config:                    HandlerConfig{MaxConcurrency: 1, Timeout: 30 * time.Second, AckMode: AckModeAuto},
		RegisteredAt:              time.Now(),
	}

	// Build combined middleware chain
	registration.Middleware = make([]Middleware, 0, len(s.globalMiddleware)+len(middleware))
	registration.Middleware = append(registration.Middleware, s.globalMiddleware...)
	registration.Middleware = append(registration.Middleware, middleware...)

	if s.handlers[topic] == nil {
		s.handlers[topic] = make([]*HandlerRegistration, 0)
	}

	s.handlers[topic] = append(s.handlers[topic], registration)

	return nil
}

// UnregisterHandler removes a handler for a specific topic.
func (s *BaseSubscriber) UnregisterHandler(topic string, handlerID string) error {
	s.handlersMutex.Lock()
	defer s.handlersMutex.Unlock()

	handlers, exists := s.handlers[topic]
	if !exists {
		return NewConfigError(fmt.Sprintf("no handlers found for topic: %s", topic), nil)
	}

	for i, registration := range handlers {
		if registration.HandlerID == handlerID {
			// Remove handler from slice
			s.handlers[topic] = append(handlers[:i], handlers[i+1:]...)

			// If no more handlers for this topic, remove the topic entry
			if len(s.handlers[topic]) == 0 {
				delete(s.handlers, topic)
			}

			return nil
		}
	}

	return NewConfigError(fmt.Sprintf("handler with ID %s not found for topic %s", handlerID, topic), nil)
}

// GetRegisteredHandlers returns all registered handlers for a topic.
func (s *BaseSubscriber) GetRegisteredHandlers(topic string) []*HandlerRegistration {
	s.handlersMutex.RLock()
	defer s.handlersMutex.RUnlock()

	handlers, exists := s.handlers[topic]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]*HandlerRegistration, len(handlers))
	copy(result, handlers)
	return result
}

// AddGlobalMiddleware adds middleware that will be applied to all handlers.
func (s *BaseSubscriber) AddGlobalMiddleware(middleware ...Middleware) {
	s.handlersMutex.Lock()
	defer s.handlersMutex.Unlock()

	s.globalMiddleware = append(s.globalMiddleware, middleware...)

	// Rebuild middleware chain for existing handler registrations
	for topic := range s.handlers {
		for _, registration := range s.handlers[topic] {
			// Rebuild combined middleware chain from global + handler-specific
			registration.Middleware = make([]Middleware, 0, len(s.globalMiddleware)+len(registration.HandlerSpecificMiddleware))
			registration.Middleware = append(registration.Middleware, s.globalMiddleware...)
			registration.Middleware = append(registration.Middleware, registration.HandlerSpecificMiddleware...)
		}
	}
}

// GetState returns the current subscriber state.
func (s *BaseSubscriber) GetState() SubscriberState {
	s.stateMutex.RLock()
	defer s.stateMutex.RUnlock()
	return s.state
}

// setState updates the subscriber state.
func (s *BaseSubscriber) setState(newState SubscriberState) {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	s.state = newState

	s.stateMetricsMutex.Lock()
	s.stateMetrics.State = newState
	s.stateMetricsMutex.Unlock()
}

// GetMetrics returns current subscriber metrics.
func (s *BaseSubscriber) GetMetrics() *SubscriberMetrics {
	s.metricsMutex.RLock()
	defer s.metricsMutex.RUnlock()

	// Create a copy to prevent external modification
	metrics := *s.metrics

	// Get state metrics and active handlers count atomically
	s.stateMetricsMutex.RLock()
	stateMetrics := *s.stateMetrics
	activeHandlers := atomic.LoadInt32(&s.activeHandlers)
	stateMetrics.ActiveHandlers = activeHandlers
	s.stateMetricsMutex.RUnlock()

	// Calculate average processing time if we have processing data
	if metrics.MessagesProcessed > 0 && stateMetrics.ProcessingTime > 0 {
		metrics.AvgProcessingTime = time.Duration(int64(stateMetrics.ProcessingTime) / metrics.MessagesProcessed)
	}

	return &metrics
}

// updateMetrics atomically updates subscriber metrics.
func (s *BaseSubscriber) updateMetrics(updateFunc func(*SubscriberMetrics)) {
	s.metricsMutex.Lock()
	defer s.metricsMutex.Unlock()
	updateFunc(s.metrics)
}

// processMessage processes a single message through the handler chain.
func (s *BaseSubscriber) processMessage(ctx context.Context, message *Message, registration *HandlerRegistration) error {
	start := time.Now()
	atomic.AddInt32(&s.activeHandlers, 1)
	defer atomic.AddInt32(&s.activeHandlers, -1)

	s.updateMetrics(func(m *SubscriberMetrics) {
		m.MessagesConsumed++
		m.LastActivity = time.Now()
	})

	// Create handler chain with middleware
	handler := s.buildHandlerChain(registration.Handler, registration.Middleware)

	// Process message with timeout
	processingCtx, cancel := context.WithTimeout(ctx, registration.Config.Timeout)
	defer cancel()

	err := handler.Handle(processingCtx, message)

	processingTime := time.Since(start)

	s.updateMetrics(func(m *SubscriberMetrics) {
		if m.MinProcessingTime == 0 || processingTime < m.MinProcessingTime {
			m.MinProcessingTime = processingTime
		}
		if processingTime > m.MaxProcessingTime {
			m.MaxProcessingTime = processingTime
		}

		if err != nil {
			m.MessagesFailed++
		} else {
			m.MessagesProcessed++
		}
	})

	// Update state metrics with processing time
	s.stateMetricsMutex.Lock()
	s.stateMetrics.ProcessingTime += processingTime
	s.stateMetricsMutex.Unlock()

	return err
}

// buildHandlerChain creates a handler chain with middleware applied in order.
func (s *BaseSubscriber) buildHandlerChain(handler MessageHandler, middleware []Middleware) MessageHandler {
	// Apply middleware in reverse order so the first middleware is outermost
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i].Wrap(handler)
	}
	return handler
}

// Subscribe starts consuming messages with the given handler (implemented by specific brokers).
func (s *BaseSubscriber) Subscribe(ctx context.Context, handler MessageHandler) error {
	return NewConfigError("Subscribe must be implemented by specific broker", nil)
}

// SubscribeWithMiddleware starts consuming with a middleware chain (implemented by specific brokers).
func (s *BaseSubscriber) SubscribeWithMiddleware(ctx context.Context, handler MessageHandler, middleware ...Middleware) error {
	return NewConfigError("SubscribeWithMiddleware must be implemented by specific broker", nil)
}

// Unsubscribe stops consuming messages (implemented by specific brokers).
func (s *BaseSubscriber) Unsubscribe(ctx context.Context) error {
	return NewConfigError("Unsubscribe must be implemented by specific broker", nil)
}

// Pause temporarily stops message consumption (implemented by specific brokers).
func (s *BaseSubscriber) Pause(ctx context.Context) error {
	return NewConfigError("Pause must be implemented by specific broker", nil)
}

// Resume resumes message consumption (implemented by specific brokers).
func (s *BaseSubscriber) Resume(ctx context.Context) error {
	return NewConfigError("Resume must be implemented by specific broker", nil)
}

// Seek seeks to a specific position (implemented by specific brokers).
func (s *BaseSubscriber) Seek(ctx context.Context, position SeekPosition) error {
	return NewConfigError("Seek must be implemented by specific broker", nil)
}

// GetLag returns the current consumer lag (implemented by specific brokers).
func (s *BaseSubscriber) GetLag(ctx context.Context) (int64, error) {
	return -1, NewConfigError("GetLag must be implemented by specific broker", nil)
}

// Close closes the subscriber and releases resources.
func (s *BaseSubscriber) Close() error {
	s.setState(SubscriberStateClosed)

	close(s.stopCh)
	s.wg.Wait()

	return nil
}
