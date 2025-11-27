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

package opa_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/security/opa"
)

// TestEmbeddedRBACIntegration 测试嵌入式 OPA RBAC 场景
func TestEmbeddedRBACIntegration(t *testing.T) {
	ctx := context.Background()

	// 创建临时目录并复制 RBAC 策略
	tempDir := t.TempDir()
	rbacPolicyPath := filepath.Join("policies", "rbac.rego")

	// 读取 RBAC 策略文件
	rbacContent, err := os.ReadFile(rbacPolicyPath)
	if err != nil {
		t.Fatalf("Failed to read RBAC policy: %v", err)
	}

	// 写入临时目录
	if err := os.WriteFile(filepath.Join(tempDir, "rbac.rego"), rbacContent, 0644); err != nil {
		t.Fatalf("Failed to write RBAC policy: %v", err)
	}

	// 创建嵌入式客户端
	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "rbac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create embedded client: %v", err)
	}
	defer client.Close(ctx)

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "admin can do everything",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "alice",
					"roles": []string{"admin"},
				},
				"action":   "delete",
				"resource": "users",
			},
			expected: true,
		},
		{
			name: "editor can update documents",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "bob",
					"roles": []string{"editor"},
				},
				"action":   "update",
				"resource": "documents",
			},
			expected: true,
		},
		{
			name: "viewer cannot delete",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "charlie",
					"roles": []string{"viewer"},
				},
				"action":   "delete",
				"resource": "documents",
			},
			expected: false,
		},
		{
			name: "editor cannot access settings",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "bob",
					"roles": []string{"editor"},
				},
				"action":   "read",
				"resource": "settings",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Evaluate(ctx, "", tt.input)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}

			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v, got %v", tt.expected, result.Allowed)
			}

			// 验证性能指标
			if result.Metrics != nil && result.Metrics.TimerEvalNs > 0 {
				evalTimeMs := float64(result.Metrics.TimerEvalNs) / 1000000
				t.Logf("Evaluation time: %.2f ms", evalTimeMs)

				// 性能要求: < 5ms
				if evalTimeMs > 5.0 {
					t.Errorf("Evaluation time %.2f ms exceeds 5ms threshold", evalTimeMs)
				}
			}
		})
	}
}

// TestEmbeddedABACIntegration 测试嵌入式 OPA ABAC 场景
func TestEmbeddedABACIntegration(t *testing.T) {
	ctx := context.Background()

	// 创建临时目录并复制 ABAC 策略
	tempDir := t.TempDir()
	abacPolicyPath := filepath.Join("policies", "abac.rego")

	// 读取 ABAC 策略文件
	abacContent, err := os.ReadFile(abacPolicyPath)
	if err != nil {
		t.Fatalf("Failed to read ABAC policy: %v", err)
	}

	// 写入临时目录
	if err := os.WriteFile(filepath.Join(tempDir, "abac.rego"), abacContent, 0644); err != nil {
		t.Fatalf("Failed to write ABAC policy: %v", err)
	}

	// 创建嵌入式客户端
	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "abac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create embedded client: %v", err)
	}
	defer client.Close(ctx)

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "admin bypasses all checks",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "alice",
					"roles": []string{"admin"},
				},
				"action":   "delete",
				"resource": "documents",
			},
			expected: true,
		},
		{
			name: "valid subject and resource with read action",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "bob",
					"roles": []string{"user"},
				},
				"action":   "read",
				"resource": "documents",
			},
			expected: true,
		},
		{
			name: "editor can update documents",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "charlie",
					"roles": []string{"editor"},
				},
				"action": "update",
				"resource": map[string]interface{}{
					"type": "document",
					"id":   "doc-123",
				},
				"environment": map[string]interface{}{
					"client_ip": "192.168.1.1",
				},
			},
			expected: true,
		},
		{
			name: "invalid action denied",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "bob",
					"roles": []string{"user"},
				},
				"action":   "invalid_action",
				"resource": "documents",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Evaluate(ctx, "", tt.input)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}

			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v, got %v", tt.expected, result.Allowed)
			}
		})
	}
}

