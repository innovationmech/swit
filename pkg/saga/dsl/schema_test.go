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

package dsl

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestSagaDefinitionUnmarshal tests unmarshaling a complete Saga definition from YAML.
func TestSagaDefinitionUnmarshal(t *testing.T) {
	yamlData := `
saga:
  id: test-saga
  name: Test Saga
  description: A test saga for validation
  version: "1.0.0"
  timeout: 5m
  mode: orchestration
  tags:
    - test
    - validation
  metadata:
    owner: test-team
    priority: high

global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true

steps:
  - id: step-1
    name: First Step
    description: The first step
    type: service
    action:
      service:
        name: test-service
        endpoint: http://test:8080
        method: POST
        path: /api/test
        headers:
          Content-Type: application/json
        body:
          key: value
        timeout: 10s
    compensation:
      type: custom
      action:
        service:
          name: test-service
          method: DELETE
          path: /api/test
      strategy: sequential
      timeout: 30s
      max_attempts: 3
    retry_policy:
      type: fixed_delay
      max_attempts: 3
      initial_delay: 5s
    timeout: 30s
`

	var def SagaDefinition
	err := yaml.Unmarshal([]byte(yamlData), &def)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Validate Saga config
	if def.Saga.ID != "test-saga" {
		t.Errorf("Expected saga ID 'test-saga', got '%s'", def.Saga.ID)
	}
	if def.Saga.Name != "Test Saga" {
		t.Errorf("Expected saga name 'Test Saga', got '%s'", def.Saga.Name)
	}
	if def.Saga.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", def.Saga.Version)
	}
	if def.Saga.Mode != ExecutionModeOrchestration {
		t.Errorf("Expected mode 'orchestration', got '%s'", def.Saga.Mode)
	}
	if def.Saga.Timeout.ToDuration() != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", def.Saga.Timeout)
	}
	if len(def.Saga.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(def.Saga.Tags))
	}

	// Validate global retry policy
	if def.GlobalRetryPolicy == nil {
		t.Fatal("Expected global retry policy, got nil")
	}
	if def.GlobalRetryPolicy.Type != "exponential_backoff" {
		t.Errorf("Expected retry type 'exponential_backoff', got '%s'", def.GlobalRetryPolicy.Type)
	}
	if def.GlobalRetryPolicy.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", def.GlobalRetryPolicy.MaxAttempts)
	}

	// Validate steps
	if len(def.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(def.Steps))
	}

	step := def.Steps[0]
	if step.ID != "step-1" {
		t.Errorf("Expected step ID 'step-1', got '%s'", step.ID)
	}
	if step.Type != StepTypeService {
		t.Errorf("Expected step type 'service', got '%s'", step.Type)
	}
	if step.Timeout.ToDuration() != 30*time.Second {
		t.Errorf("Expected step timeout 30s, got %v", step.Timeout)
	}

	// Validate action
	if step.Action.Service == nil {
		t.Fatal("Expected service action, got nil")
	}
	if step.Action.Service.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", step.Action.Service.Name)
	}
	if step.Action.Service.Method != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", step.Action.Service.Method)
	}

	// Validate compensation
	if step.Compensation == nil {
		t.Fatal("Expected compensation config, got nil")
	}
	if step.Compensation.Type != CompensationTypeCustom {
		t.Errorf("Expected compensation type 'custom', got '%s'", step.Compensation.Type)
	}
	if step.Compensation.Strategy != CompensationStrategySequential {
		t.Errorf("Expected compensation strategy 'sequential', got '%s'", step.Compensation.Strategy)
	}

	// Validate retry policy
	if step.RetryPolicy == nil {
		t.Fatal("Expected retry policy, got nil")
	}
	if step.RetryPolicy.Type != "fixed_delay" {
		t.Errorf("Expected retry type 'fixed_delay', got '%s'", step.RetryPolicy.Type)
	}
}

