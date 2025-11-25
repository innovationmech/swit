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
	"testing"
)

const rbacPolicy = `
package rbac

import rego.v1

default allow = false

allow if {
    input.subject.user == "admin"
}

allow if {
    some role in input.subject.roles
    role == "admin"
}

allow if {
    input.action == "read"
    some role in input.subject.roles
    role == "reader"
}
`

const abacPolicy = `
package abac

import rego.v1

default allow = false

allow if {
    input.subject.user == input.resource.owner
}

allow if {
    input.subject.attributes.department == "engineering"
    input.action == "read"
}

allow if {
    input.environment.ip_address == "192.168.1.1"
    input.action == "read"
}
`

func TestNewEvaluator(t *testing.T) {
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

	evaluator := NewEvaluator(client)
	if evaluator == nil {
		t.Fatal("NewEvaluator returned nil")
	}

	if evaluator.client == nil {
		t.Error("Evaluator client is nil")
	}
}

func TestNewEvaluatorWithConfig(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
	}

	evaluator, err := NewEvaluatorWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}
	defer evaluator.Close(ctx)

	if evaluator == nil {
		t.Fatal("NewEvaluatorWithConfig returned nil")
	}

	if evaluator.config != config {
		t.Error("Evaluator config not set correctly")
	}
}

func TestNewEvaluatorWithConfigInvalidConfig(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		Mode: "invalid",
	}

	_, err := NewEvaluatorWithConfig(ctx, config)
	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}
}

func TestEvaluatorEvaluate(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)
	defer evaluator.Close(ctx)

	// Load test policy
	err := evaluator.client.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		input       interface{}
		wantAllowed bool
		wantErr     bool
	}{
		{
			name: "allowed - alice read",
			path: "test.allow",
			input: map[string]interface{}{
				"user":   "alice",
				"action": "read",
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "denied - bob read",
			path: "test.allow",
			input: map[string]interface{}{
				"user":   "bob",
				"action": "read",
			},
			wantAllowed: false,
			wantErr:     false,
		},
		{
			name: "denied - alice write",
			path: "test.allow",
			input: map[string]interface{}{
				"user":   "alice",
				"action": "write",
			},
			wantAllowed: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(ctx, tt.path, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate() allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}
		})
	}
}

func TestEvaluatorIsAllowed(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)
	defer evaluator.Close(ctx)

	// Load test policy
	err := evaluator.client.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		input       interface{}
		wantAllowed bool
		wantErr     bool
	}{
		{
			name: "allowed",
			path: "test.allow",
			input: map[string]interface{}{
				"user":   "alice",
				"action": "read",
			},
			wantAllowed: true,
			wantErr:     false,
		},
		{
			name: "denied",
			path: "test.allow",
			input: map[string]interface{}{
				"user":   "bob",
				"action": "read",
			},
			wantAllowed: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := evaluator.IsAllowed(ctx, tt.path, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsAllowed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && allowed != tt.wantAllowed {
				t.Errorf("IsAllowed() = %v, want %v", allowed, tt.wantAllowed)
			}
		})
	}
}

