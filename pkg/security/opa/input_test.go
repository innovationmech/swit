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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestNewPolicyInputBuilder(t *testing.T) {
	builder := NewPolicyInputBuilder()
	assert.NotNil(t, builder)
	assert.NotNil(t, builder.input)
	assert.Equal(t, "unknown", builder.input.Request.Protocol)
	assert.NotNil(t, builder.input.User.Roles)
	assert.NotNil(t, builder.input.User.Permissions)
	assert.NotNil(t, builder.input.User.Groups)
	assert.NotNil(t, builder.input.User.Attributes)
	assert.NotNil(t, builder.input.Resource.Attributes)
	assert.NotNil(t, builder.input.Custom)
}

func TestPolicyInputBuilder_FromHTTPRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		query          string
		headers        map[string]string
		expectedMethod string
		expectedPath   string
		expectedQuery  map[string][]string
	}{
		{
			name:           "Simple GET request",
			method:         "GET",
			path:           "/api/users",
			expectedMethod: "GET",
			expectedPath:   "/api/users",
		},
		{
			name:           "POST request with query params",
			method:         "POST",
			path:           "/api/users",
			query:          "role=admin&status=active",
			expectedMethod: "POST",
			expectedPath:   "/api/users",
			expectedQuery: map[string][]string{
				"role":   {"admin"},
				"status": {"active"},
			},
		},
		{
			name:   "Request with headers",
			method: "GET",
			path:   "/api/users",
			headers: map[string]string{
				"Content-Type": "application/json",
				"User-Agent":   "test-agent",
			},
			expectedMethod: "GET",
			expectedPath:   "/api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			url := tt.path
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(tt.method, url, nil)

			// Add headers
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Create Gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Build input
			builder := NewPolicyInputBuilder()
			builder.FromHTTPRequest(c)
			input := builder.Build()

			// Verify
			assert.Equal(t, tt.expectedMethod, input.Request.Method)
			assert.Equal(t, tt.expectedPath, input.Request.Path)
			assert.Equal(t, "http", input.Request.Protocol)

			if tt.expectedQuery != nil {
				assert.Equal(t, tt.expectedQuery, input.Request.Query)
			}

			if len(tt.headers) > 0 {
				assert.NotNil(t, input.Request.Headers)
			}
		})
	}
}

func TestPolicyInputBuilder_FromGRPCContext(t *testing.T) {
	tests := []struct {
		name           string
		fullMethod     string
		metadata       map[string][]string
		peerAddr       string
		expectedMethod string
		expectedPath   string
	}{
		{
			name:           "Simple gRPC request",
			fullMethod:     "/api.UserService/GetUser",
			expectedMethod: "/api.UserService/GetUser",
			expectedPath:   "/api.UserService/GetUser",
		},
		{
			name:       "gRPC request with metadata",
			fullMethod: "/api.UserService/CreateUser",
			metadata: map[string][]string{
				"content-type": {"application/grpc"},
				"user-agent":   {"grpc-go/1.50.0"},
			},
			expectedMethod: "/api.UserService/CreateUser",
			expectedPath:   "/api.UserService/CreateUser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Add metadata if provided
			if tt.metadata != nil {
				md := metadata.New(nil)
				for k, v := range tt.metadata {
					md.Set(k, v...)
				}
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Add peer info if provided
			if tt.peerAddr != "" {
				p := &peer.Peer{
					Addr: &mockAddr{addr: tt.peerAddr},
				}
				ctx = peer.NewContext(ctx, p)
			}

			// Build input
			builder := NewPolicyInputBuilder()
			builder.FromGRPCContext(ctx, tt.fullMethod)
			input := builder.Build()

			// Verify
			assert.Equal(t, tt.expectedMethod, input.Request.Method)
			assert.Equal(t, tt.expectedPath, input.Request.Path)
			assert.Equal(t, "grpc", input.Request.Protocol)

			if tt.peerAddr != "" {
				assert.Equal(t, tt.peerAddr, input.Request.ClientIP)
			}
		})
	}
}

