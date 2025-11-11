// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"context"
	"sync"
	"time"
)

// TokenBlacklist is an interface for token blacklist implementations.
type TokenBlacklist interface {
	// IsBlacklisted checks if a token is blacklisted.
	IsBlacklisted(ctx context.Context, token string) (bool, error)

	// Blacklist adds a token to the blacklist.
	Blacklist(ctx context.Context, token string, expiry time.Time) error

	// Remove removes a token from the blacklist.
	Remove(ctx context.Context, token string) error

	// Clear clears all tokens from the blacklist.
	Clear(ctx context.Context) error

	// Size returns the number of blacklisted tokens.
	Size() int
}

// InMemoryBlacklist is an in-memory implementation of TokenBlacklist.
type InMemoryBlacklist struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
	stopCh chan struct{}
}

// NewInMemoryBlacklist creates a new in-memory blacklist.
func NewInMemoryBlacklist() *InMemoryBlacklist {
	bl := &InMemoryBlacklist{
		tokens: make(map[string]time.Time),
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go bl.startCleanup()

	return bl
}

// IsBlacklisted checks if a token is blacklisted.
func (b *InMemoryBlacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	expiry, exists := b.tokens[token]
	if !exists {
		return false, nil
	}

	// Check if token has expired
	if time.Now().After(expiry) {
		return false, nil
	}

	return true, nil
}

// Blacklist adds a token to the blacklist.
func (b *InMemoryBlacklist) Blacklist(ctx context.Context, token string, expiry time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens[token] = expiry
	return nil
}

// Remove removes a token from the blacklist.
func (b *InMemoryBlacklist) Remove(ctx context.Context, token string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.tokens, token)
	return nil
}

// Clear clears all tokens from the blacklist.
func (b *InMemoryBlacklist) Clear(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens = make(map[string]time.Time)
	return nil
}

// Size returns the number of blacklisted tokens.
func (b *InMemoryBlacklist) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.tokens)
}

// Stop stops the cleanup goroutine.
func (b *InMemoryBlacklist) Stop() {
	close(b.stopCh)
}

// startCleanup starts a goroutine that periodically removes expired tokens.
func (b *InMemoryBlacklist) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.cleanup()
		case <-b.stopCh:
			return
		}
	}
}

// cleanup removes expired tokens from the blacklist.
func (b *InMemoryBlacklist) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for token, expiry := range b.tokens {
		if now.After(expiry) {
			delete(b.tokens, token)
		}
	}
}

// WithBlacklist is a validator option that sets the token blacklist.
func WithBlacklist(blacklist TokenBlacklist) ValidatorOption {
	return func(v *Validator) {
		v.blacklist = blacklist
	}
}

