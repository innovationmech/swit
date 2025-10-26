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
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"gopkg.in/yaml.v3"
)

//go:embed fixtures/*.yaml
var fixturesFS embed.FS

// ==========================
// Fixture Core Types
// ==========================

// Fixture represents a test data fixture.
type Fixture struct {
	ID          string                 `yaml:"id" json:"id"`
	Name        string                 `yaml:"name" json:"name"`
	Description string                 `yaml:"description" json:"description"`
	Type        FixtureType            `yaml:"type" json:"type"`
	Tags        []string               `yaml:"tags" json:"tags"`
	Data        interface{}            `yaml:"data" json:"data"`
	Metadata    map[string]interface{} `yaml:"metadata" json:"metadata"`
}

// FixtureType represents the type of fixture.
type FixtureType string

const (
	// FixtureTypeSagaDefinition represents a Saga definition fixture.
	FixtureTypeSagaDefinition FixtureType = "saga_definition"

	// FixtureTypeSagaInstance represents a Saga instance state fixture.
	FixtureTypeSagaInstance FixtureType = "saga_instance"

	// FixtureTypeStepState represents a step state fixture.
	FixtureTypeStepState FixtureType = "step_state"

	// FixtureTypeSagaEvent represents a Saga event fixture.
	FixtureTypeSagaEvent FixtureType = "saga_event"

	// FixtureTypeConfig represents a configuration fixture.
	FixtureTypeConfig FixtureType = "config"

	// FixtureTypeError represents an error scenario fixture.
	FixtureTypeError FixtureType = "error"

	// FixtureTypeOrderData represents order processing data fixture.
	FixtureTypeOrderData FixtureType = "order_data"

	// FixtureTypePaymentData represents payment data fixture.
	FixtureTypePaymentData FixtureType = "payment_data"

	// FixtureTypeInventoryData represents inventory data fixture.
	FixtureTypeInventoryData FixtureType = "inventory_data"
)

// SagaDefinitionFixture represents a Saga definition fixture.
type SagaDefinitionFixture struct {
	ID                   string                  `yaml:"id" json:"id"`
	Name                 string                  `yaml:"name" json:"name"`
	Description          string                  `yaml:"description" json:"description"`
	Timeout              string                  `yaml:"timeout" json:"timeout"`
	CompensationStrategy string                  `yaml:"compensation_strategy" json:"compensation_strategy"`
	Steps                []StepDefinitionFixture `yaml:"steps" json:"steps"`
	Metadata             map[string]interface{}  `yaml:"metadata" json:"metadata"`
	RetryConfig          *RetryConfigFixture     `yaml:"retry_config,omitempty" json:"retry_config,omitempty"`
}

// StepDefinitionFixture represents a step definition in a fixture.
type StepDefinitionFixture struct {
	ID            string                 `yaml:"id" json:"id"`
	Name          string                 `yaml:"name" json:"name"`
	Type          string                 `yaml:"type" json:"type"`
	Timeout       string                 `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retryable     bool                   `yaml:"retryable" json:"retryable"`
	Compensatable bool                   `yaml:"compensatable" json:"compensatable"`
	Metadata      map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// RetryConfigFixture represents retry configuration in a fixture.
type RetryConfigFixture struct {
	MaxAttempts       int     `yaml:"max_attempts" json:"max_attempts"`
	InitialDelay      string  `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay          string  `yaml:"max_delay" json:"max_delay"`
	BackoffMultiplier float64 `yaml:"backoff_multiplier" json:"backoff_multiplier"`
	Jitter            float64 `yaml:"jitter" json:"jitter"`
}

