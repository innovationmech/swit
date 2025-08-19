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

package interfaces

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

// TypesTestSuite tests configuration types and structures
type TypesTestSuite struct {
	suite.Suite
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (s *TypesTestSuite) TestServiceConfig_JSONSerialization() {
	config := ServiceConfig{
		Name:        "test-service",
		Description: "Test service description",
		Author:      "Test Author",
		Version:     "1.0.0",
		Features: ServiceFeatures{
			Database:       true,
			Authentication: false,
			Cache:          true,
		},
		Database: DatabaseConfig{
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "testdb",
			Username: "testuser",
		},
		Auth: AuthConfig{
			Type:       "jwt",
			SecretKey:  "test-secret",
			Expiration: 15 * time.Minute,
		},
		Ports: PortConfig{
			HTTP:    8080,
			GRPC:    9090,
			Metrics: 9091,
			Health:  8081,
		},
		Metadata: map[string]string{
			"framework": "swit",
			"type":      "microservice",
		},
		ModulePath: "github.com/example/test-service",
		OutputDir:  "./output",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(config)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), jsonData)

	// Test JSON unmarshaling
	var unmarshaledConfig ServiceConfig
	err = json.Unmarshal(jsonData, &unmarshaledConfig)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), config.Name, unmarshaledConfig.Name)
	assert.Equal(s.T(), config.Description, unmarshaledConfig.Description)
	assert.Equal(s.T(), config.Features.Database, unmarshaledConfig.Features.Database)
	assert.Equal(s.T(), config.Database.Type, unmarshaledConfig.Database.Type)
}

func (s *TypesTestSuite) TestServiceConfig_YAMLSerialization() {
	config := ServiceConfig{
		Name:        "yaml-service",
		Description: "YAML test service",
		Version:     "2.0.0",
		Features: ServiceFeatures{
			Monitoring: true,
			Tracing:    true,
			Docker:     true,
			Kubernetes: false,
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(config)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), yamlData)

	// Test YAML unmarshaling
	var unmarshaledConfig ServiceConfig
	err = yaml.Unmarshal(yamlData, &unmarshaledConfig)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), config.Name, unmarshaledConfig.Name)
	assert.Equal(s.T(), config.Description, unmarshaledConfig.Description)
	assert.Equal(s.T(), config.Features.Monitoring, unmarshaledConfig.Features.Monitoring)
}

func (s *TypesTestSuite) TestStreamingType_String() {
	tests := []struct {
		name     string
		value    StreamingType
		expected StreamingType
	}{
		{"none streaming", StreamingNone, "none"},
		{"client streaming", StreamingClient, "client"},
		{"server streaming", StreamingServer, "server"},
		{"bidirectional streaming", StreamingBidi, "bidi"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, tt.value)
		})
	}
}

func (s *TypesTestSuite) TestProjectType_String() {
	tests := []struct {
		name     string
		value    ProjectType
		expected ProjectType
	}{
		{"monolith", ProjectTypeMonolith, "monolith"},
		{"microservice", ProjectTypeMicroservice, "microservice"},
		{"library", ProjectTypeLibrary, "library"},
		{"api gateway", ProjectTypeAPIGateway, "api_gateway"},
		{"cli", ProjectTypeCLI, "cli"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, tt.value)
		})
	}
}

func (s *TypesTestSuite) TestMenuOption_Structure() {
	option := MenuOption{
		Label:       "Generate Service",
		Description: "Generate a new microservice",
		Value:       "generate-service",
		Icon:        "⚙️",
		Enabled:     true,
	}

	assert.Equal(s.T(), "Generate Service", option.Label)
	assert.Equal(s.T(), "Generate a new microservice", option.Description)
	assert.Equal(s.T(), "generate-service", option.Value)
	assert.Equal(s.T(), "⚙️", option.Icon)
	assert.True(s.T(), option.Enabled)
}

