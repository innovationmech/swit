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

package jwt

import (
	"context"
	"testing"
	"time"
)

func TestNewInMemoryBlacklist(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	if bl == nil {
		t.Fatal("Expected blacklist to be non-nil")
	}
	if bl.Size() != 0 {
		t.Errorf("Expected initial size to be 0, got %d", bl.Size())
	}
}

func TestInMemoryBlacklist_Blacklist(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	token := "test-token"
	expiry := time.Now().Add(1 * time.Hour)

	err := bl.Blacklist(ctx, token, expiry)
	if err != nil {
		t.Errorf("Blacklist() error = %v", err)
	}

	if bl.Size() != 1 {
		t.Errorf("Expected size to be 1, got %d", bl.Size())
	}
}

func TestInMemoryBlacklist_IsBlacklisted(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	token := "test-token"
	expiry := time.Now().Add(1 * time.Hour)

	// Token should not be blacklisted initially
	blacklisted, err := bl.IsBlacklisted(ctx, token)
	if err != nil {
		t.Errorf("IsBlacklisted() error = %v", err)
	}
	if blacklisted {
		t.Error("Expected token to not be blacklisted")
	}

	// Blacklist the token
	err = bl.Blacklist(ctx, token, expiry)
	if err != nil {
		t.Errorf("Blacklist() error = %v", err)
	}

	// Token should now be blacklisted
	blacklisted, err = bl.IsBlacklisted(ctx, token)
	if err != nil {
		t.Errorf("IsBlacklisted() error = %v", err)
	}
	if !blacklisted {
		t.Error("Expected token to be blacklisted")
	}
}

func TestInMemoryBlacklist_Remove(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	token := "test-token"
	expiry := time.Now().Add(1 * time.Hour)

	// Blacklist the token
	err := bl.Blacklist(ctx, token, expiry)
	if err != nil {
		t.Errorf("Blacklist() error = %v", err)
	}

	// Remove the token
	err = bl.Remove(ctx, token)
	if err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	// Token should no longer be blacklisted
	blacklisted, err := bl.IsBlacklisted(ctx, token)
	if err != nil {
		t.Errorf("IsBlacklisted() error = %v", err)
	}
	if blacklisted {
		t.Error("Expected token to not be blacklisted after removal")
	}

	if bl.Size() != 0 {
		t.Errorf("Expected size to be 0, got %d", bl.Size())
	}
}

func TestInMemoryBlacklist_Clear(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	expiry := time.Now().Add(1 * time.Hour)

	// Blacklist multiple tokens
	tokens := []string{"token1", "token2", "token3"}
	for _, token := range tokens {
		err := bl.Blacklist(ctx, token, expiry)
		if err != nil {
			t.Errorf("Blacklist() error = %v", err)
		}
	}

	if bl.Size() != 3 {
		t.Errorf("Expected size to be 3, got %d", bl.Size())
	}

	// Clear all tokens
	err := bl.Clear(ctx)
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	if bl.Size() != 0 {
		t.Errorf("Expected size to be 0 after clear, got %d", bl.Size())
	}
}

func TestInMemoryBlacklist_ExpiredTokens(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()
	token := "test-token"
	expiry := time.Now().Add(-1 * time.Hour) // Already expired

	// Blacklist the token with expired time
	err := bl.Blacklist(ctx, token, expiry)
	if err != nil {
		t.Errorf("Blacklist() error = %v", err)
	}

	// Token should not be considered blacklisted because it's expired
	blacklisted, err := bl.IsBlacklisted(ctx, token)
	if err != nil {
		t.Errorf("IsBlacklisted() error = %v", err)
	}
	if blacklisted {
		t.Error("Expected expired token to not be blacklisted")
	}
}

func TestInMemoryBlacklist_Cleanup(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	ctx := context.Background()

	// Add tokens with different expiry times
	token1 := "token1"
	expiry1 := time.Now().Add(-1 * time.Hour) // Already expired
	err := bl.Blacklist(ctx, token1, expiry1)
	if err != nil {
		t.Errorf("Blacklist() error = %v", err)
	}

	token2 := "token2"
	expiry2 := time.Now().Add(1 * time.Hour) // Not expired
	err = bl.Blacklist(ctx, token2, expiry2)
	if err != nil {
		t.Errorf("Blacklist() error = %v", err)
	}

	// Size should be 2 before cleanup
	if bl.Size() != 2 {
		t.Errorf("Expected size to be 2, got %d", bl.Size())
	}

	// Manually trigger cleanup
	bl.cleanup()

	// Size should be 1 after cleanup (only non-expired token remains)
	if bl.Size() != 1 {
		t.Errorf("Expected size to be 1 after cleanup, got %d", bl.Size())
	}

	// token2 should still be blacklisted
	blacklisted, err := bl.IsBlacklisted(ctx, token2)
	if err != nil {
		t.Errorf("IsBlacklisted() error = %v", err)
	}
	if !blacklisted {
		t.Error("Expected non-expired token to still be blacklisted")
	}
}

func TestWithBlacklist(t *testing.T) {
	bl := NewInMemoryBlacklist()
	defer bl.Stop()

	config := &Config{
		Secret: "test-secret",
	}

	validator, err := NewValidator(config, WithBlacklist(bl))
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	if validator.blacklist == nil {
		t.Error("Expected validator to have blacklist configured")
	}

	if validator.blacklist != bl {
		t.Error("Expected validator blacklist to match provided blacklist")
	}
}
