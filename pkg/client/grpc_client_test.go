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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/tracing"
)

// consulInstance describes a service instance returned by the mock Consul server.
type consulInstance struct {
	Address string
	Port    int
}

// mockConsul is a fake Consul HTTP API serving health-check queries. Instances
// can be swapped at runtime to simulate topology changes between reconnects.
type mockConsul struct {
	mu        sync.Mutex
	instances map[string][]consulInstance
	server    *httptest.Server
}

func newMockConsul(t *testing.T) *mockConsul {
	t.Helper()
	mc := &mockConsul{instances: make(map[string][]consulInstance)}
	mc.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/health/service/") {
			serviceName := strings.TrimPrefix(r.URL.Path, "/v1/health/service/")

			mc.mu.Lock()
			instances := mc.instances[serviceName]
			mc.mu.Unlock()

			entries := make([]map[string]interface{}, 0, len(instances))
			for _, inst := range instances {
				entries = append(entries, map[string]interface{}{
					"Service": map[string]interface{}{
						"Address": inst.Address,
						"Port":    inst.Port,
					},
				})
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(entries)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(mc.server.Close)
	return mc
}

func (mc *mockConsul) setInstances(service string, instances []consulInstance) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.instances[service] = instances
}

func (mc *mockConsul) discovery(t *testing.T) *discovery.ServiceDiscovery {
	t.Helper()
	sd, err := discovery.NewServiceDiscovery(strings.TrimPrefix(mc.server.URL, "http://"))
	require.NoError(t, err)
	return sd
}

// startGRPCServer starts a real gRPC server with the standard health service on
// an ephemeral port and returns its address.
func startGRPCServer(t *testing.T) (string, int) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, health.NewServer())
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	host, portStr, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return host, port
}

// newTestClientConfig returns a config suited for fast tests (no background monitoring).
func newTestClientConfig(service string) *GRPCClientConfig {
	cfg := DefaultGRPCClientConfig(service)
	cfg.ConnectionTimeout = 5 * time.Second
	cfg.MonitoringInterval = 0 // disable background monitoring unless a test opts in
	cfg.EnableTracing = false
	return cfg
}

func init() {
	logger.InitLogger()
}

func TestDefaultGRPCClientConfig(t *testing.T) {
	cfg := DefaultGRPCClientConfig("user-service")

	assert.Equal(t, "user-service", cfg.ServiceName)
	assert.Equal(t, 30*time.Second, cfg.ConnectionTimeout)
	assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
	assert.Equal(t, 10*time.Second, cfg.KeepaliveTime)
	assert.Equal(t, time.Second, cfg.KeepaliveTimeout)
	assert.True(t, cfg.EnableTracing)
	assert.True(t, cfg.EnableRetry)
	assert.Equal(t, 3, cfg.MaxRetryAttempts)
	assert.Equal(t, 5*time.Second, cfg.MonitoringInterval)
}

func TestNewGRPCClient_NilConfigUsesDefaults(t *testing.T) {
	client := NewGRPCClient(nil, nil, nil)
	defer client.Close()

	assert.Equal(t, "unknown", client.config.ServiceName)
	assert.Equal(t, connectivity.Idle, client.GetState())
	assert.Nil(t, client.GetConnection())
	assert.Empty(t, client.GetTarget())
	assert.False(t, client.IsConnected())
}

func TestGRPCClient_Connect_ServiceDiscoveryFailure(t *testing.T) {
	consul := newMockConsul(t)
	// No instances registered for the service → discovery must fail
	sd := consul.discovery(t)

	client := NewGRPCClient(newTestClientConfig("missing-service"), sd, nil)
	defer client.Close()

	err := client.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service discovery failed")
	assert.Nil(t, client.GetConnection())
}

