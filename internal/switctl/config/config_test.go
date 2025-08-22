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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger implements the Logger interface for testing.
type MockLogger struct {
	Logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Fields  []interface{}
}

func (ml *MockLogger) Debug(msg string, fields ...interface{}) {
	ml.Logs = append(ml.Logs, LogEntry{Level: "debug", Message: msg, Fields: fields})
}

func (ml *MockLogger) Info(msg string, fields ...interface{}) {
	ml.Logs = append(ml.Logs, LogEntry{Level: "info", Message: msg, Fields: fields})
}

func (ml *MockLogger) Warn(msg string, fields ...interface{}) {
	ml.Logs = append(ml.Logs, LogEntry{Level: "warn", Message: msg, Fields: fields})
}

func (ml *MockLogger) Error(msg string, fields ...interface{}) {
	ml.Logs = append(ml.Logs, LogEntry{Level: "error", Message: msg, Fields: fields})
}

func (ml *MockLogger) Fatal(msg string, fields ...interface{}) {
	ml.Logs = append(ml.Logs, LogEntry{Level: "fatal", Message: msg, Fields: fields})
}

func TestNewHierarchicalConfigManager(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}

	manager := NewHierarchicalConfigManager(workDir, logger)

	assert.NotNil(t, manager)
	assert.Equal(t, workDir, manager.workDir)
	assert.Equal(t, logger, manager.logger)
	assert.Equal(t, "SWITCTL", manager.envPrefix)
	assert.NotEmpty(t, manager.defaults)
	assert.NotEmpty(t, manager.validators)
	assert.NotEmpty(t, manager.configPaths)
}

func TestConfigLevel_String(t *testing.T) {
	tests := []struct {
		level    ConfigLevel
		expected string
	}{
		{GlobalLevel, "global"},
		{UserLevel, "user"},
		{ProjectLevel, "project"},
		{EnvironmentLevel, "environment"},
		{ConfigLevel(99), "unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.level.String())
	}
}

func TestHierarchicalConfigManager_Load(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Create a project-level config file
	projectConfigPath := filepath.Join(workDir, ".switctl.yaml")
	projectConfigContent := `
logging:
  level: debug
  format: json
server:
  http:
    port: 8000
  grpc:
    port: 9000
database:
  type: postgresql
  host: db.example.com
  port: 5432
`
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfigContent), 0644))

	// Set environment variables
	os.Setenv("SWITCTL_CACHE_HOST", "redis.example.com")
	os.Setenv("SWITCTL_CACHE_PORT", "6380")
	defer func() {
		os.Unsetenv("SWITCTL_CACHE_HOST")
		os.Unsetenv("SWITCTL_CACHE_PORT")
	}()

	// Load configuration
	err := manager.Load()
	require.NoError(t, err)

	// Test merged configuration
	assert.Equal(t, "debug", manager.GetString("logging.level"))
	assert.Equal(t, "json", manager.GetString("logging.format"))
	assert.Equal(t, 8000, manager.GetInt("server.http.port"))
	assert.Equal(t, 9000, manager.GetInt("server.grpc.port"))
	assert.Equal(t, "postgresql", manager.GetString("database.type"))
	assert.Equal(t, "db.example.com", manager.GetString("database.host"))
	assert.Equal(t, 5432, manager.GetInt("database.port"))

	// Test environment variable override
	assert.Equal(t, "redis.example.com", manager.GetString("cache.host"))
	assert.Equal(t, "6380", manager.GetString("cache.port")) // Environment variables are strings

	// Test defaults for unspecified values
	assert.Equal(t, 8080, manager.GetInt("server.health.port"))
	assert.Equal(t, "redis", manager.GetString("cache.type"))
}

func TestHierarchicalConfigManager_Set(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Load initial configuration
	err := manager.Load()
	require.NoError(t, err)

	// Set a value
	manager.Set("test.key", "test.value")

	// Verify the value was set
	assert.Equal(t, "test.value", manager.GetString("test.key"))

	// Verify it was set at project level
	projectConfig := manager.GetConfigLevel(ProjectLevel)
	assert.Equal(t, "test.value", projectConfig["test"].(map[string]interface{})["key"])
}

