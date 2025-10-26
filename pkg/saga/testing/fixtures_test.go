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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

func TestFixtureLoader_Load(t *testing.T) {
	loader := GetDefaultFixtureLoader()

	tests := []struct {
		name        string
		fixtureID   string
		expectError bool
		expectType  FixtureType
	}{
		{
			name:        "Load successful saga",
			fixtureID:   "successful-saga",
			expectError: false,
			expectType:  FixtureTypeSagaDefinition,
		},
		{
			name:        "Load failing saga",
			fixtureID:   "failing-saga",
			expectError: false,
			expectType:  FixtureTypeSagaDefinition,
		},
		{
			name:        "Load compensation saga",
			fixtureID:   "compensation-saga",
			expectError: false,
			expectType:  FixtureTypeSagaDefinition,
		},
		{
			name:        "Load timeout saga",
			fixtureID:   "timeout-saga",
			expectError: false,
			expectType:  FixtureTypeSagaDefinition,
		},
		{
			name:        "Load running instance",
			fixtureID:   "saga-instance-running",
			expectError: false,
			expectType:  FixtureTypeSagaInstance,
		},
		{
			name:        "Load completed instance",
			fixtureID:   "saga-instance-completed",
			expectError: false,
			expectType:  FixtureTypeSagaInstance,
		},
		{
			name:        "Load failed instance",
			fixtureID:   "saga-instance-failed",
			expectError: false,
			expectType:  FixtureTypeSagaInstance,
		},
		{
			name:        "Load compensating instance",
			fixtureID:   "saga-instance-compensating",
			expectError: false,
			expectType:  FixtureTypeSagaInstance,
		},
		{
			name:        "Load non-existent fixture",
			fixtureID:   "non-existent-fixture",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture, err := loader.Load(tt.fixtureID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if fixture == nil {
				t.Fatal("Expected fixture but got nil")
			}

			if fixture.ID != tt.fixtureID {
				t.Errorf("Expected ID %s, got %s", tt.fixtureID, fixture.ID)
			}

			if fixture.Type != tt.expectType {
				t.Errorf("Expected type %s, got %s", tt.expectType, fixture.Type)
			}

			if fixture.Name == "" {
				t.Error("Fixture name should not be empty")
			}

			if fixture.Data == nil {
				t.Error("Fixture data should not be nil")
			}
		})
	}
}

func TestFixtureLoader_LoadByType(t *testing.T) {
	loader := GetDefaultFixtureLoader()

	tests := []struct {
		name         string
		fixtureType  FixtureType
		minExpected  int
	}{
		{
			name:        "Load saga definitions",
			fixtureType: FixtureTypeSagaDefinition,
			minExpected: 4, // successful, failing, compensation, timeout
		},
		{
			name:        "Load saga instances",
			fixtureType: FixtureTypeSagaInstance,
			minExpected: 4, // running, completed, failed, compensating
		},
		{
			name:        "Load saga events",
			fixtureType: FixtureTypeSagaEvent,
			minExpected: 5, // started, step-completed, step-failed, compensation-started, completed
		},
		{
			name:        "Load configs",
			fixtureType: FixtureTypeConfig,
			minExpected: 2, // quick-test, integration-test
		},
		{
			name:        "Load errors",
			fixtureType: FixtureTypeError,
			minExpected: 4, // network-timeout, business-logic, database-connection, compensation-failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixtures, err := loader.LoadByType(tt.fixtureType)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(fixtures) < tt.minExpected {
				t.Errorf("Expected at least %d fixtures, got %d", tt.minExpected, len(fixtures))
			}

			for _, fixture := range fixtures {
				if fixture.Type != tt.fixtureType {
					t.Errorf("Expected type %s, got %s", tt.fixtureType, fixture.Type)
				}
			}
		})
	}
}

func TestFixtureLoader_LoadByTags(t *testing.T) {
	loader := GetDefaultFixtureLoader()

	tests := []struct {
		name        string
		tags        []string
		minExpected int
	}{
		{
			name:        "Load success fixtures",
			tags:        []string{"success"},
			minExpected: 1,
		},
		{
			name:        "Load error fixtures",
			tags:        []string{"error"},
			minExpected: 5,
		},
		{
			name:        "Load compensation fixtures",
			tags:        []string{"compensation"},
			minExpected: 2,
		},
		{
			name:        "Load event fixtures",
			tags:        []string{"event"},
			minExpected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixtures, err := loader.LoadByTags(tt.tags...)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(fixtures) < tt.minExpected {
				t.Errorf("Expected at least %d fixtures, got %d", tt.minExpected, len(fixtures))
			}

			// Verify all fixtures have the required tags
			for _, fixture := range fixtures {
				for _, reqTag := range tt.tags {
					found := false
					for _, tag := range fixture.Tags {
						if tag == reqTag {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Fixture %s missing required tag %s", fixture.ID, reqTag)
					}
				}
			}
		})
	}
}

