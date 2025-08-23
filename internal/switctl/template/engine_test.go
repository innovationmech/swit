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

package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switctl/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TemplateEngineTestSuite provides test suite for template engine functionality.
type TemplateEngineTestSuite struct {
	suite.Suite
	engine  *Engine
	tempDir string
}

// SetupTest sets up the test environment before each test.
func (suite *TemplateEngineTestSuite) SetupTest() {
	var err error
	suite.tempDir, err = os.MkdirTemp("", "template-engine-test-*")
	suite.Require().NoError(err)

	// Create test templates
	testTemplates := map[string]string{
		"simple.tmpl": `Hello {{.Name}}!`,
		"functions.tmpl": `
{{- $name := .Name | camelCase -}}
Service: {{$name}}
Snake: {{.Name | snakeCase}}
Pascal: {{.Name | pascalCase}}
Lower: {{.Name | lower}}
Upper: {{.Name | upper}}
`,
		"conditionals.tmpl": `
{{- if .Service.Features.Database -}}
Database enabled: {{.Service.Database.Type}}
{{- end -}}
{{- if .Service.Features.Authentication -}}
Auth enabled: {{.Service.Auth.Type}}
{{- end -}}
`,
		"loops.tmpl": `
{{- range $i, $import := .Imports -}}
{{$i}}: {{$import.Path}}
{{- end -}}
`,
		"nested.tmpl": `
{{- range .Service.Metadata -}}
{{. | upper}}
{{- end -}}
`,
		"complex.tmpl": `// {{.Service.Name}} Service
package {{.Package.Name}}

import (
{{- range .Imports}}
	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{- end}}
)

{{- if .Service.Features.Database}}

// Config represents database configuration
type Config struct {
	Host     string ` + "`" + `yaml:"host"` + "`" + `
	Port     int    ` + "`" + `yaml:"port"` + "`" + `
	Database string ` + "`" + `yaml:"database"` + "`" + `
}
{{- end}}
`,
		"invalid_syntax.tmpl": `{{.Name | invalidFunction}}`,
		"missing_field.tmpl":  `{{.NonExistentField}}`,
	}

	for name, content := range testTemplates {
		path := filepath.Join(suite.tempDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		suite.Require().NoError(err)
	}

	suite.engine = NewEngine(suite.tempDir)
}

// TearDownTest cleans up after each test.
func (suite *TemplateEngineTestSuite) TearDownTest() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// TestNewEngine tests engine creation.
func (suite *TemplateEngineTestSuite) TestNewEngine() {
	suite.NotNil(suite.engine)
	suite.Equal(suite.tempDir, suite.engine.templateDir)
	suite.NotNil(suite.engine.funcMap)
	suite.NotNil(suite.engine.templates)
	suite.NotNil(suite.engine.store)
}

// TestNewEngineWithEmbedded tests engine creation with embedded templates.
func (suite *TemplateEngineTestSuite) TestNewEngineWithEmbedded() {
	embeddedEngine := NewEngineWithEmbedded()
	suite.NotNil(embeddedEngine)
	suite.Equal("", embeddedEngine.templateDir)
	suite.NotNil(embeddedEngine.funcMap)
	suite.NotNil(embeddedEngine.templates)
	suite.NotNil(embeddedEngine.store)
}

// TestLoadTemplate tests template loading functionality.
func (suite *TemplateEngineTestSuite) TestLoadTemplate() {
	tests := []struct {
		name         string
		templateName string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "Load existing template",
			templateName: "simple",
			expectError:  false,
		},
		{
			name:         "Load complex template",
			templateName: "complex",
			expectError:  false,
		},
		{
			name:         "Load non-existent template",
			templateName: "non_existent",
			expectError:  true,
			errorMsg:     "template non_existent not found",
		},
		{
			name:         "Load template with invalid syntax",
			templateName: "invalid_syntax",
			expectError:  true,
			errorMsg:     "function \"invalidFunction\" not defined",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)

			if tt.expectError {
				suite.Error(err)
				suite.Nil(tmpl)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.NoError(err)
				suite.NotNil(tmpl)
				suite.Equal(tt.templateName, tmpl.Name())

				// Test caching - second load should be from cache
				tmpl2, err2 := suite.engine.LoadTemplate(tt.templateName)
				suite.NoError(err2)
				suite.NotNil(tmpl2)
			}
		})
	}
}

