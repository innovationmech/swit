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
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/tracing"
)

// GRPCClientConfig holds configuration for gRPC client
type GRPCClientConfig struct {
	ServiceName        string        // Target service name for service discovery
	ConnectionTimeout  time.Duration // Connection timeout
	RequestTimeout     time.Duration // Default request timeout
	KeepaliveTime      time.Duration // Keepalive ping interval
	KeepaliveTimeout   time.Duration // Keepalive ping timeout
	EnableTracing      bool          // Enable tracing interceptors
	EnableRetry        bool          // Enable automatic retries
	MaxRetryAttempts   int           // Maximum retry attempts
	MonitoringInterval time.Duration // Connection state monitoring interval
}

// DefaultGRPCClientConfig returns default gRPC client configuration
func DefaultGRPCClientConfig(serviceName string) *GRPCClientConfig {
	return &GRPCClientConfig{
		ServiceName:        serviceName,
		ConnectionTimeout:  30 * time.Second,
		RequestTimeout:     30 * time.Second,
		KeepaliveTime:      10 * time.Second,
		KeepaliveTimeout:   time.Second,
		EnableTracing:      true,
		EnableRetry:        true,
		MaxRetryAttempts:   3,
		MonitoringInterval: 5 * time.Second,
	}
}

// GRPCClient provides a high-level gRPC client with tracing, monitoring, and reconnection capabilities
type GRPCClient struct {
	config         *GRPCClientConfig
	sd             *discovery.ServiceDiscovery
	tracingManager tracing.TracingManager

	mu     sync.RWMutex
	conn   *grpc.ClientConn
	target string
	state  connectivity.State

	// Connection monitoring
	stopMonitoring chan struct{}
	monitoringWG   sync.WaitGroup

	// Connection state callbacks
	onStateChange []func(connectivity.State)
}

// NewGRPCClient creates a new gRPC client with tracing and monitoring capabilities
func NewGRPCClient(config *GRPCClientConfig, sd *discovery.ServiceDiscovery, tracingManager tracing.TracingManager) *GRPCClient {
	if config == nil {
		config = DefaultGRPCClientConfig("unknown")
	}

	client := &GRPCClient{
		config:         config,
		sd:             sd,
		tracingManager: tracingManager,
		state:          connectivity.Idle,
		stopMonitoring: make(chan struct{}),
	}

	// Start connection monitoring if enabled
	if config.MonitoringInterval > 0 {
		client.startMonitoring()
	}

	return client
}

// Connect establishes a gRPC connection with tracing
func (c *GRPCClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create tracing span for connection establishment
	var span tracing.Span
	if c.tracingManager != nil {
		ctx, span = c.tracingManager.StartSpan(ctx, "gRPC_client_connect",
			tracing.WithSpanKind(oteltrace.SpanKindClient),
			tracing.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("service.name", c.config.ServiceName),
				attribute.String("operation.type", "connection"),
			),
		)
		defer span.End()
	}

	// Get service instance from service discovery
	target, err := c.sd.GetInstanceRoundRobin(c.config.ServiceName)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "service discovery failed")
			span.SetAttribute("error.type", "service_discovery")
			span.RecordError(err)
		}
		return fmt.Errorf("service discovery failed for %s: %w", c.config.ServiceName, err)
	}

	c.target = target
	if span != nil {
		span.SetAttribute("rpc.grpc.target", target)
	}

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
	}

	// Create connection options
	opts := c.createDialOptions()

	// Establish connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, c.config.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(connectCtx, target, opts...)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "gRPC connection failed")
			span.SetAttribute("error.type", "connection")
			span.RecordError(err)
		}
		return fmt.Errorf("failed to connect to %s at %s: %w", c.config.ServiceName, target, err)
	}

	c.conn = conn
	c.state = conn.GetState()

	if span != nil {
		span.SetAttribute("connection.state", c.state.String())
		span.SetAttribute("operation.success", true)
	}

	return nil
}

// createDialOptions creates gRPC dial options with tracing and other configurations
func (c *GRPCClient) createDialOptions() []grpc.DialOption {
	var opts []grpc.DialOption

	// Transport credentials (insecure for demo)
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// Add tracing interceptors if enabled
	if c.config.EnableTracing && c.tracingManager != nil {
		opts = append(opts,
			grpc.WithChainUnaryInterceptor(middleware.UnaryClientInterceptor(c.tracingManager)),
			grpc.WithChainStreamInterceptor(middleware.StreamClientInterceptor(c.tracingManager)),
		)
	}

	// Add keepalive parameters
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                c.config.KeepaliveTime,
		Timeout:             c.config.KeepaliveTimeout,
		PermitWithoutStream: true,
	}))

	return opts
}

