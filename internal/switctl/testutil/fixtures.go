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

// Package testutil provides common utilities and fixtures for testing switctl components.
package testutil

import (
	"os"
	"path/filepath"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// TestServiceConfig returns a sample service configuration for testing.
func TestServiceConfig() interfaces.ServiceConfig {
	return interfaces.ServiceConfig{
		Name:        "test-service",
		Description: "A test microservice for unit testing",
		Author:      "Test Author",
		Version:     "1.0.0",
		Features: interfaces.ServiceFeatures{
			Database:       true,
			Authentication: true,
			Cache:          true,
			Monitoring:     true,
			Tracing:        false,
			Docker:         true,
			Kubernetes:     false,
			GRPC:           true,
			REST:           true,
			HealthCheck:    true,
		},
		Database: interfaces.DatabaseConfig{
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
		},
		Auth: interfaces.AuthConfig{
			Type:       "jwt",
			SecretKey:  "test-secret-key",
			Expiration: 24 * time.Hour,
			Algorithm:  "HS256",
		},
		Ports: interfaces.PortConfig{
			HTTP:    8080,
			GRPC:    9090,
			Metrics: 9091,
			Health:  8081,
		},
		Metadata: map[string]string{
			"framework": "swit",
			"type":      "microservice",
			"category":  "business",
		},
		ModulePath: "github.com/example/test-service",
		OutputDir:  "./output",
	}
}

// TestAPIConfig returns a sample API configuration for testing.
func TestAPIConfig() interfaces.APIConfig {
	return interfaces.APIConfig{
		Name:     "test-api",
		Service:  "test-service",
		Version:  "v1",
		BasePath: "/api/v1",
		Methods: []interfaces.HTTPMethod{
			{
				Name:        "CreateUser",
				Method:      "POST",
				Path:        "/users",
				Description: "Create a new user",
				Request: interfaces.RequestModel{
					Type: "CreateUserRequest",
					Fields: []interfaces.Field{
						{
							Name:     "name",
							Type:     "string",
							Required: true,
							Tags:     map[string]string{"json": "name", "validate": "required"},
						},
						{
							Name:     "email",
							Type:     "string",
							Required: true,
							Tags:     map[string]string{"json": "email", "validate": "required,email"},
						},
					},
				},
				Response: interfaces.ResponseModel{
					Type: "CreateUserResponse",
					Fields: []interfaces.Field{
						{
							Name: "id",
							Type: "string",
							Tags: map[string]string{"json": "id"},
						},
						{
							Name: "created_at",
							Type: "time.Time",
							Tags: map[string]string{"json": "created_at"},
						},
					},
				},
				Auth:       true,
				Middleware: []string{"auth", "rate-limit"},
				Tags:       []string{"users", "crud"},
			},
			{
				Name:        "GetUser",
				Method:      "GET",
				Path:        "/users/{id}",
				Description: "Get user by ID",
				Request: interfaces.RequestModel{
					Type: "GetUserRequest",
					Fields: []interfaces.Field{
						{
							Name:     "id",
							Type:     "string",
							Required: true,
							Tags:     map[string]string{"uri": "id", "validate": "required"},
						},
					},
				},
				Response: interfaces.ResponseModel{
					Type: "GetUserResponse",
					Fields: []interfaces.Field{
						{
							Name: "user",
							Type: "User",
							Tags: map[string]string{"json": "user"},
						},
					},
				},
				Auth:       true,
				Middleware: []string{"auth"},
				Tags:       []string{"users"},
			},
		},
		GRPCMethods: []interfaces.GRPCMethod{
			{
				Name:        "CreateUser",
				Description: "Create a new user via gRPC",
				Request: interfaces.RequestModel{
					Type: "CreateUserRequest",
					Fields: []interfaces.Field{
						{
							Name: "name",
							Type: "string",
						},
						{
							Name: "email",
							Type: "string",
						},
					},
				},
				Response: interfaces.ResponseModel{
					Type: "CreateUserResponse",
					Fields: []interfaces.Field{
						{
							Name: "user",
							Type: "User",
						},
					},
				},
				Streaming: interfaces.StreamingNone,
			},
		},
		Models: []interfaces.Model{
			{
				Name:        "User",
				Description: "User entity model",
				Fields: []interfaces.Field{
					{
						Name:     "id",
						Type:     "string",
						Required: true,
						Unique:   true,
						Tags:     map[string]string{"json": "id", "gorm": "primaryKey"},
					},
					{
						Name:     "name",
						Type:     "string",
						Required: true,
						Tags:     map[string]string{"json": "name", "gorm": "not null"},
					},
					{
						Name:     "email",
						Type:     "string",
						Required: true,
						Unique:   true,
						Tags:     map[string]string{"json": "email", "gorm": "uniqueIndex"},
					},
					{
						Name: "created_at",
						Type: "time.Time",
						Tags: map[string]string{"json": "created_at"},
					},
					{
						Name: "updated_at",
						Type: "time.Time",
						Tags: map[string]string{"json": "updated_at"},
					},
				},
			},
		},
	}
}

// TestModelConfig returns a sample model configuration for testing.
func TestModelConfig() interfaces.ModelConfig {
	return interfaces.ModelConfig{
		Name:        "Product",
		Description: "Product entity model",
		Fields: []interfaces.Field{
			{
				Name:     "id",
				Type:     "string",
				Required: true,
				Unique:   true,
				Tags:     map[string]string{"json": "id", "gorm": "primaryKey"},
			},
			{
				Name:     "name",
				Type:     "string",
				Required: true,
				Tags:     map[string]string{"json": "name", "validate": "required"},
			},
			{
				Name: "price",
				Type: "float64",
				Tags: map[string]string{"json": "price", "validate": "min=0"},
			},
			{
				Name: "description",
				Type: "string",
				Tags: map[string]string{"json": "description"},
			},
			{
				Name: "category_id",
				Type: "string",
				Tags: map[string]string{"json": "category_id", "gorm": "index"},
			},
		},
		Relations: []interfaces.Relation{
			{
				Type:       "belongs_to",
				Model:      "Category",
				ForeignKey: "category_id",
				LocalKey:   "id",
			},
		},
		Indexes: []interfaces.Index{
			{
				Name:   "idx_product_name",
				Fields: []string{"name"},
				Unique: false,
				Type:   "btree",
			},
		},
		GenerateAPI:  true,
		GenerateCRUD: true,
		TableName:    "products",
	}
}

// TestTemplateData returns sample template data for testing.
func TestTemplateData() interfaces.TemplateData {
	return interfaces.TemplateData{
		Service: TestServiceConfig(),
		Package: interfaces.PackageInfo{
			Name:       "testservice",
			Path:       "internal/testservice",
			ModulePath: "github.com/example/test-service",
			Version:    "1.0.0",
		},
		Imports: []interfaces.ImportInfo{
			{
				Alias: "",
				Path:  "context",
			},
			{
				Alias: "",
				Path:  "fmt",
			},
			{
				Alias: "log",
				Path:  "github.com/sirupsen/logrus",
			},
		},
		Functions: []interfaces.FunctionInfo{
			{
				Name:     "NewTestService",
				Receiver: "",
				Parameters: []interfaces.ParameterInfo{
					{
						Name: "config",
						Type: "*Config",
					},
				},
				Returns: []interfaces.ReturnInfo{
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
			"template":  "service",
		},
		Timestamp: time.Now(),
		Author:    "Test Author",
		Version:   "1.0.0",
		Security: interfaces.SecurityConfig{
			TLS: interfaces.TLSConfig{
				Enabled:      true,
				Modern:       true,
				Intermediate: false,
				MutualAuth:   false,
				HSTS:         true,
				OCSP:         false,
				MinVersion:   "1.2",
				CertFile:     "/etc/ssl/certs/server.crt",
				KeyFile:      "/etc/ssl/private/server.key",
			},
			Headers: interfaces.HeadersConfig{
				Enabled:     true,
				Strict:      true,
				API:         false,
				Development: false,
				CSP:         "default-src 'self'",
				HPKP:        false,
				Conditional: false,
				HSTS:        true,
			},
			Config: interfaces.SecurityMode{
				Development: false,
				Production:  true,
			},
			Environment: "production",
			Mode:        "high",
		},
		Middleware: interfaces.MiddlewareTemplateConfig{
			CORS: interfaces.CORSMiddlewareConfig{
				Strict:      true,
				Development: false,
				Conditional: false,
				Logging:     true,
			},
			RateLimit: interfaces.RateLimitMiddlewareConfig{
				Advanced:    true,
				Sliding:     false,
				Distributed: false,
				Adaptive:    false,
			},
			RequestID: interfaces.RequestIDMiddlewareConfig{
				UUID:         true,
				Hierarchical: false,
				Sequential:   false,
				Tracing:      true,
				Correlation:  true,
			},
			Logging: interfaces.LoggingMiddlewareConfig{
				Structured:  true,
				Debug:       false,
				Conditional: false,
				Sampling:    true,
				Metrics:     true,
				Audit:       true,
			},
		},
	}
}

// TestTemplates returns a map of template content for testing.
func TestTemplates() map[string]string {
	return map[string]string{
		"service.go.tmpl": `// Package {{.Package.Name}} provides {{.Service.Description}}
package {{.Package.Name}}

import (
{{- range .Imports}}
	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{- end}}
)

// {{.Service.Name}} represents the main service
type {{.Service.Name | title}} struct {
	config *Config
}

{{range .Functions}}
// {{.Comment}}
func {{if .Receiver}}({{.Receiver.Name}} {{.Receiver.Type}}) {{end}}{{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}) ({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{if $r.Name}}{{$r.Name}} {{end}}{{$r.Type}}{{end}}) {
	{{.Body}}
}
{{end}}
`,
		"handler.go.tmpl": `// Package {{.Package.Name}} provides HTTP handlers
package {{.Package.Name}}

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Handler handles HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new HTTP handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetHealth handles health check
func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
`,
		"model.go.tmpl": `// Package models provides data models
package models

import (
	"time"
	"gorm.io/gorm"
)

// TestModel represents a test model
type TestModel struct {
	ID        uint      ` + "`" + `gorm:"primaryKey" json:"id"` + "`" + `
	Name      string    ` + "`" + `gorm:"not null" json:"name"` + "`" + `
	CreatedAt time.Time ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt time.Time ` + "`" + `json:"updated_at"` + "`" + `
}

// TableName returns the table name for TestModel
func (TestModel) TableName() string {
	return "test_models"
}
`,
		"invalid.tmpl": `{{.InvalidField}}`,
		"nested.tmpl":  `{{template "service.go.tmpl" .}}`,
	}
}

// CreateTempDir creates a temporary directory for testing.
func CreateTempDir() (string, error) {
	return os.MkdirTemp("", "switctl-test-*")
}

// CreateTestTemplateFiles creates template files in the given directory.
func CreateTestTemplateFiles(dir string) error {
	templates := TestTemplates()
	for name, content := range templates {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

// CleanupTempDir removes a temporary directory.
func CleanupTempDir(dir string) error {
	return os.RemoveAll(dir)
}

// TestFileSystemMockData returns mock data for file system operations.
func TestFileSystemMockData() map[string][]byte {
	return map[string][]byte{
		"main.go": []byte(`package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`),
		"config.yaml": []byte(`service:
  name: test-service
  port: 8080
database:
  type: mysql
  host: localhost
`),
		"README.md": []byte(`# Test Service

This is a test service generated by switctl.
`),
	}
}

// ExpectedFiles returns the expected files to be generated for different scenarios.
func ExpectedFiles() map[string][]string {
	return map[string][]string{
		"service": {
			"main.go",
			"internal/service/service.go",
			"internal/service/handler.go",
			"internal/service/config.go",
			"cmd/service/main.go",
			"go.mod",
			"Dockerfile",
			"README.md",
		},
		"api": {
			"api/handlers.go",
			"api/routes.go",
			"api/middleware.go",
			"proto/service.proto",
		},
		"model": {
			"internal/models/user.go",
			"internal/models/product.go",
			"internal/repository/user_repo.go",
			"internal/repository/product_repo.go",
		},
	}
}

// TestMiddlewareConfig returns a sample middleware configuration for testing.
func TestMiddlewareConfig() interfaces.MiddlewareConfig {
	return interfaces.MiddlewareConfig{
		Name:        "AuthMiddleware",
		Type:        "http",
		Description: "JWT authentication middleware",
		Dependencies: []string{
			"github.com/golang-jwt/jwt/v4",
			"github.com/gin-gonic/gin",
		},
		Config: map[string]interface{}{
			"secret_key":     "test-secret",
			"token_header":   "Authorization",
			"token_prefix":   "Bearer ",
			"skip_paths":     []string{"/health", "/metrics"},
			"token_duration": "24h",
		},
	}
}

// TestValidationErrors returns sample validation errors for testing.
func TestValidationErrors() []interfaces.ValidationError {
	return []interfaces.ValidationError{
		{
			Field:   "name",
			Message: "Name is required",
			Value:   "",
			Rule:    "required",
			Code:    "REQUIRED_FIELD",
		},
		{
			Field:   "email",
			Message: "Invalid email format",
			Value:   "invalid-email",
			Rule:    "email",
			Code:    "INVALID_FORMAT",
		},
		{
			Field:   "port",
			Message: "Port must be between 1024 and 65535",
			Value:   80,
			Rule:    "range",
			Code:    "OUT_OF_RANGE",
		},
	}
}

// TestCheckerResults returns sample checker results for testing.
func TestCheckerResults() map[string]interfaces.CheckResult {
	return map[string]interfaces.CheckResult{
		"style": {
			Name:     "Code Style Check",
			Status:   interfaces.CheckStatusPass,
			Message:  "All style checks passed",
			Duration: 2 * time.Second,
			Score:    100,
			MaxScore: 100,
			Fixable:  true,
		},
		"lint": {
			Name:    "Lint Check",
			Status:  interfaces.CheckStatusWarning,
			Message: "Found 3 warnings",
			Details: []interfaces.CheckDetail{
				{
					File:     "main.go",
					Line:     10,
					Column:   5,
					Message:  "Line too long",
					Rule:     "line-length",
					Severity: "warning",
				},
			},
			Duration: 3 * time.Second,
			Score:    85,
			MaxScore: 100,
			Fixable:  true,
		},
		"security": {
			Name:    "Security Check",
			Status:  interfaces.CheckStatusFail,
			Message: "Found 1 security issue",
			Details: []interfaces.CheckDetail{
				{
					File:     "config.go",
					Line:     25,
					Column:   12,
					Message:  "Hardcoded secret key",
					Rule:     "hardcoded-credentials",
					Severity: "error",
				},
			},
			Duration: 5 * time.Second,
			Score:    40,
			MaxScore: 100,
			Fixable:  false,
		},
	}
}