// TestDurationUnmarshal tests Duration unmarshaling.
func TestDurationUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected time.Duration
	}{
		{
			name:     "seconds",
			yaml:     "timeout: 30s",
			expected: 30 * time.Second,
		},
		{
			name:     "minutes",
			yaml:     "timeout: 5m",
			expected: 5 * time.Minute,
		},
		{
			name:     "hours",
			yaml:     "timeout: 2h",
			expected: 2 * time.Hour,
		},
		{
			name:     "milliseconds",
			yaml:     "timeout: 500ms",
			expected: 500 * time.Millisecond,
		},
		{
			name:     "mixed",
			yaml:     "timeout: 1h30m",
			expected: 90 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config struct {
				Timeout Duration `yaml:"timeout"`
			}
			err := yaml.Unmarshal([]byte(tt.yaml), &config)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			if config.Timeout.ToDuration() != tt.expected {
				t.Errorf("Expected duration %v, got %v", tt.expected, config.Timeout.ToDuration())
			}
		})
	}
}

// TestDurationMarshal tests Duration marshaling.
func TestDurationMarshal(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: Duration(30 * time.Second),
			expected: "30s",
		},
		{
			name:     "minutes",
			duration: Duration(5 * time.Minute),
			expected: "5m0s",
		},
		{
			name:     "hours",
			duration: Duration(2 * time.Hour),
			expected: "2h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := struct {
				Timeout Duration `yaml:"timeout"`
			}{
				Timeout: tt.duration,
			}
			data, err := yaml.Marshal(&config)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			// Just check that it marshals successfully
			if len(data) == 0 {
				t.Error("Expected non-empty marshaled data")
			}
		})
	}
}

// TestStepTypes tests step type constants.
func TestStepTypes(t *testing.T) {
	tests := []struct {
		name     string
		stepType StepType
	}{
		{"service", StepTypeService},
		{"function", StepTypeFunction},
		{"http", StepTypeHTTP},
		{"grpc", StepTypeGRPC},
		{"message", StepTypeMessage},
		{"custom", StepTypeCustom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.stepType) != tt.name {
				t.Errorf("Expected step type '%s', got '%s'", tt.name, tt.stepType)
			}
		})
	}
}

// TestExecutionModes tests execution mode constants.
func TestExecutionModes(t *testing.T) {
	tests := []struct {
		name string
		mode ExecutionMode
	}{
		{"orchestration", ExecutionModeOrchestration},
		{"choreography", ExecutionModeChoreography},
		{"hybrid", ExecutionModeHybrid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.name {
				t.Errorf("Expected mode '%s', got '%s'", tt.name, tt.mode)
			}
		})
	}
}

// TestCompensationTypes tests compensation type constants.
func TestCompensationTypes(t *testing.T) {
	tests := []struct {
		name  string
		cType CompensationType
	}{
		{"automatic", CompensationTypeAutomatic},
		{"custom", CompensationTypeCustom},
		{"skip", CompensationTypeSkip},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.cType) != tt.name {
				t.Errorf("Expected compensation type '%s', got '%s'", tt.name, tt.cType)
			}
		})
	}
}

// TestCompensationStrategies tests compensation strategy constants.
func TestCompensationStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy CompensationStrategy
	}{
		{"sequential", CompensationStrategySequential},
		{"parallel", CompensationStrategyParallel},
		{"best_effort", CompensationStrategyBestEffort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.strategy) != tt.name {
				t.Errorf("Expected strategy '%s', got '%s'", tt.name, tt.strategy)
			}
		})
	}
}