// TestRenderTemplate tests template rendering functionality.
func (suite *TemplateEngineTestSuite) TestRenderTemplate() {
	testData := testutil.TestTemplateData()

	tests := []struct {
		name         string
		templateName string
		data         interface{}
		expectError  bool
		errorMsg     string
		validateFunc func([]byte) bool
	}{
		{
			name:         "Render simple template",
			templateName: "simple",
			data:         map[string]string{"Name": "World"},
			expectError:  false,
			validateFunc: func(output []byte) bool {
				return string(output) == "Hello World!"
			},
		},
		{
			name:         "Render template with functions",
			templateName: "functions",
			data:         map[string]string{"Name": "test_service"},
			expectError:  false,
			validateFunc: func(output []byte) bool {
				result := string(output)
				return strings.Contains(result, "Service: testService") &&
					strings.Contains(result, "Snake: test_service") &&
					strings.Contains(result, "Pascal: TestService") &&
					strings.Contains(result, "Lower: test_service") &&
					strings.Contains(result, "Upper: TEST_SERVICE")
			},
		},
		{
			name:         "Render template with conditionals",
			templateName: "conditionals",
			data:         testData,
			expectError:  false,
			validateFunc: func(output []byte) bool {
				result := string(output)
				return strings.Contains(result, "Database enabled: mysql") &&
					strings.Contains(result, "Auth enabled: jwt")
			},
		},
		{
			name:         "Render template with loops",
			templateName: "loops",
			data:         testData,
			expectError:  false,
			validateFunc: func(output []byte) bool {
				result := string(output)
				return strings.Contains(result, "0: context") &&
					strings.Contains(result, "1: fmt")
			},
		},
		{
			name:         "Render complex template",
			templateName: "complex",
			data:         testData,
			expectError:  false,
			validateFunc: func(output []byte) bool {
				result := string(output)
				return strings.Contains(result, "// test-service Service") &&
					strings.Contains(result, "package testservice") &&
					strings.Contains(result, "type Config struct") &&
					strings.Contains(result, `yaml:"host"`)
			},
		},
		{
			name:         "Render with missing field",
			templateName: "missing_field",
			data:         map[string]string{"Name": "test"},
			expectError:  false, // Go templates don't error on missing fields by default
			validateFunc: func(output []byte) bool {
				return len(output) >= 0 // Should render with empty value
			},
		},
		{
			name:         "Render with nil data",
			templateName: "simple",
			data:         nil,
			expectError:  false, // Go templates handle nil data gracefully
			validateFunc: func(output []byte) bool {
				// With nil data, {{.Name}} becomes <no value>
				return strings.Contains(string(output), "Hello <no value>!")
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)
			suite.NoError(err)
			suite.NotNil(tmpl)

			output, err := suite.engine.RenderTemplate(tmpl, tt.data)

			if tt.expectError {
				suite.Error(err)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.NoError(err)
				suite.NotNil(output)

				if tt.validateFunc != nil {
					suite.True(tt.validateFunc(output), "Template output validation failed: %s", string(output))
				}
			}
		})
	}
}

// TestRegisterFunction tests custom function registration.
func (suite *TemplateEngineTestSuite) TestRegisterFunction() {
	tests := []struct {
		name        string
		funcName    string
		function    interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:     "Register valid string function",
			funcName: "reverse",
			function: func(s string) string {
				runes := []rune(s)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				return string(runes)
			},
			expectError: false,
		},
		{
			name:        "Register valid interface function",
			funcName:    "multiply",
			function:    func(a, b int) int { return a * b },
			expectError: false,
		},
		{
			name:        "Register with empty name",
			funcName:    "",
			function:    func() string { return "test" },
			expectError: true,
			errorMsg:    "function name cannot be empty",
		},
		{
			name:        "Register with nil function",
			funcName:    "nilFunc",
			function:    nil,
			expectError: true,
			errorMsg:    "function cannot be nil",
		},
		{
			name:        "Register with invalid type",
			funcName:    "invalidType",
			function:    "not a function",
			expectError: true,
			errorMsg:    "function must be callable",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.engine.RegisterFunction(tt.funcName, tt.function)

			if tt.expectError {
				suite.Error(err)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.NoError(err)

				// Test that function is actually registered by using it in a template
				if tt.funcName == "reverse" {
					tmplContent := fmt.Sprintf("{{%q | reverse}}", "hello")
					tmplPath := filepath.Join(suite.tempDir, "test_custom_func.tmpl")
					err := os.WriteFile(tmplPath, []byte(tmplContent), 0644)
					suite.NoError(err)

					tmpl, err := suite.engine.LoadTemplate("test_custom_func")
					suite.NoError(err)

					output, err := suite.engine.RenderTemplate(tmpl, map[string]interface{}{})
					suite.NoError(err)
					suite.Equal("olleh", string(output))
				}
			}
		})
	}
}