func TestHierarchicalConfigManager_SetLevel(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Test setting at user level
	err = manager.SetLevel(UserLevel, "user.setting", "user.value")
	require.NoError(t, err)
	assert.Equal(t, "user.value", manager.GetString("user.setting"))

	// Test setting at project level
	err = manager.SetLevel(ProjectLevel, "project.setting", "project.value")
	require.NoError(t, err)
	assert.Equal(t, "project.value", manager.GetString("project.setting"))

	// Test that environment level cannot be set
	err = manager.SetLevel(EnvironmentLevel, "env.setting", "env.value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set environment level configuration")
}

func TestHierarchicalConfigManager_GetTypes(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Set up test configuration
	manager.Set("string.value", "hello")
	manager.Set("int.value", 42)
	manager.Set("bool.value", true)
	manager.Set("duration.value", "5m")
	manager.Set("slice.value", []string{"a", "b", "c"})

	// Test type-specific getters
	assert.Equal(t, "hello", manager.GetString("string.value"))
	assert.Equal(t, 42, manager.GetInt("int.value"))
	assert.Equal(t, true, manager.GetBool("bool.value"))
	assert.Equal(t, 5*time.Minute, manager.GetDuration("duration.value"))
	assert.Equal(t, []string{"a", "b", "c"}, manager.GetStringSlice("slice.value"))

	// Test missing values return zero values
	assert.Equal(t, "", manager.GetString("missing.string"))
	assert.Equal(t, 0, manager.GetInt("missing.int"))
	assert.Equal(t, false, manager.GetBool("missing.bool"))
	assert.Equal(t, time.Duration(0), manager.GetDuration("missing.duration"))
	assert.Nil(t, manager.GetStringSlice("missing.slice"))
}

func TestHierarchicalConfigManager_Validation(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Add a custom validator
	manager.AddValidator("custom", func(key string, value interface{}) error {
		if value == "invalid" {
			return fmt.Errorf("value 'invalid' is not allowed for %s", key)
		}
		return nil
	})

	err := manager.Load()
	require.NoError(t, err)

	// Test successful validation
	manager.Set("custom.field", "valid")
	err = manager.Validate()
	assert.NoError(t, err)

	// Test validation failure
	manager.Set("custom.field", "invalid")
	err = manager.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value 'invalid' is not allowed")

	// Test port validation
	manager.Set("custom.field", "valid") // Reset to valid value first
	manager.Set("server.port", 80)
	err = manager.Validate()
	assert.NoError(t, err)

	manager.Set("server.port", 99999)
	err = manager.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port 99999")
}

func TestHierarchicalConfigManager_Save(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Set some values
	manager.Set("test.string", "value")
	manager.Set("test.number", 123)
	manager.Set("test.bool", true)

	// Test saving to YAML
	yamlPath := filepath.Join(workDir, "test.yaml")
	err = manager.Save(yamlPath)
	require.NoError(t, err)

	// Verify file exists and contains expected content
	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test:")
	assert.Contains(t, string(content), "string: value")
	assert.Contains(t, string(content), "number: 123")
	assert.Contains(t, string(content), "bool: true")

	// Test saving to JSON
	jsonPath := filepath.Join(workDir, "test.json")
	err = manager.SaveLevel(ProjectLevel, jsonPath)
	require.NoError(t, err)

	content, err = os.ReadFile(jsonPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "\"test\":")
	assert.Contains(t, string(content), "\"string\": \"value\"")
	assert.Contains(t, string(content), "\"number\": 123")
	assert.Contains(t, string(content), "\"bool\": true")
}

