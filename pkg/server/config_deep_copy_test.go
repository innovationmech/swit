// Copyright 2024 Swit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfig_DeepCopy(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() *ServerConfig
		verify func(t *testing.T, original, copy *ServerConfig)
	}{
		{
			name: "nil config",
			setup: func() *ServerConfig {
				return nil
			},
			verify: func(t *testing.T, original, copy *ServerConfig) {
				assert.Nil(t, copy)
			},
		},
		{
			name: "basic config with defaults",
			setup: func() *ServerConfig {
				return NewServerConfig()
			},
			verify: func(t *testing.T, original, copy *ServerConfig) {
				require.NotNil(t, copy)
				assert.NotSame(t, original, copy)
				assert.Equal(t, original.ServiceName, copy.ServiceName)
				assert.Equal(t, original.ShutdownTimeout, copy.ShutdownTimeout)
			},
		},
		{
			name: "config with custom HTTP headers",
			setup: func() *ServerConfig {
				config := NewServerConfig()
				config.HTTP.Headers = map[string]string{
					"X-Custom-Header":  "value1",
					"X-Another-Header": "value2",
				}
				return config
			},
			verify: func(t *testing.T, original, copy *ServerConfig) {
				// Verify headers are copied
				assert.Equal(t, original.HTTP.Headers, copy.HTTP.Headers)
				// Verify they are different map instances
				assert.NotSame(t, original.HTTP.Headers, copy.HTTP.Headers)

				// Modify original and verify copy is not affected
				original.HTTP.Headers["X-Modified"] = "modified"
				assert.NotContains(t, copy.HTTP.Headers, "X-Modified")
			},
		},
		{
			name: "config with CORS configuration",
			setup: func() *ServerConfig {
				config := NewServerConfig()
				config.HTTP.Middleware.CORSConfig.AllowOrigins = []string{"http://localhost:3000", "https://example.com"}
				config.HTTP.Middleware.CORSConfig.AllowMethods = []string{"GET", "POST", "PUT"}
				config.HTTP.Middleware.CORSConfig.AllowHeaders = []string{"Content-Type", "Authorization"}
				config.HTTP.Middleware.CORSConfig.ExposeHeaders = []string{"X-Total-Count"}
				return config
			},
			verify: func(t *testing.T, original, copy *ServerConfig) {
				// Verify CORS config is copied
				assert.Equal(t, original.HTTP.Middleware.CORSConfig.AllowOrigins, copy.HTTP.Middleware.CORSConfig.AllowOrigins)
				assert.Equal(t, original.HTTP.Middleware.CORSConfig.AllowMethods, copy.HTTP.Middleware.CORSConfig.AllowMethods)
				assert.Equal(t, original.HTTP.Middleware.CORSConfig.AllowHeaders, copy.HTTP.Middleware.CORSConfig.AllowHeaders)
				assert.Equal(t, original.HTTP.Middleware.CORSConfig.ExposeHeaders, copy.HTTP.Middleware.CORSConfig.ExposeHeaders)

				// Verify they are different slice instances
				assert.NotSame(t, original.HTTP.Middleware.CORSConfig.AllowOrigins, copy.HTTP.Middleware.CORSConfig.AllowOrigins)
				assert.NotSame(t, original.HTTP.Middleware.CORSConfig.AllowMethods, copy.HTTP.Middleware.CORSConfig.AllowMethods)
				assert.NotSame(t, original.HTTP.Middleware.CORSConfig.AllowHeaders, copy.HTTP.Middleware.CORSConfig.AllowHeaders)
				assert.NotSame(t, original.HTTP.Middleware.CORSConfig.ExposeHeaders, copy.HTTP.Middleware.CORSConfig.ExposeHeaders)

				// Modify original and verify copy is not affected
				original.HTTP.Middleware.CORSConfig.AllowOrigins = append(original.HTTP.Middleware.CORSConfig.AllowOrigins, "https://modified.com")
				assert.NotContains(t, copy.HTTP.Middleware.CORSConfig.AllowOrigins, "https://modified.com")
			},
		},
		{
			name: "config with custom middleware headers",
			setup: func() *ServerConfig {
				config := NewServerConfig()
				config.HTTP.Middleware.CustomHeaders = map[string]string{
					"X-API-Version":  "v1",
					"X-Service-Name": "test-service",
				}
				return config
			},
			verify: func(t *testing.T, original, copy *ServerConfig) {
				// Verify custom headers are copied
				assert.Equal(t, original.HTTP.Middleware.CustomHeaders, copy.HTTP.Middleware.CustomHeaders)
				// Verify they are different map instances
				assert.NotSame(t, original.HTTP.Middleware.CustomHeaders, copy.HTTP.Middleware.CustomHeaders)

				// Modify original and verify copy is not affected
				original.HTTP.Middleware.CustomHeaders["X-Modified"] = "modified"
				assert.NotContains(t, copy.HTTP.Middleware.CustomHeaders, "X-Modified")
			},
		},
		{
			name: "config with discovery tags",
			setup: func() *ServerConfig {
				config := NewServerConfig()
				config.Discovery.Tags = []string{"api", "v1", "production"}
				return config
			},
			verify: func(t *testing.T, original, copy *ServerConfig) {
				// Verify tags are copied
				assert.Equal(t, original.Discovery.Tags, copy.Discovery.Tags)
				// Verify they are different slice instances
				assert.NotSame(t, original.Discovery.Tags, copy.Discovery.Tags)

				// Modify original and verify copy is not affected
				original.Discovery.Tags = append(original.Discovery.Tags, "modified")
				assert.NotContains(t, copy.Discovery.Tags, "modified")
			},
		},
		{
			name: "config with all nested structures",
			setup: func() *ServerConfig {
				config := NewServerConfig()
				config.ServiceName = "test-service"
				config.ShutdownTimeout = 30 * time.Second

				// HTTP config
				config.HTTP.Port = "8080"
				config.HTTP.Headers = map[string]string{"X-Test": "value"}
				config.HTTP.Middleware.CustomHeaders = map[string]string{"X-Custom": "custom"}
				config.HTTP.Middleware.CORSConfig.AllowOrigins = []string{"http://localhost:3000"}

				// GRPC config
				config.GRPC.Port = "9090"
				config.GRPC.MaxRecvMsgSize = 1024

				// Discovery config
				config.Discovery.ServiceName = "test-discovery"
				config.Discovery.Tags = []string{"test", "api"}

				return config
			},
			verify: func(t *testing.T, original, copy *ServerConfig) {
				// Verify all fields are copied correctly
				assert.Equal(t, original.ServiceName, copy.ServiceName)
				assert.Equal(t, original.ShutdownTimeout, copy.ShutdownTimeout)
				assert.Equal(t, original.HTTP.Port, copy.HTTP.Port)
				assert.Equal(t, original.GRPC.Port, copy.GRPC.Port)
				assert.Equal(t, original.Discovery.ServiceName, copy.Discovery.ServiceName)

				// Verify nested structures are independent
				assert.NotSame(t, original.HTTP.Headers, copy.HTTP.Headers)
				assert.NotSame(t, original.HTTP.Middleware.CustomHeaders, copy.HTTP.Middleware.CustomHeaders)
				assert.NotSame(t, original.HTTP.Middleware.CORSConfig.AllowOrigins, copy.HTTP.Middleware.CORSConfig.AllowOrigins)
				assert.NotSame(t, original.Discovery.Tags, copy.Discovery.Tags)

				// Test independence by modifying original
				original.HTTP.Headers["X-Modified"] = "modified"
				original.HTTP.Middleware.CustomHeaders["X-Modified"] = "modified"
				original.HTTP.Middleware.CORSConfig.AllowOrigins = append(original.HTTP.Middleware.CORSConfig.AllowOrigins, "modified")
				original.Discovery.Tags = append(original.Discovery.Tags, "modified")

				// Verify copy is not affected
				assert.NotContains(t, copy.HTTP.Headers, "X-Modified")
				assert.NotContains(t, copy.HTTP.Middleware.CustomHeaders, "X-Modified")
				assert.NotContains(t, copy.HTTP.Middleware.CORSConfig.AllowOrigins, "modified")
				assert.NotContains(t, copy.Discovery.Tags, "modified")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.setup()
			copy := original.DeepCopy()
			tt.verify(t, original, copy)
		})
	}
}

