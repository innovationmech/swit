// Copyright (c) 2024 Six-Thirty Labs, Inc.
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

package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/open-policy-agent/opa/rego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/innovationmech/swit/pkg/security/opa"
)

// mockPolicyOPAClient 模拟 OPA 客户端
type mockPolicyOPAClient struct {
	evaluateFunc func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error)
}

func (m *mockPolicyOPAClient) Evaluate(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
	if m.evaluateFunc != nil {
		policyInput, ok := input.(*opa.PolicyInput)
		if !ok {
			return nil, errors.New("invalid input type")
		}
		return m.evaluateFunc(ctx, path, policyInput)
	}
	return &opa.Result{Allowed: true}, nil
}

func (m *mockPolicyOPAClient) Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPolicyOPAClient) PartialEvaluate(ctx context.Context, query string, input interface{}) (*opa.PartialResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPolicyOPAClient) LoadPolicy(ctx context.Context, name string, policy string) error {
	return errors.New("not implemented")
}

func (m *mockPolicyOPAClient) LoadPolicyFromFile(ctx context.Context, path string) error {
	return errors.New("not implemented")
}

func (m *mockPolicyOPAClient) LoadData(ctx context.Context, path string, data interface{}) error {
	return errors.New("not implemented")
}

func (m *mockPolicyOPAClient) RemovePolicy(ctx context.Context, name string) error {
	return errors.New("not implemented")
}

func (m *mockPolicyOPAClient) Close(ctx context.Context) error {
	return nil
}

func (m *mockPolicyOPAClient) IsEmbedded() bool {
	return false
}

// mockPolicyUnaryHandler 模拟一元 RPC 处理器
type mockPolicyUnaryHandler struct {
	called  bool
	request interface{}
	err     error
}

func (m *mockPolicyUnaryHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	m.called = true
	m.request = req
	return "success", m.err
}

// mockPolicyStreamHandler 模拟流式 RPC 处理器
type mockPolicyStreamHandler struct {
	called bool
	err    error
}

func (m *mockPolicyStreamHandler) handle(srv interface{}, stream grpc.ServerStream) error {
	m.called = true
	return m.err
}

// mockPolicyServerStream 模拟 gRPC ServerStream
type mockPolicyServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockPolicyServerStream) Context() context.Context {
	return m.ctx
}

func TestUnaryPolicyInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		setupClient    func() *mockPolicyOPAClient
		setupContext   func() context.Context
		config         []GRPCPolicyOption
		expectError    bool
		expectCode     codes.Code
		expectCalled   bool
		validateResult func(t *testing.T, resp interface{}, err error)
	}{
		{
			name: "allowed by policy",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return &opa.Result{
							Allowed:    true,
							DecisionID: "test-decision-1",
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "user_id", "user123")
				return ctx
			},
			config:       []GRPCPolicyOption{},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "denied by policy",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return &opa.Result{
							Allowed:    false,
							DecisionID: "test-decision-2",
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config:       []GRPCPolicyOption{},
			expectError:  true,
			expectCode:   codes.PermissionDenied,
			expectCalled: false,
		},
		{
			name: "policy evaluation error",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return nil, errors.New("evaluation error")
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config:       []GRPCPolicyOption{},
			expectError:  true,
			expectCode:   codes.Internal,
			expectCalled: false,
		},
		{
			name: "skip method",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						t.Error("should not call evaluate for skipped method")
						return nil, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCSkipMethods([]string{"/test.Service/TestMethod"}),
			},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "optional mode with denial",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return &opa.Result{
							Allowed:    false,
							DecisionID: "test-decision-3",
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCOptional(true),
			},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "optional mode with error",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return nil, errors.New("evaluation error")
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCOptional(true),
			},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "custom decision path",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						assert.Equal(t, "custom/allow", path)
						return &opa.Result{Allowed: true}, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCDecisionPath("custom/allow"),
			},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "dynamic decision path from context",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						assert.Equal(t, "dynamic/allow", path)
						return &opa.Result{Allowed: true}, nil
					},
				}
			},
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "policy_path", "dynamic/allow")
				return ctx
			},
			config:       []GRPCPolicyOption{},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "resource info from context",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						assert.Equal(t, "document", input.Resource.Type)
						assert.Equal(t, "doc123", input.Resource.ID)
						return &opa.Result{Allowed: true}, nil
					},
				}
			},
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "resource_type", "document")
				ctx = context.WithValue(ctx, "resource_id", "doc123")
				return ctx
			},
			config:       []GRPCPolicyOption{},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "custom error handler",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return nil, errors.New("evaluation error")
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCErrorHandler(func(ctx context.Context, err error) error {
					return status.Error(codes.Unavailable, "custom error")
				}),
			},
			expectError:  true,
			expectCode:   codes.Unavailable,
			expectCalled: false,
		},
		{
			name: "custom denied handler",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return &opa.Result{
							Allowed:    false,
							DecisionID: "test-decision-4",
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCDeniedHandler(func(ctx context.Context, result *opa.Result) error {
					return status.Error(codes.Unauthenticated, "custom denied")
				}),
			},
			expectError:  true,
			expectCode:   codes.Unauthenticated,
			expectCalled: false,
		},
		{
			name: "timeout",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						// 检查上下文是否有超时
						_, hasDeadline := ctx.Deadline()
						assert.True(t, hasDeadline, "context should have deadline")
						return &opa.Result{Allowed: true}, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCTimeout(1 * time.Second),
			},
			expectError:  false,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ctx := tt.setupContext()
			handler := &mockPolicyUnaryHandler{}

			interceptor := UnaryPolicyInterceptor(client, tt.config...)

			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			resp, err := interceptor(ctx, "test-request", info, handler.handle)

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, "success", resp)
			}

			assert.Equal(t, tt.expectCalled, handler.called)

			if tt.validateResult != nil {
				tt.validateResult(t, resp, err)
			}
		})
	}
}

func TestStreamPolicyInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		setupClient    func() *mockPolicyOPAClient
		setupContext   func() context.Context
		config         []GRPCPolicyOption
		expectError    bool
		expectCode     codes.Code
		expectCalled   bool
		validateResult func(t *testing.T, err error)
	}{
		{
			name: "allowed by policy",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return &opa.Result{
							Allowed:    true,
							DecisionID: "test-decision-1",
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "user_id", "user123")
				return ctx
			},
			config:       []GRPCPolicyOption{},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "denied by policy",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return &opa.Result{
							Allowed:    false,
							DecisionID: "test-decision-2",
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config:       []GRPCPolicyOption{},
			expectError:  true,
			expectCode:   codes.PermissionDenied,
			expectCalled: false,
		},
		{
			name: "policy evaluation error",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return nil, errors.New("evaluation error")
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config:       []GRPCPolicyOption{},
			expectError:  true,
			expectCode:   codes.Internal,
			expectCalled: false,
		},
		{
			name: "skip method",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						t.Error("should not call evaluate for skipped method")
						return nil, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCSkipMethods([]string{"/test.Service/TestMethod"}),
			},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "optional mode with denial",
			setupClient: func() *mockPolicyOPAClient {
				return &mockPolicyOPAClient{
					evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
						return &opa.Result{
							Allowed:    false,
							DecisionID: "test-decision-3",
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				return context.Background()
			},
			config: []GRPCPolicyOption{
				WithGRPCOptional(true),
			},
			expectError:  false,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ctx := tt.setupContext()
			handler := &mockPolicyStreamHandler{}
			stream := &mockPolicyServerStream{ctx: ctx}

			interceptor := StreamPolicyInterceptor(client, tt.config...)

			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			err := interceptor(nil, stream, info, handler.handle)

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectCalled, handler.called)

			if tt.validateResult != nil {
				tt.validateResult(t, err)
			}
		})
	}
}