func TestHierarchicalConfigManager_Merge(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Merge external configuration
	externalConfig := map[string]interface{}{
		"external": map[string]interface{}{
			"setting1": "value1",
			"setting2": 42,
		},
		"database": map[string]interface{}{
			"type": "mongodb",
			"host": "mongo.example.com",
		},
	}

	err = manager.Merge(UserLevel, externalConfig)
	require.NoError(t, err)

	// Test merged values
	assert.Equal(t, "value1", manager.GetString("external.setting1"))
	assert.Equal(t, 42, manager.GetInt("external.setting2"))
	assert.Equal(t, "mongodb", manager.GetString("database.type"))
	assert.Equal(t, "mongo.example.com", manager.GetString("database.host"))
}

func TestHierarchicalConfigManager_Export(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Set some values at project level
	manager.Set("test.key1", "value1")
	manager.Set("test.key2", 123)

	// Export project level configuration
	exported, err := manager.Export(ProjectLevel)
	require.NoError(t, err)

	assert.NotNil(t, exported["test"])
	testSection := exported["test"].(map[string]interface{})
	assert.Equal(t, "value1", testSection["key1"])
	assert.Equal(t, 123, testSection["key2"])

	// Test exporting non-existent level
	exported, err = manager.Export(GlobalLevel)
	require.NoError(t, err)
	assert.Empty(t, exported)
}

func TestHierarchicalConfigManager_Clone(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Set some values
	manager.Set("test.original", "value")

	// Clone the manager
	clone := manager.Clone()

	// Verify clone has same properties
	assert.Equal(t, manager.workDir, clone.workDir)
	assert.Equal(t, manager.envPrefix, clone.envPrefix)
	assert.Equal(t, manager.logger, clone.logger)

	// Verify clone has same defaults
	assert.Equal(t, len(manager.defaults), len(clone.defaults))
	for key, value := range manager.defaults {
		assert.Equal(t, value, clone.defaults[key])
	}

	// Verify clone has same config paths
	assert.Equal(t, len(manager.configPaths), len(clone.configPaths))
	for level, path := range manager.configPaths {
		assert.Equal(t, path, clone.configPaths[level])
	}
}

func TestHierarchicalConfigManager_Reset(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Set some values
	manager.Set("test.key", "value")
	assert.Equal(t, "value", manager.GetString("test.key"))

	// Reset the manager
	err = manager.Reset()
	require.NoError(t, err)

	// Verify configuration is cleared
	assert.Equal(t, "", manager.GetString("test.key"))
	assert.Empty(t, manager.configs)
	assert.Nil(t, manager.mergedConfig)
}

func TestHierarchicalConfigManager_ChangeHandlers(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Track changes
	var changes []string
	manager.AddChangeHandler(func(level ConfigLevel, key string, oldValue, newValue interface{}) error {
		changes = append(changes, fmt.Sprintf("%s:%s:%v->%v", level.String(), key, oldValue, newValue))
		return nil
	})

	// Set a value
	manager.Set("test.change", "new_value")

	// Verify change was tracked
	assert.Len(t, changes, 1)
	assert.Contains(t, changes[0], "project:test.change:")
	assert.Contains(t, changes[0], "->new_value")
}

func TestHierarchicalConfigManager_ConfigPaths(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Get initial config paths
	paths := manager.GetConfigPaths()
	assert.Contains(t, paths, ProjectLevel)
	assert.Equal(t, filepath.Join(workDir, ".switctl.yaml"), paths[ProjectLevel])

	// Set a custom config path
	customPath := filepath.Join(workDir, "custom.yaml")
	manager.SetConfigPath(ProjectLevel, customPath)

	// Verify path was updated
	updatedPaths := manager.GetConfigPaths()
	assert.Equal(t, customPath, updatedPaths[ProjectLevel])
}

func TestHierarchicalConfigManager_GetMergedConfig(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Set some values
	manager.Set("test.key1", "value1")
	manager.Set("test.key2", 123)

	// Get merged configuration
	merged := manager.GetMergedConfig()

	assert.NotNil(t, merged["test"])
	testSection := merged["test"].(map[string]interface{})
	assert.Equal(t, "value1", testSection["key1"])
	assert.Equal(t, 123, testSection["key2"])

	// Verify defaults are included
	assert.Equal(t, "info", merged["logging"].(map[string]interface{})["level"])
}

