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

package oauth2

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

// TestGeneratePKCEParams tests PKCE parameter generation.
func TestGeneratePKCEParams(t *testing.T) {
	t.Run("default S256 method", func(t *testing.T) {
		params, err := GeneratePKCEParams()
		if err != nil {
			t.Fatalf("GeneratePKCEParams() error = %v", err)
		}

		if params.CodeVerifier == "" {
			t.Error("CodeVerifier is empty")
		}
		if params.CodeChallenge == "" {
			t.Error("CodeChallenge is empty")
		}
		if params.Method != "S256" {
			t.Errorf("Method = %v, want S256", params.Method)
		}

		// Verify code challenge is correctly generated
		hash := sha256.Sum256([]byte(params.CodeVerifier))
		expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
		if params.CodeChallenge != expectedChallenge {
			t.Error("CodeChallenge does not match expected SHA256 hash")
		}
	})

	t.Run("S256 method", func(t *testing.T) {
		params, err := GeneratePKCEParamsWithMethod("S256")
		if err != nil {
			t.Fatalf("GeneratePKCEParamsWithMethod('S256') error = %v", err)
		}

		if params.Method != "S256" {
			t.Errorf("Method = %v, want S256", params.Method)
		}
	})

	t.Run("plain method", func(t *testing.T) {
		params, err := GeneratePKCEParamsWithMethod("plain")
		if err != nil {
			t.Fatalf("GeneratePKCEParamsWithMethod('plain') error = %v", err)
		}

		if params.Method != "plain" {
			t.Errorf("Method = %v, want plain", params.Method)
		}
		if params.CodeChallenge != params.CodeVerifier {
			t.Error("For plain method, CodeChallenge should equal CodeVerifier")
		}
	})

	t.Run("unsupported method", func(t *testing.T) {
		_, err := GeneratePKCEParamsWithMethod("invalid")
		if err == nil {
			t.Error("GeneratePKCEParamsWithMethod('invalid') expected error, got nil")
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		params1, _ := GeneratePKCEParams()
		params2, _ := GeneratePKCEParams()

		if params1.CodeVerifier == params2.CodeVerifier {
			t.Error("Generated PKCE params should be unique")
		}
	})
}

// TestGenerateState tests state parameter generation.
func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}

	if state1 == "" {
		t.Error("GenerateState() returned empty string")
	}

	if len(state1) != 32 {
		t.Errorf("GenerateState() length = %v, want 32", len(state1))
	}

	// Test uniqueness
	state2, _ := GenerateState()
	if state1 == state2 {
		t.Error("GenerateState() should generate unique values")
	}
}

// TestGenerateNonce tests nonce parameter generation.
func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce() error = %v", err)
	}

	if nonce1 == "" {
		t.Error("GenerateNonce() returned empty string")
	}

	if len(nonce1) != 32 {
		t.Errorf("GenerateNonce() length = %v, want 32", len(nonce1))
	}

	// Test uniqueness
	nonce2, _ := GenerateNonce()
	if nonce1 == nonce2 {
		t.Error("GenerateNonce() should generate unique values")
	}
}

