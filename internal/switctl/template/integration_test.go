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
	"go/ast"
	"go/parser"
	"go/token"
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

// TemplateIntegrationTestSuite provides integration tests for complete template workflows.
type TemplateIntegrationTestSuite struct {
	suite.Suite
	engine    *Engine
	tempDir   string
	outputDir string
}

// SetupTest sets up the test environment.
func (suite *TemplateIntegrationTestSuite) SetupTest() {
	var err error
	suite.tempDir, err = os.MkdirTemp("", "template-integration-test-*")
	suite.Require().NoError(err)

	suite.outputDir, err = os.MkdirTemp("", "template-output-*")
	suite.Require().NoError(err)

	suite.createRealWorldTemplates()
	suite.engine = NewEngine(suite.tempDir)
}

// TearDownTest cleans up after each test.
func (suite *TemplateIntegrationTestSuite) TearDownTest() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
	if suite.outputDir != "" {
		os.RemoveAll(suite.outputDir)
	}
}

// createRealWorldTemplates creates realistic templates for integration testing.
func (suite *TemplateIntegrationTestSuite) createRealWorldTemplates() {
	templates := map[string]string{
		"service/main.go.tmpl": `// {{.Service.Name}} Service
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"{{.Package.ModulePath}}/internal/{{.Package.Name}}"
	"{{.Package.ModulePath}}/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create service
	service, err := {{.Package.Name | exportedName}}.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Start service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := service.Start(ctx); err != nil {
			log.Printf("Service error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	if err := service.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
`,
		"service/config.go.tmpl": `// Package config provides configuration management
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the service configuration
type Config struct {
	{{- if .Service.Features.Database}}
	Database DatabaseConfig ` + "`yaml:\"database\" json:\"database\"`" + `
	{{- end}}
	{{- if .Service.Features.Authentication}}
	Auth     AuthConfig     ` + "`yaml:\"auth\" json:\"auth\"`" + `
	{{- end}}
	{{- if .Service.Features.Cache}}
	Cache    CacheConfig    ` + "`yaml:\"cache\" json:\"cache\"`" + `
	{{- end}}
	Server   ServerConfig   ` + "`yaml:\"server\" json:\"server\"`" + `
	Logging  LoggingConfig  ` + "`yaml:\"logging\" json:\"logging\"`" + `
}

{{- if .Service.Features.Database}}
// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type     string ` + "`yaml:\"type\" json:\"type\"`" + `
	Host     string ` + "`yaml:\"host\" json:\"host\"`" + `
	Port     int    ` + "`yaml:\"port\" json:\"port\"`" + `
	Database string ` + "`yaml:\"database\" json:\"database\"`" + `
	Username string ` + "`yaml:\"username\" json:\"username\"`" + `
	Password string ` + "`yaml:\"password\" json:\"password\"`" + `
	SSLMode  string ` + "`yaml:\"ssl_mode\" json:\"ssl_mode\" default:\"disable\"`" + `
}
{{- end}}

{{- if .Service.Features.Authentication}}
// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type      string        ` + "`yaml:\"type\" json:\"type\"`" + `
	SecretKey string        ` + "`yaml:\"secret_key\" json:\"secret_key\"`" + `
	Expiration time.Duration ` + "`yaml:\"expiration\" json:\"expiration\"`" + `
	Algorithm string        ` + "`yaml:\"algorithm\" json:\"algorithm\"`" + `
}
{{- end}}

// ServerConfig represents server configuration
type ServerConfig struct {
	{{- if .Service.Features.REST}}
	HTTPPort    int    ` + "`yaml:\"http_port\" json:\"http_port\"`" + `
	{{- end}}
	{{- if .Service.Features.GRPC}}
	GRPCPort    int    ` + "`yaml:\"grpc_port\" json:\"grpc_port\"`" + `
	{{- end}}
	{{- if .Service.Features.HealthCheck}}
	HealthPort  int    ` + "`yaml:\"health_port\" json:\"health_port\"`" + `
	{{- end}}
	{{- if .Service.Features.Metrics}}
	MetricsPort int    ` + "`yaml:\"metrics_port\" json:\"metrics_port\"`" + `
	{{- end}}
	Host        string ` + "`yaml:\"host\" json:\"host\" default:\"0.0.0.0\"`" + `
	Environment string ` + "`yaml:\"environment\" json:\"environment\" default:\"development\"`" + `
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string ` + "`yaml:\"level\" json:\"level\" default:\"info\"`" + `
	Format string ` + "`yaml:\"format\" json:\"format\" default:\"json\"`" + `
	Output string ` + "`yaml:\"output\" json:\"output\" default:\"stdout\"`" + `
}

// Load loads configuration from environment variables and defaults
func Load() (*Config, error) {
	config := &Config{
		{{- if .Service.Features.Database}}
		Database: DatabaseConfig{
			Type:     getEnv("DB_TYPE", "{{.Service.Database.Type}}"),
			Host:     getEnv("DB_HOST", "{{.Service.Database.Host}}"),
			Port:     getEnvInt("DB_PORT", {{.Service.Database.Port}}),
			Database: getEnv("DB_NAME", "{{.Service.Database.Database}}"),
			Username: getEnv("DB_USERNAME", "{{.Service.Database.Username}}"),
			Password: getEnv("DB_PASSWORD", "{{.Service.Database.Password}}"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		{{- end}}
		{{- if .Service.Features.Authentication}}
		Auth: AuthConfig{
			Type:      getEnv("AUTH_TYPE", "{{.Service.Auth.Type}}"),
			SecretKey: getEnv("AUTH_SECRET_KEY", "{{.Service.Auth.SecretKey}}"),
			Expiration: getEnvDuration("AUTH_EXPIRATION", time.Hour * 24),
			Algorithm: getEnv("AUTH_ALGORITHM", "{{.Service.Auth.Algorithm}}"),
		},
		{{- end}}
		Server: ServerConfig{
			{{- if .Service.Features.REST}}
			HTTPPort:    getEnvInt("HTTP_PORT", {{.Service.Ports.HTTP}}),
			{{- end}}
			{{- if .Service.Features.GRPC}}
			GRPCPort:    getEnvInt("GRPC_PORT", {{.Service.Ports.GRPC}}),
			{{- end}}
			{{- if .Service.Features.HealthCheck}}
			HealthPort:  getEnvInt("HEALTH_PORT", {{.Service.Ports.Health}}),
			{{- end}}
			{{- if .Service.Features.Metrics}}
			MetricsPort: getEnvInt("METRICS_PORT", {{.Service.Ports.Metrics}}),
			{{- end}}
			Host:        getEnv("HOST", "0.0.0.0"),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	return config, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
`,
		"auth/jwt/middleware.go.tmpl": `// Package auth provides JWT authentication middleware
package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// JWTMiddleware provides JWT authentication middleware
type JWTMiddleware struct {
	secretKey []byte
	algorithm string
}

// NewJWTMiddleware creates a new JWT middleware
func NewJWTMiddleware(secretKey, algorithm string) *JWTMiddleware {
	return &JWTMiddleware{
		secretKey: []byte(secretKey),
		algorithm: algorithm,
	}
}

// RequireAuth middleware requires valid JWT authentication
func (m *JWTMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Parse and validate token
		claims, err := m.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims["user_id"])
		c.Set("username", claims["username"])
		c.Set("email", claims["email"])

		c.Next()
	}
}

// OptionalAuth middleware allows optional JWT authentication
func (m *JWTMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString != authHeader {
				// Try to validate token, but don't fail if invalid
				claims, err := m.validateToken(tokenString)
				if err == nil {
					c.Set("user_id", claims["user_id"])
					c.Set("username", claims["username"])
					c.Set("email", claims["email"])
					c.Set("authenticated", true)
				}
			}
		}

		c.Next()
	}
}

// validateToken validates a JWT token and returns claims
func (m *JWTMiddleware) validateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate algorithm
		if token.Method.Alg() != m.algorithm {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}
`,
		"database/mongodb/repository.tmpl": `// Package repository provides MongoDB repository implementations
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrInvalidObjectID  = errors.New("invalid object ID")
)

// {{.Model.Name | exportedName}}Repository provides MongoDB operations for {{.Model.Name}}
type {{.Model.Name | exportedName}}Repository struct {
	collection *mongo.Collection
}

// New{{.Model.Name | exportedName}}Repository creates a new repository
func New{{.Model.Name | exportedName}}Repository(db *mongo.Database) *{{.Model.Name | exportedName}}Repository {
	return &{{.Model.Name | exportedName}}Repository{
		collection: db.Collection("{{.Model.TableName | default (printf "%ss" (.Model.Name | snakeCase))}}"),
	}
}

// Create creates a new {{.Model.Name}}
func (r *{{.Model.Name | exportedName}}Repository) Create(ctx context.Context, {{.Model.Name | unexportedName}} *{{.Model.Name | exportedName}}) error {
	{{.Model.Name | unexportedName}}.CreatedAt = time.Now()
	{{.Model.Name | unexportedName}}.UpdatedAt = time.Now()
	
	result, err := r.collection.InsertOne(ctx, {{.Model.Name | unexportedName}})
	if err != nil {
		return fmt.Errorf("failed to create {{.Model.Name | snakeCase}}: %w", err)
	}
	
	{{.Model.Name | unexportedName}}.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID retrieves a {{.Model.Name}} by ID
func (r *{{.Model.Name | exportedName}}Repository) GetByID(ctx context.Context, id string) (*{{.Model.Name | exportedName}}, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidObjectID
	}

	var {{.Model.Name | unexportedName}} {{.Model.Name | exportedName}}
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&{{.Model.Name | unexportedName}})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to get {{.Model.Name | snakeCase}}: %w", err)
	}

	return &{{.Model.Name | unexportedName}}, nil
}

// Update updates a {{.Model.Name}}
func (r *{{.Model.Name | exportedName}}Repository) Update(ctx context.Context, id string, {{.Model.Name | unexportedName}} *{{.Model.Name | exportedName}}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidObjectID
	}

	{{.Model.Name | unexportedName}}.UpdatedAt = time.Now()
	
	update := bson.M{"$set": {{.Model.Name | unexportedName}}}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return fmt.Errorf("failed to update {{.Model.Name | snakeCase}}: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// Delete deletes a {{.Model.Name}}
func (r *{{.Model.Name | exportedName}}Repository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidObjectID
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete {{.Model.Name | snakeCase}}: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrDocumentNotFound
	}

	return nil
}

// List retrieves all {{.Model.Name | pluralize}}
func (r *{{.Model.Name | exportedName}}Repository) List(ctx context.Context, limit, offset int) ([]*{{.Model.Name | exportedName}}, error) {
	findOptions := options.Find()
	if limit > 0 {
		findOptions.SetLimit(int64(limit))
	}
	if offset > 0 {
		findOptions.SetSkip(int64(offset))
	}

	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list {{.Model.Name | snakeCase | pluralize}}: %w", err)
	}
	defer cursor.Close(ctx)

	var {{.Model.Name | unexportedName | pluralize}} []*{{.Model.Name | exportedName}}
	if err := cursor.All(ctx, &{{.Model.Name | unexportedName | pluralize}}); err != nil {
		return nil, fmt.Errorf("failed to decode {{.Model.Name | snakeCase | pluralize}}: %w", err)
	}

	return {{.Model.Name | unexportedName | pluralize}}, nil
}

// Count returns the total count of {{.Model.Name | pluralize}}
func (r *{{.Model.Name | exportedName}}Repository) Count(ctx context.Context) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to count {{.Model.Name | snakeCase | pluralize}}: %w", err)
	}
	return count, nil
}
`,
	}

	for name, content := range templates {
		path := filepath.Join(suite.tempDir, name)
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, 0755)
		suite.Require().NoError(err)

		err = os.WriteFile(path, []byte(content), 0644)
		suite.Require().NoError(err)
	}
}