func (s *TypesTestSuite) TestHTTPMethod_Structure() {
	method := HTTPMethod{
		Name:        "CreateUser",
		Method:      "POST",
		Path:        "/api/v1/users",
		Description: "Create a new user",
		Request: RequestModel{
			Type: "CreateUserRequest",
			Fields: []Field{
				{
					Name:     "name",
					Type:     "string",
					Required: true,
				},
				{
					Name:     "email",
					Type:     "string",
					Required: true,
				},
			},
		},
		Response: ResponseModel{
			Type: "CreateUserResponse",
			Fields: []Field{
				{
					Name: "id",
					Type: "string",
				},
				{
					Name: "created_at",
					Type: "time.Time",
				},
			},
		},
		Auth:       true,
		Middleware: []string{"auth", "rate-limit"},
		Tags:       []string{"users", "crud"},
		Headers:    map[string]string{"Content-Type": "application/json"},
	}

	assert.Equal(s.T(), "CreateUser", method.Name)
	assert.Equal(s.T(), "POST", method.Method)
	assert.Equal(s.T(), "/api/v1/users", method.Path)
	assert.True(s.T(), method.Auth)
	assert.Len(s.T(), method.Middleware, 2)
	assert.Contains(s.T(), method.Middleware, "auth")
	assert.Contains(s.T(), method.Middleware, "rate-limit")
	assert.Len(s.T(), method.Tags, 2)
	assert.Equal(s.T(), "application/json", method.Headers["Content-Type"])
}

func (s *TypesTestSuite) TestGRPCMethod_Structure() {
	method := GRPCMethod{
		Name:        "GetUser",
		Description: "Retrieve user by ID",
		Request: RequestModel{
			Type: "GetUserRequest",
			Fields: []Field{
				{
					Name:     "id",
					Type:     "string",
					Required: true,
				},
			},
		},
		Response: ResponseModel{
			Type: "GetUserResponse",
			Fields: []Field{
				{
					Name: "user",
					Type: "User",
				},
			},
		},
		Streaming: StreamingNone,
	}

	assert.Equal(s.T(), "GetUser", method.Name)
	assert.Equal(s.T(), "Retrieve user by ID", method.Description)
	assert.Equal(s.T(), StreamingNone, method.Streaming)
	assert.Equal(s.T(), "GetUserRequest", method.Request.Type)
	assert.Equal(s.T(), "GetUserResponse", method.Response.Type)
}

func (s *TypesTestSuite) TestField_Structure() {
	field := Field{
		Name:        "username",
		Type:        "string",
		Tags:        map[string]string{"json": "username", "db": "username"},
		Required:    true,
		Unique:      true,
		Index:       true,
		Default:     "",
		Description: "User's unique username",
		Validation: []ValidationRule{
			{
				Type:    "min_length",
				Value:   3,
				Message: "Username must be at least 3 characters",
			},
			{
				Type:    "max_length",
				Value:   50,
				Message: "Username must not exceed 50 characters",
			},
		},
	}

	assert.Equal(s.T(), "username", field.Name)
	assert.Equal(s.T(), "string", field.Type)
	assert.True(s.T(), field.Required)
	assert.True(s.T(), field.Unique)
	assert.True(s.T(), field.Index)
	assert.Equal(s.T(), "", field.Default)
	assert.Len(s.T(), field.Validation, 2)
	assert.Equal(s.T(), "min_length", field.Validation[0].Type)
	assert.Equal(s.T(), 3, field.Validation[0].Value)
}

func (s *TypesTestSuite) TestIndex_Structure() {
	index := Index{
		Name:   "idx_user_email",
		Fields: []string{"email"},
		Unique: true,
		Type:   "btree",
	}

	assert.Equal(s.T(), "idx_user_email", index.Name)
	assert.Len(s.T(), index.Fields, 1)
	assert.Contains(s.T(), index.Fields, "email")
	assert.True(s.T(), index.Unique)
	assert.Equal(s.T(), "btree", index.Type)
}

func (s *TypesTestSuite) TestRelation_Structure() {
	relation := Relation{
		Type:       "has_many",
		Model:      "Post",
		ForeignKey: "user_id",
		LocalKey:   "id",
		PivotTable: "",
		Through:    "",
	}

	assert.Equal(s.T(), "has_many", relation.Type)
	assert.Equal(s.T(), "Post", relation.Model)
	assert.Equal(s.T(), "user_id", relation.ForeignKey)
	assert.Equal(s.T(), "id", relation.LocalKey)
	assert.Empty(s.T(), relation.PivotTable)
	assert.Empty(s.T(), relation.Through)
}