func TestPredefinedFixtureLoaders(t *testing.T) {
	tests := []struct {
		name       string
		loadFunc   func() (*Fixture, error)
		expectType FixtureType
	}{
		{
			name:       "Load successful saga fixture",
			loadFunc:   LoadSuccessfulSagaFixture,
			expectType: FixtureTypeSagaDefinition,
		},
		{
			name:       "Load failing saga fixture",
			loadFunc:   LoadFailingSagaFixture,
			expectType: FixtureTypeSagaDefinition,
		},
		{
			name:       "Load compensation saga fixture",
			loadFunc:   LoadCompensationSagaFixture,
			expectType: FixtureTypeSagaDefinition,
		},
		{
			name:       "Load timeout saga fixture",
			loadFunc:   LoadTimeoutSagaFixture,
			expectType: FixtureTypeSagaDefinition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture, err := tt.loadFunc()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if fixture == nil {
				t.Fatal("Expected fixture but got nil")
			}

			if fixture.Type != tt.expectType {
				t.Errorf("Expected type %s, got %s", tt.expectType, fixture.Type)
			}
		})
	}
}

func TestFixtureGenerator_GenerateSagaDefinition(t *testing.T) {
	generator := NewFixtureGenerator()

	tests := []struct {
		name      string
		sagaID    string
		sagaName  string
		stepCount int
	}{
		{
			name:      "Generate 3-step saga",
			sagaID:    "test-saga-3",
			sagaName:  "Test Saga 3",
			stepCount: 3,
		},
		{
			name:      "Generate 5-step saga",
			sagaID:    "test-saga-5",
			sagaName:  "Test Saga 5",
			stepCount: 5,
		},
		{
			name:      "Generate 10-step saga",
			sagaID:    "test-saga-10",
			sagaName:  "Test Saga 10",
			stepCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sagaDef := generator.GenerateSagaDefinition(tt.sagaID, tt.sagaName, tt.stepCount)

			if sagaDef == nil {
				t.Fatal("Expected saga definition but got nil")
			}

			if sagaDef.GetID() != tt.sagaID {
				t.Errorf("Expected ID %s, got %s", tt.sagaID, sagaDef.GetID())
			}

			if sagaDef.GetName() != tt.sagaName {
				t.Errorf("Expected name %s, got %s", tt.sagaName, sagaDef.GetName())
			}

			steps := sagaDef.GetSteps()
			if len(steps) != tt.stepCount {
				t.Errorf("Expected %d steps, got %d", tt.stepCount, len(steps))
			}
		})
	}
}

func TestFixtureGenerator_GenerateSagaInstance(t *testing.T) {
	generator := NewFixtureGenerator()

	tests := []struct {
		name  string
		state saga.SagaState
	}{
		{
			name:  "Generate pending instance",
			state: saga.StatePending,
		},
		{
			name:  "Generate running instance",
			state: saga.StateRunning,
		},
		{
			name:  "Generate completed instance",
			state: saga.StateCompleted,
		},
		{
			name:  "Generate failed instance",
			state: saga.StateFailed,
		},
		{
			name:  "Generate timed out instance",
			state: saga.StateTimedOut,
		},
		{
			name:  "Generate compensating instance",
			state: saga.StateCompensating,
		},
		{
			name:  "Generate compensated instance",
			state: saga.StateCompensated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := generator.GenerateSagaInstance("test-saga-1", "test-def-1", tt.state)

			if instance == nil {
				t.Fatal("Expected saga instance but got nil")
			}

			if instance.State != tt.state {
				t.Errorf("Expected state %s, got %s", tt.state, instance.State)
			}

			if instance.ID == "" {
				t.Error("Instance ID should not be empty")
			}

			if instance.DefinitionID == "" {
				t.Error("Definition ID should not be empty")
			}

			// Verify state-specific fields
			switch tt.state {
			case saga.StateCompleted:
				if instance.CompletedAt == nil {
					t.Error("CompletedAt should be set for completed state")
				}
				if instance.ResultData == nil {
					t.Error("ResultData should be set for completed state")
				}
			case saga.StateFailed:
				if instance.Error == nil {
					t.Error("Error should be set for failed state")
				}
			case saga.StateTimedOut:
				if instance.TimedOutAt == nil {
					t.Error("TimedOutAt should be set for timed out state")
				}
				if instance.Error == nil {
					t.Error("Error should be set for timed out state")
				}
			case saga.StateCompensating, saga.StateCompensated:
				if instance.Error == nil {
					t.Error("Error should be set for compensating/compensated state")
				}
			}
		})
	}
}

