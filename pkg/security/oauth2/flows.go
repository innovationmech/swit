// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

// PKCEParams holds PKCE (Proof Key for Code Exchange) parameters.
type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
	Method        string // "S256" or "plain"
}

// FlowState represents the state of an OAuth2 authorization flow.
type FlowState struct {
	State         string
	Nonce         string
	PKCEVerifier  string
	RedirectURL   string
	OriginalScope []string
}

// FlowManager manages OAuth2 authorization flows with state, nonce, and PKCE support.
type FlowManager struct {
	mu     sync.RWMutex
	states map[string]*FlowState
}

// NewFlowManager creates a new flow manager.
func NewFlowManager() *FlowManager {
	return &FlowManager{
		states: make(map[string]*FlowState),
	}
}

// GeneratePKCEParams generates PKCE parameters (code verifier and code challenge).
// It uses the S256 method (SHA256) by default as recommended by RFC 7636.
func GeneratePKCEParams() (*PKCEParams, error) {
	return GeneratePKCEParamsWithMethod("S256")
}

// GeneratePKCEParamsWithMethod generates PKCE parameters with a specific method.
// Supported methods: "S256" (SHA256) and "plain".
func GeneratePKCEParamsWithMethod(method string) (*PKCEParams, error) {
	if method != "S256" && method != "plain" {
		return nil, fmt.Errorf("oauth2: unsupported PKCE method: %s", method)
	}

	// Generate code verifier
	codeVerifier, err := generateRandomString(43) // 43 characters is recommended
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to generate code verifier: %w", err)
	}

	// Generate code challenge
	var codeChallenge string
	if method == "S256" {
		// SHA256 hash of code verifier, base64url encoded
		hash := sha256.Sum256([]byte(codeVerifier))
		codeChallenge = base64.RawURLEncoding.EncodeToString(hash[:])
	} else {
		// Plain method: code challenge = code verifier
		codeChallenge = codeVerifier
	}

	return &PKCEParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        method,
	}, nil
}

// GenerateState generates a random state parameter for CSRF protection.
func GenerateState() (string, error) {
	return generateRandomString(32)
}

// GenerateNonce generates a random nonce parameter for replay attack protection.
func GenerateNonce() (string, error) {
	return generateRandomString(32)
}

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

// StartFlow initiates a new OAuth2 authorization flow and returns the state parameter.
func (fm *FlowManager) StartFlow(redirectURL string, scopes []string, usePKCE bool) (*FlowState, error) {
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to generate state: %w", err)
	}

	nonce, err := GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to generate nonce: %w", err)
	}

	flowState := &FlowState{
		State:         state,
		Nonce:         nonce,
		RedirectURL:   redirectURL,
		OriginalScope: scopes,
	}

	if usePKCE {
		pkceParams, err := GeneratePKCEParams()
		if err != nil {
			return nil, fmt.Errorf("oauth2: failed to generate PKCE params: %w", err)
		}
		flowState.PKCEVerifier = pkceParams.CodeVerifier
	}

	fm.mu.Lock()
	fm.states[state] = flowState
	fm.mu.Unlock()

	return flowState, nil
}

// ValidateState validates a state parameter and returns the associated flow state.
// It removes the state from the manager after validation (one-time use).
func (fm *FlowManager) ValidateState(state string) (*FlowState, error) {
	if state == "" {
		return nil, fmt.Errorf("oauth2: state parameter is empty")
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	flowState, ok := fm.states[state]
	if !ok {
		return nil, fmt.Errorf("oauth2: invalid or expired state parameter")
	}

	// Remove state after validation (one-time use)
	delete(fm.states, state)

	return flowState, nil
}

// ValidateNonce validates a nonce parameter against the expected value from the flow state.
func ValidateNonce(expected, actual string) error {
	if expected == "" {
		return fmt.Errorf("oauth2: expected nonce is empty")
	}
	if actual == "" {
		return fmt.Errorf("oauth2: actual nonce is empty")
	}
	if expected != actual {
		return fmt.Errorf("oauth2: nonce mismatch")
	}
	return nil
}

// CleanupExpiredStates removes flow states that are no longer needed.
// This should be called periodically to prevent memory leaks.
func (fm *FlowManager) CleanupExpiredStates() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// In a production system, you would track timestamps and remove expired states.
	// For now, we just clear all states.
	fm.states = make(map[string]*FlowState)
}

// GetStateCount returns the number of active flow states.
func (fm *FlowManager) GetStateCount() int {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return len(fm.states)
}

// AuthCodeURLWithPKCE generates an authorization URL with PKCE support.
func (c *Client) AuthCodeURLWithPKCE(state string, pkceParams *PKCEParams, opts ...oauth2.AuthCodeOption) string {
	if pkceParams != nil {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", pkceParams.CodeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", pkceParams.Method),
		)
	}
	return c.oauth2Config.AuthCodeURL(state, opts...)
}

// ExchangeWithPKCE exchanges an authorization code for a token with PKCE support.
func (c *Client) ExchangeWithPKCE(ctx context.Context, code string, codeVerifier string) (*oauth2.Token, error) {
	if codeVerifier != "" {
		return c.oauth2Config.Exchange(ctx,
			code,
			oauth2.SetAuthURLParam("code_verifier", codeVerifier),
		)
	}
	return c.Exchange(ctx, code)
}

// AuthCodeURLWithState generates an authorization URL with state and optional nonce.
func (c *Client) AuthCodeURLWithState(state string, nonce string, opts ...oauth2.AuthCodeOption) string {
	if nonce != "" {
		opts = append(opts, oauth2.SetAuthURLParam("nonce", nonce))
	}
	return c.oauth2Config.AuthCodeURL(state, opts...)
}

// AuthCodeURLWithFlow generates an authorization URL with full flow support (state, nonce, PKCE).
func (c *Client) AuthCodeURLWithFlow(flowState *FlowState, opts ...oauth2.AuthCodeOption) string {
	// Add nonce if present
	if flowState.Nonce != "" {
		opts = append(opts, oauth2.SetAuthURLParam("nonce", flowState.Nonce))
	}

	// Add PKCE if present - derive challenge from existing verifier
	if flowState.PKCEVerifier != "" {
		// Compute code_challenge from the existing code_verifier (S256 method)
		hash := sha256.Sum256([]byte(flowState.PKCEVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}

	return c.oauth2Config.AuthCodeURL(flowState.State, opts...)
}

// ExchangeWithFlow exchanges an authorization code with full flow support.
func (c *Client) ExchangeWithFlow(ctx context.Context, code string, flowState *FlowState) (*oauth2.Token, error) {
	if flowState.PKCEVerifier != "" {
		return c.ExchangeWithPKCE(ctx, code, flowState.PKCEVerifier)
	}
	return c.Exchange(ctx, code)
}