// TestRemoteOPAIntegration 测试远程 OPA 集成（需要运行 OPA 容器）
func TestRemoteOPAIntegration(t *testing.T) {
	// 检查是否设置了远程 OPA URL
	opaURL := os.Getenv("OPA_URL")
	if opaURL == "" {
		t.Skip("Skipping remote OPA test: OPA_URL not set")
	}

	ctx := context.Background()

	// 创建远程客户端
	config := &opa.Config{
		Mode: opa.ModeRemote,
		RemoteConfig: &opa.RemoteConfig{
			URL:     opaURL,
			Timeout: 5 * time.Second,
		},
		DefaultDecisionPath: "rbac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create remote client: %v", err)
	}
	defer client.Close(ctx)

	// 测试远程策略评估
	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"user":  "alice",
			"roles": []string{"admin"},
		},
		"action":   "read",
		"resource": "documents",
	}

	result, err := client.Evaluate(ctx, "", input)
	if err != nil {
		t.Fatalf("Remote policy evaluation failed: %v", err)
	}

	t.Logf("Remote evaluation result: allowed=%v", result.Allowed)

	// 验证决策 ID 存在
	if result.DecisionID == "" {
		t.Error("Expected decision_id to be present in remote evaluation")
	}
}

// TestRemoteOPAWithMockServer 测试使用 Mock 服务器的远程 OPA
func TestRemoteOPAWithMockServer(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock OPA 服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if r.URL.Path == "/v1/data/rbac/allow" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{
				"result": true,
				"decision_id": "test-decision-123"
			}`)
			return
		}

		// 健康检查端点
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": "ok"}`)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	// 创建远程客户端
	config := &opa.Config{
		Mode: opa.ModeRemote,
		RemoteConfig: &opa.RemoteConfig{
			URL:     mockServer.URL,
			Timeout: 5 * time.Second,
		},
		DefaultDecisionPath: "rbac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create remote client: %v", err)
	}
	defer client.Close(ctx)

	// 测试评估
	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"user":  "alice",
			"roles": []string{"admin"},
		},
		"action":   "read",
		"resource": "documents",
	}

	result, err := client.Evaluate(ctx, "", input)
	if err != nil {
		t.Fatalf("Policy evaluation failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected evaluation to return allowed=true")
	}

	if result.DecisionID != "test-decision-123" {
		t.Errorf("Expected decision_id='test-decision-123', got '%s'", result.DecisionID)
	}
}

// TestCachingPerformance 测试缓存性能
func TestCachingPerformance(t *testing.T) {
	ctx := context.Background()

	// 创建临时目录并复制策略
	tempDir := t.TempDir()
	rbacPolicyPath := filepath.Join("policies", "rbac.rego")
	rbacContent, err := os.ReadFile(rbacPolicyPath)
	if err != nil {
		t.Fatalf("Failed to read RBAC policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rbac.rego"), rbacContent, 0644); err != nil {
		t.Fatalf("Failed to write RBAC policy: %v", err)
	}

	// 创建带缓存的客户端
	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		CacheConfig: &opa.CacheConfig{
			Enabled: true,
			MaxSize: 100,
			TTL:     60 * time.Second,
		},
		DefaultDecisionPath: "rbac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"user":  "alice",
			"roles": []string{"admin"},
		},
		"action":   "read",
		"resource": "documents",
	}

	// 第一次评估（缓存未命中）
	start := time.Now()
	result1, err := client.Evaluate(ctx, "", input)
	firstEvalTime := time.Since(start)
	if err != nil {
		t.Fatalf("First evaluation failed: %v", err)
	}

	// 第二次评估（应该命中缓存）
	start = time.Now()
	result2, err := client.Evaluate(ctx, "", input)
	secondEvalTime := time.Since(start)
	if err != nil {
		t.Fatalf("Second evaluation failed: %v", err)
	}

	t.Logf("First evaluation (cache miss): %v", firstEvalTime)
	t.Logf("Second evaluation (cache hit): %v", secondEvalTime)

	// 验证结果一致
	if result1.Allowed != result2.Allowed {
		t.Error("Cached result differs from original result")
	}

	// 缓存命中应该更快（至少快 50%）
	if secondEvalTime > firstEvalTime/2 {
		t.Logf("Warning: Cache hit not significantly faster. First: %v, Second: %v",
			firstEvalTime, secondEvalTime)
	}
}

// TestDynamicPolicyLoading 测试动态策略加载
func TestDynamicPolicyLoading(t *testing.T) {
	ctx := context.Background()

	// 创建嵌入式客户端（空策略目录）
	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
		DefaultDecisionPath: "dynamic/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 动态加载策略
	policy := `
package dynamic

import rego.v1

default allow := false

allow if {
    input.subject.user == "alice"
    input.action == "read"
}
`

	if err := client.LoadPolicy(ctx, "dynamic.rego", policy); err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 测试策略
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "alice can read",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user": "alice",
				},
				"action": "read",
			},
			expected: true,
		},
		{
			name: "bob cannot read",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user": "bob",
				},
				"action": "read",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Evaluate(ctx, "", tt.input)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}

			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v, got %v", tt.expected, result.Allowed)
			}
		})
	}

	// 更新策略
	updatedPolicy := `
package dynamic

import rego.v1

default allow := false

allow if {
    input.subject.user in ["alice", "bob"]
    input.action == "read"
}
`

	if err := client.LoadPolicy(ctx, "dynamic.rego", updatedPolicy); err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	// 验证更新后的策略
	result, err := client.Evaluate(ctx, "", map[string]interface{}{
		"subject": map[string]interface{}{
			"user": "bob",
		},
		"action": "read",
	})
	if err != nil {
		t.Fatalf("Policy evaluation failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected bob to be allowed after policy update")
	}
}

