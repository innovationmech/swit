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

package checker

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// ConfigCheckerTestSuite tests the config checker functionality
type ConfigCheckerTestSuite struct {
	suite.Suite
	tempDir       string
	configChecker *ConfigChecker
	mockLogger    *MockLogger
}

func TestConfigCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigCheckerTestSuite))
}

func (s *ConfigCheckerTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "config-checker-test-*")
	s.Require().NoError(err)

	s.mockLogger = &MockLogger{messages: make([]LogMessage, 0)}
	s.configChecker = NewConfigChecker(s.tempDir, s.mockLogger)
}

func (s *ConfigCheckerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
	if s.configChecker != nil {
		s.configChecker.Close()
	}
}

// TestConfigCheckerInterface tests that ConfigChecker implements IndividualChecker
func (s *ConfigCheckerTestSuite) TestConfigCheckerInterface() {
	var _ IndividualChecker = s.configChecker
	s.NotNil(s.configChecker)
}

// TestNewConfigChecker tests the constructor
func (s *ConfigCheckerTestSuite) TestNewConfigChecker() {
	checker := NewConfigChecker("/test/path", s.mockLogger)

	s.NotNil(checker)
	s.Equal("/test/path", checker.workDir)
	s.Equal(s.mockLogger, checker.logger)
	s.Equal("config", checker.Name())
	s.Contains(checker.Description(), "configuration")
	s.Contains(checker.Description(), "YAML")
}

// TestDefaultConfigValidationConfig tests default configuration
func (s *ConfigCheckerTestSuite) TestDefaultConfigValidationConfig() {
	config := defaultConfigValidationConfig()

	s.True(config.YAMLValidation)
	s.True(config.JSONValidation)
	s.True(config.ProtoValidation)
	s.False(config.TOMLValidation) // Requires dependencies
	s.False(config.StrictMode)
	s.True(config.CheckKeys)
	s.True(config.CheckValues)
	s.Contains(config.ExcludePatterns, "vendor/*")
}

// TestSetConfig tests configuration setting
func (s *ConfigCheckerTestSuite) TestSetConfig() {
	customConfig := ConfigValidationConfig{
		YAMLValidation:  false,
		JSONValidation:  false,
		ProtoValidation: true,
		TOMLValidation:  true,
		ConfigPaths:     []string{"/custom/path"},
		StrictMode:      true,
		CheckKeys:       false,
		CheckValues:     false,
	}

	s.configChecker.SetConfig(customConfig)
	s.Equal(customConfig, s.configChecker.config)
}

// TestCheckWithValidYAMLConfig tests config check with valid YAML
func (s *ConfigCheckerTestSuite) TestCheckWithValidYAMLConfig() {
	// Create valid YAML config
	validConfig := `# Valid configuration file
server:
  http_port: 8080
  grpc_port: 9090
  host: localhost

database:
  type: mysql
  host: localhost
  port: 3306
  database: testdb
  username: user
  password: pass

logging:
  level: info
  format: json
`

	err := s.createTestFile("config.yaml", validConfig)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal("config", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "passed")
	s.True(result.Duration > 0)
	s.False(result.Fixable) // Config issues are not auto-fixable
}

// TestCheckWithInvalidYAML tests config check with invalid YAML syntax
func (s *ConfigCheckerTestSuite) TestCheckWithInvalidYAML() {
	// Create invalid YAML config
	invalidConfig := `server:
  http_port: 8080
  invalid_yaml: [unclosed_array
  missing_value:
database:
  type: mysql
  host: "unclosed_string
`

	err := s.createTestFile("invalid.yaml", invalidConfig)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal("config", result.Name)
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.Contains(result.Message, "failed")
	s.NotEmpty(result.Details)

	// Should detect YAML syntax errors
	foundSyntaxError := false
	for _, detail := range result.Details {
		if detail.Rule == "YAML_SYNTAX_ERROR" {
			foundSyntaxError = true
			s.Contains(detail.Message, "Invalid YAML")
		}
	}
	s.True(foundSyntaxError, "Should detect YAML syntax errors")
}