// GetConnection returns the gRPC connection
func (c *GRPCClient) GetConnection() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// GetState returns the current connection state
func (c *GRPCClient) GetState() connectivity.State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn != nil {
		return c.conn.GetState()
	}
	return c.state
}

// GetTarget returns the current connection target
func (c *GRPCClient) GetTarget() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.target
}

// IsConnected returns true if the connection is ready
func (c *GRPCClient) IsConnected() bool {
	return c.GetState() == connectivity.Ready
}

// WaitForReady waits for the connection to be ready
func (c *GRPCClient) WaitForReady(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("no connection established")
	}

	// Create tracing span for wait operation
	var span tracing.Span
	if c.tracingManager != nil {
		ctx, span = c.tracingManager.StartSpan(ctx, "gRPC_client_wait_for_ready",
			tracing.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("service.name", c.config.ServiceName),
				attribute.String("operation.type", "wait_for_ready"),
			),
		)
		defer span.End()
	}

	// Wait for connection to be ready
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			if span != nil {
				span.SetAttribute("connection.state", "ready")
				span.SetAttribute("operation.success", true)
			}
			return nil
		}

		if state == connectivity.Shutdown {
			if span != nil {
				span.SetStatus(codes.Error, "connection shutdown")
			}
			return fmt.Errorf("connection is shutdown")
		}

		if !conn.WaitForStateChange(ctx, state) {
			if span != nil {
				span.SetStatus(codes.Error, "wait timeout")
			}
			return ctx.Err()
		}
	}
}

// OnStateChange registers a callback for connection state changes
func (c *GRPCClient) OnStateChange(callback func(connectivity.State)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStateChange = append(c.onStateChange, callback)
}

// startMonitoring starts connection state monitoring
func (c *GRPCClient) startMonitoring() {
	c.monitoringWG.Add(1)
	go func() {
		defer c.monitoringWG.Done()
		ticker := time.NewTicker(c.config.MonitoringInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopMonitoring:
				return
			case <-ticker.C:
				c.checkConnectionState()
			}
		}
	}()
}

// checkConnectionState monitors and reports connection state changes
func (c *GRPCClient) checkConnectionState() {
	c.mu.RLock()
	conn := c.conn
	target := c.target
	oldState := c.state
	callbacks := make([]func(connectivity.State), len(c.onStateChange))
	copy(callbacks, c.onStateChange)
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	currentState := conn.GetState()
	if currentState != oldState {
		c.mu.Lock()
		c.state = currentState
		c.mu.Unlock()

		// Create tracing span for state change
		if c.tracingManager != nil {
			_, span := c.tracingManager.StartSpan(context.Background(), "gRPC_client_state_change",
				tracing.WithAttributes(
					attribute.String("rpc.system", "grpc"),
					attribute.String("service.name", c.config.ServiceName),
					attribute.String("rpc.grpc.target", target),
					attribute.String("connection.state.old", oldState.String()),
					attribute.String("connection.state.new", currentState.String()),
				),
			)
			span.End()
		}

		// Call state change callbacks
		for _, callback := range callbacks {
			if callback != nil {
				go callback(currentState)
			}
		}
	}
}

// Reconnect attempts to reconnect to the service
func (c *GRPCClient) Reconnect(ctx context.Context) error {
	// Create tracing span for reconnection
	var span tracing.Span
	if c.tracingManager != nil {
		ctx, span = c.tracingManager.StartSpan(ctx, "gRPC_client_reconnect",
			tracing.WithSpanKind(oteltrace.SpanKindClient),
			tracing.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("service.name", c.config.ServiceName),
				attribute.String("operation.type", "reconnection"),
			),
		)
		defer span.End()
	}

	return c.Connect(ctx)
}

// Close closes the gRPC connection and stops monitoring
func (c *GRPCClient) Close() error {
	// Stop monitoring
	close(c.stopMonitoring)
	c.monitoringWG.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.state = connectivity.Shutdown
		return err
	}

	return nil
}
