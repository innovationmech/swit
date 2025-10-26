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

package testing

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/retry"
)

// ==========================
// SagaTestBuilder
// ==========================

// SagaTestBuilder provides a fluent API for building Saga test scenarios.
type SagaTestBuilder struct {
	definition *MockSagaDefinition
	steps      []*MockSagaStep
	metadata   map[string]interface{}
}

// NewSagaTestBuilder creates a new saga test builder.
func NewSagaTestBuilder(sagaID, name string) *SagaTestBuilder {
	return &SagaTestBuilder{
		definition: NewMockSagaDefinition(sagaID, name),
		steps:      make([]*MockSagaStep, 0),
		metadata:   make(map[string]interface{}),
	}
}

// WithDescription sets the saga description.
func (b *SagaTestBuilder) WithDescription(desc string) *SagaTestBuilder {
	b.definition.DescriptionValue = desc
	return b
}

// WithTimeout sets the saga timeout.
func (b *SagaTestBuilder) WithTimeout(timeout time.Duration) *SagaTestBuilder {
	b.definition.TimeoutValue = timeout
	return b
}

// WithRetryPolicy sets the saga retry policy.
func (b *SagaTestBuilder) WithRetryPolicy(policy saga.RetryPolicy) *SagaTestBuilder {
	b.definition.RetryPolicyValue = policy
	return b
}

// WithCompensationStrategy sets the compensation strategy.
func (b *SagaTestBuilder) WithCompensationStrategy(strategy saga.CompensationStrategy) *SagaTestBuilder {
	b.definition.CompensationStrategyValue = strategy
	return b
}

// WithMetadata adds metadata to the saga.
func (b *SagaTestBuilder) WithMetadata(key string, value interface{}) *SagaTestBuilder {
	b.metadata[key] = value
	return b
}

// AddStep adds a step to the saga with default behavior.
func (b *SagaTestBuilder) AddStep(stepID, stepName string) *SagaTestBuilder {
	step := NewMockSagaStep(stepID, stepName)
	b.steps = append(b.steps, step)
	b.definition.AddStep(step)
	return b
}

// AddStepWithExecute adds a step with custom execute function.
func (b *SagaTestBuilder) AddStepWithExecute(
	stepID, stepName string,
	executeFunc func(ctx context.Context, data interface{}) (interface{}, error),
) *SagaTestBuilder {
	step := NewMockSagaStep(stepID, stepName)
	step.ExecuteFunc = executeFunc
	b.steps = append(b.steps, step)
	b.definition.AddStep(step)
	return b
}

// AddStepWithCompensate adds a step with custom compensate function.
func (b *SagaTestBuilder) AddStepWithCompensate(
	stepID, stepName string,
	executeFunc func(ctx context.Context, data interface{}) (interface{}, error),
	compensateFunc func(ctx context.Context, data interface{}) error,
) *SagaTestBuilder {
	step := NewMockSagaStep(stepID, stepName)
	step.ExecuteFunc = executeFunc
	step.CompensateFunc = compensateFunc
	b.steps = append(b.steps, step)
	b.definition.AddStep(step)
	return b
}

// AddFailingStep adds a step that will fail with the given error.
func (b *SagaTestBuilder) AddFailingStep(stepID, stepName string, err error) *SagaTestBuilder {
	step := NewMockSagaStep(stepID, stepName)
	step.ExecuteError = err
	b.steps = append(b.steps, step)
	b.definition.AddStep(step)
	return b
}

