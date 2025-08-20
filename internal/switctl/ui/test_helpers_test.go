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

package ui

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestHelper provides utilities for UI testing
type TestHelper struct {
	input  *strings.Reader
	output *bytes.Buffer
	ui     *TerminalUI
}

// NewTestHelper creates a new test helper with the given input
func NewTestHelper(input string) *TestHelper {
	inputReader := strings.NewReader(input)
	outputBuffer := &bytes.Buffer{}
	ui := NewTerminalUI(
		WithInput(inputReader),
		WithOutput(outputBuffer),
		WithNoColor(true),
	)

	return &TestHelper{
		input:  inputReader,
		output: outputBuffer,
		ui:     ui,
	}
}

// GetUI returns the TerminalUI instance
func (th *TestHelper) GetUI() *TerminalUI {
	return th.ui
}

// GetOutput returns the output buffer content as string
func (th *TestHelper) GetOutput() string {
	return th.output.String()
}

// Reset clears the output buffer
func (th *TestHelper) Reset() {
	th.output.Reset()
}

// AssertContains checks if the output contains the expected text
func (th *TestHelper) AssertContains(t *testing.T, expected string) {
	assert.Contains(t, th.GetOutput(), expected)
}

// AssertNotContains checks if the output doesn't contain the text
func (th *TestHelper) AssertNotContains(t *testing.T, unexpected string) {
	assert.NotContains(t, th.GetOutput(), unexpected)
}

// MockReader implements io.Reader for testing error conditions
type MockReader struct {
	mock.Mock
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

// MockWriter implements io.Writer for testing error conditions
type MockWriter struct {
	mock.Mock
	buffer bytes.Buffer
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	if err := args.Error(1); err != nil {
		return args.Int(0), err
	}
	// If no error, also write to buffer for inspection
	m.buffer.Write(p)
	return args.Int(0), args.Error(1)
}

func (m *MockWriter) String() string {
	return m.buffer.String()
}

// MockColor implements the Color interface for testing
type MockColor struct {
	mock.Mock
}

func (m *MockColor) Sprint(a ...interface{}) string {
	args := m.Called(a...)
	return args.String(0)
}

func (m *MockColor) Sprintf(format string, a ...interface{}) string {
	args := m.Called(append([]interface{}{format}, a...)...)
	return args.String(0)
}

// MockProgressBar implements the ProgressBar interface for testing
type MockProgressBar struct {
	mock.Mock
}

func (m *MockProgressBar) Update(current int) error {
	args := m.Called(current)
	return args.Error(0)
}

func (m *MockProgressBar) SetMessage(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockProgressBar) Finish() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockProgressBar) SetTotal(total int) error {
	args := m.Called(total)
	return args.Error(0)
}

// Test helper functions

// CreateTestOptions creates a slice of test menu options
func CreateTestOptions(count int, enabledPattern ...bool) []interfaces.MenuOption {
	options := make([]interfaces.MenuOption, count)
	for i := 0; i < count; i++ {
		enabled := true
		if len(enabledPattern) > 0 {
			enabled = enabledPattern[i%len(enabledPattern)]
		}

		options[i] = interfaces.MenuOption{
			Label:       fmt.Sprintf("Option %d", i+1),
			Description: fmt.Sprintf("Description for option %d", i+1),
			Value:       fmt.Sprintf("value_%d", i+1),
			Enabled:     enabled,
		}
	}
	return options
}

// CreateTestOptionsWithIcons creates test options with icons
func CreateTestOptionsWithIcons(count int) []interfaces.MenuOption {
	icons := []string{"🔧", "⚙️", "🛠️", "🔨", "🔩", "⚡", "🌟", "💡", "🎯", "🚀"}
	options := make([]interfaces.MenuOption, count)

	for i := 0; i < count; i++ {
		icon := ""
		if i < len(icons) {
			icon = icons[i]
		}

		options[i] = interfaces.MenuOption{
			Label:       fmt.Sprintf("Option %d", i+1),
			Description: fmt.Sprintf("Description %d", i+1),
			Value:       i + 1,
			Icon:        icon,
			Enabled:     true,
		}
	}
	return options
}

// CreateFailingValidator creates a validator that always fails
func CreateFailingValidator(message string) interfaces.InputValidator {
	return func(input string) error {
		return errors.New(message)
	}
}

// CreateConditionalValidator creates a validator that fails for specific inputs
func CreateConditionalValidator(failInputs []string, message string) interfaces.InputValidator {
	failMap := make(map[string]bool)
	for _, input := range failInputs {
		failMap[input] = true
	}

	return func(input string) error {
		if failMap[input] {
			return errors.New(message)
		}
		return nil
	}
}

// SimulateUserInput simulates sequential user inputs
func SimulateUserInput(inputs ...string) string {
	return strings.Join(inputs, "\n") + "\n"
}

// CreateLargeInput creates input with repeated content for stress testing
func CreateLargeInput(content string, count int) string {
	parts := make([]string, count)
	for i := 0; i < count; i++ {
		parts[i] = content
	}
	return strings.Join(parts, "")
}