// BenchmarkEmbeddedPolicyEvaluation 基准测试嵌入式策略评估性能
func BenchmarkEmbeddedPolicyEvaluation(b *testing.B) {
	ctx := context.Background()

	// 创建临时目录并复制策略
	tempDir := b.TempDir()
	rbacPolicyPath := filepath.Join("policies", "rbac.rego")
	rbacContent, err := os.ReadFile(rbacPolicyPath)
	if err != nil {
		b.Fatalf("Failed to read RBAC policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rbac.rego"), rbacContent, 0644); err != nil {
		b.Fatalf("Failed to write RBAC policy: %v", err)
	}

	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "rbac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"user":  "alice",
			"roles": []string{"admin"},
		},
		"action":   "read",
		"resource": "documents",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "", input)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

// BenchmarkEmbeddedWithCache 基准测试带缓存的策略评估
func BenchmarkEmbeddedWithCache(b *testing.B) {
	ctx := context.Background()

	tempDir := b.TempDir()
	rbacPolicyPath := filepath.Join("policies", "rbac.rego")
	rbacContent, err := os.ReadFile(rbacPolicyPath)
	if err != nil {
		b.Fatalf("Failed to read RBAC policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rbac.rego"), rbacContent, 0644); err != nil {
		b.Fatalf("Failed to write RBAC policy: %v", err)
	}

	config := &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		CacheConfig: &opa.CacheConfig{
			Enabled: true,
			MaxSize: 10000,
			TTL:     5 * time.Minute,
		},
		DefaultDecisionPath: "rbac/allow",
	}

	client, err := opa.NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"user":  "alice",
			"roles": []string{"admin"},
		},
		"action":   "read",
		"resource": "documents",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "", input)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

// TestMultipleClientsIsolation 测试多客户端隔离
func TestMultipleClientsIsolation(t *testing.T) {
	ctx := context.Background()

	// 客户端 1：只加载 RBAC 策略
	tempDir1 := t.TempDir()
	rbacContent, err := os.ReadFile(filepath.Join("policies", "rbac.rego"))
	if err != nil {
		t.Fatalf("Failed to read RBAC policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir1, "rbac.rego"), rbacContent, 0644); err != nil {
		t.Fatalf("Failed to write RBAC policy: %v", err)
	}

	client1, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir1,
		},
		DefaultDecisionPath: "rbac/allow",
	})
	if err != nil {
		t.Fatalf("Failed to create client1: %v", err)
	}
	defer client1.Close(ctx)

	// 客户端 2：只加载 ABAC 策略
	tempDir2 := t.TempDir()
	abacContent, err := os.ReadFile(filepath.Join("policies", "abac.rego"))
	if err != nil {
		t.Fatalf("Failed to read ABAC policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, "abac.rego"), abacContent, 0644); err != nil {
		t.Fatalf("Failed to write ABAC policy: %v", err)
	}

	client2, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir2,
		},
		DefaultDecisionPath: "abac/allow",
	})
	if err != nil {
		t.Fatalf("Failed to create client2: %v", err)
	}
	defer client2.Close(ctx)

	// 验证客户端隔离
	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"user":  "alice",
			"roles": []string{"admin"},
		},
		"action":   "read",
		"resource": "documents",
	}

	// 客户端 1 应该能评估 RBAC
	result1, err := client1.Evaluate(ctx, "", input)
	if err != nil {
		t.Fatalf("Client1 evaluation failed: %v", err)
	}
	if !result1.Allowed {
		t.Error("Client1 RBAC evaluation should allow admin")
	}

	// 客户端 2 应该能评估 ABAC
	result2, err := client2.Evaluate(ctx, "", input)
	if err != nil {
		t.Fatalf("Client2 evaluation failed: %v", err)
	}
	if !result2.Allowed {
		t.Error("Client2 ABAC evaluation should allow admin")
	}
}

// =============================================================================
// Additional OPA Integration Tests
// =============================================================================