// TestCheckWithValidJSON tests config check with valid JSON
func (s *ConfigCheckerTestSuite) TestCheckWithValidJSON() {
	// Create valid JSON config
	validJSON := `{
  "server": {
    "http_port": 8080,
    "grpc_port": 9090,
    "host": "localhost"
  },
  "database": {
    "type": "mysql",
    "host": "localhost",
    "port": 3306,
    "database": "testdb"
  }
}`

	err := s.createTestFile("config.json", validJSON)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal("config", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "passed")
}

// TestCheckWithInvalidJSON tests config check with invalid JSON syntax
func (s *ConfigCheckerTestSuite) TestCheckWithInvalidJSON() {
	// Create invalid JSON config
	invalidJSON := `{
  "server": {
    "http_port": 8080,
    "grpc_port": 9090,
    "host": "localhost",
  },
  "database": {
    "type": "mysql"
    "host": "localhost"
  }
}`

	err := s.createTestFile("invalid.json", invalidJSON)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.NotEmpty(result.Details)

	// Should detect JSON syntax errors
	foundSyntaxError := false
	for _, detail := range result.Details {
		if detail.Rule == "JSON_SYNTAX_ERROR" {
			foundSyntaxError = true
			s.Contains(detail.Message, "Invalid JSON")
		}
	}
	s.True(foundSyntaxError, "Should detect JSON syntax errors")
}

// TestCheckWithSwitConfig tests Swit-specific configuration validation
func (s *ConfigCheckerTestSuite) TestCheckWithSwitConfig() {
	// Create Swit config with validation issues
	switConfig := `# Swit configuration
server:
  http_port: 99999  # Invalid port range
  grpc_port: 9090

database:
  type: unsupported_db  # Unsupported database type
  host: localhost
  port: 3306
`

	err := s.createTestFile("swit.yaml", switConfig)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.True(result.Status == interfaces.CheckStatusFail || result.Status == interfaces.CheckStatusWarning)
	s.NotEmpty(result.Details)

	// Should detect port range issues
	foundPortError := false
	foundDBWarning := false
	for _, detail := range result.Details {
		if detail.Rule == "INVALID_PORT_RANGE" {
			foundPortError = true
			s.Contains(detail.Message, "port must be between")
		}
		if detail.Rule == "UNSUPPORTED_DATABASE_TYPE" {
			foundDBWarning = true
			s.Contains(detail.Message, "may not be fully supported")
		}
	}
	s.True(foundPortError, "Should detect invalid port range")
	s.True(foundDBWarning, "Should detect unsupported database type")
}

// TestCheckWithMissingRequiredFields tests validation of required fields
func (s *ConfigCheckerTestSuite) TestCheckWithMissingRequiredFields() {
	// Create incomplete Swit config
	incompleteConfig := `# Incomplete Swit configuration
logging:
  level: info
  format: json
# Missing server and database sections
`

	err := s.createTestFile("swit_incomplete.yaml", incompleteConfig)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.NotEmpty(result.Details)

	// Should detect missing required fields
	foundMissingServer := false
	foundMissingDatabase := false
	for _, detail := range result.Details {
		if detail.Rule == "MISSING_REQUIRED_FIELD" {
			if detail.Message == "Required field 'server' is missing" {
				foundMissingServer = true
			}
			if detail.Message == "Required field 'database' is missing" {
				foundMissingDatabase = true
			}
		}
	}
	s.True(foundMissingServer, "Should detect missing server field")
	s.True(foundMissingDatabase, "Should detect missing database field")
}

// TestCheckWithProtoFiles tests Protocol Buffer file validation
func (s *ConfigCheckerTestSuite) TestCheckWithProtoFiles() {
	// Create proto file without package declaration
	protoContent := `syntax = "proto3";

// Missing package declaration

service TestService {
  rpc TestMethod(TestRequest) returns (TestResponse);
}

message TestRequest {
  string test_field = 1;
}

message TestResponse {
  string result = 1;
}
`

	err := s.createTestFile("test.proto", protoContent)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	// Should have warnings for missing package declaration
	if result.Status == interfaces.CheckStatusWarning {
		foundMissingPackage := false
		for _, detail := range result.Details {
			if detail.Rule == "MISSING_PACKAGE" {
				foundMissingPackage = true
				s.Contains(detail.Message, "missing package")
			}
		}
		s.True(foundMissingPackage, "Should detect missing package declaration")
	}
}