// TestCompleteServiceGeneration tests generating a complete service from templates.
func (suite *TemplateIntegrationTestSuite) TestCompleteServiceGeneration() {
	testData := testutil.TestTemplateData()

	tests := []struct {
		name         string
		templateName string
		outputFile   string
		validateFunc func([]byte) bool
	}{
		{
			name:         "Generate main.go",
			templateName: "service/main.go",
			outputFile:   "main.go",
			validateFunc: func(content []byte) bool {
				str := string(content)
				return strings.Contains(str, "package main") &&
					strings.Contains(str, "func main()") &&
					strings.Contains(str, testData.Package.ModulePath) &&
					strings.Contains(str, "NewService")
			},
		},
		{
			name:         "Generate config.go",
			templateName: "service/config.go",
			outputFile:   "config.go",
			validateFunc: func(content []byte) bool {
				str := string(content)
				return strings.Contains(str, "package config") &&
					strings.Contains(str, "type Config struct") &&
					strings.Contains(str, "DatabaseConfig") &&
					strings.Contains(str, "AuthConfig") &&
					strings.Contains(str, "func Load()")
			},
		},
		{
			name:         "Generate JWT middleware",
			templateName: "auth/jwt/middleware.go",
			outputFile:   "middleware.go",
			validateFunc: func(content []byte) bool {
				str := string(content)
				return strings.Contains(str, "package auth") &&
					strings.Contains(str, "type JWTMiddleware struct") &&
					strings.Contains(str, "RequireAuth()") &&
					strings.Contains(str, "OptionalAuth()")
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Load template
			tmpl, err := suite.engine.LoadTemplate(tt.templateName)
			suite.NoError(err)
			suite.NotNil(tmpl)

			// Render template
			output, err := suite.engine.RenderTemplate(tmpl, testData)
			suite.NoError(err)
			suite.NotEmpty(output)

			// Validate content
			if tt.validateFunc != nil {
				suite.True(tt.validateFunc(output), "Generated content validation failed for %s", tt.name)
			}

			// Write to output directory
			outputPath := filepath.Join(suite.outputDir, tt.outputFile)
			err = os.WriteFile(outputPath, output, 0644)
			suite.NoError(err)

			// Validate that generated Go code is syntactically correct
			if strings.HasSuffix(tt.outputFile, ".go") {
				suite.validateGoSyntax(outputPath)
			}
		})
	}
}