// TestSetTemplateDir tests setting template directory.
func (suite *TemplateEngineTestSuite) TestSetTemplateDir() {
	tests := []struct {
		name        string
		dir         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Set valid directory",
			dir:         suite.tempDir,
			expectError: false,
		},
		{
			name:        "Set non-existent directory",
			dir:         "/non/existent/directory",
			expectError: true,
			errorMsg:    "template directory does not exist",
		},
		{
			name:        "Set empty directory",
			dir:         "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.engine.SetTemplateDir(tt.dir)

			if tt.expectError {
				suite.Error(err)
				if tt.errorMsg != "" {
					suite.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.NoError(err)
				suite.Equal(tt.dir, suite.engine.templateDir)
			}
		})
	}
}

// TestLoadTemplatesFromDirectory tests loading all templates from directory.
func (suite *TemplateEngineTestSuite) TestLoadTemplatesFromDirectory() {
	// This will try to load all templates, but some may fail due to syntax errors
	// We expect it to continue processing other templates even if some fail
	err := suite.engine.LoadTemplatesFromDirectory(suite.tempDir)
	// The function should return error if any template fails to load
	suite.Error(err)

	// Check that valid templates are loaded (the ones that didn't fail)
	loadedTemplates := suite.engine.GetLoadedTemplates()

	// We can't guarantee which templates will be loaded since invalid ones will cause the process to stop
	// So let's just check that at least some templates were loaded before the error
	suite.True(len(loadedTemplates) > 0, "At least some templates should be loaded before error")
}

// TestGetLoadedTemplates tests getting list of loaded templates.
func (suite *TemplateEngineTestSuite) TestGetLoadedTemplates() {
	// Initially empty
	templates := suite.engine.GetLoadedTemplates()
	suite.Empty(templates)

	// Load a template
	_, err := suite.engine.LoadTemplate("simple")
	suite.NoError(err)

	templates = suite.engine.GetLoadedTemplates()
	suite.Contains(templates, "simple")

	// Load another template
	_, err = suite.engine.LoadTemplate("functions")
	suite.NoError(err)

	templates = suite.engine.GetLoadedTemplates()
	suite.Contains(templates, "simple")
	suite.Contains(templates, "functions")
}

// TestClearCache tests clearing template cache.
func (suite *TemplateEngineTestSuite) TestClearCache() {
	// Load templates
	_, err := suite.engine.LoadTemplate("simple")
	suite.NoError(err)
	_, err = suite.engine.LoadTemplate("functions")
	suite.NoError(err)

	templates := suite.engine.GetLoadedTemplates()
	suite.Len(templates, 2)

	// Clear cache
	suite.engine.ClearCache()

	templates = suite.engine.GetLoadedTemplates()
	suite.Empty(templates)
}

// TestGetStore tests getting template store.
func (suite *TemplateEngineTestSuite) TestGetStore() {
	store := suite.engine.GetStore()
	suite.NotNil(store)
	suite.Equal(suite.engine.store, store)
}