func TestEvaluatorEvaluateWithDefault(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)
	defer evaluator.Close(ctx)

	// Load test policy
	err := evaluator.client.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		input        interface{}
		defaultValue bool
		want         bool
	}{
		{
			name: "allowed - returns true",
			path: "test.allow",
			input: map[string]interface{}{
				"user":   "alice",
				"action": "read",
			},
			defaultValue: false,
			want:         true,
		},
		{
			name: "denied - returns false",
			path: "test.allow",
			input: map[string]interface{}{
				"user":   "bob",
				"action": "read",
			},
			defaultValue: true,
			want:         false,
		},
		{
			name: "nonexistent path - returns result from policy evaluation (false)",
			path: "nonexistent.path",
			input: map[string]interface{}{
				"user":   "alice",
				"action": "read",
			},
			defaultValue: true,
			want:         false, // OPA returns false for nonexistent paths, not an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.EvaluateWithDefault(ctx, tt.path, tt.input, tt.defaultValue)
			if result != tt.want {
				t.Errorf("EvaluateWithDefault() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestEvaluatorBatchEvaluate(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)
	defer evaluator.Close(ctx)

	// Load test policy
	err := evaluator.client.LoadPolicy(ctx, "test.rego", testPolicy)
	if err != nil {
		t.Fatalf("Failed to load policy: %v", err)
	}

	inputs := []interface{}{
		map[string]interface{}{
			"user":   "alice",
			"action": "read",
		},
		map[string]interface{}{
			"user":   "bob",
			"action": "read",
		},
		map[string]interface{}{
			"user":   "alice",
			"action": "write",
		},
	}

	results, err := evaluator.BatchEvaluate(ctx, "test.allow", inputs)
	if err != nil {
		t.Fatalf("BatchEvaluate() error = %v", err)
	}

	if len(results) != len(inputs) {
		t.Errorf("BatchEvaluate() returned %d results, want %d", len(results), len(inputs))
	}

	expectedAllowed := []bool{true, false, false}
	for i, result := range results {
		if result.Allowed != expectedAllowed[i] {
			t.Errorf("BatchEvaluate() result[%d].Allowed = %v, want %v", i, result.Allowed, expectedAllowed[i])
		}
	}
}

func TestEvaluatorBatchEvaluateEmpty(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)
	defer evaluator.Close(ctx)

	results, err := evaluator.BatchEvaluate(ctx, "test.allow", []interface{}{})
	if err != nil {
		t.Fatalf("BatchEvaluate() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("BatchEvaluate() returned %d results, want 0", len(results))
	}
}

func TestEvaluatorEvaluateRBAC(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)
	defer evaluator.Close(ctx)

	// Load RBAC policy
	err := evaluator.client.LoadPolicy(ctx, "rbac.rego", rbacPolicy)
	if err != nil {
		t.Fatalf("Failed to load RBAC policy: %v", err)
	}

	tests := []struct {
		name        string
		input       *RBACInput
		wantAllowed bool
	}{
		{
			name: "admin user allowed",
			input: &RBACInput{
				Subject: &Subject{
					User:  "admin",
					Roles: []string{},
				},
				Action:   "write",
				Resource: "document",
			},
			wantAllowed: true,
		},
		{
			name: "admin role allowed",
			input: &RBACInput{
				Subject: &Subject{
					User:  "bob",
					Roles: []string{"admin"},
				},
				Action:   "write",
				Resource: "document",
			},
			wantAllowed: true,
		},
		{
			name: "reader role read allowed",
			input: &RBACInput{
				Subject: &Subject{
					User:  "charlie",
					Roles: []string{"reader"},
				},
				Action:   "read",
				Resource: "document",
			},
			wantAllowed: true,
		},
		{
			name: "reader role write denied",
			input: &RBACInput{
				Subject: &Subject{
					User:  "charlie",
					Roles: []string{"reader"},
				},
				Action:   "write",
				Resource: "document",
			},
			wantAllowed: false,
		},
		{
			name: "no roles denied",
			input: &RBACInput{
				Subject: &Subject{
					User:  "guest",
					Roles: []string{},
				},
				Action:   "read",
				Resource: "document",
			},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateRBAC(ctx, tt.input)
			if err != nil {
				t.Fatalf("EvaluateRBAC() error = %v", err)
			}
			if result.Allowed != tt.wantAllowed {
				t.Errorf("EvaluateRBAC() allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}
		})
	}
}

func TestEvaluatorEvaluateRBACWithConfig(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
		DefaultDecisionPath: "rbac.allow",
	}

	evaluator, err := NewEvaluatorWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}
	defer evaluator.Close(ctx)

	// Load RBAC policy
	err = evaluator.client.LoadPolicy(ctx, "rbac.rego", rbacPolicy)
	if err != nil {
		t.Fatalf("Failed to load RBAC policy: %v", err)
	}

	input := &RBACInput{
		Subject: &Subject{
			User:  "admin",
			Roles: []string{},
		},
		Action:   "write",
		Resource: "document",
	}

	result, err := evaluator.EvaluateRBAC(ctx, input)
	if err != nil {
		t.Fatalf("EvaluateRBAC() error = %v", err)
	}
	if !result.Allowed {
		t.Error("EvaluateRBAC() should allow admin user")
	}
}

func TestEvaluatorEvaluateABAC(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)
	defer evaluator.Close(ctx)

	// Load ABAC policy
	err := evaluator.client.LoadPolicy(ctx, "abac.rego", abacPolicy)
	if err != nil {
		t.Fatalf("Failed to load ABAC policy: %v", err)
	}

	tests := []struct {
		name        string
		input       *ABACInput
		wantAllowed bool
	}{
		{
			name: "owner allowed",
			input: &ABACInput{
				Subject: &Subject{
					User: "alice",
				},
				Action: "write",
				Resource: &Resource{
					Type:  "document",
					ID:    "doc1",
					Owner: "alice",
				},
			},
			wantAllowed: true,
		},
		{
			name: "engineering department read allowed",
			input: &ABACInput{
				Subject: &Subject{
					User: "bob",
					Attributes: map[string]interface{}{
						"department": "engineering",
					},
				},
				Action: "read",
				Resource: &Resource{
					Type: "document",
					ID:   "doc1",
				},
			},
			wantAllowed: true,
		},
		{
			name: "engineering department write denied",
			input: &ABACInput{
				Subject: &Subject{
					User: "bob",
					Attributes: map[string]interface{}{
						"department": "engineering",
					},
				},
				Action: "write",
				Resource: &Resource{
					Type: "document",
					ID:   "doc1",
				},
			},
			wantAllowed: false,
		},
		{
			name: "ip address allowed",
			input: &ABACInput{
				Subject: &Subject{
					User: "charlie",
				},
				Action: "read",
				Resource: &Resource{
					Type: "document",
					ID:   "doc1",
				},
				Environment: &Environment{
					IPAddress: "192.168.1.1",
				},
			},
			wantAllowed: true,
		},
		{
			name: "wrong ip address denied",
			input: &ABACInput{
				Subject: &Subject{
					User: "charlie",
				},
				Action: "read",
				Resource: &Resource{
					Type: "document",
					ID:   "doc1",
				},
				Environment: &Environment{
					IPAddress: "10.0.0.1",
				},
			},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateABAC(ctx, tt.input)
			if err != nil {
				t.Fatalf("EvaluateABAC() error = %v", err)
			}
			if result.Allowed != tt.wantAllowed {
				t.Errorf("EvaluateABAC() allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}
		})
	}
}

func TestEvaluatorEvaluateABACWithConfig(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		Mode: ModeEmbedded,
		EmbeddedConfig: &EmbeddedConfig{
			PolicyDir: t.TempDir(),
		},
		DefaultDecisionPath: "abac.allow",
	}

	evaluator, err := NewEvaluatorWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}
	defer evaluator.Close(ctx)

	// Load ABAC policy
	err = evaluator.client.LoadPolicy(ctx, "abac.rego", abacPolicy)
	if err != nil {
		t.Fatalf("Failed to load ABAC policy: %v", err)
	}

	input := &ABACInput{
		Subject: &Subject{
			User: "alice",
		},
		Action: "write",
		Resource: &Resource{
			Type:  "document",
			ID:    "doc1",
			Owner: "alice",
		},
	}

	result, err := evaluator.EvaluateABAC(ctx, input)
	if err != nil {
		t.Fatalf("EvaluateABAC() error = %v", err)
	}
	if !result.Allowed {
		t.Error("EvaluateABAC() should allow owner")
	}
}

func TestEvaluatorClose(t *testing.T) {
	ctx := context.Background()
	evaluator := createTestEvaluator(t)

	err := evaluator.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// createTestEvaluator creates an evaluator for testing
func createTestEvaluator(t *testing.T) *Evaluator {
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

	return NewEvaluator(client)
}
