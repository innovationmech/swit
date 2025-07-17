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

package v1

import (
	"context"
	"testing"

	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/service/auth/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuthSrv struct {
	mock.Mock
}

func (m *MockAuthSrv) Login(ctx context.Context, username, password string) (*v1.AuthResponse, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.AuthResponse), args.Error(1)
}

func (m *MockAuthSrv) RefreshToken(ctx context.Context, refreshToken string) (*v1.AuthResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.AuthResponse), args.Error(1)
}

func (m *MockAuthSrv) ValidateToken(ctx context.Context, tokenString string) (*model.Token, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockAuthSrv) Logout(ctx context.Context, tokenString string) error {
	args := m.Called(ctx, tokenString)
	return args.Error(0)
}

func TestNewAuthController(t *testing.T) {
	tests := []struct {
		name        string
		authService v1.AuthSrv
		expected    *Controller
	}{
		{
			name:        "Create controller with valid auth service",
			authService: &MockAuthSrv{},
			expected: &Controller{
				authService: &MockAuthSrv{},
			},
		},
		{
			name:        "Create controller with nil auth service",
			authService: nil,
			expected: &Controller{
				authService: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := NewAuthController(tt.authService)

			assert.NotNil(t, controller)
			assert.IsType(t, &Controller{}, controller)

			if tt.authService != nil {
				assert.NotNil(t, controller.authService)
				assert.IsType(t, tt.authService, controller.authService)
			} else {
				assert.Nil(t, controller.authService)
			}
		})
	}
}

func TestControllerStruct(t *testing.T) {
	mockAuthService := &MockAuthSrv{}
	controller := &Controller{authService: mockAuthService}

	assert.NotNil(t, controller)
	assert.Equal(t, mockAuthService, controller.authService)
}

func TestControllerWithMockService(t *testing.T) {
	mockAuthService := &MockAuthSrv{}
	controller := NewAuthController(mockAuthService)

	assert.NotNil(t, controller)
	assert.Equal(t, mockAuthService, controller.authService)

	mockAuthService.AssertExpectations(t)
}

func TestControllerFieldAccess(t *testing.T) {
	mockAuthService := &MockAuthSrv{}
	controller := NewAuthController(mockAuthService)

	assert.NotNil(t, controller.authService)

	_, ok := controller.authService.(v1.AuthSrv)
	assert.True(t, ok, "authService should implement AuthSrv interface")
}

func BenchmarkNewAuthController(b *testing.B) {
	mockAuthService := &MockAuthSrv{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewAuthController(mockAuthService)
	}
}

func TestControllerInterfaceCompliance(t *testing.T) {
	mockAuthService := &MockAuthSrv{}
	controller := NewAuthController(mockAuthService)

	assert.Implements(t, (*v1.AuthSrv)(nil), controller.authService)
}
