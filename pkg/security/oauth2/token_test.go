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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// TestTokenIntrospection tests token introspection structure and methods.
func TestTokenIntrospection(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		introspection *TokenIntrospection
		wantExpired   bool
		wantExpiresAt time.Time
		wantIssuedAt  time.Time
		wantNotBefore time.Time
	}{
		{
			name: "active token with all fields",
			introspection: &TokenIntrospection{
				Active:    true,
				Scope:     "openid profile email",
				ClientID:  "test-client",
				Username:  "testuser",
				TokenType: "Bearer",
				Exp:       now.Add(1 * time.Hour).Unix(),
				Iat:       now.Unix(),
				Nbf:       now.Unix(),
				Sub:       "user123",
				Aud:       "test-audience",
				Iss:       "https://example.com",
				Jti:       "token-id-123",
			},
			wantExpired:   false,
			wantExpiresAt: time.Unix(now.Add(1*time.Hour).Unix(), 0),
			wantIssuedAt:  time.Unix(now.Unix(), 0),
			wantNotBefore: time.Unix(now.Unix(), 0),
		},
		{
			name: "expired token",
			introspection: &TokenIntrospection{
				Active: false,
				Exp:    now.Add(-1 * time.Hour).Unix(),
				Iat:    now.Add(-2 * time.Hour).Unix(),
			},
			wantExpired:   true,
			wantExpiresAt: time.Unix(now.Add(-1*time.Hour).Unix(), 0),
			wantIssuedAt:  time.Unix(now.Add(-2*time.Hour).Unix(), 0),
		},
		{
			name: "token without expiry",
			introspection: &TokenIntrospection{
				Active: true,
				Exp:    0,
			},
			wantExpired:   false,
			wantExpiresAt: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.introspection.IsExpired(); got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}

			if got := tt.introspection.ExpiresAt(); !got.Equal(tt.wantExpiresAt) {
				t.Errorf("ExpiresAt() = %v, want %v", got, tt.wantExpiresAt)
			}

			if tt.introspection.Iat != 0 {
				if got := tt.introspection.IssuedAt(); !got.Equal(tt.wantIssuedAt) {
					t.Errorf("IssuedAt() = %v, want %v", got, tt.wantIssuedAt)
				}
			}

			if tt.introspection.Nbf != 0 {
				if got := tt.introspection.NotBefore(); !got.Equal(tt.wantNotBefore) {
					t.Errorf("NotBefore() = %v, want %v", got, tt.wantNotBefore)
				}
			}
		})
	}
}

// TestClientIntrospectToken tests the token introspection endpoint.
func TestClientIntrospectToken(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		wantActive    bool
	}{
		{
			name:  "successful introspection - active token",
			token: "test-access-token",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/introspect" {
					http.NotFound(w, r)
					return
				}
				if r.Method != "POST" {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"active":     true,
					"scope":      "openid profile email",
					"client_id":  "test-client",
					"username":   "testuser",
					"token_type": "Bearer",
					"exp":        time.Now().Add(1 * time.Hour).Unix(),
					"iat":        time.Now().Unix(),
					"sub":        "user123",
				})
			},
			wantErr:    false,
			wantActive: true,
		},
		{
			name:  "successful introspection - inactive token",
			token: "expired-token",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/introspect" {
					http.NotFound(w, r)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"active": false,
				})
			},
			wantErr:    false,
			wantActive: false,
		},
		{
			name:  "empty token",
			token: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called with empty token")
			},
			wantErr: true,
		},
		{
			name:  "server error",
			token: "test-token",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			config := &Config{
				Enabled:      true,
				Provider:     "custom",
				ClientID:     "test-client",
				ClientSecret: "secret",
				UseDiscovery: false,
				AuthURL:      server.URL + "/auth",
				TokenURL:     server.URL + "/token",
				JWKSURL:      server.URL + "/jwks",
			}

			ctx := context.Background()
			client, err := NewClient(ctx, config)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			introspection, err := client.IntrospectToken(ctx, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Error("IntrospectToken() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("IntrospectToken() unexpected error = %v", err)
				return
			}

			if introspection.Active != tt.wantActive {
				t.Errorf("introspection.Active = %v, want %v", introspection.Active, tt.wantActive)
			}
		})
	}
}

// TestClientRevokeToken tests the token revocation endpoint.
func TestClientRevokeToken(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
	}{
		{
			name:  "successful revocation",
			token: "test-access-token",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/revoke" {
					http.NotFound(w, r)
					return
				}
				if r.Method != "POST" {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:  "empty token",
			token: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called with empty token")
			},
			wantErr: true,
		},
		{
			name:  "server error",
			token: "test-token",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name:  "invalid token (still returns 200 per RFC 7009)",
			token: "invalid-token",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Per RFC 7009, server should return 200 even for invalid tokens
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			config := &Config{
				Enabled:      true,
				Provider:     "custom",
				ClientID:     "test-client",
				ClientSecret: "secret",
				UseDiscovery: false,
				AuthURL:      server.URL + "/auth",
				TokenURL:     server.URL + "/token",
				JWKSURL:      server.URL + "/jwks",
			}

			ctx := context.Background()
			client, err := NewClient(ctx, config)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			err = client.RevokeToken(ctx, tt.token)

			if tt.wantErr {
				if err == nil {
					t.Error("RevokeToken() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("RevokeToken() unexpected error = %v", err)
			}
		})
	}
}

// TestTokenHelpers tests token helper functions.
func TestTokenHelpers(t *testing.T) {
	t.Run("ToJSON and FromJSON", func(t *testing.T) {
		token := &Token{
			Token: &oauth2.Token{
				AccessToken:  "test-access-token",
				TokenType:    "Bearer",
				RefreshToken: "test-refresh-token",
				Expiry:       time.Now().Add(1 * time.Hour),
			},
			IDToken: "test-id-token",
			Claims: map[string]interface{}{
				"sub":   "user123",
				"email": "test@example.com",
			},
		}

		// Test ToJSON
		jsonData, err := token.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON() error = %v", err)
		}

		// Test FromJSON
		parsedToken, err := FromJSON(jsonData)
		if err != nil {
			t.Fatalf("FromJSON() error = %v", err)
		}

		if parsedToken.AccessToken != token.AccessToken {
			t.Errorf("AccessToken = %v, want %v", parsedToken.AccessToken, token.AccessToken)
		}
		if parsedToken.IDToken != token.IDToken {
			t.Errorf("IDToken = %v, want %v", parsedToken.IDToken, token.IDToken)
		}
	})

	t.Run("GetClaim", func(t *testing.T) {
		token := &Token{
			Claims: map[string]interface{}{
				"sub":   "user123",
				"email": "test@example.com",
				"roles": []string{"admin", "user"},
			},
		}

		// Test GetClaim
		sub, ok := token.GetClaim("sub")
		if !ok {
			t.Error("GetClaim('sub') returned false")
		}
		if sub != "user123" {
			t.Errorf("GetClaim('sub') = %v, want %v", sub, "user123")
		}

		// Test GetStringClaim
		email, ok := token.GetStringClaim("email")
		if !ok {
			t.Error("GetStringClaim('email') returned false")
		}
		if email != "test@example.com" {
			t.Errorf("GetStringClaim('email') = %v, want %v", email, "test@example.com")
		}

		// Test GetSubject
		if token.GetSubject() != "user123" {
			t.Errorf("GetSubject() = %v, want %v", token.GetSubject(), "user123")
		}

		// Test GetEmail
		if token.GetEmail() != "test@example.com" {
			t.Errorf("GetEmail() = %v, want %v", token.GetEmail(), "test@example.com")
		}
	})
}
