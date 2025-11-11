// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestFromAuthHeader(t *testing.T) {
	extractor := FromAuthHeader()

	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{
			name:    "valid bearer token",
			header:  "Bearer test-token",
			want:    "test-token",
			wantErr: false,
		},
		{
			name:    "missing authorization header",
			header:  "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			header:  "test-token",
			wantErr: true,
		},
		{
			name:    "non-bearer scheme",
			header:  "Basic test-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			got, err := extractor(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromAuthHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FromAuthHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromHeader(t *testing.T) {
	extractor := FromHeader("X-Access-Token")

	tests := []struct {
		name       string
		headerName string
		headerVal  string
		want       string
		wantErr    bool
	}{
		{
			name:       "valid custom header",
			headerName: "X-Access-Token",
			headerVal:  "test-token",
			want:       "test-token",
			wantErr:    false,
		},
		{
			name:       "missing header",
			headerName: "X-Access-Token",
			headerVal:  "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.headerVal != "" {
				req.Header.Set(tt.headerName, tt.headerVal)
			}

			got, err := extractor(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FromHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromQuery(t *testing.T) {
	extractor := FromQuery("access_token")

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid query parameter",
			url:     "/?access_token=test-token",
			want:    "test-token",
			wantErr: false,
		},
		{
			name:    "missing query parameter",
			url:     "/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			got, err := extractor(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FromQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromCookie(t *testing.T) {
	extractor := FromCookie("access_token")

	tests := []struct {
		name    string
		cookie  *http.Cookie
		want    string
		wantErr bool
	}{
		{
			name: "valid cookie",
			cookie: &http.Cookie{
				Name:  "access_token",
				Value: "test-token",
			},
			want:    "test-token",
			wantErr: false,
		},
		{
			name:    "missing cookie",
			cookie:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			got, err := extractor(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromCookie() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FromCookie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromFirst(t *testing.T) {
	extractor := FromFirst(
		FromAuthHeader(),
		FromHeader("X-Access-Token"),
		FromQuery("access_token"),
	)

	tests := []struct {
		name        string
		setupReq    func(*http.Request)
		want        string
		wantErr     bool
		description string
	}{
		{
			name: "token in auth header",
			setupReq: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer auth-token")
			},
			want:        "auth-token",
			wantErr:     false,
			description: "Should extract from Authorization header",
		},
		{
			name: "token in custom header",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Access-Token", "custom-token")
			},
			want:        "custom-token",
			wantErr:     false,
			description: "Should extract from custom header",
		},
		{
			name: "token in query parameter",
			setupReq: func(req *http.Request) {
				q := req.URL.Query()
				q.Add("access_token", "query-token")
				req.URL.RawQuery = q.Encode()
			},
			want:        "query-token",
			wantErr:     false,
			description: "Should extract from query parameter",
		},
		{
			name:        "no token found",
			setupReq:    func(req *http.Request) {},
			wantErr:     true,
			description: "Should fail when no token found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupReq(req)

			got, err := extractor(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromFirst() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FromFirst() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMultiTokenExtractor(t *testing.T) {
	extractor := NewMultiTokenExtractor(
		FromAuthHeader(),
		FromQuery("access_token"),
	)

	if extractor == nil {
		t.Fatal("Expected extractor to be non-nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	token, err := extractor.Extract(req)
	if err != nil {
		t.Errorf("Extract() error = %v", err)
	}
	if token != "test-token" {
		t.Errorf("Extract() = %v, want test-token", token)
	}
}

func TestMultiTokenExtractor_Extract(t *testing.T) {
	extractor := NewMultiTokenExtractor(
		FromAuthHeader(),
		FromQuery("access_token"),
	)

	tests := []struct {
		name     string
		setupReq func(*http.Request)
		want     string
		wantErr  bool
	}{
		{
			name: "token from header",
			setupReq: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer header-token")
			},
			want:    "header-token",
			wantErr: false,
		},
		{
			name: "no token found",
			setupReq: func(req *http.Request) {
				// No token set
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupReq(req)

			got, err := extractor.Extract(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Extract() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultExtractor(t *testing.T) {
	extractor := DefaultExtractor()

	tests := []struct {
		name     string
		setupReq func(*http.Request)
		want     string
		wantErr  bool
	}{
		{
			name: "auth header",
			setupReq: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer test-token")
			},
			want:    "test-token",
			wantErr: false,
		},
		{
			name: "X-Access-Token header",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Access-Token", "custom-token")
			},
			want:    "custom-token",
			wantErr: false,
		},
		{
			name: "query parameter",
			setupReq: func(req *http.Request) {
				q := req.URL.Query()
				q.Add("access_token", "query-token")
				req.URL.RawQuery = q.Encode()
			},
			want:    "query-token",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupReq(req)

			got, err := extractor(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultExtractor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DefaultExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTokenFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	token, err := ExtractTokenFromRequest(req)
	if err != nil {
		t.Errorf("ExtractTokenFromRequest() error = %v", err)
	}
	if token != "test-token" {
		t.Errorf("ExtractTokenFromRequest() = %v, want test-token", token)
	}
}

func TestValidator_ExtractAndValidateToken(t *testing.T) {
	secret := "test-secret-key-for-testing"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a valid token
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	tests := []struct {
		name      string
		setupReq  func(*http.Request)
		extractor TokenExtractor
		wantErr   bool
	}{
		{
			name: "valid token with default extractor",
			setupReq: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+tokenString)
			},
			extractor: nil,
			wantErr:   false,
		},
		{
			name: "valid token with custom extractor",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Access-Token", tokenString)
			},
			extractor: FromHeader("X-Access-Token"),
			wantErr:   false,
		},
		{
			name: "missing token",
			setupReq: func(req *http.Request) {
				// No token
			},
			extractor: nil,
			wantErr:   true,
		},
		{
			name: "invalid token",
			setupReq: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid-token")
			},
			extractor: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupReq(req)

			claims, err := validator.ExtractAndValidateToken(req, tt.extractor)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractAndValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && claims == nil {
				t.Error("Expected claims to be non-nil")
			}
		})
	}
}

func TestExtractConfig_SetDefaults(t *testing.T) {
	config := &ExtractConfig{}
	config.SetDefaults()

	if config.AuthScheme != "Bearer" {
		t.Errorf("Expected default AuthScheme to be Bearer, got %s", config.AuthScheme)
	}
	if config.HeaderName != "Authorization" {
		t.Errorf("Expected default HeaderName to be Authorization, got %s", config.HeaderName)
	}
	if config.QueryParam != "access_token" {
		t.Errorf("Expected default QueryParam to be access_token, got %s", config.QueryParam)
	}
	if config.CookieName != "access_token" {
		t.Errorf("Expected default CookieName to be access_token, got %s", config.CookieName)
	}
}

func TestExtractConfig_BuildExtractor(t *testing.T) {
	tests := []struct {
		name   string
		config *ExtractConfig
		setup  func(*http.Request)
		want   string
	}{
		{
			name: "extract from header only",
			config: &ExtractConfig{
				AllowQueryParam: false,
				AllowCookie:     false,
			},
			setup: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer header-token")
			},
			want: "header-token",
		},
		{
			name: "extract from query",
			config: &ExtractConfig{
				AllowQueryParam: true,
				AllowCookie:     false,
			},
			setup: func(req *http.Request) {
				q := req.URL.Query()
				q.Add("access_token", "query-token")
				req.URL.RawQuery = q.Encode()
			},
			want: "query-token",
		},
		{
			name: "extract from cookie",
			config: &ExtractConfig{
				AllowQueryParam: false,
				AllowCookie:     true,
			},
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: "cookie-token",
				})
			},
			want: "cookie-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := tt.config.BuildExtractor()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setup(req)

			got, err := extractor(req)
			if err != nil {
				t.Errorf("BuildExtractor() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("BuildExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}
