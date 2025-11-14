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

//go:build integration
// +build integration

package opa_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
