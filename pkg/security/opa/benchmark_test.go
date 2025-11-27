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
	"fmt"
	"testing"
	"time"
)

// ============================================================================
// Simple Policy Evaluation Benchmarks
// ============================================================================

// simplePolicy is a basic RBAC policy for benchmarking.
const simplePolicy = `
package rbac

default allow = false

allow {
    input.user.roles[_] == "admin"
}

allow {
    input.user.roles[_] == input.resource.required_role
}
`

// BenchmarkEmbeddedClient_Evaluate benchmarks basic policy evaluation.
func BenchmarkEmbeddedClient_Evaluate(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the simple policy
	if err := client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
		"resource": map[string]interface{}{
			"type": "document",
			"id":   "doc123",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "rbac.allow", input)
		if err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

// BenchmarkEmbeddedClient_EvaluateWithContext benchmarks evaluation with context.
func BenchmarkEmbeddedClient_EvaluateWithContext(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the simple policy
	if err := client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
		"resource": map[string]interface{}{
			"type": "document",
			"id":   "doc123",
		},
	}

	// Create context with timeout
	evalCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(evalCtx, "rbac.allow", input)
		if err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

// ============================================================================
// RBAC Policy Benchmarks
// ============================================================================

// benchRBACPolicy is a more complex RBAC policy for benchmarking.
const benchRBACPolicy = `
package rbac

default allow = false

# Role definitions
admin_roles = {"admin", "super_admin"}
editor_roles = {"editor", "content_manager"}
viewer_roles = {"viewer", "guest"}

# Permission mappings
role_permissions = {
    "admin": ["create", "read", "update", "delete", "manage"],
    "super_admin": ["create", "read", "update", "delete", "manage", "admin"],
    "editor": ["create", "read", "update"],
    "content_manager": ["create", "read", "update", "publish"],
    "viewer": ["read"],
    "guest": ["read"]
}

# Allow if user has admin role
allow {
    admin_roles[input.user.roles[_]]
}

# Allow if user has required permission
allow {
    role := input.user.roles[_]
    permission := role_permissions[role][_]
    permission == input.action
}

# Allow if user owns the resource
allow {
    input.resource.owner == input.user.id
    input.action != "delete"
}
`

// BenchmarkEmbeddedClient_EvaluateRBAC benchmarks RBAC policy evaluation.
func BenchmarkEmbeddedClient_EvaluateRBAC(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the RBAC policy
	if err := client.LoadPolicy(ctx, "rbac.rego", benchRBACPolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"editor"},
		},
		"resource": map[string]interface{}{
			"type":  "document",
			"id":    "doc123",
			"owner": "user456",
		},
		"action": "update",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "rbac.allow", input)
		if err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

// ============================================================================
// ABAC Policy Benchmarks
// ============================================================================

// benchABACPolicy is an ABAC policy for benchmarking.
const benchABACPolicy = `
package abac

default allow = false

# Time-based access
allow {
    input.user.roles[_] == "admin"
}

# Department-based access
allow {
    input.user.department == input.resource.department
    input.action == "read"
}

# Level-based access
allow {
    input.user.level >= input.resource.required_level
    input.action == "read"
}

# Project membership
allow {
    input.resource.project_id == input.user.projects[_]
    input.action != "delete"
}

# Ownership check
allow {
    input.resource.owner == input.user.id
}
`

// BenchmarkEmbeddedClient_EvaluateABAC benchmarks ABAC policy evaluation.
func BenchmarkEmbeddedClient_EvaluateABAC(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "abac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the ABAC policy
	if err := client.LoadPolicy(ctx, "abac.rego", benchABACPolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":         "user123",
			"roles":      []string{"developer"},
			"department": "engineering",
			"level":      3,
			"projects":   []string{"proj1", "proj2", "proj3"},
		},
		"resource": map[string]interface{}{
			"type":           "document",
			"id":             "doc123",
			"department":     "engineering",
			"required_level": 2,
			"project_id":     "proj2",
			"owner":          "user456",
		},
		"action": "read",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "abac.allow", input)
		if err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

// ============================================================================
// Cache Benchmarks
// ============================================================================

// BenchmarkCache_Get benchmarks cache retrieval.
func BenchmarkCache_Get(b *testing.B) {
	ctx := context.Background()

	cacheConfig := &CacheConfig{
		Enabled: true,
		MaxSize: 1000,
		TTL:     time.Hour,
	}

	cache, err := NewCache(cacheConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close(ctx)

	// Pre-populate cache
	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
	}

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	cache.Set(ctx, "rbac.allow", input, result)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "rbac.allow", input)
	}
}