func TestValidators(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Test port validator
	portValidator := manager.validators["port"]

	// Valid port
	err := portValidator("test.port", 8080)
	assert.NoError(t, err)

	// Invalid port - too low
	err = portValidator("test.port", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port 0")

	// Invalid port - too high
	err = portValidator("test.port", 70000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port 70000")

	// Test URL validator
	urlValidator := manager.validators["url"]

	// Valid URLs
	err = urlValidator("test.url", "http://example.com")
	assert.NoError(t, err)

	err = urlValidator("test.url", "https://example.com")
	assert.NoError(t, err)

	// Invalid URL
	err = urlValidator("test.url", "ftp://example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")

	// Test required validator
	requiredValidator := manager.validators["required"]

	// Valid value
	err = requiredValidator("test.required", "value")
	assert.NoError(t, err)

	// Invalid values
	err = requiredValidator("test.required", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required field")

	err = requiredValidator("test.required", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required field")
}

func TestDeepCopy(t *testing.T) {
	original := map[string]interface{}{
		"string": "value",
		"number": 123,
		"bool":   true,
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": 456,
		},
		"slice": []interface{}{"a", "b", "c"},
	}

	copied := deepCopy(original)

	// Verify values are equal
	assert.Equal(t, original["string"], copied["string"])
	assert.Equal(t, original["number"], copied["number"])
	assert.Equal(t, original["bool"], copied["bool"])

	// Verify nested map is copied
	originalNested := original["nested"].(map[string]interface{})
	copiedNested := copied["nested"].(map[string]interface{})
	assert.Equal(t, originalNested["key1"], copiedNested["key1"])
	assert.Equal(t, originalNested["key2"], copiedNested["key2"])

	// Verify slice is copied
	originalSlice := original["slice"].([]interface{})
	copiedSlice := copied["slice"].([]interface{})
	assert.Equal(t, originalSlice, copiedSlice)

	// Verify modifications to copy don't affect original
	copiedNested["key1"] = "modified"
	assert.NotEqual(t, originalNested["key1"], copiedNested["key1"])
}

func TestConfigurationPrecedence(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Create project config
	projectConfigPath := filepath.Join(workDir, ".switctl.yaml")
	projectConfigContent := `
server:
  http:
    port: 8000
test:
  project: "project_value"
  common: "from_project"
`
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfigContent), 0644))

	// Set environment variables (highest precedence)
	os.Setenv("SWITCTL_SERVER_HTTP_PORT", "9000")
	os.Setenv("SWITCTL_TEST_COMMON", "from_env")
	defer func() {
		os.Unsetenv("SWITCTL_SERVER_HTTP_PORT")
		os.Unsetenv("SWITCTL_TEST_COMMON")
	}()

	// Load configuration
	err := manager.Load()
	require.NoError(t, err)

	// Environment should override project
	assert.Equal(t, "9000", manager.GetString("server.http.port"))
	assert.Equal(t, "from_env", manager.GetString("test.common"))

	// Project value should be available when no environment override
	assert.Equal(t, "project_value", manager.GetString("test.project"))

	// Default should be used when neither project nor environment provide value
	assert.Equal(t, "mysql", manager.GetString("database.type"))
}

// Additional comprehensive tests

func TestHierarchicalConfigManager_ConcurrentAccess(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Test concurrent read/write operations
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("test.concurrent%d", i)
			value := fmt.Sprintf("value%d", i)
			manager.Set(key, value)
			retrieved := manager.GetString(key)
			assert.Equal(t, value, retrieved)
		}(i)
	}
	wg.Wait()
}

