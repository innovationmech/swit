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
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// Assertion Functions
// ==========================

// AssertStepCount verifies that the saga has the expected number of steps.
func AssertStepCount(expected int) AssertionFunc {
	return func(result *TestResult) error {
		actual := len(result.StepStates)
		if actual != expected {
			return fmt.Errorf("expected %d steps, got %d", expected, actual)
		}
		return nil
	}
}

// AssertAllStepsCompleted verifies that all steps completed successfully.
func AssertAllStepsCompleted() AssertionFunc {
	return func(result *TestResult) error {
		for i, step := range result.StepStates {
			if step.State != saga.StepStateCompleted {
				return fmt.Errorf("step %d (%s) not completed: state=%s", i, step.Name, step.State)
			}
		}
		return nil
	}
}

// AssertStepCompleted verifies that a specific step completed successfully.
func AssertStepCompleted(stepIndex int) AssertionFunc {
	return func(result *TestResult) error {
		if stepIndex < 0 || stepIndex >= len(result.StepStates) {
			return fmt.Errorf("step index %d out of range", stepIndex)
		}
		step := result.StepStates[stepIndex]
		if step.State != saga.StepStateCompleted {
			return fmt.Errorf("step %d (%s) not completed: state=%s", stepIndex, step.Name, step.State)
		}
		return nil
	}
}

// AssertStepFailed verifies that a specific step failed.
func AssertStepFailed(stepIndex int) AssertionFunc {
	return func(result *TestResult) error {
		if stepIndex < 0 || stepIndex >= len(result.StepStates) {
			return fmt.Errorf("step index %d out of range", stepIndex)
		}
		step := result.StepStates[stepIndex]
		if step.State != saga.StepStateFailed {
			return fmt.Errorf("step %d (%s) not failed: state=%s", stepIndex, step.Name, step.State)
		}
		return nil
	}
}

// AssertStepState verifies that a specific step has the expected state.
func AssertStepState(stepIndex int, expectedState saga.StepStateEnum) AssertionFunc {
	return func(result *TestResult) error {
		if stepIndex < 0 || stepIndex >= len(result.StepStates) {
			return fmt.Errorf("step index %d out of range", stepIndex)
		}
		step := result.StepStates[stepIndex]
		if step.State != expectedState {
			return fmt.Errorf("step %d (%s) expected state=%s, got state=%s",
				stepIndex, step.Name, expectedState, step.State)
		}
		return nil
	}
}

// AssertExecutionSucceeded verifies that the saga execution succeeded.
func AssertExecutionSucceeded() AssertionFunc {
	return func(result *TestResult) error {
		if !result.Success {
			return fmt.Errorf("saga execution failed: %v", result.Error)
		}
		return nil
	}
}

// AssertExecutionFailed verifies that the saga execution failed.
func AssertExecutionFailed() AssertionFunc {
	return func(result *TestResult) error {
		if result.Success {
			return fmt.Errorf("saga execution succeeded, expected failure")
		}
		return nil
	}
}

// AssertError verifies that the execution error matches the expected error.
func AssertError(expectedError error) AssertionFunc {
	return func(result *TestResult) error {
		if result.Error == nil {
			return fmt.Errorf("expected error %v, got nil", expectedError)
		}
		if result.Error.Error() != expectedError.Error() {
			return fmt.Errorf("expected error %v, got %v", expectedError, result.Error)
		}
		return nil
	}
}

// AssertErrorContains verifies that the error message contains the expected substring.
func AssertErrorContains(substring string) AssertionFunc {
	return func(result *TestResult) error {
		if result.Error == nil {
			return fmt.Errorf("expected error containing %q, got nil", substring)
		}
		if !contains(result.Error.Error(), substring) {
			return fmt.Errorf("error %q does not contain %q", result.Error.Error(), substring)
		}
		return nil
	}
}