func TestFixtureGenerator_GenerateStepState(t *testing.T) {
	generator := NewFixtureGenerator()

	tests := []struct {
		name  string
		state saga.StepStateEnum
	}{
		{
			name:  "Generate pending step",
			state: saga.StepStatePending,
		},
		{
			name:  "Generate running step",
			state: saga.StepStateRunning,
		},
		{
			name:  "Generate completed step",
			state: saga.StepStateCompleted,
		},
		{
			name:  "Generate failed step",
			state: saga.StepStateFailed,
		},
		{
			name:  "Generate compensated step",
			state: saga.StepStateCompensated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepState := generator.GenerateStepState("test-saga-1", 0, tt.state)

			if stepState == nil {
				t.Fatal("Expected step state but got nil")
			}

			if stepState.State != tt.state {
				t.Errorf("Expected state %s, got %s", tt.state, stepState.State)
			}

			if stepState.SagaID == "" {
				t.Error("SagaID should not be empty")
			}

			// Verify state-specific fields
			switch tt.state {
			case saga.StepStateCompleted:
				if stepState.CompletedAt == nil {
					t.Error("CompletedAt should be set for completed state")
				}
				if stepState.OutputData == nil {
					t.Error("OutputData should be set for completed state")
				}
			case saga.StepStateFailed:
				if stepState.Error == nil {
					t.Error("Error should be set for failed state")
				}
			case saga.StepStateCompensated:
				if stepState.CompensationState == nil {
					t.Error("CompensationState should be set for compensated state")
				}
			}
		})
	}
}

