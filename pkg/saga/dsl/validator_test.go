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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	assert.NotNil(t, v)
	assert.NotNil(t, v.structValidator)
	assert.NotEmpty(t, v.customRules)
}

func TestValidator_ValidateNilDefinition(t *testing.T) {
	v := NewValidator()
	err := v.Validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga definition is nil")
}

func TestValidator_ValidateValidDefinition(t *testing.T) {
	v := NewValidator()
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:      "test-saga",
			Name:    "Test Saga",
			Version: "1.0.0",
			Timeout: Duration(5 * time.Minute),
			Mode:    ExecutionModeOrchestration,
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
						Path:   "/api/test",
					},
				},
				Timeout: Duration(30 * time.Second),
			},
		},
	}

	err := v.Validate(def)
	assert.NoError(t, err)
}

func TestValidateSagaConfig_EmptyID(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga.id")
	assert.Contains(t, err.Error(), "required")
}

func TestValidateSagaConfig_InvalidIDPattern(t *testing.T) {
	tests := []struct {
		name    string
		sagaID  string
		wantErr bool
	}{
		{"valid alphanumeric", "test-saga-123", false},
		{"valid with underscore", "test_saga", false},
		{"invalid starts with dash", "-test-saga", true},
		{"invalid ends with dash", "test-saga-", true},
		{"invalid special char", "test@saga", true},
		{"invalid space", "test saga", true},
		{"single char", "t", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &SagaDefinition{
				Saga: SagaConfig{
					ID:   tt.sagaID,
					Name: "Test Saga",
				},
				Steps: []StepConfig{
					{
						ID:   "step1",
						Name: "Step 1",
						Type: StepTypeService,
						Action: ActionConfig{
							Service: &ServiceConfig{
								Name:   "test-service",
								Method: "POST",
							},
						},
					},
				},
			}

			v := NewValidator()
			err := v.Validate(def)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "saga.id")
			} else {
				if err != nil && strings.Contains(err.Error(), "saga.id") {
					t.Errorf("expected no error for saga.id, got: %v", err)
				}
			}
		})
	}
}

func TestValidateSagaConfig_EmptyName(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga.name")
}

func TestValidateSagaConfig_NameTooLong(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: strings.Repeat("a", 101),
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga.name")
	assert.Contains(t, err.Error(), "100 characters")
}

func TestValidateSagaConfig_InvalidMode(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
			Mode: ExecutionMode("invalid"),
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga.mode")
	assert.Contains(t, err.Error(), "invalid execution mode")
}

func TestValidateSteps_NoSteps(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "steps")
	assert.Contains(t, err.Error(), "at least one step")
}

func TestValidateSteps_EmptyStepID(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "steps[0].id")
	assert.Contains(t, err.Error(), "required")
}

func TestValidateSteps_InvalidStepIDPattern(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "-invalid-step",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "steps[0].id")
	assert.Contains(t, err.Error(), "alphanumeric")
}

func TestValidateSteps_DuplicateStepID(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
			{
				ID:   "step1",
				Name: "Step 1 Duplicate",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate step ID")
}

func TestValidateSteps_EmptyStepName(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "steps[0].name")
}

func TestValidateSteps_InvalidStepType(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepType("invalid"),
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "steps[0].type")
	assert.Contains(t, err.Error(), "invalid step type")
}

func TestValidateStepAction_ServiceTypeMissingAction(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:     "step1",
				Name:   "Step 1",
				Type:   StepTypeService,
				Action: ActionConfig{},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service action is required")
}

func TestValidateStepAction_FunctionTypeMissingAction(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:     "step1",
				Name:   "Step 1",
				Type:   StepTypeFunction,
				Action: ActionConfig{},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "function action is required")
}

func TestValidateStepAction_MessageTypeMissingAction(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:     "step1",
				Name:   "Step 1",
				Type:   StepTypeMessage,
				Action: ActionConfig{},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message action is required")
}

func TestValidateStepAction_CustomTypeMissingAction(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:     "step1",
				Name:   "Step 1",
				Type:   StepTypeCustom,
				Action: ActionConfig{},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom action is required")
}

func TestValidateServiceConfig_EmptyName(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service name is required")
}

func TestValidateServiceConfig_InvalidNamePattern(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "InvalidService",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service name must contain only lowercase")
}

func TestValidateServiceConfig_EmptyMethod(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service method is required")
}

