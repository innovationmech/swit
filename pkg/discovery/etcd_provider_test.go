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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEtcdProvider_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *EtcdConfig
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
			name:        "missing endpoints",
			config:      &EtcdConfig{},
			wantErr:     true,
			errContains: "endpoints are required",
		},
		{
			name:   "valid config",
			config: &EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewEtcdProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, provider)
			assert.Equal(t, "etcd", provider.Name())
			assert.NoError(t, provider.Close())
		})
	}
}

func TestEtcdConfig_SetDefaults(t *testing.T) {
	config := &EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}}
	config.setDefaults()

	assert.Equal(t, defaultEtcdKeyPrefix, config.KeyPrefix)
	assert.Equal(t, defaultEtcdLeaseTTL, config.LeaseTTL)
	assert.Equal(t, defaultEtcdDialTimeout, config.DialTimeout)

	custom := &EtcdConfig{
		Endpoints:   []string{"127.0.0.1:2379"},
		KeyPrefix:   "/custom/prefix",
		LeaseTTL:    time.Minute,
		DialTimeout: time.Second,
	}
	custom.setDefaults()
	assert.Equal(t, "/custom/prefix", custom.KeyPrefix)
	assert.Equal(t, time.Minute, custom.LeaseTTL)
	assert.Equal(t, time.Second, custom.DialTimeout)
}

func TestEtcdProvider_Keys(t *testing.T) {
	provider, err := NewEtcdProvider(&EtcdConfig{
		Endpoints: []string{"127.0.0.1:2379"},
		KeyPrefix: "/swit/discovery/",
	})
	require.NoError(t, err)
	defer func() { _ = provider.Close() }()

	assert.Equal(t, "/swit/discovery/my-service/", provider.serviceKey("my-service"))

	instance := &ServiceInstance{Name: "my-service", Address: "10.0.0.1", Port: 8080}
	assert.Equal(t, "/swit/discovery/my-service/my-service-10.0.0.1-8080", provider.instanceKey(instance))

	withID := &ServiceInstance{ID: "custom", Name: "my-service", Address: "10.0.0.1", Port: 8080}
	assert.Equal(t, "/swit/discovery/my-service/custom", provider.instanceKey(withID))
}

func TestEtcdProvider_NilInstanceErrors(t *testing.T) {
	provider, err := NewEtcdProvider(&EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}})
	require.NoError(t, err)
	defer func() { _ = provider.Close() }()

	ctx := context.Background()
	assert.Error(t, provider.Register(ctx, nil))
	assert.Error(t, provider.Deregister(ctx, nil))
}

// TestEtcdProvider_Integration exercises the full register/discover/deregister
// cycle against a real etcd. It is skipped unless SWIT_ETCD_ENDPOINTS is set
// (e.g. SWIT_ETCD_ENDPOINTS=127.0.0.1:2379 go test ./pkg/discovery -run Etcd).
func TestEtcdProvider_Integration(t *testing.T) {
	endpoints := os.Getenv("SWIT_ETCD_ENDPOINTS")
	if endpoints == "" {
		t.Skip("SWIT_ETCD_ENDPOINTS not set, skipping etcd integration test")
	}

	provider, err := NewEtcdProvider(&EtcdConfig{
		Endpoints: []string{endpoints},
		KeyPrefix: "/swit/discovery-test",
		LeaseTTL:  10 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = provider.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	require.True(t, provider.IsHealthy(ctx))

	instance := &ServiceInstance{
		Name:    "etcd-it-service",
		Address: "127.0.0.1",
		Port:    18080,
		Tags:    []string{"it"},
	}
	require.NoError(t, provider.Register(ctx, instance))

	instances, err := provider.Discover(ctx, "etcd-it-service")
	require.NoError(t, err)
	require.Len(t, instances, 1)
	assert.Equal(t, "127.0.0.1:18080", instances[0].Endpoint())
	assert.Equal(t, []string{"it"}, instances[0].Tags)

	require.NoError(t, provider.Deregister(ctx, instance))

	instances, err = provider.Discover(ctx, "etcd-it-service")
	require.NoError(t, err)
	assert.Empty(t, instances)
}
