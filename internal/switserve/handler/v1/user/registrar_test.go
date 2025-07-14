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

package user

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test interface compliance without database initialization
func TestUserRouteRegistrar_Interface(t *testing.T) {
	// Create registrar with nil controller to avoid DB initialization
	registrar := &UserRouteRegistrar{controller: nil}

	// Test interface methods
	assert.Equal(t, "user-api", registrar.GetName())
	assert.Equal(t, "v1", registrar.GetVersion())
	assert.Equal(t, "", registrar.GetPrefix())
}

func TestUserInternalRouteRegistrar_Interface(t *testing.T) {
	// Create registrar with nil controller to avoid DB initialization
	registrar := &UserInternalRouteRegistrar{controller: nil}

	// Test interface methods
	assert.Equal(t, "user-internal-api", registrar.GetName())
	assert.Equal(t, "root", registrar.GetVersion())
	assert.Equal(t, "", registrar.GetPrefix())
}

func TestUserRouteRegistrar_RouteRegistration(t *testing.T) {
	// Create registrar with nil controller to avoid DB initialization
	registrar := &UserRouteRegistrar{controller: nil}

	// Create a test gin engine
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	// Create a test router group
	group := engine.Group("/api")

	// Test that registration doesn't panic
	assert.NotPanics(t, func() {
		_ = registrar.RegisterRoutes(group)
	})
}

func TestUserInternalRouteRegistrar_RouteRegistration(t *testing.T) {
	// Create registrar with nil controller to avoid DB initialization
	registrar := &UserInternalRouteRegistrar{controller: nil}

	// Create a test gin engine
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	// Create a test router group
	group := engine.Group("/api")

	// Test that registration doesn't panic
	assert.NotPanics(t, func() {
		_ = registrar.RegisterRoutes(group)
	})
}

func TestRegistrar_Constructors(t *testing.T) {
	tests := []struct {
		name            string
		constructor     func() interface{}
		expectedName    string
		expectedVersion string
		expectedPrefix  string
	}{
		{
			name:            "UserRouteRegistrar",
			constructor:     func() interface{} { return &UserRouteRegistrar{controller: nil} },
			expectedName:    "user-api",
			expectedVersion: "v1",
			expectedPrefix:  "",
		},
		{
			name:            "UserInternalRouteRegistrar",
			constructor:     func() interface{} { return &UserInternalRouteRegistrar{controller: nil} },
			expectedName:    "user-internal-api",
			expectedVersion: "root",
			expectedPrefix:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := tt.constructor()

			// Test that registrar implements required interface
			_, ok := registrar.(interface {
				GetName() string
				GetVersion() string
				GetPrefix() string
				RegisterRoutes(*gin.RouterGroup) error
			})
			assert.True(t, ok)
		})
	}
}

func TestRouteStructureMethods(t *testing.T) {
	tests := []struct {
		name      string
		registrar interface {
			GetName() string
			GetVersion() string
			GetPrefix() string
		}
		expectedName    string
		expectedVersion string
		expectedPrefix  string
	}{
		{
			name:            "UserRouteRegistrar",
			registrar:       &UserRouteRegistrar{controller: nil},
			expectedName:    "user-api",
			expectedVersion: "v1",
			expectedPrefix:  "",
		},
		{
			name:            "UserInternalRouteRegistrar",
			registrar:       &UserInternalRouteRegistrar{controller: nil},
			expectedName:    "user-internal-api",
			expectedVersion: "root",
			expectedPrefix:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, tt.registrar.GetName())
			assert.Equal(t, tt.expectedVersion, tt.registrar.GetVersion())
			assert.Equal(t, tt.expectedPrefix, tt.registrar.GetPrefix())
		})
	}
}

func TestRouteRegistrationBehavior(t *testing.T) {
	tests := []struct {
		name      string
		registrar interface {
			RegisterRoutes(*gin.RouterGroup) error
		}
		expectedError bool
	}{
		{
			name:          "UserRouteRegistrar",
			registrar:     &UserRouteRegistrar{controller: nil},
			expectedError: false,
		},
		{
			name:          "UserInternalRouteRegistrar",
			registrar:     &UserInternalRouteRegistrar{controller: nil},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test gin engine
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			group := engine.Group("/test")

			// Test registration
			if tt.expectedError {
				assert.Error(t, tt.registrar.RegisterRoutes(group))
			} else {
				assert.NoError(t, tt.registrar.RegisterRoutes(group))
			}
		})
	}
}

func TestRouteRegistrationPanicHandling(t *testing.T) {
	tests := []struct {
		name      string
		registrar interface {
			RegisterRoutes(*gin.RouterGroup) error
		}
	}{
		{
			name:      "UserRouteRegistrar with nil group",
			registrar: &UserRouteRegistrar{controller: nil},
		},
		{
			name:      "UserInternalRouteRegistrar with nil group",
			registrar: &UserInternalRouteRegistrar{controller: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Panics(t, func() {
				_ = tt.registrar.RegisterRoutes(nil)
			})
		})
	}
}
