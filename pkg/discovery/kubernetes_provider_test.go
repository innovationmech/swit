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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeK8sResolver implements kubernetesResolver for DNS mode tests.
type fakeK8sResolver struct {
	ips    map[string][]net.IP
	srvs   map[string][]*net.SRV
	ipErr  error
	srvErr error
}

func (f *fakeK8sResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if f.ipErr != nil {
		return nil, f.ipErr
	}
	return f.ips[host], nil
}

func (f *fakeK8sResolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	if f.srvErr != nil {
		return "", nil, f.srvErr
	}
	return name, f.srvs[name], nil
}

// newMockKubernetesAPI starts an httptest server emulating the Endpoints API.
func newMockKubernetesAPI(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/namespaces/default/endpoints/test-service":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"subsets": [
					{
						"addresses": [
							{"ip": "10.244.0.5", "targetRef": {"name": "test-service-abc"}},
							{"ip": "10.244.0.6"}
						],
						"ports": [
							{"name": "http", "port": 8080},
							{"name": "metrics", "port": 9090}
						]
					}
				]
			}`))
		case "/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"major":"1","minor":"31"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newEndpointsProvider(t *testing.T, server *httptest.Server, portName string) *KubernetesProvider {
	t.Helper()
	provider, err := NewKubernetesProvider(&KubernetesConfig{
		Mode:      KubernetesModeEndpoints,
		Namespace: "default",
		APIServer: server.URL,
		// Point token/CA at nonexistent files so in-cluster defaults are skipped.
		TokenFile:  "/nonexistent/token",
		CACertFile: "/nonexistent/ca.crt",
		PortName:   portName,
	})
	require.NoError(t, err)
	return provider
}

func TestNewKubernetesProvider_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *KubernetesConfig
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "cannot be nil",
		},
		{
			name:        "invalid mode",
			config:      &KubernetesConfig{Mode: "watch"},
			wantErr:     true,
			errContains: "unsupported kubernetes discovery mode",
		},
		{
			name:   "dns mode",
			config: &KubernetesConfig{Mode: KubernetesModeDNS},
		},
		{
			name:   "empty mode defaults to endpoints",
			config: &KubernetesConfig{TokenFile: "/nonexistent", CACertFile: "/nonexistent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewKubernetesProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, provider)
			assert.Equal(t, "kubernetes", provider.Name())
			assert.NoError(t, provider.Close())
		})
	}
}

func TestKubernetesConfig_SetDefaults(t *testing.T) {
	config := &KubernetesConfig{}
	config.setDefaults()

	assert.Equal(t, KubernetesModeEndpoints, config.Mode)
	assert.Equal(t, defaultKubernetesNamespace, config.Namespace)
	assert.Equal(t, defaultKubernetesAPIServer, config.APIServer)
	assert.Equal(t, defaultKubernetesDNSDomain, config.DNSDomain)
}

func TestKubernetesProvider_RegisterDeregisterNoOp(t *testing.T) {
	provider, err := NewKubernetesProvider(&KubernetesConfig{Mode: KubernetesModeDNS})
	require.NoError(t, err)

	ctx := context.Background()
	instance := &ServiceInstance{Name: "svc", Address: "10.0.0.1", Port: 8080}

	assert.NoError(t, provider.Register(ctx, instance))
	assert.NoError(t, provider.Deregister(ctx, instance))

	assert.Error(t, provider.Register(ctx, nil))
	assert.Error(t, provider.Deregister(ctx, nil))
}

func TestKubernetesProvider_DiscoverEndpoints(t *testing.T) {
	server := newMockKubernetesAPI(t)
	provider := newEndpointsProvider(t, server, "")
	ctx := context.Background()

	instances, err := provider.Discover(ctx, "test-service")
	require.NoError(t, err)
	require.Len(t, instances, 2)

	assert.Equal(t, "test-service-abc", instances[0].ID)
	assert.Equal(t, "10.244.0.5", instances[0].Address)
	assert.Equal(t, 8080, instances[0].Port)
	assert.Equal(t, "10.244.0.6", instances[1].Address)
	assert.Equal(t, 8080, instances[1].Port)
}

func TestKubernetesProvider_DiscoverEndpointsWithPortName(t *testing.T) {
	server := newMockKubernetesAPI(t)
	provider := newEndpointsProvider(t, server, "metrics")

	instances, err := provider.Discover(context.Background(), "test-service")
	require.NoError(t, err)
	require.Len(t, instances, 2)
	assert.Equal(t, 9090, instances[0].Port)
}

func TestKubernetesProvider_DiscoverEndpointsNotFound(t *testing.T) {
	server := newMockKubernetesAPI(t)
	provider := newEndpointsProvider(t, server, "")

	_, err := provider.Discover(context.Background(), "missing-service")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no healthy service instances found")
}

func TestKubernetesProvider_DiscoverDNS_SRV(t *testing.T) {
	provider, err := NewKubernetesProvider(&KubernetesConfig{
		Mode:      KubernetesModeDNS,
		Namespace: "prod",
		DNSDomain: "cluster.local",
	})
	require.NoError(t, err)

	provider.resolver = &fakeK8sResolver{
		srvs: map[string][]*net.SRV{
			"my-service.prod.svc.cluster.local": {
				{Target: "pod-0.my-service.prod.svc.cluster.local.", Port: 8080},
				{Target: "pod-1.my-service.prod.svc.cluster.local.", Port: 8080},
			},
		},
	}

	instances, err := provider.Discover(context.Background(), "my-service")
	require.NoError(t, err)
	require.Len(t, instances, 2)
	assert.Equal(t, "pod-0.my-service.prod.svc.cluster.local", instances[0].Address)
	assert.Equal(t, 8080, instances[0].Port)
}

func TestKubernetesProvider_DiscoverDNS_AFallback(t *testing.T) {
	provider, err := NewKubernetesProvider(&KubernetesConfig{
		Mode:      KubernetesModeDNS,
		Namespace: "prod",
		Port:      9090,
	})
	require.NoError(t, err)

	provider.resolver = &fakeK8sResolver{
		srvErr: fmt.Errorf("no srv records"),
		ips: map[string][]net.IP{
			"my-service.prod.svc.cluster.local": {net.ParseIP("10.96.0.10")},
		},
	}

	instances, err := provider.Discover(context.Background(), "my-service")
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, "10.96.0.10:9090", instances[0].Endpoint())
}

func TestKubernetesProvider_DiscoverDNS_NoPortConfigured(t *testing.T) {
	provider, err := NewKubernetesProvider(&KubernetesConfig{
		Mode:      KubernetesModeDNS,
		Namespace: "prod",
	})
	require.NoError(t, err)

	provider.resolver = &fakeK8sResolver{
		srvErr: fmt.Errorf("no srv records"),
		ips: map[string][]net.IP{
			"my-service.prod.svc.cluster.local": {net.ParseIP("10.96.0.10")},
		},
	}

	_, err = provider.Discover(context.Background(), "my-service")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a configured port")
}

func TestKubernetesProvider_DiscoverDNS_ResolutionError(t *testing.T) {
	provider, err := NewKubernetesProvider(&KubernetesConfig{
		Mode: KubernetesModeDNS,
		Port: 8080,
	})
	require.NoError(t, err)

	provider.resolver = &fakeK8sResolver{
		srvErr: fmt.Errorf("no srv records"),
		ipErr:  fmt.Errorf("no such host"),
	}

	_, err = provider.Discover(context.Background(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve kubernetes service")
}

func TestKubernetesProvider_IsHealthy(t *testing.T) {
	t.Run("endpoints mode with reachable API", func(t *testing.T) {
		server := newMockKubernetesAPI(t)
		provider := newEndpointsProvider(t, server, "")
		assert.True(t, provider.IsHealthy(context.Background()))
	})

	t.Run("endpoints mode with unreachable API", func(t *testing.T) {
		provider, err := NewKubernetesProvider(&KubernetesConfig{
			Mode:       KubernetesModeEndpoints,
			APIServer:  "http://127.0.0.1:1",
			TokenFile:  "/nonexistent",
			CACertFile: "/nonexistent",
		})
		require.NoError(t, err)
		assert.False(t, provider.IsHealthy(context.Background()))
	})

	t.Run("dns mode is always healthy", func(t *testing.T) {
		provider, err := NewKubernetesProvider(&KubernetesConfig{Mode: KubernetesModeDNS})
		require.NoError(t, err)
		assert.True(t, provider.IsHealthy(context.Background()))
	})
}
