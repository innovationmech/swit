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
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/security/opa"
	"github.com/open-policy-agent/opa/rego"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockOPAClient 模拟 OPA 客户端
type mockOPAClient struct {
	evaluateFunc       func(ctx context.Context, path string, input interface{}) (*opa.Result, error)
	queryFunc          func(ctx context.Context, query string, input interface{}) (rego.ResultSet, error)
	partialEvaluate    func(ctx context.Context, query string, input interface{}) (*opa.PartialResult, error)
	loadPolicyFunc     func(ctx context.Context, name string, policy string) error
	loadPolicyFileFunc func(ctx context.Context, path string) error
	loadDataFunc       func(ctx context.Context, path string, data interface{}) error
	removePolicyFunc   func(ctx context.Context, name string) error
	closeFunc          func(ctx context.Context) error
	isEmbedded         bool
}

func (m *mockOPAClient) Evaluate(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
	if m.evaluateFunc != nil {
		return m.evaluateFunc(ctx, path, input)
	}
	return &opa.Result{Allowed: true}, nil
}

func (m *mockOPAClient) Query(ctx context.Context, query string, input interface{}) (rego.ResultSet, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, input)
	}
	return nil, nil
}

func (m *mockOPAClient) PartialEvaluate(ctx context.Context, query string, input interface{}) (*opa.PartialResult, error) {
	if m.partialEvaluate != nil {
		return m.partialEvaluate(ctx, query, input)
	}
	return nil, nil
}

func (m *mockOPAClient) LoadPolicy(ctx context.Context, name string, policy string) error {
	if m.loadPolicyFunc != nil {
		return m.loadPolicyFunc(ctx, name, policy)
	}
	return nil
}

func (m *mockOPAClient) LoadPolicyFromFile(ctx context.Context, path string) error {
	if m.loadPolicyFileFunc != nil {
		return m.loadPolicyFileFunc(ctx, path)
	}
	return nil
}

func (m *mockOPAClient) LoadData(ctx context.Context, path string, data interface{}) error {
	if m.loadDataFunc != nil {
		return m.loadDataFunc(ctx, path, data)
	}
	return nil
}

func (m *mockOPAClient) RemovePolicy(ctx context.Context, name string) error {
	if m.removePolicyFunc != nil {
		return m.removePolicyFunc(ctx, name)
	}
	return nil
}

func (m *mockOPAClient) Close(ctx context.Context) error {
	if m.closeFunc != nil {
		return m.closeFunc(ctx)
	}
	return nil
}

func (m *mockOPAClient) IsEmbedded() bool {
	return m.isEmbedded
}

func TestOPAMiddleware_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return &opa.Result{
				Allowed:    true,
				DecisionID: "test-decision-id",
			}, nil
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestOPAMiddleware_Denied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return &opa.Result{
				Allowed:    false,
				DecisionID: "test-decision-id",
			}, nil
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "access denied")
}

func TestOPAMiddleware_EvaluationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return nil, errors.New("evaluation failed")
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "policy evaluation failed")
}

func TestOPAMiddleware_WithWhiteList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			// 不应该被调用
			t.Fatal("evaluate should not be called for whitelisted path")
			return nil, nil
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithWhiteList([]string{"/health", "/metrics"})))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 测试白名单路径
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestOPAMiddleware_WithCustomDecisionPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedPath string
	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			capturedPath = path
			return &opa.Result{Allowed: true}, nil
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithDecisionPath("custom/policy/allow")))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "custom/policy/allow", capturedPath)
}

func TestOPAMiddleware_WithCustomInputBuilder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedInput *opa.PolicyInput
	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			capturedInput = input.(*opa.PolicyInput)
			return &opa.Result{Allowed: true}, nil
		},
	}

	customBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		input := opa.NewPolicyInputBuilder().
			FromHTTPRequest(c).
			WithUserID("custom-user-123").
			WithCustomData("tenant_id", "tenant-456").
			Build()
		return input, nil
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithInputBuilder(customBuilder)))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedInput)
	assert.Equal(t, "custom-user-123", capturedInput.User.ID)
	assert.Equal(t, "tenant-456", capturedInput.Custom["tenant_id"])
}