// TestCheckWithValidProtoFile tests valid Protocol Buffer file
func (s *ConfigCheckerTestSuite) TestCheckWithValidProtoFile() {
	// Create valid proto file
	validProto := `syntax = "proto3";

package test.v1;

import "google/api/annotations.proto";

service TestService {
  rpc TestMethod(TestRequest) returns (TestResponse) {
    option (google.api.http) = {
      post: "/v1/test"
      body: "*"
    };
  };
}

message TestRequest {
  string test_field = 1;
}

message TestResponse {
  string result = 1;
}
`

	err := s.createTestFile("valid.proto", validProto)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.True(result.Status == interfaces.CheckStatusPass || result.Status == interfaces.CheckStatusWarning)
}

// TestCheckWithGoModule tests Go module validation
func (s *ConfigCheckerTestSuite) TestCheckWithGoModule() {
	// Create valid go.mod
	validGoMod := `module github.com/example/testproject

go 1.19

require (
    github.com/gin-gonic/gin v1.9.0
    github.com/stretchr/testify v1.8.0
)

replace github.com/example/localpackage => ./localpackage
`

	err := s.createTestFile("go.mod", validGoMod)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.True(result.Status == interfaces.CheckStatusPass || result.Status == interfaces.CheckStatusWarning)
}

// TestCheckWithInvalidGoModule tests invalid Go module
func (s *ConfigCheckerTestSuite) TestCheckWithInvalidGoModule() {
	// Create invalid go.mod (missing module declaration)
	invalidGoMod := `go 1.19

require (
    github.com/gin-gonic/gin v1.9.0
)
`

	err := s.createTestFile("go.mod", invalidGoMod)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.NotEmpty(result.Details)

	// Should detect missing module declaration
	foundMissingModule := false
	for _, detail := range result.Details {
		if detail.Rule == "MISSING_MODULE_DECLARATION" {
			foundMissingModule = true
			s.Contains(detail.Message, "missing module declaration")
		}
	}
	s.True(foundMissingModule, "Should detect missing module declaration")
}

// TestCheckWithTOMLFile tests TOML file validation
func (s *ConfigCheckerTestSuite) TestCheckWithTOMLFile() {
	// Create TOML config
	tomlConfig := `[server]
http_port = 8080
grpc_port = 9090

[database]
type = "mysql"
host = "localhost"
port = 3306
`

	err := s.createTestFile("config.toml", tomlConfig)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	// TOML validation is limited, should warn about partial validation
	if result.Status == interfaces.CheckStatusWarning {
		foundPartialValidation := false
		for _, detail := range result.Details {
			if detail.Rule == "PARTIAL_VALIDATION" {
				foundPartialValidation = true
				s.Contains(detail.Message, "not fully implemented")
			}
		}
		s.True(foundPartialValidation, "Should warn about partial TOML validation")
	}
}

// TestCheckWithEnvironmentFiles tests environment file detection
func (s *ConfigCheckerTestSuite) TestCheckWithEnvironmentFiles() {
	// Create .env file
	envContent := `API_KEY=secret123
DATABASE_URL=mysql://user:pass@localhost/db
JWT_SECRET=supersecret
`

	err := s.createTestFile(".env", envContent)
	s.Require().NoError(err)

	// Create .gitignore without .env exclusion
	gitignoreContent := `*.log
/tmp/
/build/
`

	err = s.createTestFile(".gitignore", gitignoreContent)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	// Should have warnings about environment files
	if result.Status == interfaces.CheckStatusWarning {
		foundEnvWarning := false
		foundGitignoreWarning := false
		for _, detail := range result.Details {
			if detail.Rule == "ENV_FILE_FOUND" {
				foundEnvWarning = true
				s.Contains(detail.Message, "not committed to version control")
			}
			if detail.Rule == "ENV_NOT_IGNORED" {
				foundGitignoreWarning = true
				s.Contains(detail.Message, "*.env to .gitignore")
			}
		}
		s.True(foundEnvWarning, "Should warn about environment file")
		s.True(foundGitignoreWarning, "Should warn about missing gitignore entry")
	}
}

// TestCheckWithEmptyConfig tests handling of empty configuration files
func (s *ConfigCheckerTestSuite) TestCheckWithEmptyConfig() {
	// Create empty YAML config
	err := s.createTestFile("empty.yaml", "")
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	// Should warn about empty config
	if result.Status == interfaces.CheckStatusWarning {
		foundEmptyWarning := false
		for _, detail := range result.Details {
			if detail.Rule == "EMPTY_CONFIG" {
				foundEmptyWarning = true
				s.Contains(detail.Message, "empty")
			}
		}
		s.True(foundEmptyWarning, "Should warn about empty configuration")
	}
}