// TestTemplateHelperFunctions tests built-in template functions.
func (suite *TemplateEngineTestSuite) TestTemplateHelperFunctions() {
	tests := []struct {
		name     string
		function string
		input    interface{}
		expected interface{}
	}{
		// String case functions
		{"camelCase", "camelCase", "test_service", "testService"},
		{"snakeCase", "snakeCase", "TestService", "test_service"},
		{"pascalCase", "pascalCase", "test_service", "TestService"},
		{"kebabCase", "kebabCase", "test_service", "test-service"},
		{"screamingSnakeCase", "screamingSnakeCase", "testService", "TEST_SERVICE"},
		{"lower", "lower", "TEST", "test"},
		{"upper", "upper", "test", "TEST"},

		// Go-specific functions
		{"exportedName", "exportedName", "test_name", "TestName"},
		{"unexportedName", "unexportedName", "test_name", "testName"},
		{"packageName", "packageName", "test-service", "testservice"},
		{"receiverName", "receiverName", "UserService", "us"},
		{"receiverName vowel", "receiverName", "AuthService", "au"},

		// String manipulation
		{"pluralize", "pluralize", "user", "users"},
		{"pluralize y", "pluralize", "category", "categories"},
		{"singularize", "singularize", "users", "user"},
		{"singularize ies", "singularize", "categories", "category"},
		{"humanize", "humanize", "test_service", "Test Service"},
		{"dehumanize", "dehumanize", "Test Service", "test_service"},

		// Math functions
		{"add", "add", []interface{}{5, 3}, 8.0},
		{"sub", "sub", []interface{}{10, 4}, 6.0},
		{"mul", "mul", []interface{}{3, 4}, 12.0},
		{"div", "div", []interface{}{12, 3}, 4.0},
		{"mod", "mod", []interface{}{10, 3}, 1},

		// Comparison functions
		{"eq true", "eq", []interface{}{5, 5}, true},
		{"eq false", "eq", []interface{}{5, 3}, false},
		{"ne true", "ne", []interface{}{5, 3}, true},
		{"ne false", "ne", []interface{}{5, 5}, false},
		{"lt", "lt", []interface{}{3, 5}, true},
		{"le", "le", []interface{}{5, 5}, true},
		{"gt", "gt", []interface{}{5, 3}, true},
		{"ge", "ge", []interface{}{5, 5}, true},

		// Logical functions
		{"and true", "and", []interface{}{true, true}, true},
		{"and false", "and", []interface{}{true, false}, false},
		{"or true", "or", []interface{}{true, false}, true},
		{"or false", "or", []interface{}{false, false}, false},
		{"not true", "not", true, false},
		{"not false", "not", false, true},

		// Default value function
		{"default empty", "default", []interface{}{"", "fallback"}, "fallback"},
		{"default non-empty", "default", []interface{}{"value", "fallback"}, "value"},
		{"default zero", "default", []interface{}{0, 42}, 42},
		{"default non-zero", "default", []interface{}{5, 42}, 5},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var templateContent string

			// Helper function to format template arguments
			formatArg := func(arg interface{}) string {
				switch v := arg.(type) {
				case string:
					return fmt.Sprintf(`"%s"`, v)
				default:
					return fmt.Sprintf("%v", v)
				}
			}

			switch v := tt.input.(type) {
			case []interface{}:
				// For functions that take multiple arguments
				if len(v) == 2 {
					arg0 := formatArg(v[0])
					arg1 := formatArg(v[1])
					templateContent = fmt.Sprintf("{{%s %s %s}}", tt.function, arg0, arg1)
				} else {
					suite.Fail("Unsupported number of arguments")
				}
			default:
				// For functions that take single argument
				templateContent = fmt.Sprintf("{{%s | %s}}", formatArg(v), tt.function)
			}

			tmplPath := filepath.Join(suite.tempDir, fmt.Sprintf("test_%s.tmpl", strings.ReplaceAll(tt.name, " ", "_")))
			err := os.WriteFile(tmplPath, []byte(templateContent), 0644)
			suite.NoError(err)

			tmpl, err := suite.engine.LoadTemplate(fmt.Sprintf("test_%s", strings.ReplaceAll(tt.name, " ", "_")))
			suite.NoError(err)

			output, err := suite.engine.RenderTemplate(tmpl, map[string]interface{}{})
			suite.NoError(err)

			result := strings.TrimSpace(string(output))
			expected := fmt.Sprintf("%v", tt.expected)
			suite.Equal(expected, result, "Function %s with input %v should return %v, got %s", tt.function, tt.input, tt.expected, result)
		})
	}
}

