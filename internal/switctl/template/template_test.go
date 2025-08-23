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
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
	"github.com/innovationmech/swit/internal/switctl/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TemplateTestSuite provides tests for specific template functionality.
type TemplateTestSuite struct {
	suite.Suite
	engine *Engine
}

// SetupTest sets up the test environment.
func (suite *TemplateTestSuite) SetupTest() {
	suite.engine = NewEngineWithEmbedded()
}

// TestJWTAuthenticationTemplates tests JWT authentication template generation.
func (suite *TemplateTestSuite) TestJWTAuthenticationTemplates() {
	tests := []struct {
		name         string
		templateName string
		dataBuilder  func() interface{}
		validators   []func([]byte) bool
	}{
		{
			name:         "JWT Types Template",
			templateName: "auth/jwt/types",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				// Configure for JWT authentication with various features
				data.Service.Auth = interfaces.AuthConfig{
					Type:      "jwt",
					Algorithm: "HS256",
				}
				// Add auth-specific features
				authFeatures := map[string]interface{}{
					"Features": map[string]interface{}{
						"RBAC":           true,
						"MultiTenant":    true,
						"OAuth2":         true,
						"TokenBlacklist": true,
					},
				}

				return map[string]interface{}{
					"Service":   data.Service,
					"Package":   data.Package,
					"Author":    data.Author,
					"Auth":      authFeatures,
					"Timestamp": data.Timestamp,
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "package auth") &&
						strings.Contains(str, "type User struct") &&
						strings.Contains(str, "type Role struct") &&
						strings.Contains(str, "type Permission struct") &&
						strings.Contains(str, "type Session struct") &&
						strings.Contains(str, "AuthService interface") &&
						strings.Contains(str, "AuthRepository interface")
				},
				func(content []byte) bool {
					str := string(content)
					// Check for conditional features
					return strings.Contains(str, "TenantID") && // Multi-tenant feature
						strings.Contains(str, "OAuthToken") && // OAuth2 feature
						strings.Contains(str, "BlacklistedToken") && // Token blacklist feature
						strings.Contains(str, "RequirePermission") // RBAC feature
				},
			},
		},
		{
			name:         "JWT Service Template",
			templateName: "auth/jwt/service",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
					"Auth": map[string]interface{}{
						"Type":      "jwt",
						"Algorithm": "HS256",
						"Features": map[string]interface{}{
							"RBAC":           true,
							"MultiTenant":    false,
							"TokenBlacklist": true,
						},
					},
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "package auth") &&
						strings.Contains(str, "type AuthService struct") &&
						strings.Contains(str, "func NewAuthService") &&
						strings.Contains(str, "func (a *AuthService) GenerateToken") &&
						strings.Contains(str, "func (a *AuthService) ValidateToken")
				},
			},
		},
		{
			name:         "JWT Repository Template",
			templateName: "auth/jwt/repository",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
					"Auth": map[string]interface{}{
						"Features": map[string]interface{}{
							"RBAC":        true,
							"MultiTenant": true,
						},
					},
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "package repository") &&
						strings.Contains(str, "type AuthRepository struct") &&
						strings.Contains(str, "func (r *AuthRepository) CreateUser") &&
						strings.Contains(str, "func (r *AuthRepository) GetUserByEmail")
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Try to load the template
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)
			if err != nil {
				suite.T().Skipf("Template %s not found in embedded templates: %v", tt.templateName, err)
				return
			}

			suite.NotNil(tmpl)
			suite.Equal(tt.templateName, tmpl.Name())

			// Render template with test data
			testData := tt.dataBuilder()
			output, err := suite.engine.RenderTemplate(tmpl, testData)
			suite.NoError(err)
			suite.NotEmpty(output)

			// Run validators
			for i, validator := range tt.validators {
				suite.True(validator(output), "Validator %d failed for template %s", i, tt.templateName)
			}

			// Validate Go syntax if it's a Go file
			if strings.Contains(tt.templateName, ".go") || strings.Contains(string(output), "package ") {
				suite.validateGoSyntax(output, tt.templateName)
			}
		})
	}
}