func TestGRPCResourcePolicy(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		expectError  bool
		expectCode   codes.Code
		expectCalled bool
		validateCtx  func(t *testing.T, ctx context.Context)
	}{
		{
			name:         "valid resource",
			resourceType: "document",
			resourceID:   "doc123",
			expectError:  false,
			expectCalled: true,
			validateCtx: func(t *testing.T, ctx context.Context) {
				assert.Equal(t, "document", ctx.Value("resource_type"))
				assert.Equal(t, "doc123", ctx.Value("resource_id"))
			},
		},
		{
			name:         "missing resource ID",
			resourceType: "document",
			resourceID:   "",
			expectError:  true,
			expectCode:   codes.InvalidArgument,
			expectCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockPolicyUnaryHandler{}
			var capturedCtx context.Context

			interceptor := GRPCResourcePolicy(tt.resourceType, func(ctx context.Context) string {
				return tt.resourceID
			})

			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			ctx := context.Background()
			resp, err := interceptor(ctx, "test-request", info, func(ctx context.Context, req interface{}) (interface{}, error) {
				capturedCtx = ctx
				return handler.handle(ctx, req)
			})

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, "success", resp)
			}

			assert.Equal(t, tt.expectCalled, handler.called)

			if tt.validateCtx != nil && capturedCtx != nil {
				tt.validateCtx(t, capturedCtx)
			}
		})
	}
}

func TestGRPCDynamicPolicyPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectError  bool
		expectCode   codes.Code
		expectCalled bool
		validateCtx  func(t *testing.T, ctx context.Context)
	}{
		{
			name:         "valid path",
			path:         "custom/allow",
			expectError:  false,
			expectCalled: true,
			validateCtx: func(t *testing.T, ctx context.Context) {
				assert.Equal(t, "custom/allow", ctx.Value("policy_path"))
			},
		},
		{
			name:         "empty path",
			path:         "",
			expectError:  true,
			expectCode:   codes.Internal,
			expectCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockPolicyUnaryHandler{}
			var capturedCtx context.Context

			interceptor := GRPCDynamicPolicyPath(func(ctx context.Context) string {
				return tt.path
			})

			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			ctx := context.Background()
			resp, err := interceptor(ctx, "test-request", info, func(ctx context.Context, req interface{}) (interface{}, error) {
				capturedCtx = ctx
				return handler.handle(ctx, req)
			})

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, "success", resp)
			}

			assert.Equal(t, tt.expectCalled, handler.called)

			if tt.validateCtx != nil && capturedCtx != nil {
				tt.validateCtx(t, capturedCtx)
			}
		})
	}
}

func TestGRPCStreamResourcePolicy(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceID   string
		expectError  bool
		expectCode   codes.Code
		expectCalled bool
		validateCtx  func(t *testing.T, ctx context.Context)
	}{
		{
			name:         "valid resource",
			resourceType: "document",
			resourceID:   "doc123",
			expectError:  false,
			expectCalled: true,
			validateCtx: func(t *testing.T, ctx context.Context) {
				assert.Equal(t, "document", ctx.Value("resource_type"))
				assert.Equal(t, "doc123", ctx.Value("resource_id"))
			},
		},
		{
			name:         "missing resource ID",
			resourceType: "document",
			resourceID:   "",
			expectError:  true,
			expectCode:   codes.InvalidArgument,
			expectCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockPolicyStreamHandler{}
			var capturedCtx context.Context

			interceptor := GRPCStreamResourcePolicy(tt.resourceType, func(ctx context.Context) string {
				return tt.resourceID
			})

			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			ctx := context.Background()
			stream := &mockPolicyServerStream{ctx: ctx}

			err := interceptor(nil, stream, info, func(srv interface{}, ss grpc.ServerStream) error {
				capturedCtx = ss.Context()
				return handler.handle(srv, ss)
			})

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectCalled, handler.called)

			if tt.validateCtx != nil && capturedCtx != nil {
				tt.validateCtx(t, capturedCtx)
			}
		})
	}
}

func TestGRPCStreamDynamicPolicyPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectError  bool
		expectCode   codes.Code
		expectCalled bool
		validateCtx  func(t *testing.T, ctx context.Context)
	}{
		{
			name:         "valid path",
			path:         "custom/allow",
			expectError:  false,
			expectCalled: true,
			validateCtx: func(t *testing.T, ctx context.Context) {
				assert.Equal(t, "custom/allow", ctx.Value("policy_path"))
			},
		},
		{
			name:         "empty path",
			path:         "",
			expectError:  true,
			expectCode:   codes.Internal,
			expectCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockPolicyStreamHandler{}
			var capturedCtx context.Context

			interceptor := GRPCStreamDynamicPolicyPath(func(ctx context.Context) string {
				return tt.path
			})

			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			ctx := context.Background()
			stream := &mockPolicyServerStream{ctx: ctx}

			err := interceptor(nil, stream, info, func(srv interface{}, ss grpc.ServerStream) error {
				capturedCtx = ss.Context()
				return handler.handle(srv, ss)
			})

			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectCalled, handler.called)

			if tt.validateCtx != nil && capturedCtx != nil {
				tt.validateCtx(t, capturedCtx)
			}
		})
	}
}

func TestGRPCAuditLog(t *testing.T) {
	// 测试审计日志功能
	var capturedLog *GRPCAuditLog

	client := &mockPolicyOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
			return &opa.Result{
				Allowed:    true,
				DecisionID: "test-decision-audit",
			}, nil
		},
	}

	logger, _ := zap.NewDevelopment()

	config := []GRPCPolicyOption{
		WithGRPCLogger(logger),
		WithGRPCAuditLog(func(log *GRPCAuditLog) {
			capturedLog = log
		}),
	}

	handler := &mockPolicyUnaryHandler{}
	interceptor := UnaryPolicyInterceptor(client, config...)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "user123")

	_, err := interceptor(ctx, "test-request", info, handler.handle)

	require.NoError(t, err)
	require.NotNil(t, capturedLog)

	assert.True(t, capturedLog.Allowed)
	assert.Equal(t, "test-decision-audit", capturedLog.DecisionID)
	assert.Equal(t, "authz/allow", capturedLog.DecisionPath)
	assert.Equal(t, "/test.Service/TestMethod", capturedLog.Request.Method)
	assert.NotNil(t, capturedLog.Input)
	assert.NotNil(t, capturedLog.Result)
	assert.GreaterOrEqual(t, capturedLog.Duration, int64(0))
}

func TestGRPCCustomInputBuilder(t *testing.T) {
	customBuilderCalled := false

	client := &mockPolicyOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
			// 验证自定义输入构建器的结果
			assert.Equal(t, "custom-user", input.User.ID)
			return &opa.Result{Allowed: true}, nil
		},
	}

	config := []GRPCPolicyOption{
		WithGRPCInputBuilder(func(ctx context.Context, fullMethod string, request interface{}) (*opa.PolicyInput, error) {
			customBuilderCalled = true
			return &opa.PolicyInput{
				Request: opa.RequestInfo{
					Method:   fullMethod,
					Protocol: "grpc",
				},
				User: opa.UserInfo{
					ID: "custom-user",
				},
				Custom: map[string]interface{}{
					"request": request,
				},
			}, nil
		}),
	}

	handler := &mockPolicyUnaryHandler{}
	interceptor := UnaryPolicyInterceptor(client, config...)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	ctx := context.Background()
	_, err := interceptor(ctx, "test-request", info, handler.handle)

	require.NoError(t, err)
	assert.True(t, customBuilderCalled)
	assert.True(t, handler.called)
}