// SagaInstanceFixture represents a Saga instance state fixture.
type SagaInstanceFixture struct {
	ID           string                 `yaml:"id" json:"id"`
	DefinitionID string                 `yaml:"definition_id" json:"definition_id"`
	Name         string                 `yaml:"name" json:"name"`
	Description  string                 `yaml:"description" json:"description"`
	State        string                 `yaml:"state" json:"state"`
	CurrentStep  int                    `yaml:"current_step" json:"current_step"`
	TotalSteps   int                    `yaml:"total_steps" json:"total_steps"`
	CreatedAt    string                 `yaml:"created_at" json:"created_at"`
	UpdatedAt    string                 `yaml:"updated_at" json:"updated_at"`
	StartedAt    string                 `yaml:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt  string                 `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
	TimedOutAt   string                 `yaml:"timed_out_at,omitempty" json:"timed_out_at,omitempty"`
	InitialData  interface{}            `yaml:"initial_data,omitempty" json:"initial_data,omitempty"`
	CurrentData  interface{}            `yaml:"current_data,omitempty" json:"current_data,omitempty"`
	ResultData   interface{}            `yaml:"result_data,omitempty" json:"result_data,omitempty"`
	Error        *ErrorFixture          `yaml:"error,omitempty" json:"error,omitempty"`
	Timeout      string                 `yaml:"timeout" json:"timeout"`
	StepStates   []StepStateFixture     `yaml:"step_states,omitempty" json:"step_states,omitempty"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	TraceID      string                 `yaml:"trace_id,omitempty" json:"trace_id,omitempty"`
	SpanID       string                 `yaml:"span_id,omitempty" json:"span_id,omitempty"`
	Version      int                    `yaml:"version" json:"version"`
}

// StepStateFixture represents a step state fixture.
type StepStateFixture struct {
	ID                string                    `yaml:"id" json:"id"`
	SagaID            string                    `yaml:"saga_id" json:"saga_id"`
	StepIndex         int                       `yaml:"step_index" json:"step_index"`
	Name              string                    `yaml:"name" json:"name"`
	State             string                    `yaml:"state" json:"state"`
	Attempts          int                       `yaml:"attempts" json:"attempts"`
	MaxAttempts       int                       `yaml:"max_attempts" json:"max_attempts"`
	CreatedAt         string                    `yaml:"created_at" json:"created_at"`
	StartedAt         string                    `yaml:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt       string                    `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
	LastAttemptAt     string                    `yaml:"last_attempt_at,omitempty" json:"last_attempt_at,omitempty"`
	InputData         interface{}               `yaml:"input_data,omitempty" json:"input_data,omitempty"`
	OutputData        interface{}               `yaml:"output_data,omitempty" json:"output_data,omitempty"`
	Error             *ErrorFixture             `yaml:"error,omitempty" json:"error,omitempty"`
	CompensationState *CompensationStateFixture `yaml:"compensation_state,omitempty" json:"compensation_state,omitempty"`
	Metadata          map[string]interface{}    `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// CompensationStateFixture represents compensation state in a fixture.
type CompensationStateFixture struct {
	State       string        `yaml:"state" json:"state"`
	Attempts    int           `yaml:"attempts" json:"attempts"`
	MaxAttempts int           `yaml:"max_attempts" json:"max_attempts"`
	StartedAt   string        `yaml:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt string        `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
	Error       *ErrorFixture `yaml:"error,omitempty" json:"error,omitempty"`
}

// ErrorFixture represents an error in a fixture.
type ErrorFixture struct {
	Code       string                 `yaml:"code" json:"code"`
	Message    string                 `yaml:"message" json:"message"`
	Type       string                 `yaml:"type" json:"type"`
	Retryable  bool                   `yaml:"retryable" json:"retryable"`
	Timestamp  string                 `yaml:"timestamp" json:"timestamp"`
	StackTrace string                 `yaml:"stack_trace,omitempty" json:"stack_trace,omitempty"`
	Details    map[string]interface{} `yaml:"details,omitempty" json:"details,omitempty"`
	Cause      *ErrorFixture          `yaml:"cause,omitempty" json:"cause,omitempty"`
}

// SagaEventFixture represents a Saga event fixture.
type SagaEventFixture struct {
	ID             string                 `yaml:"id" json:"id"`
	SagaID         string                 `yaml:"saga_id" json:"saga_id"`
	StepID         string                 `yaml:"step_id,omitempty" json:"step_id,omitempty"`
	Type           string                 `yaml:"type" json:"type"`
	Version        string                 `yaml:"version" json:"version"`
	Timestamp      string                 `yaml:"timestamp" json:"timestamp"`
	CorrelationID  string                 `yaml:"correlation_id,omitempty" json:"correlation_id,omitempty"`
	Data           interface{}            `yaml:"data,omitempty" json:"data,omitempty"`
	PreviousState  interface{}            `yaml:"previous_state,omitempty" json:"previous_state,omitempty"`
	NewState       interface{}            `yaml:"new_state,omitempty" json:"new_state,omitempty"`
	Error          *ErrorFixture          `yaml:"error,omitempty" json:"error,omitempty"`
	Duration       string                 `yaml:"duration,omitempty" json:"duration,omitempty"`
	Attempt        int                    `yaml:"attempt,omitempty" json:"attempt,omitempty"`
	MaxAttempts    int                    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	Metadata       map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Source         string                 `yaml:"source,omitempty" json:"source,omitempty"`
	Service        string                 `yaml:"service,omitempty" json:"service,omitempty"`
	ServiceVersion string                 `yaml:"service_version,omitempty" json:"service_version,omitempty"`
	TraceID        string                 `yaml:"trace_id,omitempty" json:"trace_id,omitempty"`
	SpanID         string                 `yaml:"span_id,omitempty" json:"span_id,omitempty"`
	ParentSpanID   string                 `yaml:"parent_span_id,omitempty" json:"parent_span_id,omitempty"`
}

// ==========================
// Fixture Loader
// ==========================

// FixtureLoader loads fixtures from files or embedded data.
type FixtureLoader struct {
	fixturesDir string
	cache       map[string]*Fixture
}

// NewFixtureLoader creates a new fixture loader.
func NewFixtureLoader(fixturesDir string) *FixtureLoader {
	return &FixtureLoader{
		fixturesDir: fixturesDir,
		cache:       make(map[string]*Fixture),
	}
}

// Load loads a fixture by ID from a file.
func (l *FixtureLoader) Load(id string) (*Fixture, error) {
	// Check cache first
	if fixture, ok := l.cache[id]; ok {
		return fixture, nil
	}

	// Try to load from embedded FS first
	data, err := fixturesFS.ReadFile(fmt.Sprintf("fixtures/%s.yaml", id))
	if err != nil {
		// Try .yml extension
		data, err = fixturesFS.ReadFile(fmt.Sprintf("fixtures/%s.yml", id))
		if err != nil {
			// Try to load from file system
			if l.fixturesDir != "" {
				data, err = os.ReadFile(filepath.Join(l.fixturesDir, id+".yaml"))
				if err != nil {
					data, err = os.ReadFile(filepath.Join(l.fixturesDir, id+".yml"))
					if err != nil {
						return nil, fmt.Errorf("failed to load fixture %s: %w", id, err)
					}
				}
			} else {
				return nil, fmt.Errorf("failed to load fixture %s: %w", id, err)
			}
		}
	}

	var fixture Fixture
	if err := yaml.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fixture %s: %w", id, err)
	}

	l.cache[id] = &fixture
	return &fixture, nil
}