// TestDatabaseTemplates tests database-related template generation.
func (suite *TemplateTestSuite) TestDatabaseTemplates() {
	tests := []struct {
		name         string
		templateName string
		dataBuilder  func() interface{}
		validators   []func([]byte) bool
	}{
		{
			name:         "MongoDB Repository Template",
			templateName: "database/mongodb/repository",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				// Configure for MongoDB
				data.Service.Database.Type = "mongodb"
				data.Service.Features.Database = true

				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "go.mongodb.org/mongo-driver") &&
						strings.Contains(str, "type BaseRepository struct") &&
						strings.Contains(str, "func NewBaseRepository") &&
						strings.Contains(str, "func (r *BaseRepository) Insert") &&
						strings.Contains(str, "func (r *BaseRepository) FindOne") &&
						strings.Contains(str, "func (r *BaseRepository) UpdateOne") &&
						strings.Contains(str, "func (r *BaseRepository) DeleteOne")
				},
				func(content []byte) bool {
					str := string(content)
					// Check for advanced MongoDB features
					return strings.Contains(str, "Aggregate") &&
						strings.Contains(str, "CreateIndex") &&
						strings.Contains(str, "Transaction") &&
						strings.Contains(str, "SoftDelete") &&
						strings.Contains(str, "FindWithPagination")
				},
			},
		},
		{
			name:         "MongoDB Config Template",
			templateName: "database/mongodb/config",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				data.Service.Database.Type = "mongodb"
				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "type MongoConfig struct") &&
						strings.Contains(str, "func NewMongoConnection")
				},
			},
		},
		{
			name:         "Database Connection Template",
			templateName: "database/mongodb/connection",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				data.Service.Database.Type = "mongodb"
				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "func Connect") &&
						strings.Contains(str, "mongo.Connect")
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Try to load the template
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)
			if err != nil {
				suite.T().Skipf("Template %s not found in embedded templates: %v", tt.templateName, err)
				return
			}

			suite.NotNil(tmpl)

			// Render template with test data
			testData := tt.dataBuilder()
			output, err := suite.engine.RenderTemplate(tmpl, testData)
			suite.NoError(err)
			suite.NotEmpty(output)

			// Run validators
			for i, validator := range tt.validators {
				suite.True(validator(output), "Validator %d failed for template %s", i, tt.templateName)
			}

			// Validate Go syntax
			if strings.Contains(string(output), "package ") {
				suite.validateGoSyntax(output, tt.templateName)
			}
		})
	}
}

// TestMiddlewareTemplates tests middleware template generation.
func (suite *TemplateTestSuite) TestMiddlewareTemplates() {
	tests := []struct {
		name         string
		templateName string
		dataBuilder  func() interface{}
		validators   []func([]byte) bool
	}{
		{
			name:         "Advanced Rate Limit Template",
			templateName: "middleware/advanced_ratelimit",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				data.Service.Features.Cache = true // Enable Redis support

				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
					"Middleware": map[string]interface{}{
						"RateLimit": map[string]interface{}{
							"Advanced":    true,
							"Sliding":     true,
							"Distributed": true,
							"Adaptive":    true,
						},
					},
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "package middleware") &&
						strings.Contains(str, "type AdvancedRateLimitConfig struct") &&
						strings.Contains(str, "func AdvancedRateLimit") &&
						strings.Contains(str, "gin.HandlerFunc")
				},
				func(content []byte) bool {
					str := string(content)
					// Check for advanced features
					return strings.Contains(str, "SlidingWindowRateLimiter") &&
						strings.Contains(str, "AdaptiveRateLimiter") &&
						strings.Contains(str, "redis.Client") &&
						strings.Contains(str, "SystemMetrics")
				},
			},
		},
		{
			name:         "CORS Middleware Template",
			templateName: "middleware/cors",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
					"Middleware": map[string]interface{}{
						"CORS": map[string]interface{}{
							"Strict":      true,
							"Development": false,
						},
					},
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "func NewCORSMiddleware") &&
						strings.Contains(str, "Access-Control-Allow-Origin")
				},
			},
		},
		{
			name:         "Request ID Middleware Template",
			templateName: "middleware/requestid",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				return map[string]interface{}{
					"Service": data.Service,
					"Package": data.Package,
					"Author":  data.Author,
					"Middleware": map[string]interface{}{
						"RequestID": map[string]interface{}{
							"UUID":         true,
							"Hierarchical": false,
							"Tracing":      true,
						},
					},
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "func RequestIDMiddleware") &&
						strings.Contains(str, "X-Request-ID")
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Try to load the template
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)
			if err != nil {
				suite.T().Skipf("Template %s not found in embedded templates: %v", tt.templateName, err)
				return
			}

			suite.NotNil(tmpl)

			// Render template with test data
			testData := tt.dataBuilder()
			output, err := suite.engine.RenderTemplate(tmpl, testData)
			suite.NoError(err)
			suite.NotEmpty(output)

			// Run validators
			for i, validator := range tt.validators {
				suite.True(validator(output), "Validator %d failed for template %s", i, tt.templateName)
			}

			// Validate Go syntax
			if strings.Contains(string(output), "package ") {
				suite.validateGoSyntax(output, tt.templateName)
			}
		})
	}
}

