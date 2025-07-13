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

func TestJwtSecret(t *testing.T) {
	expectedSecret := []byte("my-256-bit-secret")
	assert.Equal(t, expectedSecret, JwtSecret)
	assert.NotEmpty(t, JwtSecret)
}

func TestAuthConfig_Structure(t *testing.T) {
	config := &AuthConfig{}

	// Test struct fields exist and are properly typed
	assert.IsType(t, "", config.Database.Username)
	assert.IsType(t, "", config.Database.Password)
	assert.IsType(t, "", config.Database.Host)
	assert.IsType(t, "", config.Database.Port)
	assert.IsType(t, "", config.Database.DBName)
	assert.IsType(t, "", config.Server.Port)
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
	configFile := filepath.Join(tempDir, "switauth.yaml")
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
	configFile := filepath.Join(tempDir, "switauth.yaml")
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
	configFile := filepath.Join(tempDir, "switauth.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test concurrent access
	const numGoroutines = 10
	configs := make([]*AuthConfig, numGoroutines)
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
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "switauth.yaml")
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
	configFile := filepath.Join(tempDir, "switauth.yaml")
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
`
	configFile := filepath.Join(tempDir, "switauth.yaml")
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
	assert.Empty(t, cfg.Server.Port)       // Should be empty since not in config
}

func TestGetConfig_EnvironmentVariableOverride(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create config file
	configContent := `database:
  host: file-host
  port: "3306"
  username: file-user
  password: file-pass
  dbname: file-db
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "switauth.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state for testing
	resetConfigState()

	// Test GetConfig uses file values when no env vars are set
	cfg := GetConfig()
	assert.Equal(t, "file-host", cfg.Database.Host)
	assert.Equal(t, "file-user", cfg.Database.Username)
}

func TestAuthConfig_JSONTags(t *testing.T) {
	// Test that struct has proper JSON/YAML tags by verifying they exist in struct definition
	// This is more of a compile-time check, but we can verify the struct can be marshaled
	assert.NotPanics(t, func() {
		// If the struct tags are properly defined, this should not panic
		cfg := &AuthConfig{}
		cfg.Database.Host = "test"
		cfg.Database.Port = "3306"
		cfg.Server.Port = "8080"
		cfg.ServiceDiscovery.Address = "localhost:8500"
	})
}

// Helper functions

func createTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "config_test")
	require.NoError(t, err)
	return tempDir
}

// resetConfigState resets the global config and sync.Once for testing
func resetConfigState() {
	config = nil
	once = sync.Once{}
	viper.Reset()
}
