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

package opa

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testPolicy = `
package test

import rego.v1

default allow = false

allow if {
    input.user == "alice"
    input.action == "read"
}
`

const testPolicyV2 = `
package test

import rego.v1

default allow = false

allow if {
    input.user == "alice"
    input.action == "write"
}
`

const invalidPolicy = `
package test

this is invalid rego
`

func TestNewManager(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	manager := NewManager(client)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.client == nil {
		t.Error("Manager client is nil")
	}

	if manager.policies == nil {
		t.Error("Manager policies map is nil")
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	manager, err := NewManagerWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close(ctx)

	if manager == nil {
		t.Fatal("NewManagerWithConfig returned nil")
	}
}

func TestManagerRegisterPolicy(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	policy := &Policy{
		Content: testPolicy,
	}

	err := manager.RegisterPolicy(ctx, "test.rego", policy)
	if err != nil {
		t.Fatalf("Failed to register policy: %v", err)
	}

	// 验证策略已注册
	if policy.Name != "test.rego" {
		t.Errorf("Policy name not set correctly: got %s, want test.rego", policy.Name)
	}

	if policy.Version != 1 {
		t.Errorf("Policy version not set correctly: got %d, want 1", policy.Version)
	}

	if policy.CreatedAt.IsZero() {
		t.Error("Policy CreatedAt not set")
	}

	if policy.UpdatedAt.IsZero() {
		t.Error("Policy UpdatedAt not set")
	}
}

func TestManagerRegisterPolicyInvalidSyntax(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	policy := &Policy{
		Content: invalidPolicy,
	}

	err := manager.RegisterPolicy(ctx, "invalid.rego", policy)
	if err == nil {
		t.Error("Expected error for invalid policy, got nil")
	}
}

func TestManagerRegisterPolicyNil(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	err := manager.RegisterPolicy(ctx, "test.rego", nil)
	if err == nil {
		t.Error("Expected error for nil policy, got nil")
	}
}

func TestManagerLoadPolicy(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	err := manager.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 验证策略已加载
	policy, err := manager.GetPolicy("test.rego")
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if policy.Name != "test.rego" {
		t.Errorf("Policy name incorrect: got %s, want test.rego", policy.Name)
	}

	if policy.Content != testPolicy {
		t.Error("Policy content doesn't match")
	}
}

func TestManagerLoadPolicyFromFile(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 创建临时策略文件
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "test.rego")
	if err := os.WriteFile(policyPath, []byte(testPolicy), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	err := manager.LoadPolicyFromFile(ctx, "test.rego", policyPath)
	if err != nil {
		t.Fatalf("Failed to load policy from file: %v", err)
	}

	// 验证策略已加载
	policy, err := manager.GetPolicy("test.rego")
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if policy.Name != "test.rego" {
		t.Errorf("Policy name incorrect: got %s, want test.rego", policy.Name)
	}
}

func TestManagerLoadPoliciesFromDir(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// 创建多个策略文件，使用不同的包名避免冲突
	policies := map[string]string{
		"policy1.rego": `
package policy1

import rego.v1

default allow = false

allow if {
    input.user == "alice"
}
`,
		"policy2.rego": `
package policy2

import rego.v1

default allow = false

allow if {
    input.user == "bob"
}
`,
	}

	for name, content := range policies {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write policy file %s: %v", name, err)
		}
	}

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
	}

	manager, err := NewManagerWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close(ctx)

	err = manager.LoadPoliciesFromDir(ctx, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load policies from directory: %v", err)
	}

	// 验证所有策略已加载
	loadedPolicies, err := manager.ListPolicies()
	if err != nil {
		t.Fatalf("Failed to list policies: %v", err)
	}

	if len(loadedPolicies) < len(policies) {
		t.Errorf("Not all policies loaded: got %d, want at least %d", len(loadedPolicies), len(policies))
	}
}

func TestManagerUpdatePolicy(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 先注册初始策略
	policy := &Policy{
		Content: testPolicy,
	}
	err := manager.RegisterPolicy(ctx, "test.rego", policy)
	if err != nil {
		t.Fatalf("Failed to register policy: %v", err)
	}

	initialVersion := policy.Version
	initialCreatedAt := policy.CreatedAt

	// 等待一点时间以确保更新时间不同
	time.Sleep(10 * time.Millisecond)

	// 更新策略
	updatedPolicy := &Policy{
		Content: testPolicyV2,
	}
	err = manager.UpdatePolicy(ctx, "test.rego", updatedPolicy)
	if err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	// 验证版本号增加
	if updatedPolicy.Version != initialVersion+1 {
		t.Errorf("Policy version not incremented: got %d, want %d", updatedPolicy.Version, initialVersion+1)
	}

	// 验证创建时间保持不变
	if !updatedPolicy.CreatedAt.Equal(initialCreatedAt) {
		t.Error("Policy CreatedAt changed after update")
	}

	// 验证更新时间改变
	if updatedPolicy.UpdatedAt.Before(initialCreatedAt) || updatedPolicy.UpdatedAt.Equal(initialCreatedAt) {
		t.Error("Policy UpdatedAt not updated correctly")
	}
}

