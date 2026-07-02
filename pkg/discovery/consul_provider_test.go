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

package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockConsulServer starts an httptest server emulating the subset of the
// Consul HTTP API used by ConsulProvider.
func newMockConsulServer(t *testing.T) (*httptest.Server, *ConsulProvider) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/v1/agent/service/register"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/v1/agent/service/deregister"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/health/service/test-service"):
			response := []map[string]interface{}{
				{
					"Node": map[string]interface{}{"Address": "10.0.0.9"},
					"Service": map[string]interface{}{
						"ID":      "test-service-1",
						"Service": "test-service",
						"Address": "10.0.0.1",
						"Port":    8080,
						"Tags":    []string{"v1"},
						"Meta":    map[string]string{"protocol": "http"},
					},
				},
				{
					"Node": map[string]interface{}{"Address": "10.0.0.9"},
					"Service": map[string]interface{}{
						"ID":      "test-service-2",
						"Service": "test-service",
						"Address": "", // falls back to node address
						"Port":    8081,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/health/service/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/status/leader"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`"127.0.0.1:8300"`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	provider, err := NewConsulProvider(strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)
	return server, provider
}

func TestConsulProvider_Name(t *testing.T) {
	_, provider := newMockConsulServer(t)
	assert.Equal(t, "consul", provider.Name())
}

func TestConsulProvider_Register(t *testing.T) {
	_, provider := newMockConsulServer(t)
	ctx := context.Background()

	err := provider.Register(ctx, &ServiceInstance{
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
		Tags:    []string{"v1"},
		Meta:    map[string]string{"protocol": "http"},
	})
	assert.NoError(t, err)

	err = provider.Register(ctx, nil)
	assert.Error(t, err)
}

func TestConsulProvider_Deregister(t *testing.T) {
	_, provider := newMockConsulServer(t)
	ctx := context.Background()

	err := provider.Deregister(ctx, &ServiceInstance{
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	})
	assert.NoError(t, err)

	err = provider.Deregister(ctx, nil)
	assert.Error(t, err)
}

func TestConsulProvider_Discover(t *testing.T) {
	_, provider := newMockConsulServer(t)
	ctx := context.Background()

	instances, err := provider.Discover(ctx, "test-service")
	require.NoError(t, err)
	require.Len(t, instances, 2)

	assert.Equal(t, "test-service-1", instances[0].ID)
	assert.Equal(t, "10.0.0.1", instances[0].Address)
	assert.Equal(t, 8080, instances[0].Port)
	assert.Equal(t, []string{"v1"}, instances[0].Tags)

	// Empty service address falls back to the node address
	assert.Equal(t, "10.0.0.9", instances[1].Address)
	assert.Equal(t, 8081, instances[1].Port)

	empty, err := provider.Discover(ctx, "unknown-service")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestConsulProvider_IsHealthy(t *testing.T) {
	_, provider := newMockConsulServer(t)
	assert.True(t, provider.IsHealthy(context.Background()))

	unreachable, err := NewConsulProvider("127.0.0.1:1")
	require.NoError(t, err)
	assert.False(t, unreachable.IsHealthy(context.Background()))
}

func TestConsulProvider_Close(t *testing.T) {
	_, provider := newMockConsulServer(t)
	assert.NoError(t, provider.Close())
}

func TestNewConsulProviderFromServiceDiscovery(t *testing.T) {
	sd, err := NewServiceDiscovery("127.0.0.1:8500")
	require.NoError(t, err)

	provider := NewConsulProviderFromServiceDiscovery(sd)
	require.NotNil(t, provider)
	assert.Equal(t, "consul", provider.Name())
}