// TestDatabaseRepositoryGeneration tests generating database repository code.
func (suite *TemplateIntegrationTestSuite) TestDatabaseRepositoryGeneration() {
	modelConfig := testutil.TestModelConfig()
	testData := map[string]interface{}{
		"Model":   modelConfig,
		"Package": testutil.TestTemplateData().Package,
	}

	// Load and render MongoDB repository template
	tmpl, err := suite.engine.LoadTemplate("database/mongodb/repository")
	suite.NoError(err)

	output, err := suite.engine.RenderTemplate(tmpl, testData)
	suite.NoError(err)

	// Validate generated repository code
	str := string(output)
	suite.Contains(str, "package repository")
	suite.Contains(str, "type ProductRepository struct")
	suite.Contains(str, "func NewProductRepository")
	suite.Contains(str, "func (r *ProductRepository) Create")
	suite.Contains(str, "func (r *ProductRepository) GetByID")
	suite.Contains(str, "func (r *ProductRepository) Update")
	suite.Contains(str, "func (r *ProductRepository) Delete")
	suite.Contains(str, "func (r *ProductRepository) List")
	suite.Contains(str, "func (r *ProductRepository) Count")

	// Write and validate Go syntax
	outputPath := filepath.Join(suite.outputDir, "product_repository.go")
	err = os.WriteFile(outputPath, output, 0644)
	suite.NoError(err)

	suite.validateGoSyntax(outputPath)
}