func TestFixtureGenerator_GenerateEvent(t *testing.T) {
	generator := NewFixtureGenerator()

	tests := []struct {
		name      string
		eventType saga.SagaEventType
	}{
		{
			name:      "Generate started event",
			eventType: saga.EventSagaStarted,
		},
		{
			name:      "Generate completed event",
			eventType: saga.EventSagaCompleted,
		},
		{
			name:      "Generate failed event",
			eventType: saga.EventSagaFailed,
		},
		{
			name:      "Generate step completed event",
			eventType: saga.EventSagaStepCompleted,
		},
		{
			name:      "Generate step failed event",
			eventType: saga.EventSagaStepFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := generator.GenerateEvent("test-saga-1", tt.eventType)

			if event == nil {
				t.Fatal("Expected event but got nil")
			}

			if event.Type != tt.eventType {
				t.Errorf("Expected type %s, got %s", tt.eventType, event.Type)
			}

			if event.ID == "" {
				t.Error("Event ID should not be empty")
			}

			if event.SagaID == "" {
				t.Error("SagaID should not be empty")
			}

			if event.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

func TestFixtureGenerator_GenerateError(t *testing.T) {
	generator := NewFixtureGenerator()

	tests := []struct {
		name      string
		code      string
		message   string
		errorType saga.ErrorType
		retryable bool
	}{
		{
			name:      "Generate retryable network error",
			code:      "NETWORK_ERROR",
			message:   "Network request failed",
			errorType: saga.ErrorTypeNetwork,
			retryable: true,
		},
		{
			name:      "Generate non-retryable validation error",
			code:      "VALIDATION_ERROR",
			message:   "Invalid input data",
			errorType: saga.ErrorTypeValidation,
			retryable: false,
		},
		{
			name:      "Generate timeout error",
			code:      "TIMEOUT",
			message:   "Operation timed out",
			errorType: saga.ErrorTypeTimeout,
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.GenerateError(tt.code, tt.message, tt.errorType, tt.retryable)

			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			if err.Code != tt.code {
				t.Errorf("Expected code %s, got %s", tt.code, err.Code)
			}

			if err.Message != tt.message {
				t.Errorf("Expected message %s, got %s", tt.message, err.Message)
			}

			if err.Type != tt.errorType {
				t.Errorf("Expected type %s, got %s", tt.errorType, err.Type)
			}

			if err.Retryable != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, err.Retryable)
			}

			if err.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

func TestFixtureBuilder(t *testing.T) {
	builder := NewFixtureBuilder("test-fixture", "Test Fixture", FixtureTypeSagaDefinition)

	fixture := builder.
		WithDescription("A test fixture").
		WithTags("test", "example").
		WithData(map[string]interface{}{"key": "value"}).
		WithMetadata("author", "test").
		Build()

	if fixture.ID != "test-fixture" {
		t.Errorf("Expected ID 'test-fixture', got %s", fixture.ID)
	}

	if fixture.Name != "Test Fixture" {
		t.Errorf("Expected name 'Test Fixture', got %s", fixture.Name)
	}

	if fixture.Description != "A test fixture" {
		t.Errorf("Expected description 'A test fixture', got %s", fixture.Description)
	}

	if len(fixture.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(fixture.Tags))
	}

	if fixture.Data == nil {
		t.Error("Expected data but got nil")
	}

	if fixture.Metadata["author"] != "test" {
		t.Error("Expected metadata author to be 'test'")
	}
}

func TestValidateFixture(t *testing.T) {
	tests := []struct {
		name        string
		fixture     *Fixture
		expectError bool
	}{
		{
			name: "Valid fixture",
			fixture: &Fixture{
				ID:   "test-1",
				Name: "Test",
				Type: FixtureTypeSagaDefinition,
				Data: map[string]interface{}{"key": "value"},
			},
			expectError: false,
		},
		{
			name: "Missing ID",
			fixture: &Fixture{
				Name: "Test",
				Type: FixtureTypeSagaDefinition,
				Data: map[string]interface{}{"key": "value"},
			},
			expectError: true,
		},
		{
			name: "Missing name",
			fixture: &Fixture{
				ID:   "test-1",
				Type: FixtureTypeSagaDefinition,
				Data: map[string]interface{}{"key": "value"},
			},
			expectError: true,
		},
		{
			name: "Missing type",
			fixture: &Fixture{
				ID:   "test-1",
				Name: "Test",
				Data: map[string]interface{}{"key": "value"},
			},
			expectError: true,
		},
		{
			name: "Missing data",
			fixture: &Fixture{
				ID:   "test-1",
				Name: "Test",
				Type: FixtureTypeSagaDefinition,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFixture(tt.fixture)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{
			name:        "Parse seconds",
			input:       "30s",
			expected:    30 * time.Second,
			expectError: false,
		},
		{
			name:        "Parse minutes",
			input:       "5m",
			expected:    5 * time.Minute,
			expectError: false,
		},
		{
			name:        "Parse hours",
			input:       "2h",
			expected:    2 * time.Hour,
			expectError: false,
		},
		{
			name:        "Parse milliseconds",
			input:       "100ms",
			expected:    100 * time.Millisecond,
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    0,
			expectError: false,
		},
		{
			name:        "Invalid format",
			input:       "invalid",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := ParseDuration(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if duration != tt.expected {
				t.Errorf("Expected duration %v, got %v", tt.expected, duration)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Parse RFC3339",
			input:       "2025-01-15T10:00:00Z",
			expectError: false,
		},
		{
			name:        "Parse with timezone",
			input:       "2025-01-15T10:00:00+08:00",
			expectError: false,
		},
		{
			name:        "Parse date only",
			input:       "2025-01-15",
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: false,
		},
		{
			name:        "Invalid format",
			input:       "invalid-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedTime, err := ParseTime(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.input == "" && !parsedTime.IsZero() {
				t.Error("Expected zero time for empty string")
			}

			if tt.input != "" && parsedTime.IsZero() {
				t.Error("Expected non-zero time")
			}
		})
	}
}

func TestFixtureCache(t *testing.T) {
	loader := NewFixtureLoader("")

	// Load fixture first time
	fixture1, err := loader.Load("successful-saga")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Load same fixture second time (should come from cache)
	fixture2, err := loader.Load("successful-saga")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should be the same instance from cache
	if fixture1 != fixture2 {
		t.Error("Expected same fixture instance from cache")
	}

	// Clear cache
	loader.ClearCache()

	// Load again after cache clear
	fixture3, err := loader.Load("successful-saga")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should be different instance after cache clear
	if fixture1 == fixture3 {
		t.Error("Expected different fixture instance after cache clear")
	}
}