func (s *TypesTestSuite) TestCheckStatus_Values() {
	tests := []struct {
		name   string
		status CheckStatus
	}{
		{"pass", CheckStatusPass},
		{"fail", CheckStatusFail},
		{"warning", CheckStatusWarning},
		{"skip", CheckStatusSkip},
		{"error", CheckStatusError},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), CheckStatus(tt.name), tt.status)
		})
	}
}

func (s *TypesTestSuite) TestCheckResult_Structure() {
	result := CheckResult{
		Name:    "lint-check",
		Status:  CheckStatusPass,
		Message: "All lint checks passed",
		Details: []CheckDetail{
			{
				File:     "main.go",
				Line:     10,
				Column:   5,
				Message:  "Line too long",
				Rule:     "line-length",
				Severity: "warning",
				Context:  map[string]interface{}{"max_length": 120},
			},
		},
		Duration: 2 * time.Second,
		Score:    95,
		MaxScore: 100,
		Fixable:  true,
	}

	assert.Equal(s.T(), "lint-check", result.Name)
	assert.Equal(s.T(), CheckStatusPass, result.Status)
	assert.Equal(s.T(), "All lint checks passed", result.Message)
	assert.Len(s.T(), result.Details, 1)
	assert.Equal(s.T(), "main.go", result.Details[0].File)
	assert.Equal(s.T(), 10, result.Details[0].Line)
	assert.Equal(s.T(), 2*time.Second, result.Duration)
	assert.Equal(s.T(), 95, result.Score)
	assert.Equal(s.T(), 100, result.MaxScore)
	assert.True(s.T(), result.Fixable)
}

func (s *TypesTestSuite) TestTestResult_Structure() {
	result := TestResult{
		TotalTests:   100,
		PassedTests:  95,
		FailedTests:  3,
		SkippedTests: 2,
		Coverage:     85.5,
		Duration:     30 * time.Second,
		Failures: []TestFailure{
			{
				Test:    "TestUserService_CreateUser",
				Package: "github.com/example/service/user",
				Message: "Expected user ID to be generated",
				Output:  "assertion failed: expected non-empty user ID",
				File:    "user_test.go",
				Line:    45,
			},
		},
		Packages: []PackageTestResult{
			{
				Package:  "github.com/example/service/user",
				Tests:    50,
				Passed:   48,
				Failed:   2,
				Coverage: 90.0,
				Duration: 15 * time.Second,
			},
		},
	}

	assert.Equal(s.T(), 100, result.TotalTests)
	assert.Equal(s.T(), 95, result.PassedTests)
	assert.Equal(s.T(), 3, result.FailedTests)
	assert.Equal(s.T(), 2, result.SkippedTests)
	assert.Equal(s.T(), 85.5, result.Coverage)
	assert.Equal(s.T(), 30*time.Second, result.Duration)
	assert.Len(s.T(), result.Failures, 1)
	assert.Len(s.T(), result.Packages, 1)
	assert.Equal(s.T(), "TestUserService_CreateUser", result.Failures[0].Test)
	assert.Equal(s.T(), "github.com/example/service/user", result.Packages[0].Package)
}

func (s *TypesTestSuite) TestValidationResult_Error() {
	tests := []struct {
		name     string
		result   ValidationResult
		expected string
	}{
		{
			name: "valid result",
			result: ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			},
			expected: "",
		},
		{
			name: "invalid with errors",
			result: ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{
						Field:   "name",
						Message: "Name is required",
						Value:   "",
						Rule:    "required",
					},
				},
			},
			expected: "validation failed: Name is required",
		},
		{
			name: "invalid without errors",
			result: ValidationResult{
				Valid:  false,
				Errors: []ValidationError{},
			},
			expected: "validation failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, tt.result.Error())
		})
	}
}

func (s *TypesTestSuite) TestDependency_Structure() {
	dep := Dependency{
		Name:    "gin",
		Version: "v1.9.1",
		Type:    "direct",
		Direct:  true,
		ModPath: "github.com/gin-gonic/gin",
		License: "MIT",
		Updated: time.Now(),
	}

	assert.Equal(s.T(), "gin", dep.Name)
	assert.Equal(s.T(), "v1.9.1", dep.Version)
	assert.Equal(s.T(), "direct", dep.Type)
	assert.True(s.T(), dep.Direct)
	assert.Equal(s.T(), "github.com/gin-gonic/gin", dep.ModPath)
	assert.Equal(s.T(), "MIT", dep.License)
	assert.False(s.T(), dep.Updated.IsZero())
}