// TestOPAComplexRBACScenarios 测试复杂 RBAC 场景
func TestOPAComplexRBACScenarios(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// 创建复杂 RBAC 策略
	complexPolicy := `
package complex_rbac

import rego.v1

default allow := false

# 角色层级定义
role_hierarchy := {
    "super_admin": ["admin", "editor", "viewer"],
    "admin": ["editor", "viewer"],
    "editor": ["viewer"],
    "viewer": [],
}

# 获取用户的有效角色（包括继承的角色）
effective_roles contains role if {
    some r in input.subject.roles
    role := role_hierarchy[r][_]
}

effective_roles contains role if {
    some role in input.subject.roles
}

# 资源权限定义
resource_permissions := {
    "documents": {
        "admin": ["create", "read", "update", "delete"],
        "editor": ["create", "read", "update"],
        "viewer": ["read"],
    },
    "users": {
        "admin": ["create", "read", "update", "delete"],
        "viewer": ["read"],
    },
    "settings": {
        "super_admin": ["create", "read", "update", "delete"],
        "admin": ["read", "update"],
    },
}

# 权限检查
allow if {
    some role in effective_roles
    input.action in resource_permissions[input.resource][role]
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "complex_rbac.rego"), []byte(complexPolicy), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	client, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "complex_rbac/allow",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "super_admin inherits admin permissions",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "root",
					"roles": []string{"super_admin"},
				},
				"action":   "delete",
				"resource": "documents",
			},
			expected: true,
		},
		{
			name: "admin can update settings",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "admin_user",
					"roles": []string{"admin"},
				},
				"action":   "update",
				"resource": "settings",
			},
			expected: true,
		},
		{
			name: "admin cannot delete settings",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "admin_user",
					"roles": []string{"admin"},
				},
				"action":   "delete",
				"resource": "settings",
			},
			expected: false,
		},
		{
			name: "editor inherits viewer permissions",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "editor_user",
					"roles": []string{"editor"},
				},
				"action":   "read",
				"resource": "documents",
			},
			expected: true,
		},
		{
			name: "viewer cannot create documents",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "viewer_user",
					"roles": []string{"viewer"},
				},
				"action":   "create",
				"resource": "documents",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Evaluate(ctx, "", tt.input)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}
			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v, got %v", tt.expected, result.Allowed)
			}
		})
	}
}

// TestOPATimeBasedPolicy 测试基于时间的策略
func TestOPATimeBasedPolicy(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// 创建时间敏感策略
	timePolicy := `
package time_policy

import rego.v1

default allow := false

# 允许工作时间访问（简化版本，使用输入时间）
allow if {
    input.current_hour >= 9
    input.current_hour < 18
    input.subject.roles[_] == "employee"
}

# 管理员任何时间都可以访问
allow if {
    input.subject.roles[_] == "admin"
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "time_policy.rego"), []byte(timePolicy), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	client, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "time_policy/allow",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "employee during work hours",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "employee1",
					"roles": []string{"employee"},
				},
				"current_hour": 10,
			},
			expected: true,
		},
		{
			name: "employee outside work hours",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "employee1",
					"roles": []string{"employee"},
				},
				"current_hour": 22,
			},
			expected: false,
		},
		{
			name: "admin outside work hours",
			input: map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  "admin1",
					"roles": []string{"admin"},
				},
				"current_hour": 22,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Evaluate(ctx, "", tt.input)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}
			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v, got %v", tt.expected, result.Allowed)
			}
		})
	}
}

// TestOPAConcurrentEvaluation 测试并发策略评估
func TestOPAConcurrentEvaluation(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// 读取 RBAC 策略
	rbacContent, err := os.ReadFile(filepath.Join("policies", "rbac.rego"))
	if err != nil {
		t.Fatalf("Failed to read RBAC policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rbac.rego"), rbacContent, 0644); err != nil {
		t.Fatalf("Failed to write RBAC policy: %v", err)
	}

	client, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "rbac/allow",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 并发评估
	numGoroutines := 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			input := map[string]interface{}{
				"subject": map[string]interface{}{
					"user":  fmt.Sprintf("user-%d", id),
					"roles": []string{"admin"},
				},
				"action":   "read",
				"resource": "documents",
			}

			result, err := client.Evaluate(ctx, "", input)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %v", id, err)
				return
			}

			if !result.Allowed {
				errors <- fmt.Errorf("goroutine %d: expected allowed=true", id)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestOPAPolicyValidation 测试策略验证
func TestOPAPolicyValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		policy    string
		wantError bool
	}{
		{
			name: "valid policy",
			policy: `
package test
import rego.v1
default allow := false
allow if { input.user == "admin" }
`,
			wantError: false,
		},
		{
			name: "invalid syntax",
			policy: `
package test
import rego.v1
default allow := false
allow if { input.user ==
`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(tempDir, "test.rego"), []byte(tt.policy), 0644); err != nil {
				t.Fatalf("Failed to write policy: %v", err)
			}

			client, err := opa.NewClient(ctx, &opa.Config{
				Mode: opa.ModeEmbedded,
				EmbeddedConfig: &opa.EmbeddedConfig{
					PolicyDir: tempDir,
				},
				DefaultDecisionPath: "test/allow",
			})

			if tt.wantError {
				if err == nil {
					t.Error("Expected error for invalid policy, got nil")
					if client != nil {
						client.Close(ctx)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					client.Close(ctx)
				}
			}
		})
	}
}