func TestHierarchicalConfigManager_ErrorHandling(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Test loading invalid YAML
	invalidYamlPath := filepath.Join(workDir, ".switctl.yaml")
	invalidYamlContent := `invalid:
  yaml:
    content: [
`
	require.NoError(t, os.WriteFile(invalidYamlPath, []byte(invalidYamlContent), 0644))

	err := manager.Load()
	// Should not fail completely, but log warnings
	assert.NoError(t, err)

	// Verify warning was logged
	found := false
	for _, log := range logger.Logs {
		if log.Level == "warn" && strings.Contains(log.Message, "Failed to load") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should log warning for invalid YAML")
}

func TestHierarchicalConfigManager_PermissionErrors(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Set some values to ensure there's config to save
	manager.Set("test.key", "test.value")

	// Test saving to read-only directory
	readOnlyDir := filepath.Join(workDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0755))
	require.NoError(t, os.Chmod(readOnlyDir, 0555)) // Read-only

	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

	readOnlyPath := filepath.Join(readOnlyDir, "config.yaml")
	err = manager.Save(readOnlyPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write config file")
}

func TestHierarchicalConfigManager_LargeConfiguration(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Test with large configuration
	for i := 0; i < 1000; i++ {
		manager.Set(fmt.Sprintf("large.config.item%d", i), fmt.Sprintf("value%d", i))
	}

	// Verify all values are stored correctly
	for i := 0; i < 1000; i++ {
		expected := fmt.Sprintf("value%d", i)
		actual := manager.GetString(fmt.Sprintf("large.config.item%d", i))
		assert.Equal(t, expected, actual)
	}
}

func TestHierarchicalConfigManager_SpecialCharacters(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Test keys and values with special characters
	specialCases := map[string]string{
		"unicode.emoji":   "🚀💻🔧",
		"unicode.chinese": "测试配置",
		"special.chars":   "!@#$%^&*()_+-={}[]|\\:;\"'<>?,./*",
		"multiline.text":  "line1\nline2\nline3",
		"json.embedded":   `{"key": "value", "number": 123}`,
	}

	for key, value := range specialCases {
		manager.Set(key, value)
		assert.Equal(t, value, manager.GetString(key))
	}
}

func TestHierarchicalConfigManager_Performance(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Benchmark Set operations
	start := time.Now()
	for i := 0; i < 1000; i++ {
		manager.Set(fmt.Sprintf("perf.test%d", i), fmt.Sprintf("value%d", i))
	}
	setDuration := time.Since(start)

	// Benchmark Get operations
	start = time.Now()
	for i := 0; i < 1000; i++ {
		manager.GetString(fmt.Sprintf("perf.test%d", i))
	}
	getDuration := time.Since(start)

	// Performance should be reasonable (adjust thresholds as needed)
	assert.Less(t, setDuration, 10*time.Second, "Set operations should complete in reasonable time")
	assert.Less(t, getDuration, 1*time.Second, "Get operations should complete in reasonable time")
}

func TestConfigLevel_EdgeCases(t *testing.T) {
	tests := []struct {
		level    ConfigLevel
		expected string
	}{
		{ConfigLevel(-1), "unknown"},
		{ConfigLevel(100), "unknown"},
		{ConfigLevel(255), "unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.level.String())
	}
}

func TestHierarchicalConfigManager_NilHandling(t *testing.T) {
	workDir := t.TempDir()
	manager := NewHierarchicalConfigManager(workDir, nil) // nil logger

	// Should not panic with nil logger
	err := manager.Load()
	assert.NoError(t, err)

	manager.Set("test.key", "value")
	assert.Equal(t, "value", manager.GetString("test.key"))
}

func TestHierarchicalConfigManager_NestedConfiguration(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Create deeply nested configuration
	projectConfigPath := filepath.Join(workDir, ".switctl.yaml")
	projectConfigContent := `
app:
  server:
    http:
      handlers:
        timeout:
          read: 30s
          write: 30s
        middleware:
          auth:
            jwt:
              secret: "secret"
              expiration: 24h
            oauth:
              providers:
                google:
                  client_id: "client123"
                  client_secret: "secret456"
                github:
                  client_id: "gh_client"
                  client_secret: "gh_secret"
`
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfigContent), 0644))

	err := manager.Load()
	require.NoError(t, err)

	// Test deep nested access
	assert.Equal(t, "30s", manager.GetString("app.server.http.handlers.timeout.read"))
	assert.Equal(t, "secret", manager.GetString("app.server.http.handlers.middleware.auth.jwt.secret"))
	assert.Equal(t, "client123", manager.GetString("app.server.http.handlers.middleware.auth.oauth.providers.google.client_id"))
}