func TestServerConfig_DeepCopy_NilSlicesAndMaps(t *testing.T) {
	// Test that nil slices and maps are handled correctly
	config := &ServerConfig{
		ServiceName: "test",
		HTTP: HTTPConfig{
			Headers: nil, // nil map
			Middleware: HTTPMiddleware{
				CustomHeaders: nil, // nil map
				CORSConfig: CORSConfig{
					AllowOrigins:  nil, // nil slice
					AllowMethods:  nil, // nil slice
					AllowHeaders:  nil, // nil slice
					ExposeHeaders: nil, // nil slice
				},
			},
		},
		Discovery: DiscoveryConfig{
			Tags: nil, // nil slice
		},
	}

	copy := config.DeepCopy()

	require.NotNil(t, copy)
	assert.Equal(t, config.ServiceName, copy.ServiceName)
	assert.Nil(t, copy.HTTP.Headers)
	assert.Nil(t, copy.HTTP.Middleware.CustomHeaders)
	assert.Nil(t, copy.HTTP.Middleware.CORSConfig.AllowOrigins)
	assert.Nil(t, copy.HTTP.Middleware.CORSConfig.AllowMethods)
	assert.Nil(t, copy.HTTP.Middleware.CORSConfig.AllowHeaders)
	assert.Nil(t, copy.HTTP.Middleware.CORSConfig.ExposeHeaders)
	assert.Nil(t, copy.Discovery.Tags)
}

