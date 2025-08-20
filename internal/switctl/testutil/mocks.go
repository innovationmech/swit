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

package testutil

import (
	"errors"
	"sync"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/stretchr/testify/mock"
)

// MockTemplateEngine is a mock implementation of TemplateEngine for testing.
type MockTemplateEngine struct {
	mock.Mock
	mu sync.RWMutex
}

// LoadTemplate mocks the LoadTemplate method.
func (m *MockTemplateEngine) LoadTemplate(name string) (interfaces.Template, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(interfaces.Template), args.Error(1)
}

// RenderTemplate mocks the RenderTemplate method.
func (m *MockTemplateEngine) RenderTemplate(template interfaces.Template, data interface{}) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(template, data)
	return args.Get(0).([]byte), args.Error(1)
}

// RegisterFunction mocks the RegisterFunction method.
func (m *MockTemplateEngine) RegisterFunction(name string, fn interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(name, fn)
	return args.Error(0)
}

// SetTemplateDir mocks the SetTemplateDir method.
func (m *MockTemplateEngine) SetTemplateDir(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(dir)
	return args.Error(0)
}

// MockTemplate is a mock implementation of Template for testing.
type MockTemplate struct {
	mock.Mock
	name string
}

// Name returns the template name.
func (m *MockTemplate) Name() string {
	return m.name
}

// Render mocks the Render method.
func (m *MockTemplate) Render(data interface{}) ([]byte, error) {
	args := m.Called(data)
	return args.Get(0).([]byte), args.Error(1)
}

// NewMockTemplate creates a new mock template with the given name.
func NewMockTemplate(name string) *MockTemplate {
	return &MockTemplate{name: name}
}

// MockGenerator is a mock implementation of Generator for testing.
type MockGenerator struct {
	mock.Mock
	mu sync.RWMutex
}

// GenerateService mocks the GenerateService method.
func (m *MockGenerator) GenerateService(config interfaces.ServiceConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(config)
	return args.Error(0)
}

// GenerateAPI mocks the GenerateAPI method.
func (m *MockGenerator) GenerateAPI(config interfaces.APIConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(config)
	return args.Error(0)
}

// GenerateModel mocks the GenerateModel method.
func (m *MockGenerator) GenerateModel(config interfaces.ModelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(config)
	return args.Error(0)
}

// GenerateMiddleware mocks the GenerateMiddleware method.
func (m *MockGenerator) GenerateMiddleware(config interfaces.MiddlewareConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(config)
	return args.Error(0)
}

// MockFileSystem is a mock implementation of FileSystem for testing.
type MockFileSystem struct {
	mock.Mock
	files map[string][]byte
	mu    sync.RWMutex
}

// NewMockFileSystem creates a new mock file system.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
	}
}

// WriteFile mocks the WriteFile method.
func (m *MockFileSystem) WriteFile(path string, content []byte, perm int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(path, content, perm)
	if args.Error(0) == nil {
		m.files[path] = content
	}
	return args.Error(0)
}

// ReadFile mocks the ReadFile method.
func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	args := m.Called(path)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if content, exists := m.files[path]; exists {
		return content, nil
	}
	return args.Get(0).([]byte), args.Error(1)
}

// MkdirAll mocks the MkdirAll method.
func (m *MockFileSystem) MkdirAll(path string, perm int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(path, perm)
	return args.Error(0)
}

// Exists mocks the Exists method.
func (m *MockFileSystem) Exists(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	args := m.Called(path)
	return args.Bool(0)
}

// Remove mocks the Remove method.
func (m *MockFileSystem) Remove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(path)
	if args.Error(0) == nil {
		delete(m.files, path)
	}
	return args.Error(0)
}

// Copy mocks the Copy method.
func (m *MockFileSystem) Copy(src, dst string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(src, dst)
	if args.Error(0) == nil {
		if content, exists := m.files[src]; exists {
			m.files[dst] = content
		}
	}
	return args.Error(0)
}