// BenchmarkCache_Set benchmarks cache storage.
func BenchmarkCache_Set(b *testing.B) {
	ctx := context.Background()

	cacheConfig := &CacheConfig{
		Enabled: true,
		MaxSize: 10000,
		TTL:     time.Hour,
	}

	cache, err := NewCache(cacheConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close(ctx)

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
	}

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "rbac.allow", input, result)
	}
}

// BenchmarkCache_GetMiss benchmarks cache miss.
func BenchmarkCache_GetMiss(b *testing.B) {
	ctx := context.Background()

	cacheConfig := &CacheConfig{
		Enabled: true,
		MaxSize: 1000,
		TTL:     time.Hour,
	}

	cache, err := NewCache(cacheConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close(ctx)

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "rbac.allow", input)
	}
}

// BenchmarkEmbeddedClient_EvaluateWithCache benchmarks evaluation with cache enabled.
func BenchmarkEmbeddedClient_EvaluateWithCache(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		CacheConfig: &CacheConfig{
			Enabled: true,
			MaxSize: 1000,
			TTL:     time.Hour,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the simple policy
	if err := client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
		"resource": map[string]interface{}{
			"type": "document",
			"id":   "doc123",
		},
	}

	// Warm up the cache
	_, _ = client.Evaluate(ctx, "rbac.allow", input)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "rbac.allow", input)
		if err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

// ============================================================================
// Concurrent Evaluation Benchmarks
// ============================================================================

// BenchmarkEmbeddedClient_ConcurrentEvaluate benchmarks concurrent policy evaluation.
func BenchmarkEmbeddedClient_ConcurrentEvaluate(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the simple policy
	if err := client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
		"resource": map[string]interface{}{
			"type": "document",
			"id":   "doc123",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Evaluate(ctx, "rbac.allow", input)
			if err != nil {
				b.Fatalf("Evaluate() error = %v", err)
			}
		}
	})
}

// BenchmarkEmbeddedClient_ConcurrentEvaluateWithCache benchmarks concurrent evaluation with cache.
func BenchmarkEmbeddedClient_ConcurrentEvaluateWithCache(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		CacheConfig: &CacheConfig{
			Enabled: true,
			MaxSize: 1000,
			TTL:     time.Hour,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the simple policy
	if err := client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
		"resource": map[string]interface{}{
			"type": "document",
			"id":   "doc123",
		},
	}

	// Warm up the cache
	_, _ = client.Evaluate(ctx, "rbac.allow", input)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Evaluate(ctx, "rbac.allow", input)
			if err != nil {
				b.Fatalf("Evaluate() error = %v", err)
			}
		}
	})
}

// ============================================================================
// Evaluator Benchmarks
// ============================================================================

// BenchmarkEvaluator_Evaluate benchmarks evaluator-based evaluation.
func BenchmarkEvaluator_Evaluate(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	evaluator, err := NewEvaluatorWithConfig(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}
	defer evaluator.Close(ctx)

	// Load the simple policy
	if err := evaluator.client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
		"resource": map[string]interface{}{
			"type": "document",
			"id":   "doc123",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := evaluator.Evaluate(ctx, "rbac.allow", input)
		if err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

// BenchmarkEvaluator_IsAllowed benchmarks the IsAllowed convenience method.
func BenchmarkEvaluator_IsAllowed(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	evaluator, err := NewEvaluatorWithConfig(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}
	defer evaluator.Close(ctx)

	// Load the simple policy
	if err := evaluator.client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user123",
			"roles": []string{"admin"},
		},
		"resource": map[string]interface{}{
			"type": "document",
			"id":   "doc123",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := evaluator.IsAllowed(ctx, "rbac.allow", input)
		if err != nil {
			b.Fatalf("IsAllowed() error = %v", err)
		}
	}
}

// BenchmarkEvaluator_EvaluateRBAC benchmarks the EvaluateRBAC method.
func BenchmarkEvaluator_EvaluateRBAC(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	evaluator, err := NewEvaluatorWithConfig(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}
	defer evaluator.Close(ctx)

	// Load the simple policy
	if err := evaluator.client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	input := &RBACInput{
		Subject: &Subject{
			User:  "user123",
			Roles: []string{"admin"},
		},
		Action:   "read",
		Resource: "document",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := evaluator.EvaluateRBAC(ctx, input)
		if err != nil {
			b.Fatalf("EvaluateRBAC() error = %v", err)
		}
	}
}

// ============================================================================
// Policy Input Builder Benchmarks
// ============================================================================

// BenchmarkPolicyInputBuilder_Build benchmarks policy input construction.
func BenchmarkPolicyInputBuilder_Build(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		builder := NewPolicyInputBuilder().
			WithUserID("user123").
			WithUsername("testuser").
			WithUserEmail("test@example.com").
			WithUserRoles([]string{"admin", "user"}).
			WithUserPermissions([]string{"read", "write"}).
			WithResourceType("document").
			WithResourceID("doc123").
			WithResourceOwner("user456")

		_ = builder.Build()
	}
}

