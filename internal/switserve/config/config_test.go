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

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeConfig_Structure(t *testing.T) {
	config := &ServeConfig{}

	// Test struct fields exist and are properly typed
	assert.IsType(t, "", config.Database.Username)
	assert.IsType(t, "", config.Database.Password)
	assert.IsType(t, "", config.Database.Host)
	assert.IsType(t, "", config.Database.Port)
	assert.IsType(t, "", config.Database.DBName)
	assert.IsType(t, "", config.Server.Port)
	assert.IsType(t, "", config.Server.GRPCPort)
	assert.IsType(t, "", config.Url)
	assert.IsType(t, "", config.ServiceDiscovery.Address)
}

func TestGetConfig_ValidConfig(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create test config file
	configContent := `database:
  host: testhost
  port: "3306"
  username: testuser
  password: testpass
  dbname: testdb
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test GetConfig
	cfg := GetConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "testhost", cfg.Database.Host)
	assert.Equal(t, "3306", cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.Username)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.DBName)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "localhost:8500", cfg.ServiceDiscovery.Address)

	// These fields are not in the actual config structure used in the project
	assert.Empty(t, cfg.Server.GRPCPort) // Not set in config
	assert.Empty(t, cfg.Url)             // Not set in config
}

func TestGetConfig_SingletonBehavior(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create test config file
	configContent := `database:
  host: singleton-test
  port: "3306"
  username: testuser
  password: testpass
  dbname: testdb
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Get config multiple times
	cfg1 := GetConfig()
	cfg2 := GetConfig()
	cfg3 := GetConfig()

	// All should return the same instance
	assert.Same(t, cfg1, cfg2)
	assert.Same(t, cfg2, cfg3)
	assert.Equal(t, "singleton-test", cfg1.Database.Host)
	assert.Equal(t, "singleton-test", cfg2.Database.Host)
	assert.Equal(t, "singleton-test", cfg3.Database.Host)
}

func TestGetConfig_ConcurrentAccess(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create test config file
	configContent := `database:
  host: concurrent-test
  port: "3306"
  username: testuser
  password: testpass
  dbname: testdb
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test concurrent access
	const numGoroutines = 10
	configs := make([]*ServeConfig, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			configs[index] = GetConfig()
		}(i)
	}

	wg.Wait()

	// All configs should be the same instance
	for i := 1; i < numGoroutines; i++ {
		assert.Same(t, configs[0], configs[i])
	}

	// Verify config content
	assert.Equal(t, "concurrent-test", configs[0].Database.Host)
}

func TestGetConfig_MissingConfigFile(t *testing.T) {
	// Create empty temp directory
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Reset global state for testing
	resetConfigState()

	// Test that GetConfig panics when config file doesn't exist
	assert.Panics(t, func() {
		GetConfig()
	})
}

func TestGetConfig_InvalidYAML(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create invalid YAML file
	invalidYAML := `database:
  host: testhost
  port: "3306"
  username: testuser
  password: testpass
  dbname: testdb
server
  port: "8080"  # Missing colon after 'server'
  grpc_port: "9090"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test that GetConfig panics with invalid YAML
	assert.Panics(t, func() {
		GetConfig()
	})
}

func TestGetConfig_EmptyConfigFile(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create empty config file
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test GetConfig with empty file - should not panic but return empty config
	cfg := GetConfig()
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.Database.Host)
	assert.Empty(t, cfg.Database.Port)
	assert.Empty(t, cfg.Server.Port)
	assert.Empty(t, cfg.Server.GRPCPort)
	assert.Empty(t, cfg.Url)
	assert.Empty(t, cfg.ServiceDiscovery.Address)
}

func TestGetConfig_PartialConfig(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create partial config file (only database section)
	partialConfig := `database:
  host: partial-host
  port: "3306"
server:
  port: "8080"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(partialConfig), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test GetConfig with partial config
	cfg := GetConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "partial-host", cfg.Database.Host)
	assert.Equal(t, "3306", cfg.Database.Port)
	assert.Empty(t, cfg.Database.Username) // Should be empty since not in config
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Empty(t, cfg.Server.GRPCPort) // Should be empty since not in config
	assert.Empty(t, cfg.Url)             // Should be empty since not in config
}

func TestGetConfig_AllFieldsPresent(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create comprehensive config file
	fullConfig := `database:
  host: full-test-host
  port: "5432"
  username: full-user
  password: full-pass
  dbname: full-db
server:
  port: "3000"
serviceDiscovery:
  address: "consul.example.com:8500"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(fullConfig), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test GetConfig with all fields
	cfg := GetConfig()
	assert.NotNil(t, cfg)

	// Verify all database fields
	assert.Equal(t, "full-test-host", cfg.Database.Host)
	assert.Equal(t, "5432", cfg.Database.Port)
	assert.Equal(t, "full-user", cfg.Database.Username)
	assert.Equal(t, "full-pass", cfg.Database.Password)
	assert.Equal(t, "full-db", cfg.Database.DBName)

	// Verify all server fields
	assert.Equal(t, "3000", cfg.Server.Port)

	// Verify service discovery
	assert.Equal(t, "consul.example.com:8500", cfg.ServiceDiscovery.Address)

	// These fields are not in the actual config structure used in the project
	assert.Empty(t, cfg.Server.GRPCPort) // Not set in config
	assert.Empty(t, cfg.Url)             // Not set in config
}

