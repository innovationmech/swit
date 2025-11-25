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
	"crypto/ecdsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewJWKSCache(t *testing.T) {
	// Create a test server that returns a valid JWKS
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "test-key-1",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	tests := []struct {
		name    string
		config  *JWKSCacheConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &JWKSCacheConfig{
				URL:         server.URL,
				RefreshTTL:  1 * time.Hour,
				AutoRefresh: false,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty URL",
			config: &JWKSCacheConfig{
				URL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewJWKSCache(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJWKSCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if cache != nil {
				cache.Stop()
			}
		})
	}
}

func TestJWKSCache_GetKey(t *testing.T) {
	// Create a test server that returns a valid JWKS
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "test-key-1",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache, err := NewJWKSCache(&JWKSCacheConfig{
		URL:         server.URL,
		RefreshTTL:  1 * time.Hour,
		AutoRefresh: false,
	})
	if err != nil {
		t.Fatalf("Failed to create JWKS cache: %v", err)
	}
	defer cache.Stop()

	tests := []struct {
		name    string
		kid     string
		wantErr bool
	}{
		{
			name:    "existing key",
			kid:     "test-key-1",
			wantErr: false,
		},
		{
			name:    "non-existent key",
			kid:     "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := cache.GetKey(tt.kid)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && key == nil {
				t.Error("Expected key to be non-nil")
			}
		})
	}
}

func TestJWKSCache_Refresh(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "test-key-1",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache, err := NewJWKSCache(&JWKSCacheConfig{
		URL:            server.URL,
		RefreshTTL:     1 * time.Hour,
		AutoRefresh:    false,
		MinRefreshWait: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create JWKS cache: %v", err)
	}
	defer cache.Stop()

	// Initial call count should be 1 (from NewJWKSCache)
	if callCount != 1 {
		t.Errorf("Expected 1 initial call, got %d", callCount)
	}

	// Refresh should succeed
	ctx := context.Background()
	time.Sleep(150 * time.Millisecond) // Wait for min refresh wait
	if err := cache.Refresh(ctx); err != nil {
		t.Errorf("Refresh() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls after refresh, got %d", callCount)
	}
}

func TestJWKSCache_AutoRefresh(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "test-key-1",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache, err := NewJWKSCache(&JWKSCacheConfig{
		URL:            server.URL,
		RefreshTTL:     200 * time.Millisecond,
		AutoRefresh:    true,
		MinRefreshWait: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Failed to create JWKS cache: %v", err)
	}
	defer cache.Stop()

	// Initial call count should be 1
	if callCount != 1 {
		t.Errorf("Expected 1 initial call, got %d", callCount)
	}

	// Wait for auto-refresh to happen
	time.Sleep(300 * time.Millisecond)

	// Should have at least one more refresh
	if callCount < 2 {
		t.Errorf("Expected at least 2 calls after auto-refresh, got %d", callCount)
	}
}

func TestJWKSCache_KeyCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "key-1",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
				{
					Kid: "key-2",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache, err := NewJWKSCache(&JWKSCacheConfig{
		URL:         server.URL,
		RefreshTTL:  1 * time.Hour,
		AutoRefresh: false,
	})
	if err != nil {
		t.Fatalf("Failed to create JWKS cache: %v", err)
	}
	defer cache.Stop()

	count := cache.KeyCount()
	if count != 2 {
		t.Errorf("Expected 2 keys, got %d", count)
	}
}

func TestJWKSCache_LastRefreshTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "test-key-1",
					Kty: "RSA",
					Alg: "RS256",
					Use: "sig",
					N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
					E:   "AQAB",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	before := time.Now()
	cache, err := NewJWKSCache(&JWKSCacheConfig{
		URL:         server.URL,
		RefreshTTL:  1 * time.Hour,
		AutoRefresh: false,
	})
	if err != nil {
		t.Fatalf("Failed to create JWKS cache: %v", err)
	}
	defer cache.Stop()
	after := time.Now()

	lastRefresh := cache.LastRefreshTime()
	if lastRefresh.Before(before) || lastRefresh.After(after) {
		t.Errorf("LastRefreshTime() = %v, expected between %v and %v", lastRefresh, before, after)
	}
}

func TestJWKSCacheConfig_SetDefaults(t *testing.T) {
	config := &JWKSCacheConfig{
		URL: "https://example.com/.well-known/jwks.json",
	}
	config.SetDefaults()

	if config.RefreshTTL != 1*time.Hour {
		t.Errorf("Expected default RefreshTTL to be 1 hour, got %v", config.RefreshTTL)
	}
	if config.HTTPClient == nil {
		t.Error("Expected default HTTPClient to be set")
	}
	if config.MinRefreshWait != 1*time.Minute {
		t.Errorf("Expected default MinRefreshWait to be 1 minute, got %v", config.MinRefreshWait)
	}
}

func TestJWKSCacheConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *JWKSCacheConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &JWKSCacheConfig{
				URL: "https://example.com/.well-known/jwks.json",
			},
			wantErr: false,
		},
		{
			name: "empty URL",
			config: &JWKSCacheConfig{
				URL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWKSCache_ECKeys(t *testing.T) {
	// Create a test server that returns JWKS with EC keys
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := JWKSet{
			Keys: []JWK{
				{
					Kid: "ec-key-p256",
					Kty: "EC",
					Alg: "ES256",
					Use: "sig",
					Crv: "P-256",
					// Valid P-256 key
					X: "WKn-ZIGevcwGIyyrzFoZNBdaq9_TsqzGl96oc0CWuis",
					Y: "y77t-RvAHRKTsSGdIYUfweuOvwrvDD-Q3Hv5J0fSKbE",
				},
				{
					Kid: "ec-key-p384",
					Kty: "EC",
					Alg: "ES384",
					Use: "sig",
					Crv: "P-384",
					// Valid P-384 key
					X: "N81v7PoRSYA49_mZs6i0ZA1d0GM_JvVSUN5vy_gseBn27vGYvg93egeX9CQ8u6lW",
					Y: "Bxe0kV_Sfr2S9EOFz1U7AHIrcfWStljsaBjuibefei0jYog4Xw9BqdncwZxQsDVM",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache, err := NewJWKSCache(&JWKSCacheConfig{
		URL:         server.URL,
		RefreshTTL:  1 * time.Hour,
		AutoRefresh: false,
	})
	if err != nil {
		t.Fatalf("Failed to create JWKS cache: %v", err)
	}
	defer cache.Stop()

	// Test P-256 key
	key256, err := cache.GetKey("ec-key-p256")
	if err != nil {
		t.Errorf("GetKey(ec-key-p256) error = %v", err)
	}
	if key256 == nil {
		t.Fatal("Expected P-256 key to be non-nil")
	}
	if _, ok := key256.(*ecdsa.PublicKey); !ok {
		t.Errorf("Expected P-256 key to be *ecdsa.PublicKey, got %T", key256)
	}

	// Test P-384 key
	key384, err := cache.GetKey("ec-key-p384")
	if err != nil {
		t.Errorf("GetKey(ec-key-p384) error = %v", err)
	}
	if key384 == nil {
		t.Fatal("Expected P-384 key to be non-nil")
	}
	if _, ok := key384.(*ecdsa.PublicKey); !ok {
		t.Errorf("Expected P-384 key to be *ecdsa.PublicKey, got %T", key384)
	}

	// Verify we have 2 keys cached
	if cache.KeyCount() != 2 {
		t.Errorf("Expected 2 keys, got %d", cache.KeyCount())
	}
}