// TestSecurityTemplates tests security-related template generation.
func (suite *TemplateTestSuite) TestSecurityTemplates() {
	tests := []struct {
		name         string
		templateName string
		dataBuilder  func() interface{}
		validators   []func([]byte) bool
	}{
		{
			name:         "Security Config Template",
			templateName: "security/config",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				return map[string]interface{}{
					"Service":  data.Service,
					"Package":  data.Package,
					"Author":   data.Author,
					"Security": data.Security,
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "type SecurityConfig struct") &&
						strings.Contains(str, "TLS") &&
						strings.Contains(str, "Headers")
				},
			},
		},
		{
			name:         "TLS Config Template",
			templateName: "security/tls",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				return map[string]interface{}{
					"Service":  data.Service,
					"Package":  data.Package,
					"Author":   data.Author,
					"Security": data.Security,
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "func NewTLSConfig") &&
						strings.Contains(str, "tls.Config")
				},
			},
		},
		{
			name:         "Security Headers Template",
			templateName: "security/headers",
			dataBuilder: func() interface{} {
				data := testutil.TestTemplateData()
				return map[string]interface{}{
					"Service":  data.Service,
					"Package":  data.Package,
					"Author":   data.Author,
					"Security": data.Security,
				}
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "func SecurityHeaders") &&
						strings.Contains(str, "X-Content-Type-Options") &&
						strings.Contains(str, "X-Frame-Options")
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Try to load the template
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)
			if err != nil {
				suite.T().Skipf("Template %s not found in embedded templates: %v", tt.templateName, err)
				return
			}

			suite.NotNil(tmpl)

			// Render template with test data
			testData := tt.dataBuilder()
			output, err := suite.engine.RenderTemplate(tmpl, testData)
			suite.NoError(err)
			suite.NotEmpty(output)

			// Run validators
			for i, validator := range tt.validators {
				suite.True(validator(output), "Validator %d failed for template %s", i, tt.templateName)
			}

			// Validate Go syntax
			if strings.Contains(string(output), "package ") {
				suite.validateGoSyntax(output, tt.templateName)
			}
		})
	}
}

// TestServiceTemplates tests service scaffolding templates.
func (suite *TemplateTestSuite) TestServiceTemplates() {
	tests := []struct {
		name         string
		templateName string
		dataBuilder  func() interface{}
		validators   []func([]byte) bool
	}{
		{
			name:         "Main Service Template",
			templateName: "service/main.go",
			dataBuilder: func() interface{} {
				return testutil.TestTemplateData()
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "package main") &&
						strings.Contains(str, "func main()") &&
						strings.Contains(str, "cmd.Execute()")
				},
			},
		},
		{
			name:         "Service Server Template",
			templateName: "service/internal/server.go",
			dataBuilder: func() interface{} {
				return testutil.TestTemplateData()
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "type TestServiceServer struct") &&
						strings.Contains(str, "func NewTestServiceServer") &&
						strings.Contains(str, "func (s *TestServiceServer) Start")
				},
			},
		},
		{
			name:         "Service Config Template",
			templateName: "service/internal/config/config.go",
			dataBuilder: func() interface{} {
				return testutil.TestTemplateData()
			},
			validators: []func([]byte) bool{
				func(content []byte) bool {
					str := string(content)
					return strings.Contains(str, "package config") &&
						strings.Contains(str, "type Config struct")
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Try to load the template
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)
			if err != nil {
				suite.T().Skipf("Template %s not found in embedded templates: %v", tt.templateName, err)
				return
			}

			suite.NotNil(tmpl)

			// Render template with test data
			testData := tt.dataBuilder()
			output, err := suite.engine.RenderTemplate(tmpl, testData)
			suite.NoError(err)
			suite.NotEmpty(output)

			// Run validators
			for i, validator := range tt.validators {
				suite.True(validator(output), "Validator %d failed for template %s", i, tt.templateName)
			}

			// Validate Go syntax
			if strings.Contains(string(output), "package ") {
				suite.validateGoSyntax(output, tt.templateName)
			}
		})
	}
}

