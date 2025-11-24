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

package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSecurityManager implements SecurityManager interface for testing
type MockSecurityManager struct {
	enabled             bool
	initialized         bool
	oauth2Client        interface{}
	opaClient           interface{}
	auditLogger         interface{}
	httpMiddleware      []interface{}
	grpcUnaryIntercept  []interface{}
	grpcStreamIntercept []interface{}
}

func NewMockSecurityManager(enabled, initialized bool) *MockSecurityManager {
	return &MockSecurityManager{
		enabled:             enabled,
		initialized:         initialized,
		oauth2Client:        &mockOAuth2Client{},
		opaClient:           &mockOPAClient{},
		auditLogger:         &mockAuditLogger{},
		httpMiddleware:      []interface{}{},
		grpcUnaryIntercept:  []interface{}{},
		grpcStreamIntercept: []interface{}{},
	}
}

func (m *MockSecurityManager) GetOAuth2Client() interface{} {
	return m.oauth2Client
}

func (m *MockSecurityManager) GetOPAClient() interface{} {
	return m.opaClient
}

func (m *MockSecurityManager) GetAuditLogger() interface{} {
	return m.auditLogger
}

func (m *MockSecurityManager) IsEnabled() bool {
	return m.enabled
}

func (m *MockSecurityManager) IsInitialized() bool {
	return m.initialized
}

func (m *MockSecurityManager) GetHTTPSecurityMiddleware() []interface{} {
	return m.httpMiddleware
}

func (m *MockSecurityManager) GetGRPCUnaryInterceptors() []interface{} {
	return m.grpcUnaryIntercept
}

func (m *MockSecurityManager) GetGRPCStreamInterceptors() []interface{} {
	return m.grpcStreamIntercept
}

// Mock clients for testing
type mockOAuth2Client struct{}
type mockOPAClient struct{}
type mockAuditLogger struct{}

func TestTransportCoordinator_SetSecurityManager(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		initialized bool
	}{
		{
			name:        "enabled and initialized",
			enabled:     true,
			initialized: true,
		},
		{
			name:        "enabled but not initialized",
			enabled:     true,
			initialized: false,
		},
		{
			name:        "disabled",
			enabled:     false,
			initialized: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator := NewTransportCoordinator()
			secMgr := NewMockSecurityManager(tt.enabled, tt.initialized)

			coordinator.SetSecurityManager(secMgr)

			retrievedSecMgr := coordinator.GetSecurityManager()
			require.NotNil(t, retrievedSecMgr)
			assert.Equal(t, tt.enabled, retrievedSecMgr.IsEnabled())
			assert.Equal(t, tt.initialized, retrievedSecMgr.IsInitialized())
		})
	}
}

func TestTransportCoordinator_SecurityManagerInjection(t *testing.T) {
	coordinator := NewTransportCoordinator()
	secMgr := NewMockSecurityManager(true, true)
	coordinator.SetSecurityManager(secMgr)

	// Create HTTP transport
	httpTransport := NewHTTPNetworkServiceWithConfig(&HTTPTransportConfig{
		Address:     ":0",
		TestMode:    true,
		EnableReady: true,
	})

	// Register transport - security manager should be injected
	coordinator.Register(httpTransport)

	// Verify security manager was injected into HTTP transport
	assert.NotNil(t, httpTransport.config.SecurityManager)
	assert.Equal(t, secMgr, httpTransport.config.SecurityManager)

	// Create gRPC transport
	grpcConfig := DefaultGRPCConfig()
	grpcConfig.Address = ":0"
	grpcConfig.TestMode = true
	grpcTransport := NewGRPCNetworkServiceWithConfig(grpcConfig)

	// Register gRPC transport
	coordinator.Register(grpcTransport)

	// Verify security manager was injected into gRPC transport
	assert.NotNil(t, grpcTransport.config.SecurityManager)
	assert.Equal(t, secMgr, grpcTransport.config.SecurityManager)
}

func TestHTTPTransport_SecurityManagerConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		secMgr      SecurityManager
		expectLog   bool
		shouldBeNil bool
	}{
		{
			name:        "with enabled security manager",
			secMgr:      NewMockSecurityManager(true, true),
			expectLog:   true,
			shouldBeNil: false,
		},
		{
			name:        "with disabled security manager",
			secMgr:      NewMockSecurityManager(false, false),
			expectLog:   false,
			shouldBeNil: false,
		},
		{
			name:        "with nil security manager",
			secMgr:      nil,
			expectLog:   false,
			shouldBeNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HTTPTransportConfig{
				Address:         ":0",
				TestMode:        true,
				EnableReady:     true,
				SecurityManager: tt.secMgr,
			}

			transport := NewHTTPNetworkServiceWithConfig(config)
			require.NotNil(t, transport)

			if tt.shouldBeNil {
				assert.Nil(t, transport.config.SecurityManager)
			} else {
				assert.NotNil(t, transport.config.SecurityManager)
			}
		})
	}
}