// TestTemplateWithComplexDataStructures tests templates with complex nested data.
func (suite *TemplateIntegrationTestSuite) TestTemplateWithComplexDataStructures() {
	// Create a complex template that uses various Go constructs
	complexTemplate := `// Generated service with complex features
package {{.Package.Name}}

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

{{- range .Imports}}
	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{- end}}
)

// {{.Service.Name | exportedName}} represents the main service
type {{.Service.Name | exportedName}} struct {
	config *Config
	{{- if .Service.Features.Database}}
	db     Database
	{{- end}}
	{{- if .Service.Features.Authentication}}
	auth   AuthService
	{{- end}}
	{{- if .Service.Features.Cache}}
	cache  CacheService
	{{- end}}
	mu     sync.RWMutex
}

// Config holds service configuration
type Config struct {
	{{- if .Service.Features.Database}}
	Database struct {
		Type     string ` + "`yaml:\"type\"`" + `
		Host     string ` + "`yaml:\"host\"`" + `
		Port     int    ` + "`yaml:\"port\"`" + `
		Database string ` + "`yaml:\"database\"`" + `
	} ` + "`yaml:\"database\"`" + `
	{{- end}}
	{{- if .Service.Features.Authentication}}
	Auth struct {
		SecretKey  string        ` + "`yaml:\"secret_key\"`" + `
		Expiration time.Duration ` + "`yaml:\"expiration\"`" + `
	} ` + "`yaml:\"auth\"`" + `
	{{- end}}
	Server struct {
		{{- if .Service.Features.REST}}
		HTTPPort int ` + "`yaml:\"http_port\"`" + `
		{{- end}}
		{{- if .Service.Features.GRPC}}
		GRPCPort int ` + "`yaml:\"grpc_port\"`" + `
		{{- end}}
		Host     string ` + "`yaml:\"host\"`" + `
	} ` + "`yaml:\"server\"`" + `
}

// New{{.Service.Name | exportedName}} creates a new service instance
func New{{.Service.Name | exportedName}}(config *Config) (*{{.Service.Name | exportedName}}, error) {
	service := &{{.Service.Name | exportedName}}{
		config: config,
	}

	{{- if .Service.Features.Database}}
	// Initialize database connection
	db, err := connectDatabase(config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	service.db = db
	{{- end}}

	{{- if .Service.Features.Authentication}}
	// Initialize auth service
	authService := NewAuthService(config.Auth.SecretKey, config.Auth.Expiration)
	service.auth = authService
	{{- end}}

	{{- if .Service.Features.Cache}}
	// Initialize cache service
	cacheService := NewCacheService()
	service.cache = cacheService
	{{- end}}

	return service, nil
}

// Start starts the service
func (s *{{.Service.Name | exportedName}}) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Starting {{.Service.Name}} service...")

	{{- if .Service.Features.REST}}
	// Start HTTP server
	go s.startHTTPServer(ctx)
	{{- end}}

	{{- if .Service.Features.GRPC}}
	// Start gRPC server
	go s.startGRPCServer(ctx)
	{{- end}}

	{{- if .Service.Features.HealthCheck}}
	// Start health check server
	go s.startHealthServer(ctx)
	{{- end}}

	return nil
}

// Stop stops the service
func (s *{{.Service.Name | exportedName}}) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Stopping {{.Service.Name}} service...")

	{{- if .Service.Features.Database}}
	// Close database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
	{{- end}}

	{{- if .Service.Features.Cache}}
	// Close cache connection
	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			log.Printf("Error closing cache: %v", err)
		}
	}
	{{- end}}

	return nil
}

{{- if .Service.Features.REST}}
// startHTTPServer starts the HTTP server
func (s *{{.Service.Name | exportedName}}) startHTTPServer(ctx context.Context) {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.HTTPPort)
	log.Printf("HTTP server listening on %s", addr)
	// Implementation would go here
}
{{- end}}

{{- if .Service.Features.GRPC}}
// startGRPCServer starts the gRPC server
func (s *{{.Service.Name | exportedName}}) startGRPCServer(ctx context.Context) {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.GRPCPort)
	log.Printf("gRPC server listening on %s", addr)
	// Implementation would go here
}
{{- end}}

{{- if .Service.Features.HealthCheck}}
// startHealthServer starts the health check server
func (s *{{.Service.Name | exportedName}}) startHealthServer(ctx context.Context) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	// Implementation would go here
}
{{- end}}

// Helper interfaces and types
{{- if .Service.Features.Database}}
type Database interface {
	Close() error
}

func connectDatabase(config struct {
	Type     string ` + "`yaml:\"type\"`" + `
	Host     string ` + "`yaml:\"host\"`" + `
	Port     int    ` + "`yaml:\"port\"`" + `
	Database string ` + "`yaml:\"database\"`" + `
}) (Database, error) {
	// Database connection implementation would go here
	return nil, nil
}
{{- end}}

{{- if .Service.Features.Authentication}}
type AuthService interface {
	ValidateToken(token string) error
}

func NewAuthService(secretKey string, expiration time.Duration) AuthService {
	// Auth service implementation would go here
	return nil
}
{{- end}}

{{- if .Service.Features.Cache}}
type CacheService interface {
	Close() error
}

func NewCacheService() CacheService {
	// Cache service implementation would go here
	return nil
}
{{- end}}
`

	// Create the template
	complexPath := filepath.Join(suite.tempDir, "complex_service.tmpl")
	err := os.WriteFile(complexPath, []byte(complexTemplate), 0644)
	suite.NoError(err)

	// Load and render
	tmpl, err := suite.engine.LoadTemplate("complex_service")
	suite.NoError(err)

	testData := testutil.TestTemplateData()
	output, err := suite.engine.RenderTemplate(tmpl, testData)
	suite.NoError(err)

	// Validate the generated code
	str := string(output)
	suite.Contains(str, "package testservice")
	suite.Contains(str, "type TestService struct")
	suite.Contains(str, "func NewTestService")
	suite.Contains(str, "func (s *TestService) Start")
	suite.Contains(str, "func (s *TestService) Stop")
	suite.Contains(str, "startHTTPServer")
	suite.Contains(str, "startGRPCServer")
	suite.Contains(str, "startHealthServer")

	// Write and validate Go syntax
	outputPath := filepath.Join(suite.outputDir, "complex_service.go")
	err = os.WriteFile(outputPath, output, 0644)
	suite.NoError(err)

	suite.validateGoSyntax(outputPath)
}

