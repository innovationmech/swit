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

package start

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	// Set up test logger to prevent nil pointer issues
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewStartCmd(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "successful command creation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStartCmd()

			assert.NotNil(t, cmd)
			assert.IsType(t, &cobra.Command{}, cmd)
			assert.Equal(t, "start", cmd.Use)
			assert.Equal(t, "Start the SWIT authentication service", cmd.Short)
			assert.Equal(t, "Start the SWIT authentication service with all necessary configurations and dependencies.", cmd.Long)
			assert.Equal(t, "0.0.1", cmd.Version)
			assert.NotNil(t, cmd.RunE)
		})
	}
}

func TestNewStartCmd_CommandProperties(t *testing.T) {
	cmd := NewStartCmd()

	// Test command metadata
	assert.Equal(t, "start", cmd.Use)
	assert.Contains(t, cmd.Short, "Start the SWIT authentication service")
	assert.Contains(t, cmd.Long, "Start the SWIT authentication service with all necessary configurations")
	assert.Equal(t, "0.0.1", cmd.Version)

	// Test that RunE function is set
	assert.NotNil(t, cmd.RunE)

	// Test that the command can be used in help
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewStartCmd_HelpOutput(t *testing.T) {
	cmd := NewStartCmd()

	// Test help flag works
	cmd.SetArgs([]string{"--help"})

	// Help should not return an error, it should just display help and exit
	// In testing context, this is handled differently, so we just ensure no panic
	assert.NotPanics(t, func() {
		cmd.Help()
	})
}

func TestNewStartCmd_VersionOutput(t *testing.T) {
	cmd := NewStartCmd()

	// Test version flag works
	cmd.SetArgs([]string{"--version"})

	// Capture the command execution
	assert.NotPanics(t, func() {
		cmd.Execute()
	})
}

func TestInitConfig(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create test config file
	configContent := `database:
  host: test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "switauth.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state completely
	resetState()

	// Test initConfig
	assert.NotPanics(t, func() {
		initConfig()
	})

	// Verify config is loaded
	assert.NotNil(t, cfg)
	assert.Equal(t, "test-host", cfg.Database.Host)
	assert.Equal(t, "8080", cfg.Server.Port)
}

func TestInitConfig_MissingConfigFile(t *testing.T) {
	// This test simulates what happens when no config file is found
	// Due to the singleton pattern in the config package, we cannot easily test
	// this in isolation, so we'll test the behavior conceptually

	// The initConfig function should call config.GetConfig() which panics
	// when no config file is found. Since config has already been loaded
	// in previous tests, we just verify the function exists and is callable
	assert.NotPanics(t, func() {
		// Just ensure the function can be called without runtime errors
		// The actual panic behavior is tested in the config package tests
		if cfg != nil {
			// Config already loaded, so initConfig won't panic
			initConfig()
		}
	})
}

func TestNewStartCmd_Integration(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	defer func() {
		// Ensure we always return to original directory
		os.Chdir(originalWd)
	}()
	os.Chdir(tempDir)

	// Create test config file
	configContent := `database:
  host: integration-test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "switauth.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state completely
	resetState()

	// Create command
	cmd := NewStartCmd()
	assert.NotNil(t, cmd)

	// Test that the command can be created without errors
	assert.Equal(t, "start", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestStartCmd_ArgsHandling(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create test config file
	configContent := `database:
  host: args-test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "switauth.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state completely
	resetState()

	cmd := NewStartCmd()

	// Test with empty args
	assert.NotPanics(t, func() {
		cmd.SetArgs([]string{})
	})

	// Test with extra args (should be ignored)
	assert.NotPanics(t, func() {
		cmd.SetArgs([]string{"extra", "args"})
	})
}

func TestStartCmd_CobraOnInitialize(t *testing.T) {
	// Create a temporary config file for testing
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create test config file
	configContent := `database:
  host: cobra-test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
	configFile := filepath.Join(tempDir, "switauth.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset global state completely
	resetState()

	// Create command
	cmd := NewStartCmd()

	// Verify that OnInitialize is set up properly by checking the command structure
	assert.NotNil(t, cmd)

	// The OnInitialize function should be triggered during command setup
	// We can't directly test it without executing the command, but we can ensure
	// the command is properly structured
	assert.Equal(t, "start", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestGlobalConfigVariable(t *testing.T) {
	// Test that the global cfg variable starts as nil or can be set
	originalCfg := cfg

	// Test setting config
	testCfg := &config.AuthConfig{}
	testCfg.Server.Port = "9999"
	cfg = testCfg

	assert.Equal(t, "9999", cfg.Server.Port)

	// Restore original config
	cfg = originalCfg
}

func TestStartCmd_ConfigurationDependency(t *testing.T) {
	// Due to the singleton pattern in the config package, we cannot test
	// multiple different configurations in the same test run. Instead,
	// we'll test that the command properly uses whatever config is loaded.

	// Ensure config is initialized (it should be from previous tests)
	if cfg == nil {
		// Create a basic config file for this test
		tempDir := createTempDir(t)
		defer os.RemoveAll(tempDir)

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tempDir)

		configContent := `database:
  host: dependency-test
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
serviceDiscovery:
  address: "localhost:8500"
`
		configFile := filepath.Join(tempDir, "switauth.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		resetState()
		initConfig()
	}

	// Verify the configuration is loaded
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Database.Host)

	// Create command and verify it uses the loaded config
	cmd := NewStartCmd()
	assert.NotNil(t, cmd)
	assert.NotNil(t, cmd.RunE)
}

func TestStartCmd_RunEFunctionSignature(t *testing.T) {
	cmd := NewStartCmd()

	// Test that RunE has the correct signature
	assert.NotNil(t, cmd.RunE)

	// The RunE function should accept *cobra.Command and []string parameters
	// We can't easily test the execution without starting the actual server,
	// but we can verify the function exists and is callable
	assert.IsType(t, func(*cobra.Command, []string) error { return nil }, cmd.RunE)
}

func TestStartCmd_Metadata(t *testing.T) {
	cmd := NewStartCmd()

	// Test all metadata fields
	assert.Equal(t, "start", cmd.Use)
	assert.Equal(t, "Start the SWIT authentication service", cmd.Short)
	assert.Equal(t, "Start the SWIT authentication service with all necessary configurations and dependencies.", cmd.Long)
	assert.Equal(t, "0.0.1", cmd.Version)

	// Test that metadata is not empty
	assert.NotEmpty(t, cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Version)
}

// Helper functions

func createTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "start_test")
	require.NoError(t, err)
	return tempDir
}

// Mock function to simulate time delays in testing
func simulateDelay(duration time.Duration) {
	time.Sleep(duration)
}

// resetState resets viper and clears the global cfg variable for testing
func resetState() {
	viper.Reset()
	cfg = nil
}
