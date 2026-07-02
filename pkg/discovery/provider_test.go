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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProvider is a minimal in-memory Provider used for resolver tests.
type fakeProvider struct {
	instances map[string][]*ServiceInstance
	err       error
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Register(ctx context.Context, instance *ServiceInstance) error {
	f.instances[instance.Name] = append(f.instances[instance.Name], instance)
	return nil
}

func (f *fakeProvider) Deregister(ctx context.Context, instance *ServiceInstance) error {
	return nil
}

func (f *fakeProvider) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.instances[serviceName], nil
}

func (f *fakeProvider) IsHealthy(ctx context.Context) bool { return true }

func (f *fakeProvider) Close() error { return nil }

func TestServiceInstance_InstanceID(t *testing.T) {
	tests := []struct {
		name     string
		instance *ServiceInstance
		expected string
	}{
		{
			name:     "explicit ID is preserved",
			instance: &ServiceInstance{ID: "custom-id", Name: "svc", Address: "10.0.0.1", Port: 8080},
			expected: "custom-id",
		},
		{
			name:     "derived ID from name, address and port",
			instance: &ServiceInstance{Name: "svc", Address: "10.0.0.1", Port: 8080},
			expected: "svc-10.0.0.1-8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.instance.InstanceID())
		})
	}
}

func TestServiceInstance_Endpoint(t *testing.T) {
	instance := &ServiceInstance{Name: "svc", Address: "10.0.0.1", Port: 9090}
	assert.Equal(t, "10.0.0.1:9090", instance.Endpoint())
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      *ProviderConfig
		wantErr     bool
		errContains string
		wantName    string
	}{
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "cannot be nil",
		},
		{
			name:     "empty type defaults to consul",
			config:   &ProviderConfig{Address: "127.0.0.1:8500"},
			wantName: "consul",
		},
		{
			name:     "explicit consul type",
			config:   &ProviderConfig{Type: ProviderTypeConsul, Address: "127.0.0.1:8500"},
			wantName: "consul",
		},
		{
			name:     "type is case-insensitive",
			config:   &ProviderConfig{Type: "Consul", Address: "127.0.0.1:8500"},
			wantName: "consul",
		},
		{
			name: "etcd type with endpoints",
			config: &ProviderConfig{
				Type: ProviderTypeEtcd,
				Etcd: EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}},
			},
			wantName: "etcd",
		},
		{
			name:     "etcd type falls back to address",
			config:   &ProviderConfig{Type: ProviderTypeEtcd, Address: "127.0.0.1:2379"},
			wantName: "etcd",
		},
		{
			name:        "etcd type without endpoints",
			config:      &ProviderConfig{Type: ProviderTypeEtcd},
			wantErr:     true,
			errContains: "endpoints are required",
		},
		{
			name: "kubernetes type with dns mode",
			config: &ProviderConfig{
				Type:       ProviderTypeKubernetes,
				Kubernetes: KubernetesConfig{Mode: KubernetesModeDNS, Namespace: "default"},
			},
			wantName: "kubernetes",
		},
		{
			name:        "unsupported type",
			config:      &ProviderConfig{Type: "zookeeper"},
			wantErr:     true,
			errContains: "unsupported discovery provider type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, provider)
			assert.Equal(t, tt.wantName, provider.Name())
			assert.NoError(t, provider.Close())
		})
	}
}

func TestSupportedProviderTypes(t *testing.T) {
	types := SupportedProviderTypes()
	assert.ElementsMatch(t,
		[]ProviderType{ProviderTypeConsul, ProviderTypeEtcd, ProviderTypeKubernetes},
		types)
}

func TestProviderConfig_CacheKey(t *testing.T) {
	consulA := &ProviderConfig{Address: "127.0.0.1:8500"}
	consulB := &ProviderConfig{Type: ProviderTypeConsul, Address: "127.0.0.1:8500"}
	consulOther := &ProviderConfig{Address: "127.0.0.1:8501"}
	etcd := &ProviderConfig{Type: ProviderTypeEtcd, Etcd: EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}}}
	k8s := &ProviderConfig{Type: ProviderTypeKubernetes, Kubernetes: KubernetesConfig{Namespace: "prod"}}

	assert.Equal(t, consulA.cacheKey(), consulB.cacheKey())
	assert.NotEqual(t, consulA.cacheKey(), consulOther.cacheKey())
	assert.NotEqual(t, consulA.cacheKey(), etcd.cacheKey())
	assert.NotEqual(t, etcd.cacheKey(), k8s.cacheKey())
}

func TestLoadBalancedResolver_RoundRobin(t *testing.T) {
	provider := &fakeProvider{instances: map[string][]*ServiceInstance{
		"svc": {
			{Name: "svc", Address: "10.0.0.1", Port: 8080},
			{Name: "svc", Address: "10.0.0.2", Port: 8080},
			{Name: "svc", Address: "10.0.0.3", Port: 8080},
		},
	}}
	resolver := NewLoadBalancedResolver(provider)
	ctx := context.Background()

	var got []string
	for i := 0; i < 6; i++ {
		endpoint, err := resolver.GetInstanceRoundRobin(ctx, "svc")
		require.NoError(t, err)
		got = append(got, endpoint)
	}

	expected := []string{
		"10.0.0.1:8080", "10.0.0.2:8080", "10.0.0.3:8080",
		"10.0.0.1:8080", "10.0.0.2:8080", "10.0.0.3:8080",
	}
	assert.Equal(t, expected, got)
}

func TestLoadBalancedResolver_Random(t *testing.T) {
	provider := &fakeProvider{instances: map[string][]*ServiceInstance{
		"svc": {
			{Name: "svc", Address: "10.0.0.1", Port: 8080},
			{Name: "svc", Address: "10.0.0.2", Port: 8080},
		},
	}}
	resolver := NewLoadBalancedResolver(provider)
	ctx := context.Background()

	valid := map[string]bool{"10.0.0.1:8080": true, "10.0.0.2:8080": true}
	for i := 0; i < 10; i++ {
		endpoint, err := resolver.GetInstanceRandom(ctx, "svc")
		require.NoError(t, err)
		assert.True(t, valid[endpoint], "unexpected endpoint %s", endpoint)
	}
}

func TestLoadBalancedResolver_Errors(t *testing.T) {
	t.Run("no instances", func(t *testing.T) {
		provider := &fakeProvider{instances: map[string][]*ServiceInstance{}}
		resolver := NewLoadBalancedResolver(provider)

		_, err := resolver.GetInstanceRoundRobin(context.Background(), "missing")
		assert.ErrorContains(t, err, "no healthy service instances found")

		_, err = resolver.GetInstanceRandom(context.Background(), "missing")
		assert.ErrorContains(t, err, "no healthy service instances found")
	})

	t.Run("provider error propagates", func(t *testing.T) {
		provider := &fakeProvider{err: fmt.Errorf("backend down")}
		resolver := NewLoadBalancedResolver(provider)

		_, err := resolver.GetInstanceRoundRobin(context.Background(), "svc")
		assert.ErrorContains(t, err, "backend down")
	})
}