// TestMessageActionUnmarshal tests unmarshaling message action configuration.
func TestMessageActionUnmarshal(t *testing.T) {
	yamlData := `
topic: test.topic
broker: default
payload:
  key: value
  count: 42
headers:
  X-Source: test
routing_key: test.routing.key
`

	var msg MessageConfig
	err := yaml.Unmarshal([]byte(yamlData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg.Topic != "test.topic" {
		t.Errorf("Expected topic 'test.topic', got '%s'", msg.Topic)
	}
	if msg.Broker != "default" {
		t.Errorf("Expected broker 'default', got '%s'", msg.Broker)
	}
	if msg.RoutingKey != "test.routing.key" {
		t.Errorf("Expected routing key 'test.routing.key', got '%s'", msg.RoutingKey)
	}
	if len(msg.Headers) != 1 {
		t.Errorf("Expected 1 header, got %d", len(msg.Headers))
	}
	if len(msg.Payload) != 2 {
		t.Errorf("Expected 2 payload fields, got %d", len(msg.Payload))
	}
}

// TestRetryPolicyUnmarshal tests unmarshaling retry policy configuration.
func TestRetryPolicyUnmarshal(t *testing.T) {
	yamlData := `
type: exponential_backoff
max_attempts: 5
initial_delay: 1s
max_delay: 30s
multiplier: 2.5
jitter: true
retryable_errors:
  - timeout
  - network
`

	var policy RetryPolicy
	err := yaml.Unmarshal([]byte(yamlData), &policy)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if policy.Type != "exponential_backoff" {
		t.Errorf("Expected type 'exponential_backoff', got '%s'", policy.Type)
	}
	if policy.MaxAttempts != 5 {
		t.Errorf("Expected max attempts 5, got %d", policy.MaxAttempts)
	}
	if policy.InitialDelay.ToDuration() != time.Second {
		t.Errorf("Expected initial delay 1s, got %v", policy.InitialDelay)
	}
	if policy.MaxDelay.ToDuration() != 30*time.Second {
		t.Errorf("Expected max delay 30s, got %v", policy.MaxDelay)
	}
	if policy.Multiplier != 2.5 {
		t.Errorf("Expected multiplier 2.5, got %f", policy.Multiplier)
	}
	if !policy.Jitter {
		t.Error("Expected jitter to be true")
	}
	if len(policy.RetryableErrors) != 2 {
		t.Errorf("Expected 2 retryable errors, got %d", len(policy.RetryableErrors))
	}
}

// TestConditionUnmarshal tests unmarshaling condition configuration.
func TestConditionUnmarshal(t *testing.T) {
	yamlData := `
expression: "$input.amount > 1000"
variables:
  threshold: 1000
  currency: USD
`

	var condition Condition
	err := yaml.Unmarshal([]byte(yamlData), &condition)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if condition.Expression != "$input.amount > 1000" {
		t.Errorf("Expected expression '$input.amount > 1000', got '%s'", condition.Expression)
	}
	if len(condition.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(condition.Variables))
	}
}

// TestComplexSagaUnmarshal tests unmarshaling a complex Saga with multiple features.
func TestComplexSagaUnmarshal(t *testing.T) {
	yamlData := `
saga:
  id: complex-saga
  name: Complex Saga
  timeout: 10m
  mode: hybrid
  tags:
    - complex
    - test

steps:
  - id: step-1
    name: Step 1
    type: service
    action:
      service:
        name: service-1
        method: POST
    timeout: 30s

  - id: step-2
    name: Step 2
    type: grpc
    action:
      service:
        name: service-2
        method: CallMethod
    dependencies:
      - step-1
    condition:
      expression: "$output.step-1.success == true"
    async: true
    timeout: 1m

  - id: step-3
    name: Step 3
    type: message
    action:
      message:
        topic: test.topic
        payload:
          data: test
    dependencies:
      - step-1
      - step-2
    on_success:
      notifications:
        - type: email
          target: test@example.com
          message: Success
    on_failure:
      notifications:
        - type: slack
          target: "#alerts"
          message: Failed
`

	var def SagaDefinition
	err := yaml.Unmarshal([]byte(yamlData), &def)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(def.Steps) != 3 {
		t.Fatalf("Expected 3 steps, got %d", len(def.Steps))
	}

	// Check step-2 has condition and async
	step2 := def.Steps[1]
	if step2.Condition == nil {
		t.Error("Expected step-2 to have condition")
	}
	if !step2.Async {
		t.Error("Expected step-2 to be async")
	}
	if len(step2.Dependencies) != 1 {
		t.Errorf("Expected step-2 to have 1 dependency, got %d", len(step2.Dependencies))
	}

	// Check step-3 has hooks
	step3 := def.Steps[2]
	if step3.OnSuccess == nil {
		t.Error("Expected step-3 to have on_success hook")
	}
	if step3.OnFailure == nil {
		t.Error("Expected step-3 to have on_failure hook")
	}
	if len(step3.Dependencies) != 2 {
		t.Errorf("Expected step-3 to have 2 dependencies, got %d", len(step3.Dependencies))
	}
}

// TestFunctionActionUnmarshal tests unmarshaling function action configuration.
func TestFunctionActionUnmarshal(t *testing.T) {
	yamlData := `
name: validateOrder
handler: pkg.handlers.ValidateOrder
parameters:
  order_id: "12345"
  strict: true
`

	var fn FunctionConfig
	err := yaml.Unmarshal([]byte(yamlData), &fn)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if fn.Name != "validateOrder" {
		t.Errorf("Expected name 'validateOrder', got '%s'", fn.Name)
	}
	if fn.Handler != "pkg.handlers.ValidateOrder" {
		t.Errorf("Expected handler 'pkg.handlers.ValidateOrder', got '%s'", fn.Handler)
	}
	if len(fn.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(fn.Parameters))
	}
}