func TestServerConfig_DeepCopy_EmptySlicesAndMaps(t *testing.T) {
	// Test that empty slices and maps are handled correctly
	config := &ServerConfig{
		ServiceName: "test",
		HTTP: HTTPConfig{
			Headers: make(map[string]string), // empty map
			Middleware: HTTPMiddleware{
				CustomHeaders: make(map[string]string), // empty map
				CORSConfig: CORSConfig{
					AllowOrigins:  make([]string, 0), // empty slice
					AllowMethods:  make([]string, 0), // empty slice
					AllowHeaders:  make([]string, 0), // empty slice
					ExposeHeaders: make([]string, 0), // empty slice
				},
			},
		},
		Discovery: DiscoveryConfig{
			Tags: make([]string, 0), // empty slice
		},
	}

	copy := config.DeepCopy()

	require.NotNil(t, copy)
	assert.Equal(t, config.ServiceName, copy.ServiceName)
	assert.NotNil(t, copy.HTTP.Headers)
	assert.Empty(t, copy.HTTP.Headers)
	assert.NotSame(t, config.HTTP.Headers, copy.HTTP.Headers)

	assert.NotNil(t, copy.HTTP.Middleware.CustomHeaders)
	assert.Empty(t, copy.HTTP.Middleware.CustomHeaders)
	assert.NotSame(t, config.HTTP.Middleware.CustomHeaders, copy.HTTP.Middleware.CustomHeaders)

	assert.NotNil(t, copy.HTTP.Middleware.CORSConfig.AllowOrigins)
	assert.Empty(t, copy.HTTP.Middleware.CORSConfig.AllowOrigins)
	assert.NotSame(t, config.HTTP.Middleware.CORSConfig.AllowOrigins, copy.HTTP.Middleware.CORSConfig.AllowOrigins)

	assert.NotNil(t, copy.Discovery.Tags)
	assert.Empty(t, copy.Discovery.Tags)
	assert.NotSame(t, config.Discovery.Tags, copy.Discovery.Tags)
}