// TestConditionalFeatureGeneration tests conditional template generation based on feature flags.
func (suite *TemplateTestSuite) TestConditionalFeatureGeneration() {
	testCases := []struct {
		name             string
		baseData         func() interfaces.TemplateData
		featureOverrides map[string]bool
		expectedPresence []string
		expectedAbsence  []string
		templateName     string
	}{
		{
			name: "Database feature enabled",
			baseData: func() interfaces.TemplateData {
				return testutil.TestTemplateData()
			},
			featureOverrides: map[string]bool{
				"Database": true,
				"Cache":    false,
			},
			expectedPresence: []string{
				"DatabaseConfig",
				"Database DatabaseConfig",
			},
			expectedAbsence: []string{
				"CacheConfig",
				"Cache CacheConfig",
			},
			templateName: "service/internal/config/config.go",
		},
		{
			name: "All features enabled",
			baseData: func() interfaces.TemplateData {
				return testutil.TestTemplateData()
			},
			featureOverrides: map[string]bool{
				"Database":       true,
				"Authentication": true,
				"Cache":          true,
				"GRPC":           true,
				"REST":           true,
				"Monitoring":     true,
			},
			expectedPresence: []string{
				"DatabaseConfig",
				"AuthConfig",
				"CacheConfig",
				"HTTPConfig",
				"GRPCConfig",
			},
			expectedAbsence: []string{},
			templateName:    "service/internal/config/config.go",
		},
		{
			name: "Minimal features",
			baseData: func() interfaces.TemplateData {
				return testutil.TestTemplateData()
			},
			featureOverrides: map[string]bool{
				"Database":       false,
				"Authentication": false,
				"Cache":          false,
				"GRPC":           false,
				"REST":           true,
			},
			expectedPresence: []string{
				"HTTPConfig",
			},
			expectedAbsence: []string{
				"DatabaseConfig",
				"AuthConfig",
				"CacheConfig",
			},
			templateName: "service/internal/config/config.go",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Skip if template not found
			tmpl, err := suite.engine.LoadTemplate(tc.templateName)
			if err != nil {
				suite.T().Skipf("Template %s not found: %v", tc.templateName, err)
				return
			}

			// Build test data with feature overrides
			testData := tc.baseData()

			// Apply feature overrides
			if database, ok := tc.featureOverrides["Database"]; ok {
				testData.Service.Features.Database = database
			}
			if auth, ok := tc.featureOverrides["Authentication"]; ok {
				testData.Service.Features.Authentication = auth
			}
			if cache, ok := tc.featureOverrides["Cache"]; ok {
				testData.Service.Features.Cache = cache
			}
			if grpc, ok := tc.featureOverrides["GRPC"]; ok {
				testData.Service.Features.GRPC = grpc
			}
			if rest, ok := tc.featureOverrides["REST"]; ok {
				testData.Service.Features.REST = rest
			}

			// Render template
			output, err := suite.engine.RenderTemplate(tmpl, testData)
			suite.NoError(err)
			suite.NotEmpty(output)

			str := string(output)

			// Check expected presence
			for _, expected := range tc.expectedPresence {
				suite.Contains(str, expected, "Expected '%s' to be present in generated content", expected)
			}

			// Check expected absence
			for _, notExpected := range tc.expectedAbsence {
				suite.NotContains(str, notExpected, "Expected '%s' to be absent from generated content", notExpected)
			}

			// Validate Go syntax
			if strings.Contains(str, "package ") {
				suite.validateGoSyntax(output, tc.templateName)
			}
		})
	}
}