// AssertDuration verifies that the execution duration is within the expected range.
func AssertDuration(min, max time.Duration) AssertionFunc {
	return func(result *TestResult) error {
		if result.Duration < min {
			return fmt.Errorf("duration %v less than minimum %v", result.Duration, min)
		}
		if result.Duration > max {
			return fmt.Errorf("duration %v greater than maximum %v", result.Duration, max)
		}
		return nil
	}
}

// AssertDurationLessThan verifies that the execution duration is less than the maximum.
func AssertDurationLessThan(max time.Duration) AssertionFunc {
	return func(result *TestResult) error {
		if result.Duration > max {
			return fmt.Errorf("duration %v greater than maximum %v", result.Duration, max)
		}
		return nil
	}
}

// AssertNoCompensation verifies that no compensation was performed.
func AssertNoCompensation() AssertionFunc {
	return func(result *TestResult) error {
		for i, step := range result.StepStates {
			if step.CompensationState != nil && step.CompensationState.State == saga.CompensationStateCompleted {
				return fmt.Errorf("step %d (%s) was compensated", i, step.Name)
			}
		}
		return nil
	}
}

// AssertCompensationPerformed verifies that compensation was performed.
func AssertCompensationPerformed() AssertionFunc {
	return func(result *TestResult) error {
		compensated := false
		for _, step := range result.StepStates {
			if step.CompensationState != nil && step.CompensationState.State == saga.CompensationStateCompleted {
				compensated = true
				break
			}
		}
		if !compensated {
			return fmt.Errorf("no compensation was performed")
		}
		return nil
	}
}

// AssertStepCompensated verifies that a specific step was compensated.
func AssertStepCompensated(stepIndex int) AssertionFunc {
	return func(result *TestResult) error {
		if stepIndex < 0 || stepIndex >= len(result.StepStates) {
			return fmt.Errorf("step index %d out of range", stepIndex)
		}
		step := result.StepStates[stepIndex]
		if step.CompensationState == nil || step.CompensationState.State != saga.CompensationStateCompleted {
			return fmt.Errorf("step %d (%s) was not compensated", stepIndex, step.Name)
		}
		return nil
	}
}

// AssertEventPublished verifies that an event of the given type was published.
func AssertEventPublished(eventType saga.SagaEventType) AssertionFunc {
	return func(result *TestResult) error {
		for _, event := range result.Events {
			if event.Type == eventType {
				return nil
			}
		}
		return fmt.Errorf("event type %s was not published", eventType)
	}
}

// AssertEventCount verifies that the expected number of events was published.
func AssertEventCount(expected int) AssertionFunc {
	return func(result *TestResult) error {
		actual := len(result.Events)
		if actual != expected {
			return fmt.Errorf("expected %d events, got %d", expected, actual)
		}
		return nil
	}
}

// AssertEventCountByType verifies that the expected number of events of a type was published.
func AssertEventCountByType(eventType saga.SagaEventType, expected int) AssertionFunc {
	return func(result *TestResult) error {
		count := 0
		for _, event := range result.Events {
			if event.Type == eventType {
				count++
			}
		}
		if count != expected {
			return fmt.Errorf("expected %d events of type %s, got %d", expected, eventType, count)
		}
		return nil
	}
}

// AssertStorageSaveCount verifies the number of times SaveSaga was called.
func AssertStorageSaveCount(expected int) AssertionFunc {
	return func(result *TestResult) error {
		actual := result.Storage.SaveSagaCalls
		if actual != expected {
			return fmt.Errorf("expected %d SaveSaga calls, got %d", expected, actual)
		}
		return nil
	}
}

// AssertStorageGetCount verifies the number of times GetSaga was called.
func AssertStorageGetCount(expected int) AssertionFunc {
	return func(result *TestResult) error {
		actual := result.Storage.GetSagaCalls
		if actual != expected {
			return fmt.Errorf("expected %d GetSaga calls, got %d", expected, actual)
		}
		return nil
	}
}

// AssertPublisherCalls verifies the number of times PublishEvent was called.
func AssertPublisherCalls(expected int) AssertionFunc {
	return func(result *TestResult) error {
		actual := result.Publisher.PublishEventCalls
		if actual != expected {
			return fmt.Errorf("expected %d PublishEvent calls, got %d", expected, actual)
		}
		return nil
	}
}

