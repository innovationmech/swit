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
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	userv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/user/v1"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/tracing"
)

// ServiceDiscoveryInterface defines the interface for service discovery operations
type ServiceDiscoveryInterface interface {
	GetInstanceRoundRobin(serviceName string) (string, error)
}

// GRPCUserClient implements UserClient interface using gRPC with tracing support
type GRPCUserClient struct {
	sd             ServiceDiscoveryInterface
	tracingManager tracing.TracingManager
	conn           *grpc.ClientConn
	client         userv1.UserServiceClient
	serviceName    string
}

// NewGRPCUserClient creates a new gRPC user client with tracing support
func NewGRPCUserClient(sd ServiceDiscoveryInterface, tracingManager tracing.TracingManager) (UserClient, error) {
	return &GRPCUserClient{
		sd:             sd,
		tracingManager: tracingManager,
		serviceName:    "swit-serve",
	}, nil
}

// ensureConnection ensures gRPC connection is established
func (c *GRPCUserClient) ensureConnection(ctx context.Context) error {
	if c.conn != nil && c.conn.GetState() == connectivity.Ready {
		return nil
	}

	// Create tracing span for connection establishment
	var span tracing.Span
	if c.tracingManager != nil {
		ctx, span = c.tracingManager.StartSpan(ctx, "gRPC_connect_to_service",
			tracing.WithSpanKind(oteltrace.SpanKindClient),
			tracing.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("service.name", c.serviceName),
				attribute.String("operation.type", "connection_establishment"),
			),
		)
		defer span.End()
	}

	// Get service instance from service discovery
	serviceURL, err := c.sd.GetInstanceRoundRobin(c.serviceName)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "service discovery failed")
			span.SetAttribute("error.type", "service_discovery")
			span.RecordError(err)
		}
		return fmt.Errorf("unable to discover %s service: %w", c.serviceName, err)
	}

	if span != nil {
		span.SetAttribute("net.peer.name", serviceURL)
	}

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
	}

	// Create gRPC connection options with tracing interceptors
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// Add tracing interceptors if tracing is enabled
	if c.tracingManager != nil {
		opts = append(opts,
			grpc.WithChainUnaryInterceptor(middleware.UnaryClientInterceptor(c.tracingManager)),
			grpc.WithChainStreamInterceptor(middleware.StreamClientInterceptor(c.tracingManager)),
		)
	}

	// Add keepalive parameters for better connection management
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second, // Send keepalive ping every 10 seconds
		Timeout:             time.Second,      // Wait 1 second for ping ack before considering connection dead
		PermitWithoutStream: true,             // Send ping even without active streams
	}))

	// Establish gRPC connection
	conn, err := grpc.DialContext(ctx, serviceURL, opts...)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "gRPC connection failed")
			span.SetAttribute("error.type", "connection")
			span.RecordError(err)
		}
		return fmt.Errorf("failed to connect to %s service: %w", c.serviceName, err)
	}

	c.conn = conn
	c.client = userv1.NewUserServiceClient(conn)

	if span != nil {
		span.SetAttribute("connection.state", conn.GetState().String())
		span.SetAttribute("operation.success", true)
	}

	return nil
}

// ValidateUserCredentials validates user credentials via gRPC call
func (c *GRPCUserClient) ValidateUserCredentials(ctx context.Context, username, password string) (*model.User, error) {
	// Ensure gRPC connection is ready before starting the RPC span
	if err := c.ensureConnection(ctx); err != nil {
		return nil, err
	}

	// Create tracing span for the gRPC call (after connection is established)
	var span tracing.Span
	if c.tracingManager != nil {
		ctx, span = c.tracingManager.StartSpan(ctx, "gRPC_GetUserByUsername",
			tracing.WithSpanKind(oteltrace.SpanKindClient),
			tracing.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", "swit.user.v1.UserService"),
				attribute.String("rpc.method", "GetUserByUsername"),
				attribute.String("service.name", c.serviceName),
				attribute.String("user.username", username),
			),
		)
		defer span.End()
		span.SetAttribute("rpc.grpc.target", c.conn.Target())
		span.SetAttribute("connection.state", c.conn.GetState().String())
	}

	// Create gRPC request
	req := &userv1.GetUserByUsernameRequest{
		Username: username,
	}

	// Make gRPC call with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.GetUserByUsername(ctx, req)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "gRPC call failed")
			span.SetAttribute("error.type", "rpc_call")
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to get user by username via gRPC: %w", err)
	}

	if resp == nil || resp.User == nil {
		if span != nil {
			span.SetStatus(codes.Error, "user not found")
			span.SetAttribute("error.type", "not_found")
		}
		return nil, fmt.Errorf("user not found")
	}

	// Convert protobuf user to model.User
	user := convertProtoUserToModel(resp.User)

	// Validate password (this would typically be done on the server side,
	// but for this demo, we're doing basic credential validation)
	// In a real implementation, you'd have a separate gRPC method for credential validation
	if !validatePassword(password, user.PasswordHash) {
		if span != nil {
			span.SetStatus(codes.Error, "invalid credentials")
			span.SetAttribute("error.type", "authentication_failed")
		}
		return nil, fmt.Errorf("invalid credentials")
	}

	// Record successful operation
	if span != nil {
		span.SetAttribute("operation.success", true)
		span.SetAttribute("user.id", user.ID.String())
		span.SetStatus(codes.Ok, "user validation successful")
	}

	return user, nil
}

// Close closes the gRPC connection
func (c *GRPCUserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetConnectionState returns the current connection state
func (c *GRPCUserClient) GetConnectionState() connectivity.State {
	if c.conn != nil {
		return c.conn.GetState()
	}
	return connectivity.Idle
}

// convertProtoUserToModel converts protobuf User to model.User
func convertProtoUserToModel(protoUser *userv1.User) *model.User {
	if protoUser == nil {
		return nil
	}

	user := &model.User{
		Username: protoUser.Username,
		Email:    protoUser.Email,
		IsActive: protoUser.Status == userv1.UserStatus_USER_STATUS_ACTIVE,
	}

	// Parse UUID from string
	if protoUser.Id != "" {
		if id, err := uuid.Parse(protoUser.Id); err == nil {
			user.ID = id
		}
	}

	return user
}

// validatePassword validates the provided password against the stored hash
// This is a placeholder implementation - in reality, you'd use proper password hashing
func validatePassword(password, hash string) bool {
	// This is a demo implementation - in practice, use bcrypt or similar
	// For now, just check if password is not empty
	return password != "" && hash != ""
}
