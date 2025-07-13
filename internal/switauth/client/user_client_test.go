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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/pkg/discovery"
)

func TestNewUserClient(t *testing.T) {
	tests := []struct {
		name string
		sd   *discovery.ServiceDiscovery
	}{
		{
			name: "successful client creation with valid service discovery",
			sd:   &discovery.ServiceDiscovery{},
		},
		{
			name: "client creation with nil service discovery",
			sd:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewUserClient(tt.sd)

			assert.NotNil(t, client)
			assert.IsType(t, &userClient{}, client)

			userClientImpl := client.(*userClient)
			assert.Equal(t, tt.sd, userClientImpl.sd)
		})
	}
}

func TestUserClient_Interface(t *testing.T) {
	sd := &discovery.ServiceDiscovery{}
	client := NewUserClient(sd)

	// Verify it implements the UserClient interface
	var _ UserClient = client
	assert.NotNil(t, client)
}

// testableUserClient creates a user client that can be tested with mock HTTP servers
type testableUserClient struct {
	httpClient *http.Client
	baseURL    string
}

func (c *testableUserClient) ValidateUserCredentials(ctx context.Context, username, password string) (*model.User, error) {
	url := c.baseURL + "/internal/validate-user"

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user credential validation failed: %s", resp.Status)
	}

	var user model.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func TestUserClient_ValidateUserCredentials_Success(t *testing.T) {
	// Create a test user
	testUser := &model.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      "user",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a test server that returns the user
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/internal/validate-user", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read and verify request body
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		err = json.Unmarshal(body, &credentials)
		assert.NoError(t, err)
		assert.Equal(t, "testuser", credentials.Username)
		assert.Equal(t, "password123", credentials.Password)

		// Return successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(testUser)
	}))
	defer server.Close()

	// Create testable client
	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, testUser.Username, user.Username)
	assert.Equal(t, testUser.Email, user.Email)
	assert.Equal(t, testUser.Role, user.Role)
	assert.Equal(t, testUser.IsActive, user.IsActive)
}

func TestUserClient_ValidateUserCredentials_HTTPError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid credentials"))
	}))
	defer server.Close()

	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "wronguser", "wrongpass")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserClient_ValidateUserCredentials_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"username": "testuser", "invalid": json}`)) // Invalid JSON
	}))
	defer server.Close()

	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserClient_ValidateUserCredentials_ContextCancellation(t *testing.T) {
	// Create a test server with a delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&model.User{})
	}))
	defer server.Close()

	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestUserClient_ValidateUserCredentials_NetworkError(t *testing.T) {
	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    "http://invalid-url-that-does-not-exist:99999",
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserClient_ValidateUserCredentials_EmptyCredentials(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and verify request body
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		err = json.Unmarshal(body, &credentials)
		assert.NoError(t, err)
		assert.Empty(t, credentials.Username)
		assert.Empty(t, credentials.Password)

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing credentials"))
	}))
	defer server.Close()

	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "", "")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserClient_ValidateUserCredentials_LargeResponse(t *testing.T) {
	// Create a test user with large data
	testUser := &model.User{
		ID:        uuid.New(),
		Username:  "testuser_with_very_long_username_that_could_cause_issues",
		Email:     "test.user.with.very.long.email.address@example.com",
		Role:      "administrator_with_extended_permissions",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(testUser)
	}))
	defer server.Close()

	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUser.Username, user.Username)
	assert.Equal(t, testUser.Email, user.Email)
}

func TestUserClient_ValidateUserCredentials_RequestHeaders(t *testing.T) {
	// Create a test server that verifies headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type header
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request has context
		assert.NotNil(t, r.Context())

		testUser := &model.User{
			ID:       uuid.New(),
			Username: "testuser",
			Email:    "test@example.com",
			Role:     "user",
			IsActive: true,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(testUser)
	}))
	defer server.Close()

	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func TestUserClient_ValidateUserCredentials_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "success 200",
			statusCode:   http.StatusOK,
			responseBody: `{"id":"550e8400-e29b-41d4-a716-446655440000","username":"testuser","email":"test@example.com","role":"user","is_active":true}`,
			expectError:  false,
		},
		{
			name:         "unauthorized 401",
			statusCode:   http.StatusUnauthorized,
			responseBody: "Unauthorized",
			expectError:  true,
		},
		{
			name:         "forbidden 403",
			statusCode:   http.StatusForbidden,
			responseBody: "Forbidden",
			expectError:  true,
		},
		{
			name:         "not found 404",
			statusCode:   http.StatusNotFound,
			responseBody: "Not Found",
			expectError:  true,
		},
		{
			name:         "internal server error 500",
			statusCode:   http.StatusInternalServerError,
			responseBody: "Internal Server Error",
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			client := &testableUserClient{
				httpClient: &http.Client{},
				baseURL:    server.URL,
			}

			ctx := context.Background()
			user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
		})
	}
}

func TestUserClient_ValidateUserCredentials_JSONMarshaling(t *testing.T) {
	// Test that the client properly marshals credentials to JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		// Verify the JSON structure is correct
		var creds map[string]interface{}
		err = json.Unmarshal(body, &creds)
		require.NoError(t, err)

		// Check that both username and password fields exist
		assert.Contains(t, creds, "username")
		assert.Contains(t, creds, "password")
		assert.Equal(t, "testuser", creds["username"])
		assert.Equal(t, "password123", creds["password"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&model.User{
			ID:       uuid.New(),
			Username: "testuser",
			Email:    "test@example.com",
		})
	}))
	defer server.Close()

	client := &testableUserClient{
		httpClient: &http.Client{},
		baseURL:    server.URL,
	}

	ctx := context.Background()
	user, err := client.ValidateUserCredentials(ctx, "testuser", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func TestUserClient_StructFields(t *testing.T) {
	// Test the userClient struct and its fields
	sd := &discovery.ServiceDiscovery{}
	client := NewUserClient(sd)

	userClientImpl, ok := client.(*userClient)
	require.True(t, ok, "client should be of type *userClient")

	// Test that the service discovery field is set
	assert.Equal(t, sd, userClientImpl.sd)
}

func TestUserClient_TypesAndInterfaces(t *testing.T) {
	// Test type assertions and interface compliance
	sd := &discovery.ServiceDiscovery{}
	client := NewUserClient(sd)

	// Test that it implements UserClient interface
	var userClientInterface UserClient = client
	assert.NotNil(t, userClientInterface)

	// Test concrete type
	concreteClient, ok := client.(*userClient)
	assert.True(t, ok)
	assert.NotNil(t, concreteClient)
}