// GetFiles returns all files in the mock file system.
func (m *MockFileSystem) GetFiles() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string][]byte)
	for k, v := range m.files {
		result[k] = v
	}
	return result
}

// MockTemplateStore is a mock implementation of a template store for testing.
type MockTemplateStore struct {
	mock.Mock
	templates map[string]interfaces.Template
	mu        sync.RWMutex
}

// NewMockTemplateStore creates a new mock template store.
func NewMockTemplateStore() *MockTemplateStore {
	return &MockTemplateStore{
		templates: make(map[string]interfaces.Template),
	}
}

// LoadTemplate mocks loading a template.
func (m *MockTemplateStore) LoadTemplate(name string) (interfaces.Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	args := m.Called(name)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if template, exists := m.templates[name]; exists {
		return template, nil
	}
	return args.Get(0).(interfaces.Template), args.Error(1)
}

// StoreTemplate stores a template in the mock store.
func (m *MockTemplateStore) StoreTemplate(name string, template interfaces.Template) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(name, template)
	if args.Error(0) == nil {
		m.templates[name] = template
	}
	return args.Error(0)
}

// ListTemplates mocks listing all templates.
func (m *MockTemplateStore) ListTemplates() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	args := m.Called()
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	var names []string
	for name := range m.templates {
		names = append(names, name)
	}
	return names, nil
}

// RemoveTemplate mocks removing a template.
func (m *MockTemplateStore) RemoveTemplate(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called(name)
	if args.Error(0) == nil {
		delete(m.templates, name)
	}
	return args.Error(0)
}

// ClearCache mocks clearing the template cache.
func (m *MockTemplateStore) ClearCache() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	args := m.Called()
	if args.Error(0) == nil {
		m.templates = make(map[string]interfaces.Template)
	}
	return args.Error(0)
}

// GetTemplate returns a stored template.
func (m *MockTemplateStore) GetTemplate(name string) (interfaces.Template, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	template, exists := m.templates[name]
	return template, exists
}

// MockInteractiveUI is a mock implementation of InteractiveUI for testing.
type MockInteractiveUI struct {
	mock.Mock
	responses map[string]interface{}
	mu        sync.RWMutex
}

// NewMockInteractiveUI creates a new mock interactive UI.
func NewMockInteractiveUI() *MockInteractiveUI {
	return &MockInteractiveUI{
		responses: make(map[string]interface{}),
	}
}

// ShowWelcome mocks the ShowWelcome method.
func (m *MockInteractiveUI) ShowWelcome() error {
	args := m.Called()
	return args.Error(0)
}

// PromptInput mocks the PromptInput method.
func (m *MockInteractiveUI) PromptInput(prompt string, validator interfaces.InputValidator) (string, error) {
	args := m.Called(prompt, validator)
	return args.String(0), args.Error(1)
}

// ShowMenu mocks the ShowMenu method.
func (m *MockInteractiveUI) ShowMenu(title string, options []interfaces.MenuOption) (int, error) {
	args := m.Called(title, options)
	return args.Int(0), args.Error(1)
}

// ShowProgress mocks the ShowProgress method.
func (m *MockInteractiveUI) ShowProgress(title string, total int) interfaces.ProgressBar {
	args := m.Called(title, total)
	return args.Get(0).(interfaces.ProgressBar)
}

