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

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/security/opa"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDocumentService(t *testing.T) {
	logger := zap.NewNop()
	service := NewDocumentService(logger)

	t.Run("初始化文档存在", func(t *testing.T) {
		if len(service.documents) != 3 {
			t.Errorf("期望 3 个文档, 得到 %d", len(service.documents))
		}

		expectedDocs := []string{"doc-1", "doc-2", "doc-3"}
		for _, id := range expectedDocs {
			if _, exists := service.documents[id]; !exists {
				t.Errorf("文档 %s 不存在", id)
			}
		}
	})

	t.Run("文档所有者正确", func(t *testing.T) {
		tests := []struct {
			docID string
			owner string
		}{
			{"doc-1", "system"},
			{"doc-2", "alice"},
			{"doc-3", "bob"},
		}

		for _, tt := range tests {
			doc := service.documents[tt.docID]
			if doc.Owner != tt.owner {
				t.Errorf("文档 %s 所有者应为 %s, 得到 %s", tt.docID, tt.owner, doc.Owner)
			}
		}
	})
}

func TestCreateOPAClient(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("嵌入式模式", func(t *testing.T) {
		// 获取策略目录的绝对路径
		testPolicyDir, err := filepath.Abs("./policies")
		if err != nil {
			t.Fatalf("无法解析策略目录: %v", err)
		}

		mode := "embedded"
		oldMode := *opaMode
		oldPolicyDir := *policyDir
		oldPolicyType := *policyType

		*opaMode = mode
		*policyDir = testPolicyDir
		*policyType = "rbac"

		defer func() {
			*opaMode = oldMode
			*policyDir = oldPolicyDir
			*policyType = oldPolicyType
		}()

		client, err := createOPAClient(ctx, logger)
		if err != nil {
			t.Fatalf("创建嵌入式 OPA 客户端失败: %v", err)
		}
		defer client.Close(ctx)

		if client == nil {
			t.Error("客户端不应为 nil")
		}
	})
}

func TestHTTPEndpoints(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	// 创建嵌入式 OPA 客户端
	testPolicyDir, err := filepath.Abs("./policies")
	if err != nil {
		t.Fatalf("无法解析策略目录: %v", err)
	}

	mode := "embedded"
	oldMode := *opaMode
	oldPolicyDir := *policyDir
	oldPolicyType := *policyType

	*opaMode = mode
	*policyDir = testPolicyDir
	*policyType = "rbac"

	defer func() {
		*opaMode = oldMode
		*policyDir = oldPolicyDir
		*policyType = oldPolicyType
	}()

	opaClient, err := createOPAClient(ctx, logger)
	if err != nil {
		t.Fatalf("创建 OPA 客户端失败: %v", err)
	}
	defer opaClient.Close(ctx)

	// 创建文档服务
	docService := NewDocumentService(logger)

	// 设置 HTTP 服务器
	router := setupHTTPServer(opaClient, docService, logger).Handler.(*gin.Engine)

	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "健康检查",
			method:         "GET",
			path:           "/api/v1/health",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"healthy"`,
		},
		{
			name:   "管理员访问文档列表",
			method: "GET",
			path:   "/api/v1/documents",
			headers: map[string]string{
				"X-User":  "alice",
				"X-Roles": "admin",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "编辑者访问文档列表",
			method: "GET",
			path:   "/api/v1/documents",
			headers: map[string]string{
				"X-User":  "bob",
				"X-Roles": "editor",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "查看者访问文档列表",
			method: "GET",
			path:   "/api/v1/documents",
			headers: map[string]string{
				"X-User":  "charlie",
				"X-Roles": "viewer",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "匿名用户访问文档列表应被拒绝",
			method:         "GET",
			path:           "/api/v1/documents",
			headers:        map[string]string{},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "编辑者删除文档应被拒绝",
			method: "DELETE",
			path:   "/api/v1/documents/doc-1",
			headers: map[string]string{
				"X-User":  "bob",
				"X-Roles": "editor",
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "查看者创建文档应被拒绝",
			method: "POST",
			path:   "/api/v1/documents",
			headers: map[string]string{
				"X-User":       "charlie",
				"X-Roles":      "viewer",
				"Content-Type": "application/json",
			},
			body:           `{"id":"doc-test","title":"Test","content":"Test"}`,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "管理员创建文档",
			method: "POST",
			path:   "/api/v1/documents",
			headers: map[string]string{
				"X-User":       "alice",
				"X-Roles":      "admin",
				"Content-Type": "application/json",
			},
			body:           `{"id":"doc-admin","title":"Admin Doc","content":"Content"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "管理员删除文档",
			method: "DELETE",
			path:   "/api/v1/documents/doc-admin",
			headers: map[string]string{
				"X-User":  "alice",
				"X-Roles": "admin",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("期望状态码 %d, 得到 %d, 响应: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("响应不包含期望内容: %s, 实际响应: %s",
					tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestRBACPolicy(t *testing.T) {
	ctx := context.Background()

	testPolicyDir, err := filepath.Abs("./policies")
	if err != nil {
		t.Fatalf("无法解析策略目录: %v", err)
	}

	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: testPolicyDir,
		},
		DefaultDecisionPath: "rbac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("创建 OPA 客户端失败: %v", err)
	}
	defer client.Close(ctx)

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "管理员允许所有操作",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "alice",
					"roles":    []string{"admin"},
				},
				"request": map[string]interface{}{
					"method": "DELETE",
					"path":   "/api/v1/documents",
				},
				"resource": map[string]interface{}{
					"path": "/api/v1/documents",
					"type": "document",
				},
			},
			expected: true,
		},
		{
			name: "编辑者可以读取",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "bob",
					"roles":    []string{"editor"},
				},
				"request": map[string]interface{}{
					"method": "GET",
					"path":   "/api/v1/documents",
				},
				"resource": map[string]interface{}{
					"path": "/api/v1/documents",
					"type": "document",
				},
			},
			expected: true,
		},
		{
			name: "编辑者不能删除",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "bob",
					"roles":    []string{"editor"},
				},
				"request": map[string]interface{}{
					"method": "DELETE",
					"path":   "/api/v1/documents",
				},
				"resource": map[string]interface{}{
					"path": "/api/v1/documents",
					"type": "document",
				},
			},
			expected: false,
		},
		{
			name: "查看者只能读取",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "charlie",
					"roles":    []string{"viewer"},
				},
				"request": map[string]interface{}{
					"method": "GET",
					"path":   "/api/v1/documents",
				},
				"resource": map[string]interface{}{
					"path": "/api/v1/documents",
					"type": "document",
				},
			},
			expected: true,
		},
		{
			name: "查看者不能写入",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "charlie",
					"roles":    []string{"viewer"},
				},
				"request": map[string]interface{}{
					"method": "POST",
					"path":   "/api/v1/documents",
				},
				"resource": map[string]interface{}{
					"path": "/api/v1/documents",
					"type": "document",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Evaluate(ctx, "", tt.input)
			if err != nil {
				t.Fatalf("策略评估失败: %v", err)
			}

			if result.Allowed != tt.expected {
				inputJSON, _ := json.MarshalIndent(tt.input, "", "  ")
				t.Errorf("期望 %v, 得到 %v\n输入:\n%s\n决策: %+v",
					tt.expected, result.Allowed, string(inputJSON), result.Decision)
			}
		})
	}
}