func TestGRPCInputBuilderError(t *testing.T) {
	client := &mockPolicyOPAClient{}

	config := []GRPCPolicyOption{
		WithGRPCInputBuilder(func(ctx context.Context, fullMethod string, request interface{}) (*opa.PolicyInput, error) {
			return nil, errors.New("input builder error")
		}),
	}

	handler := &mockPolicyUnaryHandler{}
	interceptor := UnaryPolicyInterceptor(client, config...)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	ctx := context.Background()
	_, err := interceptor(ctx, "test-request", info, handler.handle)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.False(t, handler.called)
}

func TestGRPCWildcardSkipMethod(t *testing.T) {
	client := &mockPolicyOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
			t.Error("should not call evaluate for wildcard skipped method")
			return nil, nil
		},
	}

	config := []GRPCPolicyOption{
		WithGRPCSkipMethods([]string{"/grpc.health.v1.Health/*"}),
	}

	handler := &mockPolicyUnaryHandler{}
	interceptor := UnaryPolicyInterceptor(client, config...)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	ctx := context.Background()
	_, err := interceptor(ctx, "test-request", info, handler.handle)

	require.NoError(t, err)
	assert.True(t, handler.called)
}

func TestGRPCRequestPayloadInInputBuilder(t *testing.T) {
	// 测试请求负载是否正确传递给 InputBuilder
	type TestRequest struct {
		UserID string
		Action string
	}

	testReq := &TestRequest{
		UserID: "user123",
		Action: "update",
	}

	var capturedRequest interface{}

	client := &mockPolicyOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
			// 验证请求在 custom data 中
			if req, ok := input.Custom["request"]; ok {
				if testReq, ok := req.(*TestRequest); ok {
					assert.Equal(t, "user123", testReq.UserID)
					assert.Equal(t, "update", testReq.Action)
				}
			}
			return &opa.Result{Allowed: true}, nil
		},
	}

	config := []GRPCPolicyOption{
		WithGRPCInputBuilder(func(ctx context.Context, fullMethod string, request interface{}) (*opa.PolicyInput, error) {
			capturedRequest = request
			// 构建包含请求数据的策略输入
			return &opa.PolicyInput{
				Request: opa.RequestInfo{
					Method:   fullMethod,
					Protocol: "grpc",
				},
				Custom: map[string]interface{}{
					"request": request,
				},
			}, nil
		}),
	}

	handler := &mockPolicyUnaryHandler{}
	interceptor := UnaryPolicyInterceptor(client, config...)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	ctx := context.Background()
	_, err := interceptor(ctx, testReq, info, handler.handle)

	require.NoError(t, err)
	assert.True(t, handler.called)
	assert.NotNil(t, capturedRequest)

	// 验证传递的是正确的请求对象
	if req, ok := capturedRequest.(*TestRequest); ok {
		assert.Equal(t, "user123", req.UserID)
		assert.Equal(t, "update", req.Action)
	} else {
		t.Error("captured request is not of expected type")
	}
}

func TestGRPCDefaultInputBuilderWithRequest(t *testing.T) {
	// 测试默认 InputBuilder 是否正确处理请求负载
	type TestRequest struct {
		ResourceID string
	}

	testReq := &TestRequest{
		ResourceID: "resource123",
	}

	client := &mockPolicyOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input *opa.PolicyInput) (*opa.Result, error) {
			// 验证默认 builder 将请求放入 custom data
			assert.NotNil(t, input.Custom)
			if req, ok := input.Custom["request"]; ok {
				assert.NotNil(t, req)
			} else {
				t.Error("request should be in custom data")
			}
			return &opa.Result{Allowed: true}, nil
		},
	}

	handler := &mockPolicyUnaryHandler{}
	interceptor := UnaryPolicyInterceptor(client) // 使用默认 InputBuilder

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	ctx := context.Background()
	_, err := interceptor(ctx, testReq, info, handler.handle)

	require.NoError(t, err)
	assert.True(t, handler.called)
}