// TestFlowManager tests the flow manager.
func TestFlowManager(t *testing.T) {
	t.Run("start and validate flow without PKCE", func(t *testing.T) {
		fm := NewFlowManager()

		flowState, err := fm.StartFlow("https://example.com/callback", []string{"openid", "profile"}, false)
		if err != nil {
			t.Fatalf("StartFlow() error = %v", err)
		}

		if flowState.State == "" {
			t.Error("FlowState.State is empty")
		}
		if flowState.Nonce == "" {
			t.Error("FlowState.Nonce is empty")
		}
		if flowState.PKCEVerifier != "" {
			t.Error("FlowState.PKCEVerifier should be empty when usePKCE is false")
		}
		if flowState.RedirectURL != "https://example.com/callback" {
			t.Errorf("FlowState.RedirectURL = %v, want %v", flowState.RedirectURL, "https://example.com/callback")
		}

		// Validate state
		validatedState, err := fm.ValidateState(flowState.State)
		if err != nil {
			t.Fatalf("ValidateState() error = %v", err)
		}

		if validatedState.State != flowState.State {
			t.Errorf("ValidateState() state = %v, want %v", validatedState.State, flowState.State)
		}

		// Try to validate the same state again (should fail - one-time use)
		_, err = fm.ValidateState(flowState.State)
		if err == nil {
			t.Error("ValidateState() should fail for already used state")
		}
	})

	t.Run("start and validate flow with PKCE", func(t *testing.T) {
		fm := NewFlowManager()

		flowState, err := fm.StartFlow("https://example.com/callback", []string{"openid", "profile"}, true)
		if err != nil {
			t.Fatalf("StartFlow() error = %v", err)
		}

		if flowState.PKCEVerifier == "" {
			t.Error("FlowState.PKCEVerifier should not be empty when usePKCE is true")
		}
	})

	t.Run("validate invalid state", func(t *testing.T) {
		fm := NewFlowManager()

		_, err := fm.ValidateState("invalid-state")
		if err == nil {
			t.Error("ValidateState() should fail for invalid state")
		}
	})

	t.Run("validate empty state", func(t *testing.T) {
		fm := NewFlowManager()

		_, err := fm.ValidateState("")
		if err == nil {
			t.Error("ValidateState() should fail for empty state")
		}
	})

	t.Run("get state count", func(t *testing.T) {
		fm := NewFlowManager()

		if fm.GetStateCount() != 0 {
			t.Errorf("GetStateCount() = %v, want 0", fm.GetStateCount())
		}

		fm.StartFlow("https://example.com/callback", []string{"openid"}, false)
		if fm.GetStateCount() != 1 {
			t.Errorf("GetStateCount() = %v, want 1", fm.GetStateCount())
		}

		fm.StartFlow("https://example.com/callback", []string{"openid"}, false)
		if fm.GetStateCount() != 2 {
			t.Errorf("GetStateCount() = %v, want 2", fm.GetStateCount())
		}
	})

	t.Run("cleanup expired states", func(t *testing.T) {
		fm := NewFlowManager()

		fm.StartFlow("https://example.com/callback", []string{"openid"}, false)
		fm.StartFlow("https://example.com/callback", []string{"openid"}, false)

		if fm.GetStateCount() != 2 {
			t.Errorf("GetStateCount() = %v, want 2", fm.GetStateCount())
		}

		fm.CleanupExpiredStates()

		if fm.GetStateCount() != 0 {
			t.Errorf("GetStateCount() = %v, want 0 after cleanup", fm.GetStateCount())
		}
	})
}