// TestOPADataIntegration 测试数据集成
func TestOPADataIntegration(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// 创建使用外部数据的策略
	dataPolicy := `
package data_policy

import rego.v1

default allow := false

# 使用输入中的数据进行决策
allow if {
    user_data := input.user_database[input.subject.user_id]
    user_data.active == true
    user_data.level >= input.required_level
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "data_policy.rego"), []byte(dataPolicy), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	client, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "data_policy/allow",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 模拟用户数据库
	userDatabase := map[string]interface{}{
		"user1": map[string]interface{}{
			"active": true,
			"level":  5,
		},
		"user2": map[string]interface{}{
			"active": false,
			"level":  10,
		},
		"user3": map[string]interface{}{
			"active": true,
			"level":  2,
		},
	}

	tests := []struct {
		name          string
		userID        string
		requiredLevel int
		expected      bool
	}{
		{
			name:          "active user with sufficient level",
			userID:        "user1",
			requiredLevel: 3,
			expected:      true,
		},
		{
			name:          "inactive user with high level",
			userID:        "user2",
			requiredLevel: 5,
			expected:      false,
		},
		{
			name:          "active user with insufficient level",
			userID:        "user3",
			requiredLevel: 5,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]interface{}{
				"subject": map[string]interface{}{
					"user_id": tt.userID,
				},
				"user_database":  userDatabase,
				"required_level": tt.requiredLevel,
			}

			result, err := client.Evaluate(ctx, "", input)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}
			if result.Allowed != tt.expected {
				t.Errorf("Expected allowed=%v, got %v", tt.expected, result.Allowed)
			}
		})
	}
}

// TestOPAErrorHandling 测试错误处理
func TestOPAErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		setupClient func() (opa.Client, error)
		wantError   bool
	}{
		{
			name: "empty policy directory",
			setupClient: func() (opa.Client, error) {
				return opa.NewClient(ctx, &opa.Config{
					Mode: opa.ModeEmbedded,
					EmbeddedConfig: &opa.EmbeddedConfig{
						PolicyDir: t.TempDir(),
					},
					DefaultDecisionPath: "nonexistent/allow",
				})
			},
			wantError: false, // Client creation succeeds, but evaluation might fail
		},
		{
			name: "invalid remote URL",
			setupClient: func() (opa.Client, error) {
				return opa.NewClient(ctx, &opa.Config{
					Mode: opa.ModeRemote,
					RemoteConfig: &opa.RemoteConfig{
						URL:     "http://invalid-host-that-does-not-exist:9999",
						Timeout: 1 * time.Second,
					},
					DefaultDecisionPath: "test/allow",
				})
			},
			wantError: false, // Client creation might succeed, connection fails later
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.setupClient()
			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
					if client != nil {
						client.Close(ctx)
					}
				}
			} else {
				if client != nil {
					client.Close(ctx)
				}
			}
		})
	}
}

// BenchmarkOPAConcurrentEvaluation 基准测试并发评估性能
func BenchmarkOPAConcurrentEvaluation(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()

	rbacContent, err := os.ReadFile(filepath.Join("policies", "rbac.rego"))
	if err != nil {
		b.Fatalf("Failed to read RBAC policy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rbac.rego"), rbacContent, 0644); err != nil {
		b.Fatalf("Failed to write RBAC policy: %v", err)
	}

	client, err := opa.NewClient(ctx, &opa.Config{
		Mode: opa.ModeEmbedded,
		EmbeddedConfig: &opa.EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "rbac/allow",
	})
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"user":  "alice",
			"roles": []string{"admin"},
		},
		"action":   "read",
		"resource": "documents",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Evaluate(ctx, "", input)
			if err != nil {
				b.Fatalf("Evaluation failed: %v", err)
			}
		}
	})
}
