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

package opa

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClientEmbedded(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建简单的策略文件
	policyContent := `
package test

import rego.v1

default allow = false

allow = true if {
    input.user == "alice"
}
`
	policyPath := filepath.Join(tempDir, "test.rego")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tempDir,
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	if !client.IsEmbedded() {
		t.Error("Expected client to be embedded mode")
	}
}

// TestEmbeddedClientEvaluate tests embedded client evaluation
func TestEmbeddedClientEvaluate(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建简单的策略文件用于加载测试
	policyContent := `
package test

import rego.v1

default allow = false

allow = true if {
    input.user == "alice"
    input.action == "read"
}
`
	policyPath := filepath.Join(tempDir, "test.rego")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tempDir,
		},
		DefaultDecisionPath: "test/allow",
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 满足所有条件的输入应被允许
	result, err := client.Evaluate(ctx, "", map[string]interface{}{
		"user":   "alice",
		"action": "read",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate policy: %v", err)
	}
	if !result.Allowed {
		t.Errorf("Expected policy to allow alice/read, got decision: %+v", result.Decision)
	}
	if allowed, ok := result.Decision.(bool); !ok || !allowed {
		t.Errorf("Expected decision to be true, got: %+v", result.Decision)
	}

	// 不满足条件的输入应被拒绝（命中 default allow = false）
	result, err = client.Evaluate(ctx, "", map[string]interface{}{
		"user":   "bob",
		"action": "read",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate policy: %v", err)
	}
	if result.Allowed {
		t.Errorf("Expected policy to deny bob/read, got decision: %+v", result.Decision)
	}
}

// TestEmbeddedClientLoadPolicy tests dynamic policy loading
func TestEmbeddedClientLoadPolicy(t *testing.T) {
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	policy := `
package dynamic

import rego.v1

default allow = false

allow = true if {
    input.role == "admin"
}
`

	// 测试加载策略功能
	if err := client.LoadPolicy(ctx, "dynamic.rego", policy); err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 验证策略已加载：admin 角色应被允许
	result, err := client.Evaluate(ctx, "dynamic/allow", map[string]interface{}{
		"role": "admin",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate dynamically loaded policy: %v", err)
	}
	if !result.Allowed {
		t.Errorf("Expected policy to allow role=admin, got decision: %+v", result.Decision)
	}

	// 非 admin 角色应被拒绝
	result, err = client.Evaluate(ctx, "dynamic/allow", map[string]interface{}{
		"role": "guest",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate dynamically loaded policy: %v", err)
	}
	if result.Allowed {
		t.Errorf("Expected policy to deny role=guest, got decision: %+v", result.Decision)
	}
}

func TestEmbeddedClientRemovePolicy(t *testing.T) {
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	policy := `
package temp

import rego.v1

default allow = true
`

	// 加载策略
	if err := client.LoadPolicy(ctx, "temp.rego", policy); err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 移除策略
	if err := client.RemovePolicy(ctx, "temp.rego"); err != nil {
		t.Fatalf("Failed to remove policy: %v", err)
	}
}

func TestNewClientWithOptions(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	client, err := NewClientWithOptions(ctx,
		WithEmbeddedMode(tempDir),
		WithDefaultDecisionPath("test/allow"),
	)

	if err != nil {
		t.Fatalf("Failed to create client with options: %v", err)
	}
	defer client.Close(ctx)

	if !client.IsEmbedded() {
		t.Error("Expected client to be embedded mode")
	}
}

func TestEmbeddedClientLoadPolicyFromFile(t *testing.T) {
	tempDir := t.TempDir()

	// 创建策略文件
	policyContent := `
package filepolicy

import rego.v1

default allow = false

allow = true if {
    input.action == "write"
    input.user == "admin"
}
`
	policyPath := filepath.Join(tempDir, "file_policy.rego")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 测试 LoadPolicyFromFile
	if err := client.LoadPolicyFromFile(ctx, policyPath); err != nil {
		t.Fatalf("Failed to load policy from file: %v", err)
	}

	// 验证策略已加载且评估结果正确
	result, err := client.Evaluate(ctx, "filepolicy/allow", map[string]interface{}{
		"action": "write",
		"user":   "admin",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate policy loaded from file: %v", err)
	}
	if !result.Allowed {
		t.Errorf("Expected policy to allow admin/write, got decision: %+v", result.Decision)
	}
}

func TestEmbeddedClientLoadData(t *testing.T) {
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 加载测试数据
	testData := map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": "1", "name": "alice", "role": "admin"},
			{"id": "2", "name": "bob", "role": "user"},
		},
	}

	if err := client.LoadData(ctx, "testdata", testData); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	// 加载使用数据的策略
	policy := `
package datatest

import rego.v1

default allow = false

allow = true if {
    some user in data.testdata.users
    user.name == input.username
    user.role == "admin"
}
`

	if err := client.LoadPolicy(ctx, "datatest.rego", policy); err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 评估策略：alice 是 admin，应被允许
	result, err := client.Evaluate(ctx, "datatest/allow", map[string]interface{}{
		"username": "alice",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate policy with data: %v", err)
	}
	if !result.Allowed {
		t.Errorf("Expected policy to allow alice (admin), got decision: %+v", result.Decision)
	}

	// bob 不是 admin，应被拒绝
	result, err = client.Evaluate(ctx, "datatest/allow", map[string]interface{}{
		"username": "bob",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate policy with data: %v", err)
	}
	if result.Allowed {
		t.Errorf("Expected policy to deny bob (user), got decision: %+v", result.Decision)
	}
}

func TestEmbeddedClientLoadDataWithLeadingSlash(t *testing.T) {
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	testData := map[string]string{
		"alice": "admin",
		"bob":   "user",
	}

	// 测试用例：验证带前导斜杠和不带前导斜杠的路径都能正常工作
	testCases := []struct {
		name string
		path string
	}{
		{"without leading slash", "roles"},
		{"with leading slash", "/roles"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := client.LoadData(ctx, tc.path, testData); err != nil {
				t.Errorf("LoadData with path %q failed: %v", tc.path, err)
			}
		})
	}

	// 验证数据已加载（使用策略访问数据）
	policy := `
package pathtest

import rego.v1

default allow = false

allow = true if {
    data.roles[input.user] == "admin"
}
`

	if err := client.LoadPolicy(ctx, "pathtest.rego", policy); err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	result, err := client.Evaluate(ctx, "pathtest/allow", map[string]interface{}{
		"user": "alice",
	})
	if err != nil {
		t.Fatalf("Failed to evaluate policy: %v", err)
	}
	if !result.Allowed {
		t.Errorf("Expected policy to allow alice (admin role), got decision: %+v", result.Decision)
	}
}

func TestEmbeddedClientQuery(t *testing.T) {
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 加载策略
	policy := `
package querytest

import rego.v1

result = {"allowed": true, "reason": "test"} if {
    input.value > 10
}

result = {"allowed": false, "reason": "too small"} if {
    input.value <= 10
}
`

	if err := client.LoadPolicy(ctx, "querytest.rego", policy); err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 测试 Query 方法
	rs, err := client.Query(ctx, "data.querytest.result", map[string]interface{}{
		"value": 15,
	})

	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(rs) == 0 {
		t.Fatal("Expected query to return a non-empty result set")
	}
	if len(rs[0].Expressions) == 0 {
		t.Fatal("Expected query result to contain expressions")
	}

	value, ok := rs[0].Expressions[0].Value.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected query result to be a map, got: %T", rs[0].Expressions[0].Value)
	}
	if allowed, _ := value["allowed"].(bool); !allowed {
		t.Errorf("Expected allowed=true for value=15, got: %+v", value)
	}
	if reason, _ := value["reason"].(string); reason != "test" {
		t.Errorf("Expected reason=test, got: %+v", value["reason"])
	}
}

func TestEmbeddedClientWithCache(t *testing.T) {
	tempDir := t.TempDir()

	// 创建策略文件
	policyContent := `
package cachetest

import rego.v1

default allow = false

allow = true if {
    input.user == "alice"
}
`
	policyPath := filepath.Join(tempDir, "cache_test.rego")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tempDir,
		},
		CacheConfig: &CacheConfig{
			Enabled: true,
			MaxSize: 100,
			TTL:     60,
		},
		DefaultDecisionPath: "cachetest/allow",
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	input := map[string]interface{}{
		"user": "alice",
	}

	// 第一次评估（缓存未命中）
	first, err := client.Evaluate(ctx, "", input)
	if err != nil {
		t.Fatalf("First evaluation failed: %v", err)
	}
	if !first.Allowed {
		t.Errorf("Expected first evaluation to allow alice, got decision: %+v", first.Decision)
	}

	// 第二次评估（应该命中缓存且结果一致）
	second, err := client.Evaluate(ctx, "", input)
	if err != nil {
		t.Fatalf("Second evaluation failed: %v", err)
	}
	if second.Allowed != first.Allowed {
		t.Errorf("Expected cached result to match: first=%v, second=%v", first.Allowed, second.Allowed)
	}
}

func TestEmbeddedClientPartialEvaluate(t *testing.T) {
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// 加载策略
	policy := `
package partialtest

import rego.v1

allow = true if {
    input.user == "alice"
    input.action == "read"
    data.roles[input.user] == "admin"
}
`

	if err := client.LoadPolicy(ctx, "partialtest.rego", policy); err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 加载角色数据
	if err := client.LoadData(ctx, "roles", map[string]string{
		"alice": "admin",
		"bob":   "user",
	}); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	// 测试部分评估 - 只提供 user，不提供 action
	result, err := client.PartialEvaluate(ctx, "data.partialtest.allow", map[string]interface{}{
		"user": "alice",
	})

	if err != nil {
		t.Fatalf("Failed to partial evaluate: %v", err)
	}

	if len(result.Queries) == 0 {
		t.Log("Partial evaluation returned empty queries (may be optimized out)")
	} else {
		t.Logf("Partial evaluation result: %d queries, %d support rules", len(result.Queries), len(result.Support))
		for i, q := range result.Queries {
			t.Logf("Query %d: %v", i, q)
		}
	}
}