// BenchmarkPolicyInputBuilder_WithCustomData benchmarks adding custom data.
func BenchmarkPolicyInputBuilder_WithCustomData(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		builder := NewPolicyInputBuilder().
			WithUserID("user123").
			WithCustomData("tenant_id", "tenant123").
			WithCustomData("department", "engineering").
			WithCustomData("location", "us-west-2")

		_ = builder.Build()
	}
}

// ============================================================================
// Large Input Benchmarks
// ============================================================================

// BenchmarkEmbeddedClient_EvaluateLargeInput benchmarks evaluation with large input.
func BenchmarkEmbeddedClient_EvaluateLargeInput(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	client, err := NewClient(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(ctx)

	// Load the simple policy
	if err := client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	// Create large input with many roles and attributes
	roles := make([]string, 50)
	for i := 0; i < 50; i++ {
		roles[i] = fmt.Sprintf("role_%d", i)
	}
	roles = append(roles, "admin") // Ensure match

	attributes := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		attributes[fmt.Sprintf("attr_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	input := map[string]interface{}{
		"user": map[string]interface{}{
			"id":         "user123",
			"roles":      roles,
			"attributes": attributes,
		},
		"resource": map[string]interface{}{
			"type":       "document",
			"id":         "doc123",
			"attributes": attributes,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Evaluate(ctx, "rbac.allow", input)
		if err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

// ============================================================================
// Batch Evaluation Benchmarks
// ============================================================================

// BenchmarkEvaluator_BatchEvaluate benchmarks batch policy evaluation.
func BenchmarkEvaluator_BatchEvaluate(b *testing.B) {
	ctx := context.Background()

	// Create a temporary directory for policies
	tmpDir := b.TempDir()

	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: tmpDir,
		},
		DefaultDecisionPath: "rbac.allow",
	}

	evaluator, err := NewEvaluatorWithConfig(ctx, config)
	if err != nil {
		b.Fatalf("Failed to create evaluator: %v", err)
	}
	defer evaluator.Close(ctx)

	// Load the simple policy
	if err := evaluator.client.LoadPolicy(ctx, "rbac.rego", simplePolicy); err != nil {
		b.Fatalf("Failed to load policy: %v", err)
	}

	// Create batch of inputs
	inputs := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		inputs[i] = map[string]interface{}{
			"user": map[string]interface{}{
				"id":    fmt.Sprintf("user%d", i),
				"roles": []string{"admin"},
			},
			"resource": map[string]interface{}{
				"type": "document",
				"id":   fmt.Sprintf("doc%d", i),
			},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := evaluator.BatchEvaluate(ctx, "rbac.allow", inputs)
		if err != nil {
			b.Fatalf("BatchEvaluate() error = %v", err)
		}
	}
}

// ============================================================================
// Cache Statistics Benchmarks
// ============================================================================

// BenchmarkCache_Stats benchmarks cache statistics retrieval.
func BenchmarkCache_Stats(b *testing.B) {
	ctx := context.Background()

	cacheConfig := &CacheConfig{
		Enabled: true,
		MaxSize: 1000,
		TTL:     time.Hour,
	}

	cache, err := NewCache(cacheConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close(ctx)

	// Pre-populate cache
	for i := 0; i < 500; i++ {
		input := map[string]interface{}{
			"user": map[string]interface{}{
				"id": fmt.Sprintf("user%d", i),
			},
		}
		result := &Result{
			Decision: true,
			Allowed:  true,
		}
		cache.Set(ctx, fmt.Sprintf("path%d", i), input, result)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cache.Stats()
	}
}
