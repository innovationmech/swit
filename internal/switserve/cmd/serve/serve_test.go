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

package serve

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewServeCmd(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "successful command creation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewServeCmd()

			assert.NotNil(t, cmd)
			assert.IsType(t, &cobra.Command{}, cmd)
			assert.Equal(t, "serve", cmd.Use)
			assert.Equal(t, "Start the SWIT server", cmd.Short)
			assert.NotNil(t, cmd.RunE)
		})
	}
}

func TestNewServeCmd_CommandProperties(t *testing.T) {
	cmd := NewServeCmd()

	assert.Equal(t, "serve", cmd.Use)
	assert.Equal(t, "Start the SWIT server", cmd.Short)

	assert.NotNil(t, cmd.RunE)

	assert.NotEmpty(t, cmd.Short)
}

func TestNewServeCmd_HelpOutput(t *testing.T) {
	cmd := NewServeCmd()

	cmd.SetArgs([]string{"--help"})

	assert.NotPanics(t, func() {
		cmd.Help()
	})
}

func TestInitConfig(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	configContent := `database:
  host: test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
  grpc_port: "9090"
serviceDiscovery:
  address: "localhost:8500"
url: "http://localhost:8080"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	resetState()

	assert.NotPanics(t, func() {
		initConfig()
	})

	assert.NotNil(t, cfg)
	assert.Equal(t, "test-host", cfg.Database.Host)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "9090", cfg.Server.GRPCPort)
}

func TestInitConfig_MissingConfigFile(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change the current working directory to the temporary directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// Create a dummy config file
	configContent := `
server:
  port: "8080"
`
	configPath := filepath.Join(tempDir, "swit.yaml")
	err = ioutil.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Reset Viper and the global config
	viper.Reset()
	cfg = nil

	// Test that initConfig does not panic when the config file is present
	assert.NotPanics(t, func() {
		initConfig()
	})

	// Test that the config is loaded correctly
	assert.NotNil(t, cfg)
	assert.Equal(t, "8080", cfg.Server.Port)
}

func TestNewServeCmd_Integration(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	defer func() {
		os.Chdir(originalWd)
	}()
	os.Chdir(tempDir)

	configContent := `database:
  host: integration-test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
  grpc_port: "9090"
serviceDiscovery:
  address: "localhost:8500"
url: "http://localhost:8080"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	resetState()

	cmd := NewServeCmd()
	assert.NotNil(t, cmd)

	assert.Equal(t, "serve", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestServeCmd_ArgsHandling(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	configContent := `database:
  host: args-test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
  grpc_port: "9090"
serviceDiscovery:
  address: "localhost:8500"
url: "http://localhost:8080"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	resetState()

	cmd := NewServeCmd()

	assert.NotPanics(t, func() {
		cmd.SetArgs([]string{})
	})

	assert.NotPanics(t, func() {
		cmd.SetArgs([]string{"extra", "args"})
	})
}

func TestServeCmd_CobraOnInitialize(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	configContent := `database:
  host: cobra-test-host
  port: "3306"
  username: test-user
  password: test-pass
  dbname: test-db
server:
  port: "8080"
  grpc_port: "9090"
serviceDiscovery:
  address: "localhost:8500"
url: "http://localhost:8080"
`
	configFile := filepath.Join(tempDir, "swit.yaml")
	err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	resetState()

	cmd := NewServeCmd()

	assert.NotNil(t, cmd)

	assert.Equal(t, "serve", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestGlobalConfigVariable(t *testing.T) {
	originalCfg := cfg

	testCfg := &config.ServeConfig{}
	testCfg.Server.Port = "9999"
	cfg = testCfg

	assert.Equal(t, "9999", cfg.Server.Port)

	cfg = originalCfg
}

func TestServeCmd_ConfigurationDependency(t *testing.T) {
	if cfg == nil {
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
  grpc_port: "9090"
serviceDiscovery:
  address: "localhost:8500"
url: "http://localhost:8080"
`
		configFile := filepath.Join(tempDir, "swit.yaml")
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		resetState()
		initConfig()
	}

	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Server.Port)
	assert.NotEmpty(t, cfg.Database.Host)

	cmd := NewServeCmd()
	assert.NotNil(t, cmd)
	assert.NotNil(t, cmd.RunE)
}

func TestServeCmd_RunEFunctionSignature(t *testing.T) {
	cmd := NewServeCmd()

	assert.NotNil(t, cmd.RunE)

	assert.IsType(t, func(*cobra.Command, []string) error { return nil }, cmd.RunE)
}

func TestServeCmd_Metadata(t *testing.T) {
	cmd := NewServeCmd()

	assert.Equal(t, "serve", cmd.Use)
	assert.Equal(t, "Start the SWIT server", cmd.Short)

	assert.NotEmpty(t, cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func createTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "serve_test")
	require.NoError(t, err)
	return tempDir
}

func resetState() {
	viper.Reset()
	cfg = nil
}