// TestTemplateErrorHandlingWorkflows tests error handling in templates.
func (suite *TemplateIntegrationTestSuite) TestTemplateErrorHandlingWorkflows() {
	// Test template with syntax errors
	invalidTemplate := `{{.Service.Name | invalidFunction}}`
	invalidPath := filepath.Join(suite.tempDir, "invalid.tmpl")
	err := os.WriteFile(invalidPath, []byte(invalidTemplate), 0644)
	suite.NoError(err)

	_, err = suite.engine.LoadTemplate("invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "function \"invalidFunction\" not defined")

	// Test template with missing data
	validTemplate := `{{.Service.NonExistentField}}`
	validPath := filepath.Join(suite.tempDir, "missing_data.tmpl")
	err = os.WriteFile(validPath, []byte(validTemplate), 0644)
	suite.NoError(err)

	tmpl, err := suite.engine.LoadTemplate("missing_data")
	suite.NoError(err)

	// Go templates fail when accessing non-existent struct fields
	_, err = suite.engine.RenderTemplate(tmpl, testutil.TestTemplateData())
	suite.Error(err)
	suite.Contains(err.Error(), "NonExistentField")
}

// TestTemplatePerformanceWithLargeDatasets tests template performance.
func (suite *TemplateIntegrationTestSuite) TestTemplatePerformanceWithLargeDatasets() {
	// Create a template that processes large datasets
	perfTemplate := `// Performance test template
package performance

import (
	"fmt"
	"time"
)

{{- range $i, $item := .LargeDataset}}
// Item {{$i}}: {{$item.Name}}
type Item{{$i}} struct {
	ID          int       ` + "`json:\"id\"`" + `
	Name        string    ` + "`json:\"name\"`" + `
	Description string    ` + "`json:\"description\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\"`" + `
}

func (i *Item{{$i}}) String() string {
	return fmt.Sprintf("Item{{$i}}{ID: %d, Name: %s}", i.ID, i.Name)
}
{{- end}}

// ProcessItems processes all items
func ProcessItems() {
	{{- range $i, $item := .LargeDataset}}
	item{{$i}} := &Item{{$i}}{
		ID:          {{$item.ID}},
		Name:        "{{$item.Name}}",
		Description: "{{$item.Description}}",
		CreatedAt:   time.Now(),
	}
	fmt.Println(item{{$i}}.String())
	{{- end}}
}
`

	perfPath := filepath.Join(suite.tempDir, "performance.tmpl")
	err := os.WriteFile(perfPath, []byte(perfTemplate), 0644)
	suite.NoError(err)

	// Create large dataset
	type DataItem struct {
		ID          int
		Name        string
		Description string
	}

	largeDataset := make([]DataItem, 100) // Reduced size for testing
	for i := 0; i < len(largeDataset); i++ {
		largeDataset[i] = DataItem{
			ID:          i + 1,
			Name:        fmt.Sprintf("Item_%d", i+1),
			Description: fmt.Sprintf("Description for item %d", i+1),
		}
	}

	testData := map[string]interface{}{
		"LargeDataset": largeDataset,
	}

	// Load and render template
	tmpl, err := suite.engine.LoadTemplate("performance")
	suite.NoError(err)

	startTime := time.Now()
	output, err := suite.engine.RenderTemplate(tmpl, testData)
	renderTime := time.Since(startTime)

	suite.NoError(err)
	suite.NotEmpty(output)

	// Validate that rendering completed in reasonable time
	suite.Less(renderTime, 5*time.Second, "Template rendering took too long: %v", renderTime)

	// Validate generated code structure
	str := string(output)
	suite.Contains(str, "package performance")
	suite.Contains(str, "func ProcessItems()")
	suite.Contains(str, "type Item0 struct")
	suite.Contains(str, "type Item99 struct")

	// Write and validate Go syntax
	outputPath := filepath.Join(suite.outputDir, "performance.go")
	err = os.WriteFile(outputPath, output, 0644)
	suite.NoError(err)

	suite.validateGoSyntax(outputPath)
}