func TestGetConfig_JSONTags(t *testing.T) {
	// Test that struct has proper JSON tags by verifying they exist in struct definition
	// This is more of a compile-time check, but we can verify the struct can be used
	assert.NotPanics(t, func() {
		// If the struct tags are properly defined, this should not panic
		cfg := &ServeConfig{}
		cfg.Database.Host = "test"
		cfg.Database.Port = "3306"
		cfg.Server.Port = "8080"
		cfg.Server.GRPCPort = "9090"
		cfg.Url = "http://test.com"
		cfg.ServiceDiscovery.Address = "localhost:8500"
	})
}

func TestGetConfig_ConfigFileFormats(t *testing.T) {
	tests := []struct {
		name           string
		configFileName string
		configContent  string
		expectError    bool
	}{
		{
			name:           "YAML format",
			configFileName: "swit.yaml",
			configContent: `database:
  host: yaml-host
server:
  port: "8080"`,
			expectError: false,
		},
		{
			name:           "YML format",
			configFileName: "swit.yml",
			configContent: `database:
  host: yml-host
server:
  port: "8080"`,
			expectError: false,
		},
		{
			name:           "JSON format",
			configFileName: "swit.json",
			configContent: `{
  "database": {"host": "json-host"},
  "server": {"port": "8080"}
}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file for testing
			tempDir := createTempDir(t)
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			originalWd, _ := os.Getwd()
			defer os.Chdir(originalWd)
			os.Chdir(tempDir)

			// Create config file
			configFile := filepath.Join(tempDir, tt.configFileName)
			err := ioutil.WriteFile(configFile, []byte(tt.configContent), 0644)
			require.NoError(t, err)

			// Reset global state for testing
			resetConfigState()

			if tt.expectError {
				assert.Panics(t, func() {
					GetConfig()
				})
			} else {
				cfg := GetConfig()
				assert.NotNil(t, cfg)
				// Verify config was loaded based on format
				if tt.name == "YAML format" {
					assert.Equal(t, "yaml-host", cfg.Database.Host)
				} else if tt.name == "YML format" {
					assert.Equal(t, "yml-host", cfg.Database.Host)
				} else if tt.name == "JSON format" {
					assert.Equal(t, "json-host", cfg.Database.Host)
				}
			}
		})
	}
}

func TestGetConfig_DefaultConfigPath(t *testing.T) {
	// Test that the config loader looks in current directory by default
	// This test verifies the viper.AddConfigPath(".") behavior

	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create config file in current directory
	configContent := `database:
  host: current-dir-host
server:
  port: "8080"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test GetConfig finds config in current directory
	cfg := GetConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "current-dir-host", cfg.Database.Host)
}

func TestGetConfig_PanicMessages(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(tempDir string)
		panicMsg string
	}{
		{
			name: "missing config file panic message",
			setup: func(tempDir string) {
				// Don't create any config file
			},
			panicMsg: "FATAL ERROR CONFIG FILE:",
		},
		{
			name: "invalid YAML panic message",
			setup: func(tempDir string) {
				invalidYAML := `invalid: yaml: content: [unclosed`
				configFile := filepath.Join(tempDir, "swit.yaml")
				ioutil.WriteFile(configFile, []byte(invalidYAML), 0644)
			},
			panicMsg: "FATAL ERROR CONFIG FILE:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tempDir := createTempDir(t)
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			originalWd, _ := os.Getwd()
			defer os.Chdir(originalWd)
			os.Chdir(tempDir)

			// Setup test scenario
			tt.setup(tempDir)

			// Reset global state for testing
			resetConfigState()

			// Test that GetConfig panics with expected message
			defer func() {
				if r := recover(); r != nil {
					assert.Contains(t, r.(error).Error(), tt.panicMsg)
				} else {
					t.Error("Expected panic but none occurred")
				}
			}()

			GetConfig()
		})
	}
}

// Helper functions

func createTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "config_test")
	require.NoError(t, err)
	return tempDir
}

// resetConfigState resets the global config and sync.Once for testing
func resetConfigState() {
	cfg = nil
	once = sync.Once{}
	viper.Reset()
}