// AssertRetryAttempts verifies that a step was retried the expected number of times.
func AssertRetryAttempts(stepIndex int, expectedAttempts int) AssertionFunc {
	return func(result *TestResult) error {
		if stepIndex < 0 || stepIndex >= len(result.StepStates) {
			return fmt.Errorf("step index %d out of range", stepIndex)
		}
		step := result.StepStates[stepIndex]
		if step.Attempts != expectedAttempts {
			return fmt.Errorf("step %d (%s) expected %d attempts, got %d",
				stepIndex, step.Name, expectedAttempts, step.Attempts)
		}
		return nil
	}
}

// AssertStepData verifies that a step has the expected output data.
func AssertStepData(stepIndex int, expectedData interface{}) AssertionFunc {
	return func(result *TestResult) error {
		if stepIndex < 0 || stepIndex >= len(result.StepStates) {
			return fmt.Errorf("step index %d out of range", stepIndex)
		}
		step := result.StepStates[stepIndex]
		if !deepEqual(step.OutputData, expectedData) {
			return fmt.Errorf("step %d (%s) data mismatch: expected %v, got %v",
				stepIndex, step.Name, expectedData, step.OutputData)
		}
		return nil
	}
}

// AssertMetadata verifies that the saga has the expected metadata.
func AssertMetadata(key string, expectedValue interface{}) AssertionFunc {
	return func(result *TestResult) error {
		metadata := result.SagaInstance.GetMetadata()
		actualValue, ok := metadata[key]
		if !ok {
			return fmt.Errorf("metadata key %q not found", key)
		}
		if !deepEqual(actualValue, expectedValue) {
			return fmt.Errorf("metadata %q mismatch: expected %v, got %v",
				key, expectedValue, actualValue)
		}
		return nil
	}
}

// AssertMetadataExists verifies that the saga has the specified metadata key.
func AssertMetadataExists(key string) AssertionFunc {
	return func(result *TestResult) error {
		metadata := result.SagaInstance.GetMetadata()
		if _, ok := metadata[key]; !ok {
			return fmt.Errorf("metadata key %q not found", key)
		}
		return nil
	}
}

// ==========================
// Composite Assertions
// ==========================

// And combines multiple assertions - all must pass.
func And(assertions ...AssertionFunc) AssertionFunc {
	return func(result *TestResult) error {
		for i, assertion := range assertions {
			if err := assertion(result); err != nil {
				return fmt.Errorf("assertion %d failed: %w", i+1, err)
			}
		}
		return nil
	}
}

// Or combines multiple assertions - at least one must pass.
func Or(assertions ...AssertionFunc) AssertionFunc {
	return func(result *TestResult) error {
		var errors []error
		for _, assertion := range assertions {
			if err := assertion(result); err == nil {
				return nil
			} else {
				errors = append(errors, err)
			}
		}
		return fmt.Errorf("all %d assertions failed: %v", len(assertions), errors)
	}
}

// Not negates an assertion.
func Not(assertion AssertionFunc) AssertionFunc {
	return func(result *TestResult) error {
		if err := assertion(result); err == nil {
			return fmt.Errorf("assertion passed but was expected to fail")
		}
		return nil
	}
}

// ==========================
// Helper Functions
// ==========================

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// deepEqual compares two values for equality.
// This is a simplified version - for production use, consider reflect.DeepEqual.
func deepEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// ==========================
// Custom Assertion Helpers
// ==========================

// AssertCustom creates a custom assertion with a user-defined check function.
func AssertCustom(name string, checkFunc func(*TestResult) bool, errorMsg string) AssertionFunc {
	return func(result *TestResult) error {
		if !checkFunc(result) {
			return fmt.Errorf("%s: %s", name, errorMsg)
		}
		return nil
	}
}

// AssertFunc creates an assertion from a simple function.
func AssertFunc(fn func(*TestResult) error) AssertionFunc {
	return fn
}