// LoadFromFile loads a fixture from a specific file path.
func (l *FixtureLoader) LoadFromFile(filePath string) (*Fixture, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file %s: %w", filePath, err)
	}

	var fixture Fixture
	if err := yaml.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fixture from %s: %w", filePath, err)
	}

	return &fixture, nil
}

// LoadAll loads all fixtures from the fixtures directory.
func (l *FixtureLoader) LoadAll() ([]*Fixture, error) {
	fixtures := make([]*Fixture, 0)

	// Load from embedded FS
	entries, err := fixturesFS.ReadDir("fixtures")
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
				id := name[:len(name)-len(filepath.Ext(name))]
				fixture, err := l.Load(id)
				if err != nil {
					return nil, err
				}
				fixtures = append(fixtures, fixture)
			}
		}
	}

	// Load from file system if specified
	if l.fixturesDir != "" {
		files, err := os.ReadDir(l.fixturesDir)
		if err != nil {
			return fixtures, nil // Return what we have from embedded FS
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := file.Name()
			if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
				fixture, err := l.LoadFromFile(filepath.Join(l.fixturesDir, name))
				if err != nil {
					return nil, err
				}
				fixtures = append(fixtures, fixture)
			}
		}
	}

	return fixtures, nil
}

// LoadByType loads all fixtures of a specific type.
func (l *FixtureLoader) LoadByType(fixtureType FixtureType) ([]*Fixture, error) {
	allFixtures, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	filtered := make([]*Fixture, 0)
	for _, fixture := range allFixtures {
		if fixture.Type == fixtureType {
			filtered = append(filtered, fixture)
		}
	}

	return filtered, nil
}