func TestValidateFunctionConfig_EmptyName(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeFunction,
				Action: ActionConfig{
					Function: &FunctionConfig{
						Name:    "",
						Handler: "handler",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "function name is required")
}

func TestValidateFunctionConfig_EmptyHandler(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeFunction,
				Action: ActionConfig{
					Function: &FunctionConfig{
						Name:    "test-function",
						Handler: "",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "function handler is required")
}

func TestValidateMessageConfig_EmptyTopic(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeMessage,
				Action: ActionConfig{
					Message: &MessageConfig{
						Topic: "",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message topic is required")
}

func TestValidateStepDependencies_NonExistentStep(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
			{
				ID:   "step2",
				Name: "Step 2",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Dependencies: []string{"non-existent-step"},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "depends on non-existent step")
	assert.Contains(t, err.Error(), "non-existent-step")
}

func TestValidateStepDependencies_CircularDependency(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Dependencies: []string{"step3"},
			},
			{
				ID:   "step2",
				Name: "Step 2",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Dependencies: []string{"step1"},
			},
			{
				ID:   "step3",
				Name: "Step 3",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Dependencies: []string{"step2"},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestValidateTimeouts_SagaTimeoutTooShort(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:      "test-saga",
			Name:    "Test Saga",
			Timeout: Duration(500 * time.Millisecond),
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga.timeout")
	assert.Contains(t, err.Error(), "at least 1 second")
}

func TestValidateTimeouts_SagaTimeoutTooLong(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:      "test-saga",
			Name:    "Test Saga",
			Timeout: Duration(25 * time.Hour),
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga.timeout")
	assert.Contains(t, err.Error(), "24 hours")
}

func TestValidateTimeouts_StepTimeoutTooShort(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:      "step1",
				Name:    "Step 1",
				Type:    StepTypeService,
				Timeout: Duration(500 * time.Millisecond),
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "steps[0].timeout")
	assert.Contains(t, err.Error(), "at least 1 second")
}

func TestValidateTimeouts_StepTimeoutExceedsSagaTimeout(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:      "test-saga",
			Name:    "Test Saga",
			Timeout: Duration(1 * time.Minute),
		},
		Steps: []StepConfig{
			{
				ID:      "step1",
				Name:    "Step 1",
				Type:    StepTypeService,
				Timeout: Duration(2 * time.Minute),
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "steps[0].timeout")
	assert.Contains(t, err.Error(), "exceeds saga timeout")
}

func TestValidateTimeouts_ServiceTimeoutTooShort(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:    "test-service",
						Method:  "POST",
						Timeout: Duration(50 * time.Millisecond),
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service.timeout")
	assert.Contains(t, err.Error(), "at least 100 milliseconds")
}

func TestValidateTimeouts_ServiceTimeoutExceedsStepTimeout(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:      "step1",
				Name:    "Step 1",
				Type:    StepTypeService,
				Timeout: Duration(10 * time.Second),
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:    "test-service",
						Method:  "POST",
						Timeout: Duration(20 * time.Second),
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service.timeout")
	assert.Contains(t, err.Error(), "exceeds step timeout")
}

func TestValidateRetryPolicy_InvalidType(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		GlobalRetryPolicy: &RetryPolicy{
			Type:        "invalid_type",
			MaxAttempts: 3,
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry")
	assert.Contains(t, err.Error(), "invalid retry type")
}

func TestValidateRetryPolicy_MaxAttemptsTooSmall(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		GlobalRetryPolicy: &RetryPolicy{
			Type:        "exponential_backoff",
			MaxAttempts: 0,
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_attempts")
	assert.Contains(t, err.Error(), "at least 1")
}

func TestValidateRetryPolicy_MaxAttemptsTooLarge(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		GlobalRetryPolicy: &RetryPolicy{
			Type:        "exponential_backoff",
			MaxAttempts: 101,
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_attempts")
	assert.Contains(t, err.Error(), "not exceed 100")
}

func TestValidateRetryPolicy_ExponentialBackoff_InvalidInitialDelay(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		GlobalRetryPolicy: &RetryPolicy{
			Type:         "exponential_backoff",
			MaxAttempts:  3,
			InitialDelay: Duration(5 * time.Millisecond),
			Multiplier:   2.0,
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "initial_delay")
	assert.Contains(t, err.Error(), "at least 10 milliseconds")
}

func TestValidateRetryPolicy_ExponentialBackoff_InvalidMultiplier(t *testing.T) {
	tests := []struct {
		name       string
		multiplier float64
		wantErr    bool
		errMsg     string
	}{
		{"too small", 0.5, true, "at least 1"},
		{"too large", 11.0, true, "not exceed 10"},
		{"valid", 2.0, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &SagaDefinition{
				Saga: SagaConfig{
					ID:   "test-saga",
					Name: "Test Saga",
				},
				GlobalRetryPolicy: &RetryPolicy{
					Type:         "exponential_backoff",
					MaxAttempts:  3,
					InitialDelay: Duration(100 * time.Millisecond),
					Multiplier:   tt.multiplier,
				},
				Steps: []StepConfig{
					{
						ID:   "step1",
						Name: "Step 1",
						Type: StepTypeService,
						Action: ActionConfig{
							Service: &ServiceConfig{
								Name:   "test-service",
								Method: "POST",
							},
						},
					},
				},
			}

			v := NewValidator()
			err := v.Validate(def)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "multiplier")
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				if err != nil && strings.Contains(err.Error(), "multiplier") {
					t.Errorf("expected no error for multiplier, got: %v", err)
				}
			}
		})
	}
}

func TestValidateRetryPolicy_ExponentialBackoff_MaxDelayLessThanInitial(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		GlobalRetryPolicy: &RetryPolicy{
			Type:         "exponential_backoff",
			MaxAttempts:  3,
			InitialDelay: Duration(1 * time.Second),
			MaxDelay:     Duration(500 * time.Millisecond),
			Multiplier:   2.0,
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max_delay")
	assert.Contains(t, err.Error(), "greater than initial delay")
}

func TestValidateCompensation_CustomWithoutAction(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Compensation: &CompensationAction{
					Type: CompensationTypeCustom,
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compensation")
	assert.Contains(t, err.Error(), "custom compensation requires an action")
}

func TestValidateCompensation_SkipWithAction(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Compensation: &CompensationAction{
					Type: CompensationTypeSkip,
					Action: &ActionConfig{
						Service: &ServiceConfig{
							Name:   "test-service",
							Method: "POST",
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compensation")
	assert.Contains(t, err.Error(), "should not have an action")
}

func TestValidateCompensation_InvalidType(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Compensation: &CompensationAction{
					Type: CompensationType("invalid"),
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compensation")
	assert.Contains(t, err.Error(), "invalid compensation type")
}

func TestValidateCompensation_InvalidStrategy(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Compensation: &CompensationAction{
					Type:     CompensationTypeCustom,
					Strategy: CompensationStrategy("invalid"),
					Action: &ActionConfig{
						Service: &ServiceConfig{
							Name:   "test-service",
							Method: "POST",
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compensation")
	assert.Contains(t, err.Error(), "invalid compensation strategy")
}

func TestValidateCompensation_InvalidOnFailureAction(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
				Compensation: &CompensationAction{
					Type: CompensationTypeCustom,
					Action: &ActionConfig{
						Service: &ServiceConfig{
							Name:   "test-service",
							Method: "POST",
						},
					},
					OnFailure: &CompensationFailurePolicy{
						Action: "invalid_action",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "on_failure.action")
	assert.Contains(t, err.Error(), "invalid on_failure action")
}

func TestValidateGlobalCompensation_InvalidStrategy(t *testing.T) {
	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		GlobalCompensation: &CompensationConfig{
			Strategy: CompensationStrategy("invalid"),
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	v := NewValidator()
	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global_compensation")
	assert.Contains(t, err.Error(), "invalid compensation strategy")
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   string
	}{
		{
			name:   "no errors",
			errors: ValidationErrors{},
			want:   "no validation errors",
		},
		{
			name: "single error",
			errors: ValidationErrors{
				{Field: "saga.id", Message: "field is required"},
			},
			want: "validation error at 'saga.id': field is required",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "saga.id", Message: "field is required"},
				{Field: "saga.name", Message: "field is required"},
			},
			want: "validation failed with 2 error(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errors.Error()
			assert.Contains(t, got, tt.want)
		})
	}
}

func TestCustomValidationRule(t *testing.T) {
	customRule := func(def *SagaDefinition) ValidationErrors {
		if def.Saga.Version == "" {
			return ValidationErrors{
				{
					Field:   "saga.version",
					Message: "version is recommended",
					Rule:    "custom",
				},
			}
		}
		return nil
	}

	v := NewValidator()
	v.RegisterRule(customRule)

	def := &SagaDefinition{
		Saga: SagaConfig{
			ID:   "test-saga",
			Name: "Test Saga",
		},
		Steps: []StepConfig{
			{
				ID:   "step1",
				Name: "Step 1",
				Type: StepTypeService,
				Action: ActionConfig{
					Service: &ServiceConfig{
						Name:   "test-service",
						Method: "POST",
					},
				},
			},
		},
	}

	err := v.Validate(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saga.version")
	assert.Contains(t, err.Error(), "version is recommended")
}