// TestValidateNonce tests nonce validation.
func TestValidateNonce(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		wantErr  bool
	}{
		{
			name:     "valid nonce",
			expected: "test-nonce",
			actual:   "test-nonce",
			wantErr:  false,
		},
		{
			name:     "nonce mismatch",
			expected: "nonce1",
			actual:   "nonce2",
			wantErr:  true,
		},
		{
			name:     "empty expected nonce",
			expected: "",
			actual:   "test-nonce",
			wantErr:  true,
		},
		{
			name:     "empty actual nonce",
			expected: "test-nonce",
			actual:   "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonce(tt.expected, tt.actual)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNonce() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestClientAuthCodeURLWithPKCE tests authorization URL generation with PKCE.
func TestClientAuthCodeURLWithPKCE(t *testing.T) {
	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		RedirectURL:  "https://example.com/callback",
		UseDiscovery: false,
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		JWKSURL:      "https://example.com/jwks",
		Scopes:       []string{"openid", "profile"},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Run("with PKCE params", func(t *testing.T) {
		pkceParams, err := GeneratePKCEParams()
		if err != nil {
			t.Fatalf("GeneratePKCEParams() error = %v", err)
		}

		authURL := client.AuthCodeURLWithPKCE("test-state", pkceParams)

		if authURL == "" {
			t.Error("AuthCodeURLWithPKCE returned empty string")
		}

		if !strings.Contains(authURL, "code_challenge=") {
			t.Error("AuthCodeURLWithPKCE missing code_challenge parameter")
		}

		if !strings.Contains(authURL, "code_challenge_method=") {
			t.Error("AuthCodeURLWithPKCE missing code_challenge_method parameter")
		}

		if !strings.Contains(authURL, "state=test-state") {
			t.Error("AuthCodeURLWithPKCE missing state parameter")
		}
	})

	t.Run("without PKCE params", func(t *testing.T) {
		authURL := client.AuthCodeURLWithPKCE("test-state", nil)

		if authURL == "" {
			t.Error("AuthCodeURLWithPKCE returned empty string")
		}

		if strings.Contains(authURL, "code_challenge=") {
			t.Error("AuthCodeURLWithPKCE should not have code_challenge parameter when pkceParams is nil")
		}
	})
}

// TestClientAuthCodeURLWithState tests authorization URL generation with state and nonce.
func TestClientAuthCodeURLWithState(t *testing.T) {
	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		RedirectURL:  "https://example.com/callback",
		UseDiscovery: false,
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		JWKSURL:      "https://example.com/jwks",
		Scopes:       []string{"openid", "profile"},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Run("with nonce", func(t *testing.T) {
		authURL := client.AuthCodeURLWithState("test-state", "test-nonce")

		if authURL == "" {
			t.Error("AuthCodeURLWithState returned empty string")
		}

		if !strings.Contains(authURL, "nonce=test-nonce") {
			t.Error("AuthCodeURLWithState missing nonce parameter")
		}

		if !strings.Contains(authURL, "state=test-state") {
			t.Error("AuthCodeURLWithState missing state parameter")
		}
	})

	t.Run("without nonce", func(t *testing.T) {
		authURL := client.AuthCodeURLWithState("test-state", "")

		if authURL == "" {
			t.Error("AuthCodeURLWithState returned empty string")
		}

		if strings.Contains(authURL, "nonce=") {
			t.Error("AuthCodeURLWithState should not have nonce parameter when nonce is empty")
		}
	})
}

// TestClientAuthCodeURLWithFlow tests authorization URL generation with full flow support.
func TestClientAuthCodeURLWithFlow(t *testing.T) {
	config := &Config{
		Enabled:      true,
		Provider:     "custom",
		ClientID:     "test-client",
		ClientSecret: "secret",
		RedirectURL:  "https://example.com/callback",
		UseDiscovery: false,
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		JWKSURL:      "https://example.com/jwks",
		Scopes:       []string{"openid", "profile"},
	}

	ctx := context.Background()
	client, err := NewClient(ctx, config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Run("with full flow (state, nonce, PKCE)", func(t *testing.T) {
		flowState := &FlowState{
			State:         "test-state",
			Nonce:         "test-nonce",
			PKCEVerifier:  "test-verifier",
			RedirectURL:   "https://example.com/callback",
			OriginalScope: []string{"openid", "profile"},
		}

		// Store original verifier to ensure it doesn't change
		originalVerifier := flowState.PKCEVerifier

		authURL := client.AuthCodeURLWithFlow(flowState)

		if authURL == "" {
			t.Error("AuthCodeURLWithFlow returned empty string")
		}

		if !strings.Contains(authURL, "state=test-state") {
			t.Error("AuthCodeURLWithFlow missing state parameter")
		}

		if !strings.Contains(authURL, "nonce=test-nonce") {
			t.Error("AuthCodeURLWithFlow missing nonce parameter")
		}

		if !strings.Contains(authURL, "code_challenge=") {
			t.Error("AuthCodeURLWithFlow missing code_challenge parameter")
		}

		// Verify that the verifier wasn't modified
		if flowState.PKCEVerifier != originalVerifier {
			t.Errorf("AuthCodeURLWithFlow modified PKCEVerifier: got %v, want %v", flowState.PKCEVerifier, originalVerifier)
		}

		// Call again to verify idempotency - URL should be identical
		authURL2 := client.AuthCodeURLWithFlow(flowState)
		if authURL != authURL2 {
			t.Error("AuthCodeURLWithFlow is not idempotent - generates different URLs on subsequent calls")
		}
	})

	t.Run("without PKCE", func(t *testing.T) {
		flowState := &FlowState{
			State:         "test-state",
			Nonce:         "test-nonce",
			RedirectURL:   "https://example.com/callback",
			OriginalScope: []string{"openid", "profile"},
		}

		authURL := client.AuthCodeURLWithFlow(flowState)

		if authURL == "" {
			t.Error("AuthCodeURLWithFlow returned empty string")
		}

		if strings.Contains(authURL, "code_challenge=") {
			t.Error("AuthCodeURLWithFlow should not have code_challenge when PKCEVerifier is empty")
		}
	})
}