// ShowSuccess mocks the ShowSuccess method.
func (m *MockInteractiveUI) ShowSuccess(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

// ShowError mocks the ShowError method.
func (m *MockInteractiveUI) ShowError(err error) error {
	args := m.Called(err)
	return args.Error(0)
}

// ShowTable mocks the ShowTable method.
func (m *MockInteractiveUI) ShowTable(headers []string, rows [][]string) error {
	args := m.Called(headers, rows)
	return args.Error(0)
}

// SetResponse sets a predefined response for testing.
func (m *MockInteractiveUI) SetResponse(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[key] = value
}

// MockProgressBar is a mock implementation of ProgressBar for testing.
type MockProgressBar struct {
	mock.Mock
	current int
	total   int
	mu      sync.RWMutex
}

// NewMockProgressBar creates a new mock progress bar.
func NewMockProgressBar(total int) *MockProgressBar {
	return &MockProgressBar{total: total}
}

// Update mocks the Update method.
func (m *MockProgressBar) Update(current int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = current
	args := m.Called(current)
	return args.Error(0)
}

// SetMessage mocks the SetMessage method.
func (m *MockProgressBar) SetMessage(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

// Finish mocks the Finish method.
func (m *MockProgressBar) Finish() error {
	args := m.Called()
	return args.Error(0)
}

// SetTotal mocks the SetTotal method.
func (m *MockProgressBar) SetTotal(total int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total = total
	args := m.Called(total)
	return args.Error(0)
}

// GetCurrent returns the current progress.
func (m *MockProgressBar) GetCurrent() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// GetTotal returns the total progress.
func (m *MockProgressBar) GetTotal() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.total
}

// ErrorTemplateEngine is a mock that always returns errors for testing error scenarios.
type ErrorTemplateEngine struct{}

// LoadTemplate always returns an error.
func (e *ErrorTemplateEngine) LoadTemplate(name string) (interfaces.Template, error) {
	return nil, errors.New("template not found: " + name)
}

// RenderTemplate always returns an error.
func (e *ErrorTemplateEngine) RenderTemplate(template interfaces.Template, data interface{}) ([]byte, error) {
	return nil, errors.New("template rendering failed")
}

// RegisterFunction always returns an error.
func (e *ErrorTemplateEngine) RegisterFunction(name string, fn interface{}) error {
	return errors.New("function registration failed: " + name)
}

// SetTemplateDir always returns an error.
func (e *ErrorTemplateEngine) SetTemplateDir(dir string) error {
	return errors.New("invalid template directory: " + dir)
}

// ErrorTemplate is a mock template that always returns errors.
type ErrorTemplate struct {
	name string
}

// NewErrorTemplate creates a new error template.
func NewErrorTemplate(name string) *ErrorTemplate {
	return &ErrorTemplate{name: name}
}

// Name returns the template name.
func (e *ErrorTemplate) Name() string {
	return e.name
}

// Render always returns an error.
func (e *ErrorTemplate) Render(data interface{}) ([]byte, error) {
	return nil, errors.New("template render error: " + e.name)
}

// FailingFileSystem is a mock file system that simulates various file operation failures.
type FailingFileSystem struct {
	FailOn map[string]bool // Map of operations that should fail
}

// NewFailingFileSystem creates a new failing file system mock.
func NewFailingFileSystem(failOn ...string) *FailingFileSystem {
	fails := make(map[string]bool)
	for _, op := range failOn {
		fails[op] = true
	}
	return &FailingFileSystem{FailOn: fails}
}

// WriteFile simulates write failures.
func (f *FailingFileSystem) WriteFile(path string, content []byte, perm int) error {
	if f.FailOn["write"] {
		return errors.New("write operation failed")
	}
	return nil
}

// ReadFile simulates read failures.
func (f *FailingFileSystem) ReadFile(path string) ([]byte, error) {
	if f.FailOn["read"] {
		return nil, errors.New("read operation failed")
	}
	return []byte("mock content"), nil
}

// MkdirAll simulates directory creation failures.
func (f *FailingFileSystem) MkdirAll(path string, perm int) error {
	if f.FailOn["mkdir"] {
		return errors.New("mkdir operation failed")
	}
	return nil
}

// Exists simulates existence check failures.
func (f *FailingFileSystem) Exists(path string) bool {
	if f.FailOn["exists"] {
		return false
	}
	return true
}

// Remove simulates removal failures.
func (f *FailingFileSystem) Remove(path string) error {
	if f.FailOn["remove"] {
		return errors.New("remove operation failed")
	}
	return nil
}

// Copy simulates copy failures.
func (f *FailingFileSystem) Copy(src, dst string) error {
	if f.FailOn["copy"] {
		return errors.New("copy operation failed")
	}
	return nil
}