// LoadByTags loads all fixtures that have all the specified tags.
func (l *FixtureLoader) LoadByTags(tags ...string) ([]*Fixture, error) {
	allFixtures, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	filtered := make([]*Fixture, 0)
	for _, fixture := range allFixtures {
		if hasAllTags(fixture.Tags, tags) {
			filtered = append(filtered, fixture)
		}
	}

	return filtered, nil
}

// ClearCache clears the fixture cache.
func (l *FixtureLoader) ClearCache() {
	l.cache = make(map[string]*Fixture)
}

// hasAllTags checks if a slice contains all specified tags.
func hasAllTags(fixtureTags, requiredTags []string) bool {
	tagMap := make(map[string]bool)
	for _, tag := range fixtureTags {
		tagMap[tag] = true
	}

	for _, required := range requiredTags {
		if !tagMap[required] {
			return false
		}
	}

	return true
}

// ==========================
// Fixture Generator
// ==========================

// FixtureGenerator generates fixtures programmatically.
type FixtureGenerator struct{}

// NewFixtureGenerator creates a new fixture generator.
func NewFixtureGenerator() *FixtureGenerator {
	return &FixtureGenerator{}
}

// GenerateSagaDefinition generates a Saga definition fixture.
func (g *FixtureGenerator) GenerateSagaDefinition(id, name string, stepCount int) *MockSagaDefinition {
	builder := NewSagaTestBuilder(id, name).
		WithTimeout(5 * time.Minute)

	for i := 0; i < stepCount; i++ {
		stepID := fmt.Sprintf("step-%d", i+1)
		stepName := fmt.Sprintf("Step %d", i+1)
		builder.AddStep(stepID, stepName)
	}

	return builder.Build()
}

// GenerateSagaInstance generates a Saga instance fixture in a specific state.
func (g *FixtureGenerator) GenerateSagaInstance(id, definitionID string, state saga.SagaState) *saga.SagaInstanceData {
	now := time.Now()
	startedAt := now.Add(-5 * time.Minute)

	instance := &saga.SagaInstanceData{
		ID:           id,
		DefinitionID: definitionID,
		Name:         "Test Saga",
		Description:  "Generated test saga instance",
		State:        state,
		CurrentStep:  0,
		TotalSteps:   3,
		CreatedAt:    now.Add(-10 * time.Minute),
		UpdatedAt:    now,
		StartedAt:    &startedAt,
		InitialData: map[string]interface{}{
			"test": "data",
		},
		Timeout:  5 * time.Minute,
		Metadata: make(map[string]interface{}),
		Version:  1,
	}

	// Set state-specific fields
	switch state {
	case saga.StateCompleted:
		completedAt := now
		instance.CompletedAt = &completedAt
		instance.CurrentStep = 3
		instance.ResultData = map[string]interface{}{
			"result": "success",
		}
	case saga.StateFailed:
		instance.Error = &saga.SagaError{
			Code:      "TEST_ERROR",
			Message:   "Test error message",
			Type:      saga.ErrorTypeService,
			Retryable: false,
			Timestamp: now,
		}
	case saga.StateTimedOut:
		timedOutAt := now
		instance.TimedOutAt = &timedOutAt
		instance.Error = &saga.SagaError{
			Code:      "TIMEOUT",
			Message:   "Saga timed out",
			Type:      saga.ErrorTypeTimeout,
			Retryable: false,
			Timestamp: now,
		}
	case saga.StateCompensating:
		instance.Error = &saga.SagaError{
			Code:      "STEP_FAILED",
			Message:   "Step execution failed, compensating",
			Type:      saga.ErrorTypeService,
			Retryable: false,
			Timestamp: now,
		}
	case saga.StateCompensated:
		completedAt := now
		instance.CompletedAt = &completedAt
		instance.Error = &saga.SagaError{
			Code:      "STEP_FAILED",
			Message:   "Step execution failed, compensation successful",
			Type:      saga.ErrorTypeService,
			Retryable: false,
			Timestamp: now.Add(-2 * time.Minute),
		}
	}

	return instance
}