func TestHierarchicalConfigManager_ValidationComplexity(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	// Add multiple complex validators
	manager.AddValidator("email", func(key string, value interface{}) error {
		if email, ok := value.(string); ok {
			if !strings.Contains(email, "@") {
				return fmt.Errorf("invalid email format for %s", key)
			}
		}
		return nil
	})

	manager.AddValidator("range", func(key string, value interface{}) error {
		if num, ok := value.(int); ok {
			if num < 1 || num > 100 {
				return fmt.Errorf("value %d for %s must be between 1 and 100", num, key)
			}
		}
		return nil
	})

	err := manager.Load()
	require.NoError(t, err)

	// Test successful validation
	manager.Set("user.email", "test@example.com")
	manager.Set("config.range", 50)
	err = manager.Validate()
	assert.NoError(t, err)

	// Test validation failures
	manager.Set("user.email", "invalid-email")
	err = manager.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email format")

	manager.Set("user.email", "test@example.com") // Fix email
	manager.Set("config.range", 150)
	err = manager.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be between 1 and 100")
}

func TestGetGlobalConfigDir(t *testing.T) {
	// This is a unit test for the helper function
	// Note: We can't easily mock os.Getuid() in Go, so we test the actual behavior

	// Test with current user (likely non-root in test environment)
	dir := getGlobalConfigDir()
	assert.NotEmpty(t, dir)
}

func TestGetUserConfigDir(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHOME := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		os.Setenv("HOME", originalHOME)
	}()

	// Test with XDG_CONFIG_HOME
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	os.Unsetenv("HOME")
	dir := getUserConfigDir()
	assert.Equal(t, "/custom/config", dir)

	// Test with HOME fallback
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", "/home/user")
	dir = getUserConfigDir()
	assert.Equal(t, "/home/user", dir)

	// Test with neither set
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("HOME")
	dir = getUserConfigDir()
	assert.Equal(t, "", dir)
}

func TestDeepCopyEdgeCases(t *testing.T) {
	// Test with nil values
	result := deepCopyValue(nil)
	assert.Nil(t, result)

	// Test with various types
	tests := []struct {
		name  string
		value interface{}
	}{
		{"string", "test"},
		{"int", 123},
		{"float", 3.14},
		{"bool", true},
		{"empty_map", map[string]interface{}{}},
		{"empty_slice", []interface{}{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := deepCopyValue(test.value)
			assert.Equal(t, test.value, result)
		})
	}
}

func TestHierarchicalConfigManager_ChangeHandlerErrors(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Add a change handler that returns an error
	manager.AddChangeHandler(func(level ConfigLevel, key string, oldValue, newValue interface{}) error {
		return fmt.Errorf("change handler error")
	})

	// Set a value - should still succeed but log error
	manager.Set("test.key", "value")

	// Verify error was logged
	found := false
	for _, log := range logger.Logs {
		if log.Level == "error" && strings.Contains(log.Message, "Configuration change handler error") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should log error from change handler")

	// Value should still be set
	assert.Equal(t, "value", manager.GetString("test.key"))
}

func TestHierarchicalConfigManager_SetLevelChangeHandler(t *testing.T) {
	workDir := t.TempDir()
	logger := &MockLogger{}
	manager := NewHierarchicalConfigManager(workDir, logger)

	err := manager.Load()
	require.NoError(t, err)

	// Add a change handler that returns an error for SetLevel
	manager.AddChangeHandler(func(level ConfigLevel, key string, oldValue, newValue interface{}) error {
		return fmt.Errorf("setlevel change handler error")
	})

	// SetLevel should return the error
	err = manager.SetLevel(UserLevel, "test.key", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration change handler error")
}