// TestUIScenario represents a common UI testing scenario
type TestUIScenario struct {
	Name           string
	Input          string
	ExpectedOutput []string
	ShouldError    bool
	ErrorMessage   string
}

// RunUIScenarios runs a set of UI test scenarios
func RunUIScenarios(t *testing.T, scenarios []TestUIScenario, testFunc func(*TestHelper) error) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			helper := NewTestHelper(scenario.Input)

			err := testFunc(helper)

			if scenario.ShouldError {
				assert.Error(t, err)
				if scenario.ErrorMessage != "" {
					assert.Contains(t, err.Error(), scenario.ErrorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			output := helper.GetOutput()
			for _, expected := range scenario.ExpectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

// Test the test helpers themselves

func TestTestHelper(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		helper := NewTestHelper("test input\n")

		assert.NotNil(t, helper.GetUI())
		assert.Equal(t, "", helper.GetOutput()) // Should be empty initially

		// Write something to output
		helper.GetUI().ShowInfo("Test message")

		output := helper.GetOutput()
		assert.Contains(t, output, "Test message")

		// Test reset
		helper.Reset()
		assert.Equal(t, "", helper.GetOutput())
	})

	t.Run("assert methods", func(t *testing.T) {
		helper := NewTestHelper("input\n")
		helper.GetUI().ShowSuccess("Success message")

		helper.AssertContains(t, "Success message")
		helper.AssertNotContains(t, "Error message")
	})
}

func TestMockReader(t *testing.T) {
	mockReader := &MockReader{}

	// Setup expectations
	testData := []byte("test data")
	mockReader.On("Read", mock.AnythingOfType("[]uint8")).Return(len(testData), nil).Once()

	buffer := make([]byte, 10)
	n, err := mockReader.Read(buffer)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	mockReader.AssertExpectations(t)
}

func TestMockWriter(t *testing.T) {
	mockWriter := &MockWriter{}

	// Setup expectations
	testData := []byte("test output")
	mockWriter.On("Write", testData).Return(len(testData), nil)

	n, err := mockWriter.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, "test output", mockWriter.String())
	mockWriter.AssertExpectations(t)
}

func TestMockColor(t *testing.T) {
	mockColor := &MockColor{}

	// Setup expectations
	mockColor.On("Sprint", mock.Anything).Return("colored text")
	mockColor.On("Sprintf", "format %s", mock.Anything).Return("formatted colored text")

	result1 := mockColor.Sprint("test")
	result2 := mockColor.Sprintf("format %s", "test")

	assert.Equal(t, "colored text", result1)
	assert.Equal(t, "formatted colored text", result2)
	mockColor.AssertExpectations(t)
}

func TestMockProgressBar(t *testing.T) {
	mockPB := &MockProgressBar{}

	// Setup expectations
	mockPB.On("Update", 50).Return(nil)
	mockPB.On("SetMessage", "Processing...").Return(nil)
	mockPB.On("SetTotal", 100).Return(nil)
	mockPB.On("Finish").Return(nil)

	// Test all methods
	assert.NoError(t, mockPB.Update(50))
	assert.NoError(t, mockPB.SetMessage("Processing..."))
	assert.NoError(t, mockPB.SetTotal(100))
	assert.NoError(t, mockPB.Finish())

	mockPB.AssertExpectations(t)
}

func TestCreateTestOptions(t *testing.T) {
	t.Run("all enabled", func(t *testing.T) {
		options := CreateTestOptions(3)
		assert.Len(t, options, 3)

		for i, option := range options {
			assert.Equal(t, fmt.Sprintf("Option %d", i+1), option.Label)
			assert.True(t, option.Enabled)
			assert.NotEmpty(t, option.Description)
			assert.NotEmpty(t, option.Value)
		}
	})

	t.Run("alternating enabled pattern", func(t *testing.T) {
		options := CreateTestOptions(4, true, false) // true, false, true, false
		assert.Len(t, options, 4)

		assert.True(t, options[0].Enabled)
		assert.False(t, options[1].Enabled)
		assert.True(t, options[2].Enabled)
		assert.False(t, options[3].Enabled)
	})
}

func TestCreateTestOptionsWithIcons(t *testing.T) {
	options := CreateTestOptionsWithIcons(5)
	assert.Len(t, options, 5)

	for i, option := range options {
		assert.Equal(t, fmt.Sprintf("Option %d", i+1), option.Label)
		assert.True(t, option.Enabled)
		assert.Equal(t, i+1, option.Value)
		if i < 10 { // We have 10 icons available
			assert.NotEmpty(t, option.Icon)
		}
	}
}

func TestValidatorHelpers(t *testing.T) {
	t.Run("failing validator", func(t *testing.T) {
		validator := CreateFailingValidator("test error")
		err := validator("any input")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test error")
	})

	t.Run("conditional validator", func(t *testing.T) {
		validator := CreateConditionalValidator([]string{"bad", "invalid"}, "validation failed")

		assert.NoError(t, validator("good"))
		assert.Error(t, validator("bad"))
		assert.Error(t, validator("invalid"))
		assert.NoError(t, validator("other"))
	})
}

