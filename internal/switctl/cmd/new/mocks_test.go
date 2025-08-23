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

package new

import (
	"fmt"

	"github.com/stretchr/testify/mock"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// MockDependencyContainer is a mock implementation of interfaces.DependencyContainer
type MockDependencyContainer struct {
	mock.Mock
}

func (m *MockDependencyContainer) Register(name string, factory interfaces.ServiceFactory) error {
	args := m.Called(name, factory)
	return args.Error(0)
}

func (m *MockDependencyContainer) RegisterSingleton(name string, factory interfaces.ServiceFactory) error {
	args := m.Called(name, factory)
	return args.Error(0)
}

func (m *MockDependencyContainer) GetService(name string) (interface{}, error) {
	args := m.Called(name)
	return args.Get(0), args.Error(1)
}

func (m *MockDependencyContainer) HasService(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockDependencyContainer) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockTerminalUI is a mock implementation of ui.TerminalUI
type MockTerminalUI struct {
	mock.Mock
}

func (m *MockTerminalUI) ShowWelcome() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTerminalUI) PromptInput(prompt string, validator interfaces.InputValidator) (string, error) {
	args := m.Called(prompt, validator)
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) PromptPassword(prompt string) (string, error) {
	args := m.Called(prompt)
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) PromptConfirm(prompt string, defaultValue bool) (bool, error) {
	args := m.Called(prompt, defaultValue)
	return args.Bool(0), args.Error(1)
}

func (m *MockTerminalUI) ShowFeatureSelectionMenu() (interfaces.ServiceFeatures, error) {
	args := m.Called()
	return args.Get(0).(interfaces.ServiceFeatures), args.Error(1)
}

func (m *MockTerminalUI) ShowDatabaseSelectionMenu() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) ShowAuthSelectionMenu() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTerminalUI) ShowMultiSelectMenu(title string, options []interfaces.MenuOption) ([]int, error) {
	args := m.Called(title, options)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockTerminalUI) ShowMenu(title string, options []interfaces.MenuOption) (int, error) {
	args := m.Called(title, options)
	return args.Int(0), args.Error(1)
}

func (m *MockTerminalUI) ShowProgress(title string, total int) interfaces.ProgressBar {
	args := m.Called(title, total)
	return args.Get(0).(interfaces.ProgressBar)
}

func (m *MockTerminalUI) ShowSuccess(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockTerminalUI) ShowError(err error) error {
	args := m.Called(err)
	return args.Error(0)
}

func (m *MockTerminalUI) ShowInfo(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockTerminalUI) ShowWarning(message string) {
	m.Called(message)
}

func (m *MockTerminalUI) PrintHeader(text string) {
	m.Called(text)
}

func (m *MockTerminalUI) PrintSubHeader(text string) {
	m.Called(text)
}

func (m *MockTerminalUI) PrintSeparator() {
	m.Called()
}

func (m *MockTerminalUI) ShowTable(headers []string, rows [][]string) error {
	args := m.Called(headers, rows)
	return args.Error(0)
}

func (m *MockTerminalUI) GetStyle() *interfaces.UIStyle {
	args := m.Called()
	return args.Get(0).(*interfaces.UIStyle)
}

// MockLogger is a mock implementation of interfaces.Logger
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	args := make([]interface{}, 1+len(fields))
	args[0] = msg
	copy(args[1:], fields)
	m.Called(args...)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	args := make([]interface{}, 1+len(fields))
	args[0] = msg
	copy(args[1:], fields)
	m.Called(args...)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	args := make([]interface{}, 1+len(fields))
	args[0] = msg
	copy(args[1:], fields)
	m.Called(args...)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	args := make([]interface{}, 1+len(fields))
	args[0] = msg
	copy(args[1:], fields)
	m.Called(args...)
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	args := make([]interface{}, 1+len(fields))
	args[0] = msg
	copy(args[1:], fields)
	m.Called(args...)
}

// MockFileSystem is a mock implementation of interfaces.FileSystem
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) WriteFile(path string, content []byte, perm int) error {
	args := m.Called(path, content, perm)
	return args.Error(0)
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockFileSystem) MkdirAll(path string, perm int) error {
	args := m.Called(path, perm)
	return args.Error(0)
}

func (m *MockFileSystem) Exists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockFileSystem) Remove(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystem) Copy(src, dst string) error {
	args := m.Called(src, dst)
	return args.Error(0)
}

// MockTemplateEngine is a mock implementation of interfaces.TemplateEngine
type MockTemplateEngine struct {
	mock.Mock
}

func (m *MockTemplateEngine) LoadTemplate(name string) (interfaces.Template, error) {
	args := m.Called(name)
	return args.Get(0).(interfaces.Template), args.Error(1)
}

func (m *MockTemplateEngine) RenderTemplate(template interfaces.Template, data interface{}) ([]byte, error) {
	args := m.Called(template, data)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTemplateEngine) RegisterFunction(name string, fn interface{}) error {
	args := m.Called(name, fn)
	return args.Error(0)
}

func (m *MockTemplateEngine) SetTemplateDir(dir string) error {
	args := m.Called(dir)
	return args.Error(0)
}

// MockTemplate is a mock implementation of interfaces.Template
type MockTemplate struct {
	mock.Mock
	name string
}

func (m *MockTemplate) Name() string {
	if m.name != "" {
		return m.name
	}
	args := m.Called()
	return args.String(0)
}

func (m *MockTemplate) Render(data interface{}) ([]byte, error) {
	args := m.Called(data)
	return args.Get(0).([]byte), args.Error(1)
}