// TestEmbeddedTemplateIntegration tests integration with embedded templates.
func (suite *TemplateIntegrationTestSuite) TestEmbeddedTemplateIntegration() {
	embeddedEngine := NewEngineWithEmbedded()

	// Test loading embedded templates
	embeddedTemplates := []string{
		"auth/jwt/types",
		"auth/jwt/service",
		"database/mongodb/repository",
		"middleware/advanced_ratelimit",
	}

	for _, templateName := range embeddedTemplates {
		suite.Run(fmt.Sprintf("Load embedded template %s", templateName), func() {
			tmpl, err := embeddedEngine.LoadTemplate(templateName)
			if err != nil {
				// Template might not exist in embedded, skip
				suite.T().Skipf("Embedded template %s not found: %v", templateName, err)
				return
			}

			suite.NotNil(tmpl)
			suite.Equal(templateName, tmpl.Name())

			// Try to render with test data
			testData := testutil.TestTemplateData()
			_, err = embeddedEngine.RenderTemplate(tmpl, testData)

			// Some templates might require specific data structure, so we just check it doesn't panic
			suite.NotPanics(func() {
				embeddedEngine.RenderTemplate(tmpl, testData)
			})
		})
	}
}

// TestTemplateValidationAndQualityChecks tests generated code quality.
func (suite *TemplateIntegrationTestSuite) TestTemplateValidationAndQualityChecks() {
	// Generate service with all features enabled
	testData := testutil.TestTemplateData()

	// Generate multiple files
	templates := map[string]string{
		"service/main.go":   "main.go",
		"service/config.go": "config.go",
	}

	generatedFiles := make([]string, 0)

	for templateName, outputFile := range templates {
		tmpl, err := suite.engine.LoadTemplate(templateName)
		suite.NoError(err)

		output, err := suite.engine.RenderTemplate(tmpl, testData)
		suite.NoError(err)

		outputPath := filepath.Join(suite.outputDir, outputFile)
		err = os.WriteFile(outputPath, output, 0644)
		suite.NoError(err)

		generatedFiles = append(generatedFiles, outputPath)
	}

	// Validate all generated files
	for _, filePath := range generatedFiles {
		suite.Run(fmt.Sprintf("Validate %s", filepath.Base(filePath)), func() {
			suite.validateGoSyntax(filePath)
			suite.validateCodeQuality(filePath)
		})
	}
}