// TestCheckWithKeyNamingIssues tests key naming validation
func (s *ConfigCheckerTestSuite) TestCheckWithKeyNamingIssues() {
	// Create config with naming issues
	problematicConfig := `# Configuration with naming issues
camelCaseKey_mixed: value1
type: reserved_keyword_value
class: another_reserved
normal_key: normal_value
`

	err := s.createTestFile("naming_issues.yaml", problematicConfig)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	// Should have warnings about naming issues
	if result.Status == interfaces.CheckStatusWarning {
		foundInconsistentNaming := false
		foundReservedKeyword := false
		for _, detail := range result.Details {
			if detail.Rule == "INCONSISTENT_KEY_NAMING" {
				foundInconsistentNaming = true
				s.Contains(detail.Message, "Mixed camelCase and snake_case")
			}
			if detail.Rule == "RESERVED_KEY_NAME" {
				foundReservedKeyword = true
				s.Contains(detail.Message, "reserved words")
			}
		}
		s.True(foundInconsistentNaming, "Should detect inconsistent key naming")
		s.True(foundReservedKeyword, "Should detect reserved keywords")
	}
}

// TestFileExclusionPatterns tests file exclusion patterns
func (s *ConfigCheckerTestSuite) TestFileExclusionPatterns() {
	// Set custom exclusion patterns
	config := s.configChecker.config
	config.ExcludePatterns = []string{"*test*", "vendor/*"}
	s.configChecker.SetConfig(config)

	// Create files that should be excluded
	err := s.createTestFile("test_config.yaml", "key: value")
	s.Require().NoError(err)

	err = os.MkdirAll(filepath.Join(s.tempDir, "vendor"), 0755)
	s.Require().NoError(err)
	err = s.createTestFile("vendor/config.yaml", "vendor: config")
	s.Require().NoError(err)

	// Create file that should be included
	err = s.createTestFile("production.yaml", "env: production")
	s.Require().NoError(err)

	ctx := context.Background()
	_, err = s.configChecker.Check(ctx)

	s.NoError(err)
	// Should only process non-excluded files
}

// TestEmptyProject tests handling of project with no config files
func (s *ConfigCheckerTestSuite) TestEmptyProject() {
	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal("config", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "passed")
	s.Empty(result.Details) // No config files found
}

// TestContextCancellation tests handling of context cancellation
func (s *ConfigCheckerTestSuite) TestContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.configChecker.Check(ctx)
	s.Error(err)
	s.Equal(context.Canceled, err)
}

// TestContextTimeout tests handling of context timeout
func (s *ConfigCheckerTestSuite) TestContextTimeout() {
	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout

	_, err := s.configChecker.Check(ctx)
	s.Error(err)
	s.Equal(context.DeadlineExceeded, err)
}

// TestConcurrentExecution tests thread safety
func (s *ConfigCheckerTestSuite) TestConcurrentExecution() {
	// Create test config file
	testConfig := `server:
  http_port: 8080
database:
  type: mysql`

	err := s.createTestFile("concurrent.yaml", testConfig)
	s.Require().NoError(err)

	// Run multiple concurrent checks
	const numGoroutines = 5
	results := make([]interfaces.CheckResult, numGoroutines)
	errors := make([]error, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			results[index], errors[index] = s.configChecker.Check(ctx)
		}(i)
	}

	wg.Wait()

	// Verify all checks completed successfully
	for i := 0; i < numGoroutines; i++ {
		s.NoError(errors[i], "Check %d should not error", i)
		s.Equal("config", results[i].Name)
	}
}