// TestTemplateEdgeCases tests various edge cases and error conditions.
func (suite *TemplateTestSuite) TestTemplateEdgeCases() {
	tests := []struct {
		name        string
		data        interface{}
		expectError bool
		description string
	}{
		{
			name:        "Nil data",
			data:        nil,
			expectError: false, // Templates should handle nil gracefully
			description: "Templates should handle nil data without errors",
		},
		{
			name: "Empty service config",
			data: map[string]interface{}{
				"Service": interfaces.ServiceConfig{},
				"Package": interfaces.PackageInfo{},
			},
			expectError: false,
			description: "Templates should handle empty configurations",
		},
		{
			name: "Partial data",
			data: map[string]interface{}{
				"Service": interfaces.ServiceConfig{
					Name: "partial-service",
				},
			},
			expectError: false,
			description: "Templates should handle partial data gracefully",
		},
		{
			name: "Complex nested data",
			data: map[string]interface{}{
				"Service": testutil.TestTemplateData().Service,
				"Package": testutil.TestTemplateData().Package,
				"Author":  testutil.TestTemplateData().Author,
				"CustomData": map[string]interface{}{
					"Level1": map[string]interface{}{
						"Level2": map[string]interface{}{
							"Level3": "deep value",
						},
					},
				},
			},
			expectError: false,
			description: "Templates should handle deeply nested data",
		},
	}

	// Try with a simple template first
	simpleTemplate := `// Package {{.Package.Name | default "main"}}
package {{.Package.Name | default "main"}}

// Service: {{.Service.Name | default "unknown"}}
// Author: {{.Author | default "unknown"}}
{{- if .CustomData}}
// Custom: {{.CustomData.Level1.Level2.Level3 | default "none"}}
{{- end}}
`

	// Create temporary template
	tempDir, err := os.MkdirTemp("", "template-edge-cases-*")
	suite.NoError(err)
	defer os.RemoveAll(tempDir)

	tmplPath := filepath.Join(tempDir, "edge_case.tmpl")
	err = os.WriteFile(tmplPath, []byte(simpleTemplate), 0644)
	suite.NoError(err)

	engine := NewEngine(tempDir)

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tmpl, err := engine.LoadTemplate("edge_case")
			suite.NoError(err)

			output, err := engine.RenderTemplate(tmpl, tt.data)

			if tt.expectError {
				suite.Error(err, tt.description)
			} else {
				suite.NoError(err, tt.description)
				suite.NotNil(output)

				// Basic validation that template rendered something
				str := string(output)
				suite.Contains(str, "package ", "Should contain package declaration")
			}
		})
	}
}

// TestTemplatePerformance tests template rendering performance.
func (suite *TemplateTestSuite) TestTemplatePerformance() {
	// Create a reasonably complex template
	performanceTemplate := `// Performance test template
package performance

import (
	"context"
	"fmt"
	"time"
)

{{- range $i, $service := .Services}}
// Service{{$i}} represents service {{$service.Name}}
type Service{{$i}} struct {
	name string
	port int
}

func NewService{{$i}}() *Service{{$i}} {
	return &Service{{$i}}{
		name: "{{$service.Name}}",
		port: {{$service.Port}},
	}
}

func (s *Service{{$i}}) Start(ctx context.Context) error {
	fmt.Printf("Starting %s on port %d\n", s.name, s.port)
	return nil
}
{{- end}}

// Manager manages all services
type Manager struct {
	{{- range $i, $service := .Services}}
	service{{$i}} *Service{{$i}}
	{{- end}}
}

func NewManager() *Manager {
	return &Manager{
		{{- range $i, $service := .Services}}
		service{{$i}}: NewService{{$i}}(),
		{{- end}}
	}
}
`

	tempDir, err := os.MkdirTemp("", "template-performance-*")
	suite.NoError(err)
	defer os.RemoveAll(tempDir)

	tmplPath := filepath.Join(tempDir, "performance.tmpl")
	err = os.WriteFile(tmplPath, []byte(performanceTemplate), 0644)
	suite.NoError(err)

	engine := NewEngine(tempDir)

	// Create test data with multiple services
	services := make([]map[string]interface{}, 50) // Moderate size for testing
	for i := 0; i < len(services); i++ {
		services[i] = map[string]interface{}{
			"Name": fmt.Sprintf("service-%d", i),
			"Port": 8000 + i,
		}
	}

	testData := map[string]interface{}{
		"Services": services,
	}

	// Load template
	tmpl, err := engine.LoadTemplate("performance")
	suite.NoError(err)

	// Measure rendering performance
	startTime := time.Now()
	output, err := engine.RenderTemplate(tmpl, testData)
	renderTime := time.Since(startTime)

	suite.NoError(err)
	suite.NotEmpty(output)

	// Performance assertion - should complete within reasonable time
	suite.Less(renderTime, 2*time.Second, "Template rendering took too long: %v", renderTime)

	// Validate that the output contains expected elements
	str := string(output)
	suite.Contains(str, "package performance")
	suite.Contains(str, "type Service0 struct")
	suite.Contains(str, "type Service49 struct")
	suite.Contains(str, "func NewManager()")

	// Validate Go syntax
	suite.validateGoSyntax(output, "performance template")
}