func TestABACPolicy(t *testing.T) {
	ctx := context.Background()

	testPolicyDir, err := filepath.Abs("./policies")
	if err != nil {
		t.Fatalf("无法解析策略目录: %v", err)
	}

	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: testPolicyDir,
		},
		DefaultDecisionPath: "abac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("创建 OPA 客户端失败: %v", err)
	}
	defer client.Close(ctx)

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "管理员绕过所有检查",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "alice",
					"roles":    []string{"admin"},
				},
				"request": map[string]interface{}{
					"method":    "DELETE",
					"path":      "/api/v1/documents",
					"client_ip": "10.0.0.1",
				},
				"resource": map[string]interface{}{
					"path": "/api/v1/documents",
					"type": "document",
				},
			},
			expected: true,
		},
		{
			name: "编辑者在允许的IP可以访问",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "bob",
					"roles":    []string{"editor"},
				},
				"request": map[string]interface{}{
					"method":    "GET",
					"path":      "/api/v1/documents",
					"client_ip": "192.168.1.100",
				},
				"resource": map[string]interface{}{
					"path": "/api/v1/documents",
					"type": "document",
				},
			},
			expected: true,
		},
		{
			name: "资源所有者可以访问自己的文档",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "bob",
					"roles":    []string{"editor"},
				},
				"request": map[string]interface{}{
					"method":    "PUT",
					"path":      "/api/v1/documents/doc-2",
					"client_ip": "192.168.1.100",
				},
				"resource": map[string]interface{}{
					"path":  "/api/v1/documents/doc-2",
					"type":  "document",
					"owner": "bob",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Evaluate(ctx, "", tt.input)
			if err != nil {
				t.Fatalf("策略评估失败: %v", err)
			}

			if result.Allowed != tt.expected {
				inputJSON, _ := json.MarshalIndent(tt.input, "", "  ")
				t.Errorf("期望 %v, 得到 %v\n输入:\n%s\n决策: %+v",
					tt.expected, result.Allowed, string(inputJSON), result.Decision)
			}
		})
	}
}

func TestSanitizeLogValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "移除换行符",
			input:    "hello\nworld",
			expected: "helloworld",
		},
		{
			name:     "移除回车符",
			input:    "hello\rworld",
			expected: "helloworld",
		},
		{
			name:     "替换制表符为空格",
			input:    "hello\tworld",
			expected: "hello world",
		},
		{
			name:     "处理多个特殊字符",
			input:    "hello\n\r\tworld",
			expected: "hello world",
		},
		{
			name:     "正常字符串不变",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLogValue(tt.input)
			if result != tt.expected {
				t.Errorf("期望 %q, 得到 %q", tt.expected, result)
			}
		})
	}
}