func TestInputHelpers(t *testing.T) {
	t.Run("simulate user input", func(t *testing.T) {
		input := SimulateUserInput("first", "second", "third")
		expected := "first\nsecond\nthird\n"
		assert.Equal(t, expected, input)
	})

	t.Run("create large input", func(t *testing.T) {
		input := CreateLargeInput("x", 1000)
		assert.Equal(t, 1000, len(input))
		assert.Equal(t, strings.Repeat("x", 1000), input)
	})
}

func TestRunUIScenarios(t *testing.T) {
	scenarios := []TestUIScenario{
		{
			Name:           "success scenario",
			Input:          "test\n",
			ExpectedOutput: []string{"test"},
			ShouldError:    false,
		},
		{
			Name:         "error scenario",
			Input:        "",
			ShouldError:  true,
			ErrorMessage: "no input",
		},
	}

	testFunc := func(helper *TestHelper) error {
		input := helper.input
		buffer := make([]byte, 1024)
		n, err := input.Read(buffer)
		if err == io.EOF && n == 0 {
			return errors.New("no input provided")
		}

		helper.ui.ShowInfo(strings.TrimSpace(string(buffer[:n])))
		return nil
	}

	RunUIScenarios(t, scenarios, testFunc)
}

// Integration test helpers

// MockUIStyle creates a mock UI style for testing
func MockUIStyle() interfaces.UIStyle {
	return interfaces.UIStyle{
		Primary:   &MockColor{},
		Success:   &MockColor{},
		Warning:   &MockColor{},
		Error:     &MockColor{},
		Info:      &MockColor{},
		Highlight: &MockColor{},
	}
}

// CreateUIWithMocks creates a TerminalUI with mocked components
func CreateUIWithMocks(input string) (*TerminalUI, *MockWriter) {
	mockWriter := &MockWriter{}

	// Set up basic expectations for write operations
	mockWriter.On("Write", mock.Anything).Return(0, nil).Maybe()

	ui := NewTerminalUI(
		WithInput(strings.NewReader(input)),
		WithOutput(mockWriter),
		WithNoColor(true),
	)

	return ui, mockWriter
}

// Benchmark helpers

// BenchmarkHelper provides utilities for benchmarking UI operations
type BenchmarkHelper struct {
	inputs  []string
	current int
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(inputs []string) *BenchmarkHelper {
	return &BenchmarkHelper{
		inputs: inputs,
	}
}

// NextInput returns the next input string for benchmarking
func (bh *BenchmarkHelper) NextInput() string {
	if len(bh.inputs) == 0 {
		return "\n"
	}

	input := bh.inputs[bh.current%len(bh.inputs)]
	bh.current++
	return input
}

// CreateBenchmarkOptions creates options for benchmarking
func CreateBenchmarkOptions(count int) []interfaces.MenuOption {
	options := make([]interfaces.MenuOption, count)
	for i := 0; i < count; i++ {
		options[i] = interfaces.MenuOption{
			Label:   fmt.Sprintf("Benchmark Option %d", i+1),
			Value:   fmt.Sprintf("bench_%d", i+1),
			Enabled: true,
		}
	}
	return options
}

func TestBenchmarkHelper(t *testing.T) {
	helper := NewBenchmarkHelper([]string{"input1", "input2"})

	assert.Equal(t, "input1", helper.NextInput())
	assert.Equal(t, "input2", helper.NextInput())
	assert.Equal(t, "input1", helper.NextInput()) // Should cycle
}

// Example usage patterns for test helpers

func ExampleTestHelper() {
	// Create a test helper with user input
	helper := NewTestHelper("test-service\ny\n1\n")

	// Use the UI for testing
	ui := helper.GetUI()

	// Test prompt input
	serviceName, _ := ui.PromptInput("Service name", ServiceNameValidator)
	fmt.Println("Service name:", serviceName)

	// Test confirmation
	confirmed, _ := ui.PromptConfirm("Continue?", false)
	fmt.Println("Confirmed:", confirmed)

	// Test menu selection
	options := CreateTestOptions(3)
	selectedIdx, _ := ui.ShowMenu("Choose option", options)
	fmt.Println("Selected:", selectedIdx)

	// Check output
	output := helper.GetOutput()
	fmt.Println("Contains service name prompt:", strings.Contains(output, "Service name"))

	// Output:
	// Service name: test-service
	// Confirmed: true
	// Selected: 0
	// Contains service name prompt: true
}

func ExampleMockColor() {
	mockColor := &MockColor{}

	// Set up expectations
	mockColor.On("Sprint", "test").Return("COLORED[test]")
	mockColor.On("Sprintf", "Hello %s", "world").Return("COLORED[Hello world]")

	// Use in tests
	result1 := mockColor.Sprint("test")
	result2 := mockColor.Sprintf("Hello %s", "world")

	fmt.Println(result1) // COLORED[test]
	fmt.Println(result2) // COLORED[Hello world]

	// Output:
	// COLORED[test]
	// COLORED[Hello world]
}