func (s *TypesTestSuite) TestTemplateData_Structure() {
	now := time.Now()
	data := TemplateData{
		Service: ServiceConfig{
			Name:    "test-service",
			Version: "1.0.0",
		},
		Package: PackageInfo{
			Name:       "testservice",
			Path:       "internal/testservice",
			ModulePath: "github.com/example/testservice",
			Version:    "1.0.0",
		},
		Imports: []ImportInfo{
			{
				Alias: "",
				Path:  "context",
			},
			{
				Alias: "log",
				Path:  "github.com/sirupsen/logrus",
			},
		},
		Functions: []FunctionInfo{
			{
				Name:     "NewTestService",
				Receiver: "",
				Parameters: []ParameterInfo{
					{
						Name: "config",
						Type: "*Config",
					},
				},
				Returns: []ReturnInfo{
					{
						Name: "",
						Type: "*TestService",
					},
					{
						Name: "",
						Type: "error",
					},
				},
				Body:    "return &TestService{config: config}, nil",
				Comment: "NewTestService creates a new test service instance",
			},
		},
		Metadata: map[string]string{
			"generator": "switctl",
			"version":   "0.1.0",
		},
		Timestamp: now,
		Author:    "Test Author",
		Version:   "1.0.0",
	}

	assert.Equal(s.T(), "test-service", data.Service.Name)
	assert.Equal(s.T(), "testservice", data.Package.Name)
	assert.Len(s.T(), data.Imports, 2)
	assert.Equal(s.T(), "context", data.Imports[0].Path)
	assert.Equal(s.T(), "log", data.Imports[1].Alias)
	assert.Len(s.T(), data.Functions, 1)
	assert.Equal(s.T(), "NewTestService", data.Functions[0].Name)
	assert.Equal(s.T(), "switctl", data.Metadata["generator"])
	assert.Equal(s.T(), now, data.Timestamp)
}

func (s *TypesTestSuite) TestProjectInfo_JSONSerialization() {
	now := time.Now()
	info := ProjectInfo{
		Name:          "test-project",
		Type:          "microservice",
		Version:       "1.0.0",
		Description:   "Test project description",
		Author:        "Test Author",
		License:       "MIT",
		ModulePath:    "github.com/example/test-project",
		GoVersion:     "1.19",
		CreatedAt:     now,
		UpdatedAt:     now,
		ServicesCount: 3,
		Dependencies:  []string{"gin", "gorm", "redis"},
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(info)
	assert.NoError(s.T(), err)

	var unmarshaled ProjectInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), info.Name, unmarshaled.Name)
	assert.Equal(s.T(), info.ServicesCount, unmarshaled.ServicesCount)
	assert.Equal(s.T(), info.Dependencies, unmarshaled.Dependencies)
}

// Benchmark tests
func BenchmarkServiceConfig_JSONMarshal(b *testing.B) {
	config := ServiceConfig{
		Name:        "benchmark-service",
		Description: "Benchmark test service",
		Version:     "1.0.0",
		Features: ServiceFeatures{
			Database:   true,
			Cache:      true,
			Monitoring: true,
			Tracing:    true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(config)
	}
}

func BenchmarkValidationResult_Error(b *testing.B) {
	result := ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{
				Field:   "test_field",
				Message: "Test validation error message",
				Value:   "invalid_value",
				Rule:    "test_rule",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.Error()
	}
}

// Edge case tests
func (s *TypesTestSuite) TestEmptyStructures() {
	// Test empty service config
	var config ServiceConfig
	jsonData, err := json.Marshal(config)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), jsonData)

	// Test empty validation result
	var result ValidationResult
	assert.Equal(s.T(), "validation failed", result.Error())

	// Test zero values
	assert.Equal(s.T(), StreamingNone, StreamingType("none"))
	assert.Equal(s.T(), ProjectTypeMonolith, ProjectType("monolith"))
}

func (s *TypesTestSuite) TestNilPointers() {
	// Test that structures handle nil pointers gracefully
	var data *TemplateData
	assert.Nil(s.T(), data)

	// Test nil slices
	var methods []HTTPMethod
	assert.Empty(s.T(), methods)
	assert.Len(s.T(), methods, 0)
}