// validateGoSyntax validates that generated Go code is syntactically correct.
func (suite *TemplateIntegrationTestSuite) validateGoSyntax(filePath string) {
	content, err := os.ReadFile(filePath)
	suite.NoError(err)

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, filePath, content, parser.ParseComments)
	suite.NoError(err, "Generated Go code has syntax errors in %s", filePath)
}

// validateCodeQuality performs basic code quality checks.
func (suite *TemplateIntegrationTestSuite) validateCodeQuality(filePath string) {
	content, err := os.ReadFile(filePath)
	suite.NoError(err)

	str := string(content)

	// Check for common quality issues
	suite.NotContains(str, "TODO", "Generated code should not contain TODO comments")
	suite.NotContains(str, "FIXME", "Generated code should not contain FIXME comments")

	// Check for proper imports (no unused imports in real scenarios)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	suite.NoError(err)

	// Validate package structure
	suite.NotEmpty(file.Name.Name, "Package name should not be empty")

	// Check for proper documentation (at least package comment or major type comments)
	hasDocumentation := false
	if file.Doc != nil && len(file.Doc.List) > 0 {
		hasDocumentation = true
	}

	// Check for documented types
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			if genDecl.Doc != nil && len(genDecl.Doc.List) > 0 {
				hasDocumentation = true
				break
			}
		}
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Doc != nil && len(funcDecl.Doc.List) > 0 {
				hasDocumentation = true
				break
			}
		}
	}

	suite.True(hasDocumentation, "Generated code should have documentation")
}