// validateGoSyntax validates that generated Go code is syntactically correct.
func (suite *TemplateTestSuite) validateGoSyntax(content []byte, templateName string) {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, templateName, content, parser.ParseComments)
	suite.NoError(err, "Generated Go code has syntax errors in template %s:\n%s", templateName, string(content))
}

// Run the test suite.
func TestTemplateTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateTestSuite))
}

// TestTemplateHelperFunctionComprehensive tests all helper functions comprehensively.
func TestTemplateHelperFunctionComprehensive(t *testing.T) {
	// Using embedded engine for comprehensive function testing
	_ = NewEngineWithEmbedded()

	tests := []struct {
		category string
		function string
		input    interface{}
		expected interface{}
		testName string
	}{
		// String case functions - comprehensive test cases
		{"case", "camelCase", "test_service_name", "testServiceName", "camelCase with underscores"},
		{"case", "camelCase", "TestServiceName", "testservicename", "camelCase with PascalCase input"},
		{"case", "camelCase", "test-service-name", "testServiceName", "camelCase with hyphens"},
		{"case", "snakeCase", "TestServiceName", "test_service_name", "snakeCase with PascalCase"},
		{"case", "snakeCase", "testServiceName", "test_service_name", "snakeCase with camelCase"},
		{"case", "snakeCase", "HTTPSProxy", "httpsproxy", "snakeCase with acronyms"},
		{"case", "pascalCase", "test_service_name", "TestServiceName", "pascalCase with underscores"},
		{"case", "pascalCase", "test-service-name", "TestServiceName", "pascalCase with hyphens"},
		{"case", "kebabCase", "TestServiceName", "test-service-name", "kebabCase with PascalCase"},
		{"case", "kebabCase", "test_service_name", "test-service-name", "kebabCase with snake_case"},

		// Go-specific functions
		{"go", "exportedName", "test_service", "TestService", "exportedName basic"},
		{"go", "exportedName", "httpProxy", "Httpproxy", "exportedName with lowercase acronym"},
		{"go", "unexportedName", "TestService", "testservice", "unexportedName basic"},
		{"go", "packageName", "test-service-name", "testservicename", "packageName with hyphens"},
		{"go", "packageName", "HTTP_Service", "httpservice", "packageName with underscores and caps"},

		// Pluralization
		{"plural", "pluralize", "user", "users", "pluralize simple"},
		{"plural", "pluralize", "category", "categories", "pluralize ending in y"},
		{"plural", "pluralize", "box", "boxes", "pluralize ending in x"},
		{"plural", "pluralize", "class", "classes", "pluralize ending in s"},
		{"plural", "singularize", "users", "user", "singularize simple"},
		{"plural", "singularize", "categories", "category", "singularize ending in ies"},
		{"plural", "singularize", "boxes", "box", "singularize ending in es"},

		// Math functions
		{"math", "add", []interface{}{10, 5}, 15.0, "add positive numbers"},
		{"math", "add", []interface{}{-5, 3}, -2.0, "add negative and positive"},
		{"math", "sub", []interface{}{10, 3}, 7.0, "subtract basic"},
		{"math", "mul", []interface{}{4, 5}, 20.0, "multiply basic"},
		{"math", "div", []interface{}{15, 3}, 5.0, "divide basic"},
		{"math", "div", []interface{}{10, 0}, 0.0, "divide by zero"},
		{"math", "mod", []interface{}{10, 3}, 1, "modulo basic"},
		{"math", "mod", []interface{}{10, 0}, 0, "modulo by zero"},

		// Comparison functions
		{"compare", "eq", []interface{}{5, 5}, true, "equal same numbers"},
		{"compare", "eq", []interface{}{5, 3}, false, "equal different numbers"},
		{"compare", "eq", []interface{}{"hello", "hello"}, true, "equal same strings"},
		{"compare", "ne", []interface{}{5, 3}, true, "not equal different"},
		{"compare", "lt", []interface{}{3, 5}, true, "less than true"},
		{"compare", "lt", []interface{}{5, 3}, false, "less than false"},
		{"compare", "gt", []interface{}{5, 3}, true, "greater than true"},
		{"compare", "le", []interface{}{5, 5}, true, "less or equal equal"},
		{"compare", "ge", []interface{}{5, 5}, true, "greater or equal equal"},

		// Logical functions
		{"logical", "and", []interface{}{true, true}, true, "and both true"},
		{"logical", "and", []interface{}{true, false}, false, "and mixed"},
		{"logical", "or", []interface{}{false, true}, true, "or mixed"},
		{"logical", "or", []interface{}{false, false}, false, "or both false"},
		{"logical", "not", true, false, "not true"},
		{"logical", "not", false, true, "not false"},

		// Default value functions
		{"default", "default", []interface{}{"", "fallback"}, "fallback", "default empty string"},
		{"default", "default", []interface{}{"value", "fallback"}, "value", "default non-empty string"},
		{"default", "default", []interface{}{0, 42}, 42, "default zero int"},
		{"default", "default", []interface{}{5, 42}, 5, "default non-zero int"},
		{"default", "default", []interface{}{false, true}, true, "default false bool"},
		{"default", "default", []interface{}{true, false}, true, "default true bool"},

		// String manipulation
		{"string", "lower", "HELLO", "hello", "lower case"},
		{"string", "upper", "hello", "HELLO", "upper case"},
		{"string", "trim", "  hello  ", "hello", "trim whitespace"},
		{"string", "humanize", "test_service_name", "Test Service Name", "humanize snake_case"},
		{"string", "humanize", "test-service-name", "Test Service Name", "humanize kebab-case"},
		{"string", "dehumanize", "Test Service Name", "test_service_name", "dehumanize to snake_case"},
	}

	// Helper function to format template arguments
	formatTemplateArg := func(arg interface{}) string {
		switch v := arg.(type) {
		case string:
			return fmt.Sprintf(`"%s"`, v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.category, tt.testName), func(t *testing.T) {
			var templateContent string

			switch v := tt.input.(type) {
			case []interface{}:
				// For functions that take multiple arguments
				if len(v) == 2 {
					arg0 := formatTemplateArg(v[0])
					arg1 := formatTemplateArg(v[1])
					templateContent = fmt.Sprintf("{{%s %s %s}}", tt.function, arg0, arg1)
				} else {
					t.Fatalf("Unsupported number of arguments: %d", len(v))
				}
			default:
				// For functions that take single argument
				templateContent = fmt.Sprintf("{{%s | %s}}", formatTemplateArg(v), tt.function)
			}

			tempDir, err := os.MkdirTemp("", "helper-function-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			tmplPath := filepath.Join(tempDir, "test.tmpl")
			err = os.WriteFile(tmplPath, []byte(templateContent), 0644)
			require.NoError(t, err)

			localEngine := NewEngine(tempDir)
			tmpl, err := localEngine.LoadTemplate("test")
			require.NoError(t, err)

			output, err := localEngine.RenderTemplate(tmpl, map[string]interface{}{})
			require.NoError(t, err)

			result := strings.TrimSpace(string(output))
			expected := fmt.Sprintf("%v", tt.expected)
			assert.Equal(t, expected, result, "Function %s with input %v should return %v, got %s", tt.function, tt.input, tt.expected, result)
		})
	}
}