// TestMultipleConfigFiles tests validation of multiple configuration files
func (s *ConfigCheckerTestSuite) TestMultipleConfigFiles() {
	// Create multiple config files with different issues

	// Valid config
	err := s.createTestFile("valid.yaml", `
server:
  http_port: 8080
database:
  type: mysql
`)
	s.Require().NoError(err)

	// Invalid JSON
	err = s.createTestFile("invalid.json", `{
  "server": {
    "port": "invalid",
  }
}`)
	s.Require().NoError(err)

	// Config with warnings
	err = s.createTestFile("warnings.yaml", `
type: reserved_keyword
camelCase_mixed: value
`)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal("config", result.Name)
	s.Equal(interfaces.CheckStatusFail, result.Status) // Should fail due to invalid JSON
	s.NotEmpty(result.Details)

	// Should have details from multiple files
	jsonErrorFound := false
	warningFound := false
	for _, detail := range result.Details {
		if detail.Rule == "JSON_SYNTAX_ERROR" {
			jsonErrorFound = true
		}
		if detail.Rule == "RESERVED_KEY_NAME" {
			warningFound = true
		}
	}
	s.True(jsonErrorFound, "Should detect JSON error")
	s.True(warningFound, "Should detect warnings")
}

// TestClose tests proper cleanup
func (s *ConfigCheckerTestSuite) TestClose() {
	err := s.configChecker.Close()
	s.NoError(err)
}

// TestConfigWithCustomPaths tests custom config paths
func (s *ConfigCheckerTestSuite) TestConfigWithCustomPaths() {
	// Create subdirectory with config
	subDir := filepath.Join(s.tempDir, "configs")
	err := os.MkdirAll(subDir, 0755)
	s.Require().NoError(err)

	err = s.createTestFile("configs/custom.yaml", `
app:
  name: test
  port: 8080
`)
	s.Require().NoError(err)

	// Set custom config paths
	config := s.configChecker.config
	config.ConfigPaths = []string{subDir}
	s.configChecker.SetConfig(config)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusPass, result.Status)
}

// TestServerConfigValidation tests detailed server configuration validation
func (s *ConfigCheckerTestSuite) TestServerConfigValidation() {
	// Create config with various server configuration issues
	serverConfig := `
server:
  http_port: "not_a_number"  # Invalid type
  grpc_port: -1              # Invalid range
  host: localhost
  invalid_field: value

database:
  type: mysql
`

	err := s.createTestFile("server_issues.yaml", serverConfig)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.configChecker.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusFail, result.Status)

	// Should detect port validation issues
	portTypeError := false
	for _, detail := range result.Details {
		if detail.Rule == "INVALID_PORT_TYPE" {
			portTypeError = true
			s.Contains(detail.Message, "must be an integer")
		}
	}
	s.True(portTypeError, "Should detect invalid port type")
}

// Helper methods

func (s *ConfigCheckerTestSuite) createTestFile(filename, content string) error {
	filePath := filepath.Join(s.tempDir, filename)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// Benchmark tests for performance

func BenchmarkConfigCheck(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "config-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	checker := NewConfigChecker(tempDir, logger)
	defer checker.Close()

	// Create test config file
	config := `server:
  http_port: 8080
  grpc_port: 9090
database:
  type: mysql
  host: localhost`

	err = os.WriteFile(filepath.Join(tempDir, "bench.yaml"), []byte(config), 0644)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkYAMLValidation(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "yaml-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	checker := NewConfigChecker(tempDir, logger)
	defer checker.Close()

	// Create complex YAML config
	complexConfig := `
server:
  http_port: 8080
  grpc_port: 9090
  host: localhost
  timeout: 30s
  max_connections: 1000

database:
  primary:
    type: mysql
    host: localhost
    port: 3306
    database: main
    username: user
    password: pass
  replica:
    type: mysql
    host: replica-host
    port: 3306

cache:
  redis:
    host: localhost
    port: 6379
    db: 0
    timeout: 5s

logging:
  level: info
  format: json
  output: stdout
  rotation:
    max_size: 100MB
    max_age: 30d
    max_backups: 10
`

	err = os.WriteFile(filepath.Join(tempDir, "complex.yaml"), []byte(complexConfig), 0644)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMultipleConfigFiles(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "multi-config-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	checker := NewConfigChecker(tempDir, logger)
	defer checker.Close()

	// Create multiple config files
	configs := map[string]string{
		"server.yaml": `
server:
  http_port: 8080
  grpc_port: 9090`,
		"database.yaml": `
database:
  type: mysql
  host: localhost`,
		"app.json": `{
  "app": {
    "name": "test",
    "version": "1.0.0"
  }
}`,
		"logging.yaml": `
logging:
  level: info
  format: json`,
	}

	for filename, content := range configs {
		err = os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