// MockServiceGenerator is a mock implementation of generator.ServiceGenerator
type MockServiceGenerator struct {
	mock.Mock
}

func (m *MockServiceGenerator) GenerateService(config interfaces.ServiceConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockServiceGenerator) GenerateAPI(config interfaces.APIConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockServiceGenerator) GenerateModel(config interfaces.ModelConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockServiceGenerator) GenerateMiddleware(config interfaces.MiddlewareConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

// MockProgressBar is a mock implementation of interfaces.ProgressBar
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

// MockColor is a mock implementation of interfaces.Color
type MockColor struct {
	mock.Mock
}

func (m *MockColor) Sprint(a ...interface{}) string {
	args := m.Called(a)
	return args.String(0)
}

func (m *MockColor) Sprintf(format string, a ...interface{}) string {
	args := make([]interface{}, 1+len(a))
	args[0] = format
	copy(args[1:], a)
	result := m.Called(args...)
	return result.String(0)
}

// Helper functions for creating test data

// CreateTestServiceConfig creates a test service configuration
func CreateTestServiceConfig() interfaces.ServiceConfig {
	return interfaces.ServiceConfig{
		Name:        "test-service",
		Description: "Test service description",
		Author:      "Test Author <test@example.com>",
		Version:     "0.1.0",
		GoVersion:   "1.19",
		ModulePath:  "github.com/test/test-service",
		OutputDir:   "test-service",
		Ports: interfaces.PortConfig{
			HTTP: 9000,
			GRPC: 10000,
		},
		Features: interfaces.ServiceFeatures{
			Database:       true,
			Authentication: false,
			Cache:          false,
			MessageQueue:   false,
			Monitoring:     true,
			Tracing:        true,
			Logging:        true,
			HealthCheck:    true,
			Metrics:        true,
			Docker:         true,
			Kubernetes:     false,
		},
		Database: interfaces.DatabaseConfig{
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "test_service",
			Username: "user",
			Password: "password",
		},
		Metadata: map[string]string{
			"created_by": "test",
		},
	}
}

// CreateTestServiceFeatures creates test service features
func CreateTestServiceFeatures() interfaces.ServiceFeatures {
	return interfaces.ServiceFeatures{
		Database:       true,
		Authentication: true,
		Cache:          false,
		MessageQueue:   false,
		Monitoring:     true,
		Tracing:        true,
		Logging:        true,
		HealthCheck:    true,
		Metrics:        true,
		Docker:         true,
		Kubernetes:     true,
	}
}

// CreateTestUIStyle creates a test UI style with mock colors
func CreateTestUIStyle() *interfaces.UIStyle {
	return &interfaces.UIStyle{
		Primary:   &MockColor{},
		Success:   &MockColor{},
		Warning:   &MockColor{},
		Error:     &MockColor{},
		Info:      &MockColor{},
		Highlight: &MockColor{},
	}
}

// SetupMockUIStyle sets up mock UI style with default behaviors
func SetupMockUIStyle(style *interfaces.UIStyle) {
	if mockColor, ok := style.Primary.(*MockColor); ok {
		mockColor.On("Sprint", mock.Anything).Return("mocked text")
		mockColor.On("Sprintf", mock.Anything, mock.Anything).Return("mocked formatted text")
	}
	if mockColor, ok := style.Success.(*MockColor); ok {
		mockColor.On("Sprint", mock.Anything).Return("✓")
		mockColor.On("Sprintf", mock.Anything, mock.Anything).Return("✓ mocked")
	}
	if mockColor, ok := style.Error.(*MockColor); ok {
		mockColor.On("Sprint", mock.Anything).Return("✗")
		mockColor.On("Sprintf", mock.Anything, mock.Anything).Return("✗ mocked")
	}
	if mockColor, ok := style.Info.(*MockColor); ok {
		mockColor.On("Sprint", mock.Anything).Return("mocked info")
		mockColor.On("Sprintf", mock.Anything, mock.Anything).Return("mocked info formatted")
	}
	if mockColor, ok := style.Warning.(*MockColor); ok {
		mockColor.On("Sprint", mock.Anything).Return("⚠ mocked")
		mockColor.On("Sprintf", mock.Anything, mock.Anything).Return("⚠ mocked formatted")
	}
	if mockColor, ok := style.Highlight.(*MockColor); ok {
		mockColor.On("Sprint", mock.Anything).Return("mocked highlight")
		mockColor.On("Sprintf", mock.Anything, mock.Anything).Return("mocked highlight formatted")
	}
}

// CreateMenuOptions creates test menu options
func CreateMenuOptions() []interfaces.MenuOption {
	return []interfaces.MenuOption{
		{
			Label:       "Option 1",
			Description: "First option",
			Value:       "option1",
			Icon:        "1",
			Enabled:     true,
		},
		{
			Label:       "Option 2",
			Description: "Second option",
			Value:       "option2",
			Icon:        "2",
			Enabled:     true,
		},
		{
			Label:       "Option 3",
			Description: "Third option",
			Value:       "option3",
			Icon:        "3",
			Enabled:     false,
		},
	}
}

// ErrorMatcher is a helper for matching specific error messages in tests
type ErrorMatcher struct {
	Expected string
}

func (e ErrorMatcher) Matches(x interface{}) bool {
	if err, ok := x.(error); ok {
		return err.Error() == e.Expected
	}
	return false
}

func (e ErrorMatcher) String() string {
	return fmt.Sprintf("error with message: %s", e.Expected)
}

// NewErrorMatcher creates a new error matcher
func NewErrorMatcher(message string) ErrorMatcher {
	return ErrorMatcher{Expected: message}
}