// AddSlowStep adds a step with a simulated delay.
func (b *SagaTestBuilder) AddSlowStep(stepID, stepName string, delay time.Duration) *SagaTestBuilder {
	step := NewMockSagaStep(stepID, stepName)
	step.ExecuteFunc = func(ctx context.Context, data interface{}) (interface{}, error) {
		select {
		case <-time.After(delay):
			return data, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	b.steps = append(b.steps, step)
	b.definition.AddStep(step)
	return b
}

// GetStep returns the step at the given index for further configuration.
func (b *SagaTestBuilder) GetStep(index int) *MockSagaStep {
	if index < 0 || index >= len(b.steps) {
		return nil
	}
	return b.steps[index]
}

// Build returns the constructed saga definition.
func (b *SagaTestBuilder) Build() *MockSagaDefinition {
	b.definition.MetadataValue = b.metadata
	return b.definition
}

// ==========================
// ScenarioBuilder
// ==========================

// ScenarioBuilder provides a fluent API for building test scenarios.
type ScenarioBuilder struct {
	name        string
	description string
	saga        *MockSagaDefinition
	storage     *MockStateStorage
	publisher   *MockEventPublisher
	inputData   interface{}
	assertions  []AssertionFunc
}

// NewScenarioBuilder creates a new scenario builder.
func NewScenarioBuilder(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		name:       name,
		storage:    NewMockStateStorage(),
		publisher:  NewMockEventPublisher(),
		assertions: make([]AssertionFunc, 0),
	}
}

// WithDescription sets the scenario description.
func (s *ScenarioBuilder) WithDescription(desc string) *ScenarioBuilder {
	s.description = desc
	return s
}

// WithSaga sets the saga definition for this scenario.
func (s *ScenarioBuilder) WithSaga(definition *MockSagaDefinition) *ScenarioBuilder {
	s.saga = definition
	return s
}

// WithStorage sets the storage for this scenario.
func (s *ScenarioBuilder) WithStorage(storage *MockStateStorage) *ScenarioBuilder {
	s.storage = storage
	return s
}

// WithPublisher sets the event publisher for this scenario.
func (s *ScenarioBuilder) WithPublisher(publisher *MockEventPublisher) *ScenarioBuilder {
	s.publisher = publisher
	return s
}

// WithInputData sets the input data for the saga.
func (s *ScenarioBuilder) WithInputData(data interface{}) *ScenarioBuilder {
	s.inputData = data
	return s
}

// AddAssertion adds an assertion function to verify the scenario results.
func (s *ScenarioBuilder) AddAssertion(fn AssertionFunc) *ScenarioBuilder {
	s.assertions = append(s.assertions, fn)
	return s
}

// Build returns the constructed test scenario.
func (s *ScenarioBuilder) Build() *TestScenario {
	return &TestScenario{
		Name:        s.name,
		Description: s.description,
		Saga:        s.saga,
		Storage:     s.storage,
		Publisher:   s.publisher,
		InputData:   s.inputData,
		Assertions:  s.assertions,
	}
}

// ==========================
// TestScenario
// ==========================

// TestScenario represents a complete test scenario with saga, storage, and assertions.
type TestScenario struct {
	Name        string
	Description string
	Saga        *MockSagaDefinition
	Storage     *MockStateStorage
	Publisher   *MockEventPublisher
	InputData   interface{}
	Assertions  []AssertionFunc
}

// AssertionFunc is a function that performs assertions on a test result.
type AssertionFunc func(result *TestResult) error

// TestResult contains the results of a test scenario execution.
type TestResult struct {
	Scenario     *TestScenario
	Success      bool
	Error        error
	Duration     time.Duration
	SagaInstance saga.SagaInstance
	Storage      *MockStateStorage
	Publisher    *MockEventPublisher
	StepStates   []*saga.StepState
	Events       []*saga.SagaEvent
}

// ==========================
// Predefined Scenarios
// ==========================

// SuccessfulSagaScenario creates a scenario with a simple successful saga.
func SuccessfulSagaScenario() *TestScenario {
	saga := NewSagaTestBuilder("success-saga", "Successful Saga").
		WithDescription("A simple saga that completes successfully").
		WithTimeout(1*time.Minute).
		AddStep("step1", "First Step").
		AddStep("step2", "Second Step").
		AddStep("step3", "Third Step").
		Build()

	return NewScenarioBuilder("Successful Saga").
		WithDescription("Test a saga that completes all steps successfully").
		WithSaga(saga).
		WithInputData(map[string]interface{}{"test": "data"}).
		AddAssertion(AssertStepCount(3)).
		AddAssertion(AssertAllStepsCompleted()).
		AddAssertion(AssertNoCompensation()).
		Build()
}

// FailingStepScenario creates a scenario where a step fails.
func FailingStepScenario() *TestScenario {
	saga := NewSagaTestBuilder("failing-saga", "Failing Saga").
		WithDescription("A saga with a failing step").
		WithTimeout(1*time.Minute).
		AddStep("step1", "First Step").
		AddFailingStep("step2", "Failing Step", fmt.Errorf("simulated failure")).
		AddStep("step3", "Third Step").
		Build()

	return NewScenarioBuilder("Failing Step").
		WithDescription("Test a saga where the second step fails").
		WithSaga(saga).
		WithInputData(map[string]interface{}{"test": "data"}).
		AddAssertion(AssertExecutionFailed()).
		AddAssertion(AssertStepFailed(1)). // step2 at index 1
		Build()
}

// CompensationScenario creates a scenario that triggers compensation.
func CompensationScenario() *TestScenario {
	compensationCalled := make(map[string]bool)

	saga := NewSagaTestBuilder("compensation-saga", "Compensation Saga").
		WithDescription("A saga that requires compensation").
		WithTimeout(1*time.Minute).
		AddStepWithCompensate("step1", "First Step",
			func(ctx context.Context, data interface{}) (interface{}, error) {
				return data, nil
			},
			func(ctx context.Context, data interface{}) error {
				compensationCalled["step1"] = true
				return nil
			}).
		AddStepWithCompensate("step2", "Second Step",
			func(ctx context.Context, data interface{}) (interface{}, error) {
				return data, nil
			},
			func(ctx context.Context, data interface{}) error {
				compensationCalled["step2"] = true
				return nil
			}).
		AddFailingStep("step3", "Failing Step", fmt.Errorf("trigger compensation")).
		Build()

	return NewScenarioBuilder("Compensation").
		WithDescription("Test compensation execution after a failure").
		WithSaga(saga).
		WithInputData(map[string]interface{}{"test": "data"}).
		AddAssertion(AssertExecutionFailed()).
		Build()
}

// TimeoutScenario creates a scenario that triggers a timeout.
func TimeoutScenario() *TestScenario {
	saga := NewSagaTestBuilder("timeout-saga", "Timeout Saga").
		WithDescription("A saga that times out").
		WithTimeout(100*time.Millisecond).
		AddStep("step1", "First Step").
		AddSlowStep("step2", "Slow Step", 500*time.Millisecond).
		AddStep("step3", "Third Step").
		Build()

	return NewScenarioBuilder("Timeout").
		WithDescription("Test saga timeout behavior").
		WithSaga(saga).
		WithInputData(map[string]interface{}{"test": "data"}).
		AddAssertion(AssertExecutionFailed()).
		Build()
}

// RetryScenario creates a scenario with retryable failures.
func RetryScenario() *TestScenario {
	attemptCount := 0

	retryConfig := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	saga := NewSagaTestBuilder("retry-saga", "Retry Saga").
		WithDescription("A saga with retryable failures").
		WithTimeout(1*time.Minute).
		WithRetryPolicy(retry.NewExponentialBackoffPolicy(retryConfig, 2.0, 0.1)).
		AddStep("step1", "First Step").
		AddStepWithExecute("step2", "Retryable Step",
			func(ctx context.Context, data interface{}) (interface{}, error) {
				attemptCount++
				if attemptCount < 3 {
					return nil, fmt.Errorf("temporary failure")
				}
				return data, nil
			}).
		AddStep("step3", "Third Step").
		Build()

	return NewScenarioBuilder("Retry").
		WithDescription("Test retry behavior with temporary failures").
		WithSaga(saga).
		WithInputData(map[string]interface{}{"test": "data"}).
		AddAssertion(AssertAllStepsCompleted()).
		Build()
}

// ParallelStepsScenario creates a scenario with multiple sagas running in parallel.
func ParallelStepsScenario(sagaCount int) *TestScenario {
	saga := NewSagaTestBuilder("parallel-saga", "Parallel Saga").
		WithDescription("A saga for parallel execution testing").
		WithTimeout(1*time.Minute).
		AddStep("step1", "First Step").
		AddStep("step2", "Second Step").
		AddStep("step3", "Third Step").
		Build()

	return NewScenarioBuilder("Parallel Execution").
		WithDescription(fmt.Sprintf("Test %d sagas running in parallel", sagaCount)).
		WithSaga(saga).
		WithInputData(map[string]interface{}{"test": "data"}).
		AddAssertion(AssertAllStepsCompleted()).
		Build()
}

// ComplexSagaScenario creates a complex saga with multiple steps and conditions.
func ComplexSagaScenario() *TestScenario {
	sagaDef := NewSagaTestBuilder("complex-saga", "Complex Saga").
		WithDescription("A complex saga with multiple steps").
		WithTimeout(5*time.Minute).
		AddStep("validate", "Validate Input").
		AddStep("reserve-inventory", "Reserve Inventory").
		AddStep("process-payment", "Process Payment").
		AddStep("create-order", "Create Order").
		AddStep("send-notification", "Send Notification").
		AddStep("update-analytics", "Update Analytics").
		Build()

	return NewScenarioBuilder("Complex Saga").
		WithDescription("Test a complex multi-step saga").
		WithSaga(sagaDef).
		WithInputData(map[string]interface{}{
			"order_id": "ORD-12345",
			"user_id":  "USER-789",
			"items":    []string{"ITEM-1", "ITEM-2"},
			"amount":   199.99,
			"payment":  "credit_card",
		}).
		AddAssertion(AssertStepCount(6)).
		AddAssertion(AssertAllStepsCompleted()).
		AddAssertion(AssertEventPublished(saga.EventSagaStarted)).
		AddAssertion(AssertEventPublished(saga.EventSagaCompleted)).
		Build()
}