func TestOPAMiddleware_WithCustomErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return nil, errors.New("evaluation failed")
		},
	}

	customErrorCalled := false
	customErrorHandler := func(c *gin.Context, err error) {
		customErrorCalled = true
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error":   "custom error message",
			"details": err.Error(),
		})
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithErrorHandler(customErrorHandler)))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, customErrorCalled)
	assert.Contains(t, w.Body.String(), "custom error message")
}

func TestOPAMiddleware_WithCustomDeniedHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return &opa.Result{
				Allowed:    false,
				DecisionID: "test-decision-id",
			}, nil
		},
	}

	customDeniedCalled := false
	customDeniedHandler := func(c *gin.Context, result *opa.Result) {
		customDeniedCalled = true
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error":       "custom denied message",
			"decision_id": result.DecisionID,
		})
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithDeniedHandler(customDeniedHandler)))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, customDeniedCalled)
	assert.Contains(t, w.Body.String(), "custom denied message")
}

func TestOPAMiddleware_WithAuditLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return &opa.Result{
				Allowed:    true,
				DecisionID: "test-decision-id",
			}, nil
		},
	}

	var auditLogCaptured *AuditLog
	auditHandler := func(log *AuditLog) {
		auditLogCaptured = log
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithAuditLog(auditHandler)))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, auditLogCaptured)
	assert.True(t, auditLogCaptured.Allowed)
	assert.Equal(t, "test-decision-id", auditLogCaptured.DecisionID)
	assert.Equal(t, "authz/allow", auditLogCaptured.DecisionPath)
	assert.Equal(t, "/api/resource", auditLogCaptured.Request.Path)
}

func TestOPAMiddleware_WithTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			// 模拟长时间评估
			select {
			case <-time.After(200 * time.Millisecond):
				return &opa.Result{Allowed: true}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithTimeout(100*time.Millisecond)))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOPAMiddleware_WithLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return &opa.Result{Allowed: true}, nil
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithLogger(logger)))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResourcePolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		resourceType   string
		resourceID     string
		expectedStatus int
	}{
		{
			name:           "valid_resource",
			resourceType:   "document",
			resourceID:     "doc-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty_resource_id",
			resourceType:   "document",
			resourceID:     "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ResourcePolicy(tt.resourceType, func(c *gin.Context) string {
				return tt.resourceID
			}))
			router.GET("/api/resource", func(c *gin.Context) {
				resourceType, _ := c.Get("resource_type")
				resourceID, _ := c.Get("resource_id")
				c.JSON(http.StatusOK, gin.H{
					"resource_type": resourceType,
					"resource_id":   resourceID,
				})
			})

			req, _ := http.NewRequest("GET", "/api/resource", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestResourcePolicy_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedInput *opa.PolicyInput
	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			capturedInput = input.(*opa.PolicyInput)
			return &opa.Result{Allowed: true}, nil
		},
	}

	router := gin.New()
	// 先应用 ResourcePolicy，然后应用 OPAMiddleware
	router.Use(ResourcePolicy("document", func(c *gin.Context) string {
		return c.Param("id")
	}))
	router.Use(OPAMiddleware(mockClient))
	router.GET("/api/documents/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/documents/doc-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedInput)
	assert.Equal(t, "document", capturedInput.Resource.Type)
	assert.Equal(t, "doc-123", capturedInput.Resource.ID)
}

func TestDynamicPolicyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		policyPath     string
		expectedStatus int
	}{
		{
			name:           "valid_policy_path",
			policyPath:     "custom/allow",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty_policy_path",
			policyPath:     "",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(DynamicPolicyPath(func(c *gin.Context) string {
				return tt.policyPath
			}))
			router.GET("/api/resource", func(c *gin.Context) {
				policyPath, _ := c.Get("policy_path")
				c.JSON(http.StatusOK, gin.H{
					"policy_path": policyPath,
				})
			})

			req, _ := http.NewRequest("GET", "/api/resource", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestWithResourceInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(WithResourceInfo("document", "doc-123"))
	router.GET("/api/resource", func(c *gin.Context) {
		resourceType, _ := c.Get("resource_type")
		resourceID, _ := c.Get("resource_id")
		c.JSON(http.StatusOK, gin.H{
			"resource_type": resourceType,
			"resource_id":   resourceID,
		})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "document")
	assert.Contains(t, w.Body.String(), "doc-123")
}

func TestWithCustomAttribute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(WithCustomAttribute("tenant_id", func(c *gin.Context) interface{} {
		return "tenant-456"
	}))
	router.GET("/api/resource", func(c *gin.Context) {
		tenantID, _ := c.Get("custom_tenant_id")
		c.JSON(http.StatusOK, gin.H{
			"tenant_id": tenantID,
		})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "tenant-456")
}

func TestOPAMiddleware_UserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedInput *opa.PolicyInput
	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			capturedInput = input.(*opa.PolicyInput)
			return &opa.Result{Allowed: true}, nil
		},
	}

	router := gin.New()
	// 模拟认证中间件设置用户信息
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("username", "testuser")
		c.Set("roles", []string{"admin", "user"})
		c.Next()
	})
	router.Use(OPAMiddleware(mockClient))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedInput)
	assert.Equal(t, "user-123", capturedInput.User.ID)
	assert.Equal(t, "testuser", capturedInput.User.Username)
	assert.Equal(t, []string{"admin", "user"}, capturedInput.User.Roles)
}

func TestOPAMiddleware_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var evaluateCount int32
	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			atomic.AddInt32(&evaluateCount, 1)
			time.Sleep(10 * time.Millisecond) // 模拟评估延迟
			return &opa.Result{Allowed: true}, nil
		},
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 并发发送多个请求
	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/api/resource", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < numRequests; i++ {
		<-done
	}

	assert.Equal(t, int32(numRequests), atomic.LoadInt32(&evaluateCount))
}

func TestAuditLog_AllFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var auditLogCaptured *AuditLog
	mockClient := &mockOPAClient{
		evaluateFunc: func(ctx context.Context, path string, input interface{}) (*opa.Result, error) {
			return &opa.Result{
				Allowed:    false,
				DecisionID: "test-decision-id-123",
			}, nil
		},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-456")
		c.Next()
	})
	router.Use(OPAMiddleware(mockClient, WithAuditLog(func(log *AuditLog) {
		auditLogCaptured = log
	})))
	router.GET("/api/sensitive", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/sensitive", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotNil(t, auditLogCaptured)
	assert.False(t, auditLogCaptured.Allowed)
	assert.Equal(t, "test-decision-id-123", auditLogCaptured.DecisionID)
	assert.Equal(t, "authz/allow", auditLogCaptured.DecisionPath)
	assert.Equal(t, "GET", auditLogCaptured.Request.Method)
	assert.Equal(t, "/api/sensitive", auditLogCaptured.Request.Path)
	assert.Equal(t, "TestAgent/1.0", auditLogCaptured.Request.UserAgent)
	assert.NotNil(t, auditLogCaptured.Input)
	assert.Equal(t, "user-456", auditLogCaptured.Input.User.ID)
	assert.GreaterOrEqual(t, auditLogCaptured.Duration, int64(0))
}

func TestOPAMiddleware_InputBuilderError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := &mockOPAClient{}

	customBuilder := func(c *gin.Context) (*opa.PolicyInput, error) {
		return nil, errors.New("failed to build input")
	}

	router := gin.New()
	router.Use(OPAMiddleware(mockClient, WithInputBuilder(customBuilder)))
	router.GET("/api/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/resource", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDefaultAuditLogHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	handler := defaultAuditLogHandler(logger)

	auditLog := &AuditLog{
		Timestamp:    time.Now(),
		DecisionID:   "test-id",
		Allowed:      true,
		DecisionPath: "authz/allow",
		Duration:     100,
	}

	// 测试不应该panic
	assert.NotPanics(t, func() {
		handler(auditLog)
	})
}

func TestIsInWhiteList(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		whiteList []string
		expected  bool
	}{
		{
			name:      "path_in_whitelist",
			path:      "/health",
			whiteList: []string{"/health", "/metrics"},
			expected:  true,
		},
		{
			name:      "path_not_in_whitelist",
			path:      "/api/resource",
			whiteList: []string{"/health", "/metrics"},
			expected:  false,
		},
		{
			name:      "empty_whitelist",
			path:      "/health",
			whiteList: []string{},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInWhiteList(tt.path, tt.whiteList)
			assert.Equal(t, tt.expected, result)
		})
	}
}