func TestGRPCTransport_SecurityManagerConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		secMgr      SecurityManager
		expectLog   bool
		shouldBeNil bool
	}{
		{
			name:        "with enabled security manager",
			secMgr:      NewMockSecurityManager(true, true),
			expectLog:   true,
			shouldBeNil: false,
		},
		{
			name:        "with disabled security manager",
			secMgr:      NewMockSecurityManager(false, false),
			expectLog:   false,
			shouldBeNil: false,
		},
		{
			name:        "with nil security manager",
			secMgr:      nil,
			expectLog:   false,
			shouldBeNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGRPCConfig()
			config.Address = ":0"
			config.TestMode = true
			config.SecurityManager = tt.secMgr

			transport := NewGRPCNetworkServiceWithConfig(config)
			require.NotNil(t, transport)

			if tt.shouldBeNil {
				assert.Nil(t, transport.config.SecurityManager)
			} else {
				assert.NotNil(t, transport.config.SecurityManager)
			}
		})
	}
}

func TestSecurityManager_ClientRetrieval(t *testing.T) {
	secMgr := NewMockSecurityManager(true, true)

	// Test OAuth2 client retrieval
	oauth2Client := secMgr.GetOAuth2Client()
	assert.NotNil(t, oauth2Client)
	_, ok := oauth2Client.(*mockOAuth2Client)
	assert.True(t, ok, "OAuth2 client should be of type *mockOAuth2Client")

	// Test OPA client retrieval
	opaClient := secMgr.GetOPAClient()
	assert.NotNil(t, opaClient)
	_, ok = opaClient.(*mockOPAClient)
	assert.True(t, ok, "OPA client should be of type *mockOPAClient")

	// Test Audit logger retrieval
	auditLogger := secMgr.GetAuditLogger()
	assert.NotNil(t, auditLogger)
	_, ok = auditLogger.(*mockAuditLogger)
	assert.True(t, ok, "Audit logger should be of type *mockAuditLogger")
}

func TestTransportCoordinator_SecurityManagerWithExistingTransports(t *testing.T) {
	coordinator := NewTransportCoordinator()

	// Register HTTP transport before setting security manager
	httpTransport := NewHTTPNetworkServiceWithConfig(&HTTPTransportConfig{
		Address:     ":0",
		TestMode:    true,
		EnableReady: true,
	})
	coordinator.Register(httpTransport)

	// Verify no security manager initially
	assert.Nil(t, httpTransport.config.SecurityManager)

	// Set security manager after transports are registered
	secMgr := NewMockSecurityManager(true, true)
	coordinator.SetSecurityManager(secMgr)

	// Verify security manager was injected into existing transport
	assert.NotNil(t, httpTransport.config.SecurityManager)
	assert.Equal(t, secMgr, httpTransport.config.SecurityManager)
}

func TestHTTPTransport_RegisterSecurityMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		secMgr         SecurityManager
		shouldRegister bool
	}{
		{
			name:           "enabled and initialized security manager",
			secMgr:         NewMockSecurityManager(true, true),
			shouldRegister: true,
		},
		{
			name:           "enabled but not initialized",
			secMgr:         NewMockSecurityManager(true, false),
			shouldRegister: false,
		},
		{
			name:           "disabled security manager",
			secMgr:         NewMockSecurityManager(false, false),
			shouldRegister: false,
		},
		{
			name:           "nil security manager",
			secMgr:         nil,
			shouldRegister: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &HTTPTransportConfig{
				Address:         ":0",
				TestMode:        true,
				EnableReady:     true,
				SecurityManager: tt.secMgr,
			}

			transport := NewHTTPNetworkServiceWithConfig(config)
			require.NotNil(t, transport)

			// Call registerSecurityMiddleware explicitly for testing
			transport.registerSecurityMiddleware()

			// Verify router is still functional
			assert.NotNil(t, transport.router)
		})
	}
}

func TestSecurityManager_InterfaceCompliance(t *testing.T) {
	// Verify that MockSecurityManager implements SecurityManager interface
	var _ SecurityManager = (*MockSecurityManager)(nil)
}