func TestManagerUpdatePolicyNotFound(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	policy := &Policy{
		Content: testPolicy,
	}

	err := manager.UpdatePolicy(ctx, "nonexistent.rego", policy)
	if err == nil {
		t.Error("Expected error for updating non-existent policy, got nil")
	}
}

func TestManagerDeletePolicy(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 先注册策略
	err := manager.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 删除策略
	err = manager.DeletePolicy(ctx, "test.rego")
	if err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// 验证策略已删除
	_, err = manager.GetPolicy("test.rego")
	if err == nil {
		t.Error("Expected error for deleted policy, got nil")
	}
}

func TestManagerRemovePolicy(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 先注册策略
	err := manager.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 移除策略
	err = manager.RemovePolicy(ctx, "test.rego")
	if err != nil {
		t.Fatalf("Failed to remove policy: %v", err)
	}

	// 验证策略已移除
	_, err = manager.GetPolicy("test.rego")
	if err == nil {
		t.Error("Expected error for removed policy, got nil")
	}
}

func TestManagerListPolicies(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 注册多个策略，使用不同的包名避免冲突
	policies := map[string]string{
		"policy1.rego": `
package policy1

import rego.v1

default allow = false
`,
		"policy2.rego": `
package policy2

import rego.v1

default allow = false
`,
		"policy3.rego": `
package policy3

import rego.v1

default allow = false
`,
	}

	for name, content := range policies {
		err := manager.LoadPolicy(ctx, name, content)
		if err != nil {
			t.Fatalf("Failed to load policy %s: %v", name, err)
		}
	}

	policyNames := []string{"policy1.rego", "policy2.rego", "policy3.rego"}

	// 列出所有策略
	loadedPolicies, err := manager.ListPolicies()
	if err != nil {
		t.Fatalf("Failed to list policies: %v", err)
	}

	if len(loadedPolicies) != len(policyNames) {
		t.Errorf("Wrong number of policies: got %d, want %d", len(loadedPolicies), len(policyNames))
	}

	// 验证策略名称
	policyMap := make(map[string]bool)
	for _, p := range loadedPolicies {
		policyMap[p.Name] = true
	}

	for _, name := range policyNames {
		if !policyMap[name] {
			t.Errorf("Policy %s not found in list", name)
		}
	}
}

func TestManagerGetPolicy(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 注册策略
	err := manager.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	// 获取策略
	policy, err := manager.GetPolicy("test.rego")
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if policy.Name != "test.rego" {
		t.Errorf("Policy name incorrect: got %s, want test.rego", policy.Name)
	}

	if policy.Content != testPolicy {
		t.Error("Policy content doesn't match")
	}
}

func TestManagerGetPolicyNotFound(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	_, err := manager.GetPolicy("nonexistent.rego")
	if err == nil {
		t.Error("Expected error for non-existent policy, got nil")
	}
}

func TestManagerValidatePolicy(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 测试有效策略
	err := manager.ValidatePolicy(testPolicy)
	if err != nil {
		t.Errorf("Valid policy failed validation: %v", err)
	}

	// 测试无效策略
	err = manager.ValidatePolicy(invalidPolicy)
	if err == nil {
		t.Error("Invalid policy passed validation")
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	manager := createTestManager(t)
	defer manager.Close(ctx)

	// 并发注册策略，使用不同的包名避免冲突
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			pkgName := "test" + string(rune('a'+index))
			name := pkgName + ".rego"
			policyContent := fmt.Sprintf(`
package %s

import rego.v1

default allow = false

allow if {
    input.user == "user%d"
}
`, pkgName, index)
			err := manager.LoadPolicy(ctx, name, policyContent)
			if err != nil {
				t.Errorf("Failed to load policy %s: %v", name, err)
			}
			done <- true
		}(i)
	}

	// 等待所有协程完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有策略已注册
	policies, err := manager.ListPolicies()
	if err != nil {
		t.Fatalf("Failed to list policies: %v", err)
	}

	if len(policies) != 10 {
		t.Errorf("Expected 10 policies, got %d", len(policies))
	}
}

// createTestManager 创建测试用的管理器
func createTestManager(t *testing.T) *Manager {
	ctx := context.Background()
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return NewManager(client)
}