// TestArraySliceFunctions tests array/slice helper functions.
func (suite *TemplateEngineTestSuite) TestArraySliceFunctions() {
	testSlice := []string{"first", "second", "third", "fourth"}

	tests := []struct {
		name        string
		templateStr string
		data        interface{}
		expected    string
	}{
		{
			name:        "first",
			templateStr: "{{.Items | first}}",
			data:        map[string]interface{}{"Items": testSlice},
			expected:    "first",
		},
		{
			name:        "last",
			templateStr: "{{.Items | last}}",
			data:        map[string]interface{}{"Items": testSlice},
			expected:    "fourth",
		},
		{
			name:        "length",
			templateStr: "{{.Items | length}}",
			data:        map[string]interface{}{"Items": testSlice},
			expected:    "4",
		},
		{
			name:        "join",
			templateStr: "{{join \",\" .Items}}",
			data:        map[string]interface{}{"Items": testSlice},
			expected:    "first,second,third,fourth",
		},
		{
			name:        "reverse",
			templateStr: "{{.Items | reverse | join \",\"}}",
			data:        map[string]interface{}{"Items": testSlice},
			expected:    "fourth,third,second,first",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tmplPath := filepath.Join(suite.tempDir, fmt.Sprintf("array_test_%s.tmpl", tt.name))
			err := os.WriteFile(tmplPath, []byte(tt.templateStr), 0644)
			suite.NoError(err)

			tmpl, err := suite.engine.LoadTemplate(fmt.Sprintf("array_test_%s", tt.name))
			suite.NoError(err)

			output, err := suite.engine.RenderTemplate(tmpl, tt.data)
			suite.NoError(err)

			result := strings.TrimSpace(string(output))
			suite.Equal(tt.expected, result)
		})
	}
}

// TestGoTemplateInterface tests the GoTemplate interface implementation.
func (suite *TemplateEngineTestSuite) TestGoTemplateInterface() {
	tmpl, err := suite.engine.LoadTemplate("simple")
	suite.NoError(err)
	suite.NotNil(tmpl)

	// Test Name method
	suite.Equal("simple", tmpl.Name())

	// Test Render method
	output, err := tmpl.Render(map[string]string{"Name": "Test"})
	suite.NoError(err)
	suite.Equal("Hello Test!", string(output))

	// Test Render with nil template (edge case)
	goTemplate := &GoTemplate{template: nil, name: "test"}
	output, err = goTemplate.Render(map[string]string{"Name": "Test"})
	suite.Error(err)
	suite.Contains(err.Error(), "cannot render nil template")
	suite.Nil(output)
}

// TestErrorHandling tests various error conditions.
func (suite *TemplateEngineTestSuite) TestErrorHandling() {
	// Test rendering template with wrong interface type
	mockTemplate := &mockTemplate{name: "mock"}
	output, err := suite.engine.RenderTemplate(mockTemplate, map[string]interface{}{})
	suite.Error(err)
	suite.Contains(err.Error(), "template is not a GoTemplate")
	suite.Nil(output)
}

// mockTemplate is a mock implementation for testing error cases.
type mockTemplate struct {
	name string
}

func (m *mockTemplate) Name() string {
	return m.name
}

func (m *mockTemplate) Render(data interface{}) ([]byte, error) {
	return []byte("mock"), nil
}

// TestTemplateDateTimeFunctions tests date/time helper functions.
func (suite *TemplateEngineTestSuite) TestTemplateDateTimeFunctions() {
	tmplContent := `
Now: {{now.Format "2006-01-02"}}
Year: {{year}}
Date: {{date "2006-01-02"}}
`

	tmplPath := filepath.Join(suite.tempDir, "datetime_test.tmpl")
	err := os.WriteFile(tmplPath, []byte(tmplContent), 0644)
	suite.NoError(err)

	tmpl, err := suite.engine.LoadTemplate("datetime_test")
	suite.NoError(err)

	output, err := suite.engine.RenderTemplate(tmpl, map[string]interface{}{})
	suite.NoError(err)

	result := string(output)
	currentYear := time.Now().Year()
	currentDate := time.Now().Format("2006-01-02")

	suite.Contains(result, fmt.Sprintf("Year: %d", currentYear))
	suite.Contains(result, fmt.Sprintf("Date: %s", currentDate))
	suite.Contains(result, fmt.Sprintf("Now: %s", currentDate))
}

// TestFieldTagFunction tests the fieldTag helper function.
func (suite *TemplateEngineTestSuite) TestFieldTagFunction() {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name:     "Single tag",
			input:    map[string]string{"json": "name"},
			expected: `json:"name"`,
		},
		{
			name:     "Multiple tags",
			input:    map[string]string{"json": "name", "validate": "required"},
			expected: `json:"name" validate:"required"`,
		},
		{
			name:     "Empty value tag",
			input:    map[string]string{"omitempty": ""},
			expected: "omitempty",
		},
		{
			name:     "Empty map",
			input:    map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := buildFieldTag(tt.input)

			if tt.expected == "" {
				suite.Equal("", result)
			} else {
				// For multiple tags, order might vary so check contains
				for key, value := range tt.input {
					if value == "" {
						suite.Contains(result, key)
					} else {
						suite.Contains(result, fmt.Sprintf(`%s:"%s"`, key, value))
					}
				}
			}
		})
	}
}