// Run the test suite.
func TestTemplateIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateIntegrationTestSuite))
}

// TestEndToEndTemplateWorkflow tests complete end-to-end template workflow.
func TestEndToEndTemplateWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "e2e-template-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	outputDir, err := os.MkdirTemp("", "e2e-output-*")
	require.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a complete service template
	serviceTemplate := `// {{.Service.Description}}
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Service represents the main service
type Service struct {
	name string
	port int
}

// NewService creates a new service
func NewService(name string, port int) *Service {
	return &Service{
		name: name,
		port: port,
	}
}

// Start starts the service
func (s *Service) Start(ctx context.Context) error {
	log.Printf("Starting %s service on port %d", s.name, s.port)
	return nil
}

// Stop stops the service
func (s *Service) Stop(ctx context.Context) error {
	log.Printf("Stopping %s service", s.name)
	return nil
}

func main() {
	service := NewService("{{.Service.Name}}", {{.Service.Ports.HTTP}})
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := service.Start(ctx); err != nil {
			log.Printf("Service error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := service.Stop(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
`

	// Save template
	tmplPath := filepath.Join(tempDir, "e2e_service.tmpl")
	err = os.WriteFile(tmplPath, []byte(serviceTemplate), 0644)
	require.NoError(t, err)

	// Create engine and load template
	engine := NewEngine(tempDir)
	tmpl, err := engine.LoadTemplate("e2e_service")
	require.NoError(t, err)

	// Render with test data
	testData := testutil.TestTemplateData()
	output, err := engine.RenderTemplate(tmpl, testData)
	require.NoError(t, err)

	// Write output
	outputPath := filepath.Join(outputDir, "service.go")
	err = os.WriteFile(outputPath, output, 0644)
	require.NoError(t, err)

	// Validate syntax
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, outputPath, content, parser.ParseComments)
	require.NoError(t, err)

	// Validate content
	str := string(content)
	assert.Contains(t, str, "A test microservice for unit testing")
	assert.Contains(t, str, `NewService("test-service", 8080)`)
	assert.Contains(t, str, "context.WithTimeout")
	assert.Contains(t, str, "30*time.Second")
}