func TestGRPCClient_Connect_Success(t *testing.T) {
	host, port := startGRPCServer(t)
	consul := newMockConsul(t)
	consul.setInstances("test-service", []consulInstance{{Address: host, Port: port}})
	sd := consul.discovery(t)

	client := NewGRPCClient(newTestClientConfig("test-service"), sd, nil)
	defer client.Close()

	require.NoError(t, client.Connect(context.Background()))
	assert.Equal(t, fmt.Sprintf("%s:%d", host, port), client.GetTarget())
	require.NotNil(t, client.GetConnection())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.WaitForReady(ctx))
	assert.True(t, client.IsConnected())
	assert.Equal(t, connectivity.Ready, client.GetState())
}

func TestGRPCClient_Connect_RPCThroughConnection(t *testing.T) {
	host, port := startGRPCServer(t)
	consul := newMockConsul(t)
	consul.setInstances("rpc-service", []consulInstance{{Address: host, Port: port}})
	sd := consul.discovery(t)

	client := NewGRPCClient(newTestClientConfig("rpc-service"), sd, nil)
	defer client.Close()

	require.NoError(t, client.Connect(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.WaitForReady(ctx))

	// Issue a real RPC via the managed connection
	healthClient := healthpb.NewHealthClient(client.GetConnection())
	resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}

func TestGRPCClient_Connect_WithTracing(t *testing.T) {
	host, port := startGRPCServer(t)
	consul := newMockConsul(t)
	consul.setInstances("traced-service", []consulInstance{{Address: host, Port: port}})
	sd := consul.discovery(t)

	cfg := newTestClientConfig("traced-service")
	cfg.EnableTracing = true
	// Uninitialized tracing manager returns no-op spans; the tracing code paths
	// and client interceptors must still work.
	tm := tracing.NewTracingManager()

	client := NewGRPCClient(cfg, sd, tm)
	defer client.Close()

	require.NoError(t, client.Connect(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.WaitForReady(ctx))

	// RPC goes through the tracing interceptor chain
	healthClient := healthpb.NewHealthClient(client.GetConnection())
	resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}

func TestGRPCClient_Connect_TracingSpanOnDiscoveryFailure(t *testing.T) {
	consul := newMockConsul(t)
	sd := consul.discovery(t)

	cfg := newTestClientConfig("missing-service")
	cfg.EnableTracing = true
	client := NewGRPCClient(cfg, sd, tracing.NewTracingManager())
	defer client.Close()

	err := client.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service discovery failed")
}

func TestGRPCClient_CreateDialOptions(t *testing.T) {
	t.Run("without tracing", func(t *testing.T) {
		cfg := newTestClientConfig("svc")
		client := NewGRPCClient(cfg, nil, nil)
		defer client.Close()

		opts := client.createDialOptions()
		// insecure credentials + keepalive
		assert.Len(t, opts, 2)
	})

	t.Run("with tracing enabled and manager present", func(t *testing.T) {
		cfg := newTestClientConfig("svc")
		cfg.EnableTracing = true
		client := NewGRPCClient(cfg, nil, tracing.NewTracingManager())
		defer client.Close()

		opts := client.createDialOptions()
		// insecure credentials + unary/stream interceptors + keepalive
		assert.Len(t, opts, 4)
	})

	t.Run("tracing enabled but no manager", func(t *testing.T) {
		cfg := newTestClientConfig("svc")
		cfg.EnableTracing = true
		client := NewGRPCClient(cfg, nil, nil)
		defer client.Close()

		opts := client.createDialOptions()
		assert.Len(t, opts, 2, "interceptors must not be added without a tracing manager")
	})
}

func TestGRPCClient_Reconnect_SwitchesTarget(t *testing.T) {
	hostA, portA := startGRPCServer(t)
	hostB, portB := startGRPCServer(t)

	consul := newMockConsul(t)
	consul.setInstances("failover-service", []consulInstance{{Address: hostA, Port: portA}})
	sd := consul.discovery(t)

	client := NewGRPCClient(newTestClientConfig("failover-service"), sd, nil)
	defer client.Close()

	require.NoError(t, client.Connect(context.Background()))
	firstConn := client.GetConnection()
	assert.Equal(t, fmt.Sprintf("%s:%d", hostA, portA), client.GetTarget())

	// Simulate instance replacement in the registry, then reconnect
	consul.setInstances("failover-service", []consulInstance{{Address: hostB, Port: portB}})
	require.NoError(t, client.Reconnect(context.Background()))

	assert.Equal(t, fmt.Sprintf("%s:%d", hostB, portB), client.GetTarget())
	assert.NotSame(t, firstConn, client.GetConnection(), "reconnect must create a new connection")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.WaitForReady(ctx))
	assert.True(t, client.IsConnected())
}

func TestGRPCClient_Reconnect_WithTracing(t *testing.T) {
	host, port := startGRPCServer(t)
	consul := newMockConsul(t)
	consul.setInstances("retrace-service", []consulInstance{{Address: host, Port: port}})
	sd := consul.discovery(t)

	cfg := newTestClientConfig("retrace-service")
	cfg.EnableTracing = true
	client := NewGRPCClient(cfg, sd, tracing.NewTracingManager())
	defer client.Close()

	require.NoError(t, client.Connect(context.Background()))
	require.NoError(t, client.Reconnect(context.Background()))
	assert.NotNil(t, client.GetConnection())
}

func TestGRPCClient_WaitForReady_NoConnection(t *testing.T) {
	client := NewGRPCClient(newTestClientConfig("svc"), nil, nil)
	defer client.Close()

	err := client.WaitForReady(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no connection established")
}

func TestGRPCClient_WaitForReady_ContextTimeout(t *testing.T) {
	consul := newMockConsul(t)
	// Register an address that is not listening so the connection can never become ready
	consul.setInstances("dead-service", []consulInstance{{Address: "127.0.0.1", Port: 1}})
	sd := consul.discovery(t)

	client := NewGRPCClient(newTestClientConfig("dead-service"), sd, nil)
	defer client.Close()

	require.NoError(t, client.Connect(context.Background()), "lazy dial must succeed even for a dead endpoint")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err := client.WaitForReady(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGRPCClient_StateMonitoringCallback(t *testing.T) {
	host, port := startGRPCServer(t)
	consul := newMockConsul(t)
	consul.setInstances("monitored-service", []consulInstance{{Address: host, Port: port}})
	sd := consul.discovery(t)

	cfg := newTestClientConfig("monitored-service")
	cfg.MonitoringInterval = 20 * time.Millisecond

	client := NewGRPCClient(cfg, sd, nil)
	defer client.Close()

	var mu sync.Mutex
	var observed []connectivity.State
	client.OnStateChange(func(state connectivity.State) {
		mu.Lock()
		defer mu.Unlock()
		observed = append(observed, state)
	})

	require.NoError(t, client.Connect(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.WaitForReady(ctx))

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, s := range observed {
			if s == connectivity.Ready {
				return true
			}
		}
		return false
	}, 3*time.Second, 20*time.Millisecond, "state-change callback must observe the Ready state")
}

func TestGRPCClient_Close(t *testing.T) {
	host, port := startGRPCServer(t)
	consul := newMockConsul(t)
	consul.setInstances("closable-service", []consulInstance{{Address: host, Port: port}})
	sd := consul.discovery(t)

	client := NewGRPCClient(newTestClientConfig("closable-service"), sd, nil)
	require.NoError(t, client.Connect(context.Background()))

	require.NoError(t, client.Close())
	assert.Nil(t, client.GetConnection())
	assert.Equal(t, connectivity.Shutdown, client.GetState())
}

func TestGRPCClient_Close_WithoutConnection(t *testing.T) {
	client := NewGRPCClient(newTestClientConfig("svc"), nil, nil)
	assert.NoError(t, client.Close())
}

func TestGRPCClient_Close_StopsMonitoring(t *testing.T) {
	cfg := newTestClientConfig("svc")
	cfg.MonitoringInterval = 10 * time.Millisecond

	client := NewGRPCClient(cfg, nil, nil)
	// Close must stop the background monitoring goroutine without deadlocking
	done := make(chan struct{})
	go func() {
		_ = client.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Close did not stop the monitoring goroutine in time")
	}
}