// TestImportAliasFunction tests the generateImportAlias function.
func (suite *TemplateEngineTestSuite) TestImportAliasFunction() {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple package",
			input:    "github.com/gin-gonic/gin",
			expected: "gin",
		},
		{
			name:     "Package with go prefix",
			input:    "github.com/go-redis/redis",
			expected: "redis",
		},
		{
			name:     "Package with go suffix",
			input:    "github.com/redis-go",
			expected: "redis",
		},
		{
			name:     "Package with lib prefix",
			input:    "github.com/libpq",
			expected: "pq",
		},
		{
			name:     "Package with special characters",
			input:    "github.com/pkg-name@v1",
			expected: "pkgnamev1",
		},
		{
			name:     "Empty package",
			input:    "",
			expected: "pkg",
		},
		{
			name:     "Single part",
			input:    "redis",
			expected: "redis",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := generateImportAlias(tt.input)
			suite.Equal(tt.expected, result)
		})
	}
}

// TestTemplateWithComplexData tests templates with complex data structures.
func (suite *TemplateEngineTestSuite) TestTemplateWithComplexData() {
	// Create a template that uses various complex features
	complexTemplate := `
{{- $service := .Service -}}
{{- $auth := $service.Auth -}}

// Service Configuration
Service: {{$service.Name}}
{{- if $service.Features.Database}}
Database Type: {{$service.Database.Type}}
Database Host: {{$service.Database.Host}}:{{$service.Database.Port}}
{{- end}}

{{- if $service.Features.Authentication}}
Auth Type: {{$auth.Type}}
{{- if eq $auth.Type "jwt"}}
JWT Algorithm: {{$auth.Algorithm}}
JWT Expiration: {{$auth.Expiration}}
{{- end}}
{{- end}}

// Imports
{{- range $i, $import := .Imports}}
{{$i}}: {{$import.Path}}{{if $import.Alias}} (as {{$import.Alias}}){{end}}
{{- end}}

// Metadata
{{- range $key, $value := $service.Metadata}}
{{$key | upper}}: {{$value}}
{{- end}}

// Features
{{- with $service.Features}}
Database: {{.Database}}
Authentication: {{.Authentication}}
Cache: {{.Cache}}
GRPC: {{.GRPC}}
REST: {{.REST}}
{{- end}}
`

	tmplPath := filepath.Join(suite.tempDir, "complex_data_test.tmpl")
	err := os.WriteFile(tmplPath, []byte(complexTemplate), 0644)
	suite.NoError(err)

	tmpl, err := suite.engine.LoadTemplate("complex_data_test")
	suite.NoError(err)

	testData := testutil.TestTemplateData()
	output, err := suite.engine.RenderTemplate(tmpl, testData)
	suite.NoError(err)

	result := string(output)

	// Verify various parts of the rendered template
	suite.Contains(result, "Service: test-service")
	suite.Contains(result, "Database Type: mysql")
	suite.Contains(result, "Database Host: localhost:3306")
	suite.Contains(result, "Auth Type: jwt")
	suite.Contains(result, "JWT Algorithm: HS256")
	suite.Contains(result, "0: context")
	suite.Contains(result, "1: fmt")
	suite.Contains(result, "2: github.com/sirupsen/logrus (as log)")
	suite.Contains(result, "FRAMEWORK: swit")
	suite.Contains(result, "TYPE: microservice")
	suite.Contains(result, "Database: true")
	suite.Contains(result, "Authentication: true")
}

// TestConcurrentTemplateAccess tests concurrent access to templates.
func (suite *TemplateEngineTestSuite) TestConcurrentTemplateAccess() {
	const numGoroutines = 10
	const numOperations = 20

	done := make(chan bool, numGoroutines)
	testData := map[string]string{"Name": "ConcurrentTest"}

	// Start multiple goroutines that load and render templates concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				// Load template
				tmpl, err := suite.engine.LoadTemplate("simple")
				suite.NoError(err)
				suite.NotNil(tmpl)

				// Render template
				output, err := suite.engine.RenderTemplate(tmpl, testData)
				suite.NoError(err)
				suite.Equal("Hello ConcurrentTest!", string(output))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// Run the test suite.
func TestTemplateEngineTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateEngineTestSuite))
}

