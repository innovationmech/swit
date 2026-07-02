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

package servertest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerConfigDefaults(t *testing.T) {
	config := NewServerConfig("test-service")

	assert.Equal(t, "test-service", config.ServiceName)
	assert.True(t, config.HTTP.Enabled)
	assert.Equal(t, "0", config.HTTP.Port)
	assert.True(t, config.HTTP.TestMode)
	assert.True(t, config.GRPC.Enabled)
	assert.Equal(t, "0", config.GRPC.Port)
	assert.False(t, config.Discovery.Enabled)
	require.NoError(t, config.Validate())
}

func TestNormalizeAddr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ipv6 wildcard", "[::]:8080", "localhost:8080"},
		{"ipv4 wildcard", "0.0.0.0:8080", "localhost:8080"},
		{"port only", ":8080", "localhost:8080"},
		{"explicit host", "127.0.0.1:8080", "127.0.0.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeAddr(tt.input))
		})
	}
}

func TestDependencyContainer(t *testing.T) {
	deps := NewDependencyContainer()
	deps.AddService("database", "fake-db")

	service, err := deps.GetService("database")
	require.NoError(t, err)
	assert.Equal(t, "fake-db", service)

	_, err = deps.GetService("missing")
	assert.Error(t, err)

	require.NoError(t, deps.Initialize(context.Background()))
	assert.True(t, deps.IsInitialized())

	require.NoError(t, deps.Close())
	assert.True(t, deps.IsClosed())
}

func TestHealthCheckToggle(t *testing.T) {
	check := NewHealthCheck("toggle-service", true)
	ctx := context.Background()

	assert.Equal(t, "toggle-service", check.GetServiceName())
	assert.NoError(t, check.Check(ctx))

	check.SetHealthy(false)
	assert.Error(t, check.Check(ctx))
}

func TestStartServerServesHTTPRoutes(t *testing.T) {
	handler := NewHTTPHandler("ping-service")
	handler.AddRoute(http.MethodGet, "/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	grpcService := NewGRPCService("noop-grpc", nil)
	deps := NewDependencyContainer()

	registrar := NewServiceRegistrar()
	registrar.AddHTTPHandler(handler)
	registrar.AddGRPCService(grpcService)
	registrar.AddHealthCheck(NewHealthCheck("ping-service", true))

	srv := StartServer(t, NewServerConfig("servertest-http"), registrar, deps)

	baseURL := HTTPBaseURL(srv)
	WaitForHTTPReady(t, baseURL+"/ping", 5*time.Second)

	resp, err := http.Get(baseURL + "/ping")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "pong", body["message"])

	assert.True(t, grpcService.IsRegistered())
	assert.NotEmpty(t, srv.GetGRPCAddress())
}