// GenerateStepState generates a step state fixture.
func (g *FixtureGenerator) GenerateStepState(sagaID string, stepIndex int, state saga.StepStateEnum) *saga.StepState {
	now := time.Now()
	createdAt := now.Add(-5 * time.Minute)
	startedAt := now.Add(-3 * time.Minute)

	stepState := &saga.StepState{
		ID:          fmt.Sprintf("%s-step-%d", sagaID, stepIndex),
		SagaID:      sagaID,
		StepIndex:   stepIndex,
		Name:        fmt.Sprintf("Step %d", stepIndex+1),
		State:       state,
		Attempts:    1,
		MaxAttempts: 3,
		CreatedAt:   createdAt,
		StartedAt:   &startedAt,
		InputData: map[string]interface{}{
			"input": "test",
		},
		Metadata: make(map[string]interface{}),
	}

	// Set state-specific fields
	switch state {
	case saga.StepStateCompleted:
		completedAt := now
		stepState.CompletedAt = &completedAt
		stepState.OutputData = map[string]interface{}{
			"output": "success",
		}
	case saga.StepStateFailed:
		lastAttemptAt := now
		stepState.LastAttemptAt = &lastAttemptAt
		stepState.Error = &saga.SagaError{
			Code:      "STEP_ERROR",
			Message:   "Step execution failed",
			Type:      saga.ErrorTypeService,
			Retryable: false,
			Timestamp: now,
		}
	case saga.StepStateCompensated:
		completedAt := now
		stepState.CompletedAt = &completedAt
		stepState.CompensationState = &saga.CompensationState{
			State:       saga.CompensationStateCompleted,
			Attempts:    1,
			MaxAttempts: 3,
			StartedAt:   &startedAt,
			CompletedAt: &completedAt,
		}
	}

	return stepState
}

// GenerateEvent generates a Saga event fixture.
func (g *FixtureGenerator) GenerateEvent(sagaID string, eventType saga.SagaEventType) *saga.SagaEvent {
	now := time.Now()

	event := &saga.SagaEvent{
		ID:        fmt.Sprintf("event-%d", now.UnixNano()),
		SagaID:    sagaID,
		Type:      eventType,
		Version:   "1.0",
		Timestamp: now,
		Data: map[string]interface{}{
			"test": "data",
		},
		Metadata:       make(map[string]interface{}),
		Source:         "test-service",
		Service:        "test-service",
		ServiceVersion: "1.0.0",
	}

	return event
}

