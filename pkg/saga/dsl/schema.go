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

// Package dsl provides a declarative DSL for defining Saga workflows using YAML.
// It allows users to define complex distributed transactions in a human-readable format.
package dsl

import (
	"time"
)

// SagaDefinition represents the top-level definition of a Saga workflow in YAML.
// It contains the configuration for the Saga and all its steps.
type SagaDefinition struct {
	// Saga contains the top-level Saga configuration
	Saga SagaConfig `yaml:"saga" validate:"required"`

	// Steps contains the ordered list of steps to execute
	Steps []StepConfig `yaml:"steps" validate:"required,min=1,dive"`

	// GlobalRetryPolicy defines the default retry policy for all steps
	// Individual steps can override this with their own retry policies
	GlobalRetryPolicy *RetryPolicy `yaml:"global_retry_policy,omitempty"`

	// GlobalCompensation defines the default compensation configuration
	GlobalCompensation *CompensationConfig `yaml:"global_compensation,omitempty"`
}

// SagaConfig contains the top-level configuration for a Saga.
type SagaConfig struct {
	// ID is the unique identifier for this Saga definition
	ID string `yaml:"id" validate:"required"`

	// Name is the human-readable name of the Saga
	Name string `yaml:"name" validate:"required"`

	// Description describes what this Saga does
	Description string `yaml:"description,omitempty"`

	// Version is the version of this Saga definition
	Version string `yaml:"version,omitempty"`

	// Timeout is the maximum duration for the entire Saga execution
	// Format: duration string (e.g., "30s", "5m", "1h")
	Timeout Duration `yaml:"timeout,omitempty" validate:"omitempty,min=0"`

	// Mode defines the execution mode: orchestration, choreography, or hybrid
	Mode ExecutionMode `yaml:"mode,omitempty" validate:"omitempty,oneof=orchestration choreography hybrid"`

	// Tags are custom labels for categorizing and filtering Sagas
	Tags []string `yaml:"tags,omitempty"`

	// Metadata contains additional custom metadata
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

// StepConfig defines the configuration for a single step in a Saga.
type StepConfig struct {
	// ID is the unique identifier for this step
	ID string `yaml:"id" validate:"required"`

	// Name is the human-readable name of the step
	Name string `yaml:"name" validate:"required"`

	// Description describes what this step does
	Description string `yaml:"description,omitempty"`

	// Type specifies the step type: service, function, http, grpc, message, etc.
	Type StepType `yaml:"type" validate:"required,oneof=service function http grpc message custom"`

	// Action defines the action to execute for this step
	Action ActionConfig `yaml:"action" validate:"required"`

	// Compensation defines the compensation action for this step
	Compensation *CompensationAction `yaml:"compensation,omitempty"`

	// RetryPolicy defines the retry policy for this step
	// If not specified, uses the global retry policy
	RetryPolicy *RetryPolicy `yaml:"retry_policy,omitempty"`

	// Timeout is the maximum duration for this step execution
	Timeout Duration `yaml:"timeout,omitempty" validate:"omitempty,min=0"`

	// Condition defines when this step should be executed (conditional logic)
	Condition *Condition `yaml:"condition,omitempty"`

	// Dependencies lists step IDs that must complete before this step
	Dependencies []string `yaml:"dependencies,omitempty"`

	// Async indicates if this step should be executed asynchronously
	Async bool `yaml:"async,omitempty"`

	// OnSuccess defines actions to execute when the step succeeds
	OnSuccess *HookConfig `yaml:"on_success,omitempty"`

	// OnFailure defines actions to execute when the step fails
	OnFailure *HookConfig `yaml:"on_failure,omitempty"`

	// Metadata contains additional custom metadata for this step
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

// StepType represents the type of step execution.
type StepType string

const (
	// StepTypeService invokes a service via RPC or REST
	StepTypeService StepType = "service"

	// StepTypeFunction executes a local function
	StepTypeFunction StepType = "function"

	// StepTypeHTTP performs an HTTP request
	StepTypeHTTP StepType = "http"

	// StepTypeGRPC performs a gRPC call
	StepTypeGRPC StepType = "grpc"

	// StepTypeMessage publishes a message to a broker
	StepTypeMessage StepType = "message"

	// StepTypeCustom allows custom step implementations
	StepTypeCustom StepType = "custom"
)

// ExecutionMode defines how the Saga is executed.
type ExecutionMode string

const (
	// ExecutionModeOrchestration uses centralized orchestration
	ExecutionModeOrchestration ExecutionMode = "orchestration"

	// ExecutionModeChoreography uses event-driven choreography
	ExecutionModeChoreography ExecutionMode = "choreography"

	// ExecutionModeHybrid combines orchestration and choreography
	ExecutionModeHybrid ExecutionMode = "hybrid"
)

// ActionConfig defines the action to execute for a step.
type ActionConfig struct {
	// Service specifies the service to call (for service/grpc/http types)
	Service *ServiceConfig `yaml:"service,omitempty"`

	// Function specifies the function to execute (for function type)
	Function *FunctionConfig `yaml:"function,omitempty"`

	// Message specifies the message to publish (for message type)
	Message *MessageConfig `yaml:"message,omitempty"`

	// Custom allows custom action configuration (for custom type)
	Custom map[string]interface{} `yaml:"custom,omitempty"`
}

// ServiceConfig defines a service call action.
type ServiceConfig struct {
	// Name is the name of the service
	Name string `yaml:"name" validate:"required"`

	// Endpoint is the service endpoint URL
	Endpoint string `yaml:"endpoint,omitempty"`

	// Method is the HTTP method or RPC method name
	Method string `yaml:"method" validate:"required"`

	// Path is the URL path for HTTP requests
	Path string `yaml:"path,omitempty"`

	// Headers contains HTTP headers or metadata
	Headers map[string]string `yaml:"headers,omitempty"`

	// Body specifies the request body or parameters
	Body map[string]interface{} `yaml:"body,omitempty"`

	// Timeout is the timeout for this service call
	Timeout Duration `yaml:"timeout,omitempty" validate:"omitempty,min=0"`
}

// FunctionConfig defines a function execution action.
type FunctionConfig struct {
	// Name is the name of the function to execute
	Name string `yaml:"name" validate:"required"`

	// Handler is the handler identifier or path
	Handler string `yaml:"handler" validate:"required"`

	// Parameters are the input parameters
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
}

// MessageConfig defines a message publishing action.
type MessageConfig struct {
	// Topic is the message topic or queue name
	Topic string `yaml:"topic" validate:"required"`

	// Broker is the message broker to use
	Broker string `yaml:"broker,omitempty"`

	// Payload is the message payload
	Payload map[string]interface{} `yaml:"payload,omitempty"`

	// Headers contains message headers
	Headers map[string]string `yaml:"headers,omitempty"`

	// RoutingKey is the routing key for topic exchanges
	RoutingKey string `yaml:"routing_key,omitempty"`
}

// CompensationAction defines the compensation action for a step.
type CompensationAction struct {
	// Type specifies the compensation type
	Type CompensationType `yaml:"type" validate:"required,oneof=automatic custom skip"`

	// Action defines the compensation action (similar to step action)
	Action *ActionConfig `yaml:"action,omitempty"`

	// Strategy defines the compensation strategy
	Strategy CompensationStrategy `yaml:"strategy,omitempty" validate:"omitempty,oneof=sequential parallel best_effort"`

	// Timeout is the timeout for compensation
	Timeout Duration `yaml:"timeout,omitempty" validate:"omitempty,min=0"`

	// MaxAttempts is the maximum number of compensation attempts
	MaxAttempts int `yaml:"max_attempts,omitempty" validate:"omitempty,min=0"`

	// OnFailure defines what to do if compensation fails
	OnFailure *CompensationFailurePolicy `yaml:"on_failure,omitempty"`
}

// CompensationType represents the type of compensation.
type CompensationType string

const (
	// CompensationTypeAutomatic uses automatic compensation
	CompensationTypeAutomatic CompensationType = "automatic"

	// CompensationTypeCustom uses custom compensation logic
	CompensationTypeCustom CompensationType = "custom"

	// CompensationTypeSkip skips compensation for this step
	CompensationTypeSkip CompensationType = "skip"
)

// CompensationStrategy defines how compensation is executed.
type CompensationStrategy string

const (
	// CompensationStrategySequential executes compensation in reverse order
	CompensationStrategySequential CompensationStrategy = "sequential"

	// CompensationStrategyParallel executes all compensations in parallel
	CompensationStrategyParallel CompensationStrategy = "parallel"

	// CompensationStrategyBestEffort attempts all compensations even if some fail
	CompensationStrategyBestEffort CompensationStrategy = "best_effort"
)

// CompensationConfig defines global compensation configuration.
type CompensationConfig struct {
	// Strategy defines the default compensation strategy
	Strategy CompensationStrategy `yaml:"strategy" validate:"required,oneof=sequential parallel best_effort"`

	// Timeout is the timeout for compensation operations
	Timeout Duration `yaml:"timeout,omitempty" validate:"omitempty,min=0"`

	// MaxAttempts is the default maximum compensation attempts
	MaxAttempts int `yaml:"max_attempts,omitempty" validate:"omitempty,min=1"`
}

// CompensationFailurePolicy defines what to do when compensation fails.
type CompensationFailurePolicy struct {
	// Action defines the action to take: ignore, retry, fail, alert
	Action string `yaml:"action" validate:"required,oneof=ignore retry fail alert"`

	// RetryPolicy for compensation failures
	RetryPolicy *RetryPolicy `yaml:"retry_policy,omitempty"`
}

// RetryPolicy defines the retry policy for failed operations.
type RetryPolicy struct {
	// Type is the retry policy type: fixed_delay, exponential_backoff, linear_backoff, no_retry
	Type string `yaml:"type" validate:"required,oneof=fixed_delay exponential_backoff linear_backoff no_retry"`

	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `yaml:"max_attempts" validate:"required,min=1"`

	// InitialDelay is the initial delay before the first retry
	InitialDelay Duration `yaml:"initial_delay,omitempty" validate:"omitempty,min=0"`

	// MaxDelay is the maximum delay between retries
	MaxDelay Duration `yaml:"max_delay,omitempty" validate:"omitempty,min=0"`

	// Multiplier is the multiplier for exponential backoff
	Multiplier float64 `yaml:"multiplier,omitempty" validate:"omitempty,min=1"`

	// Increment is the increment for linear backoff
	Increment Duration `yaml:"increment,omitempty" validate:"omitempty,min=0"`

	// Jitter adds random jitter to retry delays
	Jitter bool `yaml:"jitter,omitempty"`

	// RetryableErrors lists error codes/types that should be retried
	RetryableErrors []string `yaml:"retryable_errors,omitempty"`
}

// Condition defines a conditional expression for step execution.
type Condition struct {
	// Expression is the condition expression (e.g., "$input.amount > 100")
	Expression string `yaml:"expression" validate:"required"`

	// Variables are variables available in the condition context
	Variables map[string]interface{} `yaml:"variables,omitempty"`
}

// HookConfig defines hooks to execute on step events.
type HookConfig struct {
	// Actions are the actions to execute
	Actions []ActionConfig `yaml:"actions,omitempty"`

	// Notifications are notifications to send
	Notifications []NotificationConfig `yaml:"notifications,omitempty"`
}

// NotificationConfig defines a notification to send.
type NotificationConfig struct {
	// Type is the notification type: email, slack, webhook, etc.
	Type string `yaml:"type" validate:"required"`

	// Target is the notification target (email address, webhook URL, etc.)
	Target string `yaml:"target" validate:"required"`

	// Message is the notification message template
	Message string `yaml:"message,omitempty"`

	// Metadata contains additional notification metadata
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

// Duration is a wrapper around time.Duration for YAML unmarshaling.
type Duration time.Duration

// UnmarshalYAML implements the yaml.Unmarshaler interface for Duration.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface for Duration.
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// String returns the string representation of the Duration.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// ToDuration converts Duration to time.Duration.
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d)
}