func TestPolicyInputBuilder_WithUser(t *testing.T) {
	builder := NewPolicyInputBuilder()

	user := UserInfo{
		ID:          "user-123",
		Username:    "testuser",
		Email:       "test@example.com",
		Roles:       []string{"admin", "user"},
		Permissions: []string{"read", "write"},
		Groups:      []string{"group1"},
		Attributes: map[string]interface{}{
			"department": "engineering",
		},
	}

	input := builder.WithUser(user).Build()

	assert.Equal(t, "user-123", input.User.ID)
	assert.Equal(t, "testuser", input.User.Username)
	assert.Equal(t, "test@example.com", input.User.Email)
	assert.Equal(t, []string{"admin", "user"}, input.User.Roles)
	assert.Equal(t, []string{"read", "write"}, input.User.Permissions)
	assert.Equal(t, []string{"group1"}, input.User.Groups)
	assert.Equal(t, "engineering", input.User.Attributes["department"])
}

func TestPolicyInputBuilder_WithUserFields(t *testing.T) {
	builder := NewPolicyInputBuilder()

	input := builder.
		WithUserID("user-456").
		WithUsername("anotheruser").
		WithUserEmail("another@example.com").
		WithUserRoles([]string{"viewer"}).
		WithUserPermissions([]string{"read"}).
		WithUserGroups([]string{"group2", "group3"}).
		WithUserAttribute("location", "US").
		WithUserAttribute("level", 5).
		Build()

	assert.Equal(t, "user-456", input.User.ID)
	assert.Equal(t, "anotheruser", input.User.Username)
	assert.Equal(t, "another@example.com", input.User.Email)
	assert.Equal(t, []string{"viewer"}, input.User.Roles)
	assert.Equal(t, []string{"read"}, input.User.Permissions)
	assert.Equal(t, []string{"group2", "group3"}, input.User.Groups)
	assert.Equal(t, "US", input.User.Attributes["location"])
	assert.Equal(t, 5, input.User.Attributes["level"])
}

func TestPolicyInputBuilder_WithResource(t *testing.T) {
	builder := NewPolicyInputBuilder()

	resource := ResourceInfo{
		Type:  "document",
		ID:    "doc-123",
		Name:  "test-document",
		Owner: "user-789",
		Attributes: map[string]interface{}{
			"confidential": true,
		},
	}

	input := builder.WithResource(resource).Build()

	assert.Equal(t, "document", input.Resource.Type)
	assert.Equal(t, "doc-123", input.Resource.ID)
	assert.Equal(t, "test-document", input.Resource.Name)
	assert.Equal(t, "user-789", input.Resource.Owner)
	assert.Equal(t, true, input.Resource.Attributes["confidential"])
}

func TestPolicyInputBuilder_WithResourceFields(t *testing.T) {
	builder := NewPolicyInputBuilder()

	input := builder.
		WithResourceType("project").
		WithResourceID("proj-001").
		WithResourceName("secret-project").
		WithResourceOwner("user-111").
		WithResourceAttribute("status", "active").
		WithResourceAttribute("priority", 10).
		Build()

	assert.Equal(t, "project", input.Resource.Type)
	assert.Equal(t, "proj-001", input.Resource.ID)
	assert.Equal(t, "secret-project", input.Resource.Name)
	assert.Equal(t, "user-111", input.Resource.Owner)
	assert.Equal(t, "active", input.Resource.Attributes["status"])
	assert.Equal(t, 10, input.Resource.Attributes["priority"])
}

func TestPolicyInputBuilder_WithCustomData(t *testing.T) {
	builder := NewPolicyInputBuilder()

	input := builder.
		WithCustomData("environment", "production").
		WithCustomData("tenant_id", "tenant-123").
		WithCustomData("features", []string{"feature1", "feature2"}).
		Build()

	assert.Equal(t, "production", input.Custom["environment"])
	assert.Equal(t, "tenant-123", input.Custom["tenant_id"])
	assert.Equal(t, []string{"feature1", "feature2"}, input.Custom["features"])
}

func TestExtractUserFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		setup    func(*gin.Context)
		expected UserInfo
	}{
		{
			name: "Extract basic user info",
			setup: func(c *gin.Context) {
				c.Set("user_id", "user-123")
				c.Set("username", "testuser")
				c.Set("email", "test@example.com")
			},
			expected: UserInfo{
				ID:          "user-123",
				Username:    "testuser",
				Email:       "test@example.com",
				Roles:       []string{},
				Permissions: []string{},
				Groups:      []string{},
				Attributes:  map[string]interface{}{},
			},
		},
		{
			name: "Extract user with roles and permissions",
			setup: func(c *gin.Context) {
				c.Set("user_id", "user-456")
				c.Set("roles", []string{"admin", "editor"})
				c.Set("permissions", []string{"read", "write", "delete"})
				c.Set("groups", []string{"team-alpha"})
			},
			expected: UserInfo{
				ID:          "user-456",
				Username:    "",
				Email:       "",
				Roles:       []string{"admin", "editor"},
				Permissions: []string{"read", "write", "delete"},
				Groups:      []string{"team-alpha"},
				Attributes:  map[string]interface{}{},
			},
		},
		{
			name: "Extract from claims",
			setup: func(c *gin.Context) {
				claims := map[string]interface{}{
					"sub":        "user-789",
					"email":      "claims@example.com",
					"department": "engineering",
					"level":      5,
				}
				c.Set("claims", claims)
			},
			expected: UserInfo{
				ID:          "user-789",
				Username:    "",
				Email:       "claims@example.com",
				Roles:       []string{},
				Permissions: []string{},
				Groups:      []string{},
				Attributes: map[string]interface{}{
					"department": "engineering",
					"level":      5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			user := ExtractUserFromContext(c)

			assert.Equal(t, tt.expected.ID, user.ID)
			assert.Equal(t, tt.expected.Username, user.Username)
			assert.Equal(t, tt.expected.Email, user.Email)
			assert.Equal(t, tt.expected.Roles, user.Roles)
			assert.Equal(t, tt.expected.Permissions, user.Permissions)
			assert.Equal(t, tt.expected.Groups, user.Groups)

			for k, v := range tt.expected.Attributes {
				assert.Equal(t, v, user.Attributes[k])
			}
		})
	}
}

func TestExtractUserFromGRPCContext(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected UserInfo
	}{
		{
			name: "Extract basic user info",
			setup: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "user_id", "user-123")
				ctx = context.WithValue(ctx, "username", "grpcuser")
				ctx = context.WithValue(ctx, "email", "grpc@example.com")
				return ctx
			},
			expected: UserInfo{
				ID:          "user-123",
				Username:    "grpcuser",
				Email:       "grpc@example.com",
				Roles:       []string{},
				Permissions: []string{},
				Groups:      []string{},
				Attributes:  map[string]interface{}{},
			},
		},
		{
			name: "Extract user with roles",
			setup: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "user_id", "user-456")
				ctx = context.WithValue(ctx, "roles", []string{"viewer"})
				ctx = context.WithValue(ctx, "permissions", []string{"read"})
				return ctx
			},
			expected: UserInfo{
				ID:          "user-456",
				Username:    "",
				Email:       "",
				Roles:       []string{"viewer"},
				Permissions: []string{"read"},
				Groups:      []string{},
				Attributes:  map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			user := ExtractUserFromGRPCContext(ctx)

			assert.Equal(t, tt.expected.ID, user.ID)
			assert.Equal(t, tt.expected.Username, user.Username)
			assert.Equal(t, tt.expected.Email, user.Email)
			assert.Equal(t, tt.expected.Roles, user.Roles)
			assert.Equal(t, tt.expected.Permissions, user.Permissions)
			assert.Equal(t, tt.expected.Groups, user.Groups)
		})
	}
}

func TestExtractSafeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		expected map[string][]string
	}{
		{
			name: "Safe headers only",
			headers: http.Header{
				"Content-Type": []string{"application/json"},
				"User-Agent":   []string{"test-agent"},
				"Accept":       []string{"application/json"},
			},
			expected: map[string][]string{
				"Content-Type": {"application/json"},
				"User-Agent":   {"test-agent"},
				"Accept":       {"application/json"},
			},
		},
		{
			name: "Mix of safe and unsafe headers",
			headers: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer token123"},
				"Cookie":        []string{"session=abc123"},
				"User-Agent":    []string{"test-agent"},
			},
			expected: map[string][]string{
				"Content-Type": {"application/json"},
				"User-Agent":   {"test-agent"},
			},
		},
		{
			name:     "Empty headers",
			headers:  http.Header{},
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSafeHeaders(tt.headers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSafeMetadataKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"Safe key - content-type", "content-type", true},
		{"Safe key - user-agent", "user-agent", true},
		{"Safe key - x-request-id", "x-request-id", true},
		{"Safe key - x-forwarded-for", "x-forwarded-for", true},
		{"Safe key - grpc-timeout", "grpc-timeout", true},
		{"Unsafe key - authorization", "authorization", false},
		{"Unsafe key - cookie", "cookie", false},
		{"Unsafe key - x-api-key", "x-api-key", false},
		{"Unsafe key - x-auth-token", "x-auth-token", false},
		{"Unsafe key - grpcgateway-authorization", "grpcgateway-authorization", false},
		{"Unsafe key - authorization-bin", "authorization-bin", false},
		{"Unsafe key - custom-secret", "custom-secret", false},
		{"Unsafe key - any-unknown-key", "any-unknown-key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSafeMetadataKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildPolicyInputFromHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("GET", "/api/users?status=active", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set user context
	c.Set("user_id", "user-123")
	c.Set("username", "testuser")
	c.Set("roles", []string{"admin"})

	input := BuildPolicyInputFromHTTP(c)

	assert.NotNil(t, input)
	assert.Equal(t, "GET", input.Request.Method)
	assert.Equal(t, "/api/users", input.Request.Path)
	assert.Equal(t, "http", input.Request.Protocol)
	assert.Equal(t, "user-123", input.User.ID)
	assert.Equal(t, "testuser", input.User.Username)
	assert.Equal(t, []string{"admin"}, input.User.Roles)
	assert.NotNil(t, input.Request.Query)
	assert.Equal(t, []string{"active"}, input.Request.Query["status"])
}

func TestBuildPolicyInputFromGRPC(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "user-456")
	ctx = context.WithValue(ctx, "roles", []string{"viewer"})

	md := metadata.New(map[string]string{
		"content-type": "application/grpc",
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	input := BuildPolicyInputFromGRPC(ctx, "/api.UserService/GetUser")

	assert.NotNil(t, input)
	assert.Equal(t, "/api.UserService/GetUser", input.Request.Method)
	assert.Equal(t, "/api.UserService/GetUser", input.Request.Path)
	assert.Equal(t, "grpc", input.Request.Protocol)
	assert.Equal(t, "user-456", input.User.ID)
	assert.Equal(t, []string{"viewer"}, input.User.Roles)
}

func TestPolicyInputBuilder_ChainedCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/api/documents/123", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	input := NewPolicyInputBuilder().
		FromHTTPRequest(c).
		WithUserID("user-789").
		WithUserRoles([]string{"editor"}).
		WithResourceType("document").
		WithResourceID("123").
		WithResourceOwner("user-456").
		WithCustomData("tenant_id", "tenant-001").
		Build()

	assert.Equal(t, "POST", input.Request.Method)
	assert.Equal(t, "/api/documents/123", input.Request.Path)
	assert.Equal(t, "user-789", input.User.ID)
	assert.Equal(t, []string{"editor"}, input.User.Roles)
	assert.Equal(t, "document", input.Resource.Type)
	assert.Equal(t, "123", input.Resource.ID)
	assert.Equal(t, "user-456", input.Resource.Owner)
	assert.Equal(t, "tenant-001", input.Custom["tenant_id"])
}

func TestPolicyInputBuilder_TimeStamp(t *testing.T) {
	builder := NewPolicyInputBuilder()
	input := builder.Build()

	// Verify that time is set and recent
	assert.False(t, input.Request.Time.IsZero())
	assert.True(t, time.Since(input.Request.Time) < time.Second)
}

// mockAddr implements net.Addr for testing
type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return m.addr
}