// TestServiceActionUnmarshal tests unmarshaling service action configuration.
func TestServiceActionUnmarshal(t *testing.T) {
	yamlData := `
name: user-service
endpoint: http://user-service:8080
method: POST
path: /api/users
headers:
  Content-Type: application/json
  Authorization: Bearer token123
body:
  username: testuser
  email: test@example.com
timeout: 30s
`

	var svc ServiceConfig
	err := yaml.Unmarshal([]byte(yamlData), &svc)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if svc.Name != "user-service" {
		t.Errorf("Expected name 'user-service', got '%s'", svc.Name)
	}
	if svc.Endpoint != "http://user-service:8080" {
		t.Errorf("Expected endpoint 'http://user-service:8080', got '%s'", svc.Endpoint)
	}
	if svc.Method != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", svc.Method)
	}
	if svc.Path != "/api/users" {
		t.Errorf("Expected path '/api/users', got '%s'", svc.Path)
	}
	if len(svc.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(svc.Headers))
	}
	if len(svc.Body) != 2 {
		t.Errorf("Expected 2 body fields, got %d", len(svc.Body))
	}
	if svc.Timeout.ToDuration() != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", svc.Timeout)
	}
}

// TestGlobalCompensationUnmarshal tests unmarshaling global compensation configuration.
func TestGlobalCompensationUnmarshal(t *testing.T) {
	yamlData := `
strategy: parallel
timeout: 5m
max_attempts: 5
`

	var comp CompensationConfig
	err := yaml.Unmarshal([]byte(yamlData), &comp)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if comp.Strategy != CompensationStrategyParallel {
		t.Errorf("Expected strategy 'parallel', got '%s'", comp.Strategy)
	}
	if comp.Timeout.ToDuration() != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", comp.Timeout)
	}
	if comp.MaxAttempts != 5 {
		t.Errorf("Expected max attempts 5, got %d", comp.MaxAttempts)
	}
}

// TestNotificationConfigUnmarshal tests unmarshaling notification configuration.
func TestNotificationConfigUnmarshal(t *testing.T) {
	yamlData := `
type: slack
target: "#alerts"
message: "Alert: Step failed"
metadata:
  severity: high
  channel_id: "C12345"
`

	var notif NotificationConfig
	err := yaml.Unmarshal([]byte(yamlData), &notif)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if notif.Type != "slack" {
		t.Errorf("Expected type 'slack', got '%s'", notif.Type)
	}
	if notif.Target != "#alerts" {
		t.Errorf("Expected target '#alerts', got '%s'", notif.Target)
	}
	if notif.Message != "Alert: Step failed" {
		t.Errorf("Expected message 'Alert: Step failed', got '%s'", notif.Message)
	}
	if len(notif.Metadata) != 2 {
		t.Errorf("Expected 2 metadata fields, got %d", len(notif.Metadata))
	}
}

// TestDurationString tests Duration string conversion.
func TestDurationString(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
	}{
		{"30s", Duration(30 * time.Second)},
		{"5m", Duration(5 * time.Minute)},
		{"2h", Duration(2 * time.Hour)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.duration.String()
			if str == "" {
				t.Error("Expected non-empty string")
			}
		})
	}
}