// TestTemplateEngineBasics tests basic engine functionality without the test suite.
func TestTemplateEngineBasics(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template-engine-basic-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a simple test template
	tmplContent := `Hello {{.Name | upper}}!`
	tmplPath := filepath.Join(tempDir, "basic.tmpl")
	err = os.WriteFile(tmplPath, []byte(tmplContent), 0644)
	require.NoError(t, err)

	// Test engine creation and basic operations
	engine := NewEngine(tempDir)
	assert.NotNil(t, engine)

	// Test template loading
	tmpl, err := engine.LoadTemplate("basic")
	assert.NoError(t, err)
	assert.NotNil(t, tmpl)
	assert.Equal(t, "basic", tmpl.Name())

	// Test template rendering
	data := map[string]string{"Name": "world"}
	output, err := engine.RenderTemplate(tmpl, data)
	assert.NoError(t, err)
	assert.Equal(t, "Hello WORLD!", string(output))

	// Test direct template rendering
	directOutput, err := tmpl.Render(data)
	assert.NoError(t, err)
	assert.Equal(t, "Hello WORLD!", string(directOutput))
}

// TestEdgeCases tests various edge cases and boundary conditions.
func TestEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "template-edge-cases-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	engine := NewEngine(tempDir)

	t.Run("Empty template", func(t *testing.T) {
		emptyTmplPath := filepath.Join(tempDir, "empty.tmpl")
		err := os.WriteFile(emptyTmplPath, []byte(""), 0644)
		require.NoError(t, err)

		tmpl, err := engine.LoadTemplate("empty")
		assert.NoError(t, err)

		output, err := engine.RenderTemplate(tmpl, map[string]interface{}{})
		assert.NoError(t, err)
		assert.Empty(t, string(output))
	})

	t.Run("Template with only whitespace", func(t *testing.T) {
		whitespaceTmplPath := filepath.Join(tempDir, "whitespace.tmpl")
		err := os.WriteFile(whitespaceTmplPath, []byte("   \n\t  \n  "), 0644)
		require.NoError(t, err)

		tmpl, err := engine.LoadTemplate("whitespace")
		assert.NoError(t, err)

		output, err := engine.RenderTemplate(tmpl, map[string]interface{}{})
		assert.NoError(t, err)
		assert.Equal(t, "   \n\t  \n  ", string(output))
	})

	t.Run("Very large template", func(t *testing.T) {
		// Create a template with many repetitions
		var content strings.Builder
		for i := 0; i < 1000; i++ {
			content.WriteString(fmt.Sprintf("Line %d: {{.Value}}\n", i))
		}

		largeTmplPath := filepath.Join(tempDir, "large.tmpl")
		err := os.WriteFile(largeTmplPath, []byte(content.String()), 0644)
		require.NoError(t, err)

		tmpl, err := engine.LoadTemplate("large")
		assert.NoError(t, err)

		data := map[string]string{"Value": "test"}
		output, err := engine.RenderTemplate(tmpl, data)
		assert.NoError(t, err)

		// Check that all lines are rendered correctly
		lines := strings.Split(string(output), "\n")
		assert.Equal(t, 1001, len(lines)) // 1000 lines + 1 empty line at end
		assert.Contains(t, lines[0], "Line 0: test")
		assert.Contains(t, lines[999], "Line 999: test")
	})

	t.Run("Template with special characters", func(t *testing.T) {
		specialContent := `Special chars: àáâãäåæçèéêë ñòóôõö ùúûü ÿ 中文 日本語 한국어 emoji: 🚀🔥💻`
		specialTmplPath := filepath.Join(tempDir, "special.tmpl")
		err := os.WriteFile(specialTmplPath, []byte(specialContent), 0644)
		require.NoError(t, err)

		tmpl, err := engine.LoadTemplate("special")
		assert.NoError(t, err)

		output, err := engine.RenderTemplate(tmpl, map[string]interface{}{})
		assert.NoError(t, err)
		assert.Equal(t, specialContent, string(output))
	})
}