// GenerateError generates an error fixture.
func (g *FixtureGenerator) GenerateError(code, message string, errorType saga.ErrorType, retryable bool) *saga.SagaError {
	return &saga.SagaError{
		Code:      code,
		Message:   message,
		Type:      errorType,
		Retryable: retryable,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// ==========================
// Predefined Fixtures
// ==========================

// GetDefaultFixtureLoader returns a default fixture loader.
func GetDefaultFixtureLoader() *FixtureLoader {
	return NewFixtureLoader("")
}

// LoadSuccessfulSagaFixture loads a successful saga fixture.
func LoadSuccessfulSagaFixture() (*Fixture, error) {
	loader := GetDefaultFixtureLoader()
	return loader.Load("successful-saga")
}

// LoadFailingSagaFixture loads a failing saga fixture.
func LoadFailingSagaFixture() (*Fixture, error) {
	loader := GetDefaultFixtureLoader()
	return loader.Load("failing-saga")
}

// LoadCompensationSagaFixture loads a compensation saga fixture.
func LoadCompensationSagaFixture() (*Fixture, error) {
	loader := GetDefaultFixtureLoader()
	return loader.Load("compensation-saga")
}

// LoadTimeoutSagaFixture loads a timeout saga fixture.
func LoadTimeoutSagaFixture() (*Fixture, error) {
	loader := GetDefaultFixtureLoader()
	return loader.Load("timeout-saga")
}

// ==========================
// Fixture Builder
// ==========================

// FixtureBuilder builds fixtures fluently.
type FixtureBuilder struct {
	fixture *Fixture
}

// NewFixtureBuilder creates a new fixture builder.
func NewFixtureBuilder(id, name string, fixtureType FixtureType) *FixtureBuilder {
	return &FixtureBuilder{
		fixture: &Fixture{
			ID:       id,
			Name:     name,
			Type:     fixtureType,
			Tags:     make([]string, 0),
			Metadata: make(map[string]interface{}),
		},
	}
}

// WithDescription sets the fixture description.
func (b *FixtureBuilder) WithDescription(desc string) *FixtureBuilder {
	b.fixture.Description = desc
	return b
}

// WithTags adds tags to the fixture.
func (b *FixtureBuilder) WithTags(tags ...string) *FixtureBuilder {
	b.fixture.Tags = append(b.fixture.Tags, tags...)
	return b
}

// WithData sets the fixture data.
func (b *FixtureBuilder) WithData(data interface{}) *FixtureBuilder {
	b.fixture.Data = data
	return b
}

// WithMetadata adds metadata to the fixture.
func (b *FixtureBuilder) WithMetadata(key string, value interface{}) *FixtureBuilder {
	b.fixture.Metadata[key] = value
	return b
}

// Build returns the constructed fixture.
func (b *FixtureBuilder) Build() *Fixture {
	return b.fixture
}

// SaveToFile saves the fixture to a file.
func (b *FixtureBuilder) SaveToFile(filePath string) error {
	data, err := yaml.Marshal(b.fixture)
	if err != nil {
		return fmt.Errorf("failed to marshal fixture: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write fixture file: %w", err)
	}

	return nil
}

// ==========================
// Fixture Validation
// ==========================

// ValidateFixture validates a fixture.
func ValidateFixture(fixture *Fixture) error {
	if fixture.ID == "" {
		return fmt.Errorf("fixture ID is required")
	}

	if fixture.Name == "" {
		return fmt.Errorf("fixture name is required")
	}

	if fixture.Type == "" {
		return fmt.Errorf("fixture type is required")
	}

	if fixture.Data == nil {
		return fmt.Errorf("fixture data is required")
	}

	return nil
}

// ==========================
// Helper Functions
// ==========================

// ConvertToJSON converts a fixture to JSON bytes.
func ConvertToJSON(fixture *Fixture) ([]byte, error) {
	return json.MarshalIndent(fixture, "", "  ")
}

// ConvertToYAML converts a fixture to YAML bytes.
func ConvertToYAML(fixture *Fixture) ([]byte, error) {
	return yaml.Marshal(fixture)
}

// ParseDuration parses a duration string from fixture.
func ParseDuration(durationStr string) (time.Duration, error) {
	if durationStr == "" {
		return 0, nil
	}
	return time.ParseDuration(durationStr)
}

// ParseTime parses a time string from fixture.
func ParseTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}

	// Try RFC3339 format first
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}

// CreateMockFromFixture creates a mock saga definition from a fixture.
func CreateMockFromFixture(fixture *SagaDefinitionFixture) (*MockSagaDefinition, error) {
	mock := NewMockSagaDefinition(fixture.ID, fixture.Name)
	mock.DescriptionValue = fixture.Description
	mock.MetadataValue = fixture.Metadata

	if fixture.Timeout != "" {
		timeout, err := ParseDuration(fixture.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout: %w", err)
		}
		mock.TimeoutValue = timeout
	}

	// Add steps
	for _, stepFixture := range fixture.Steps {
		step := NewMockSagaStep(stepFixture.ID, stepFixture.Name)

		// Configure step based on fixture
		step.ExecuteFunc = func(ctx context.Context, data interface{}) (interface{}, error) {
			return data, nil
		}

		if stepFixture.Compensatable {
			step.CompensateFunc = func(ctx context.Context, data interface{}) error {
				return nil
			}
		}

		mock.AddStep(step)
	}

	return mock, nil
}
