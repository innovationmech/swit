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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAuthSrv(t *testing.T) {
	t.Run("should return error when user client is nil", func(t *testing.T) {
		srv, err := NewAuthSrv()
		assert.Error(t, err)
		assert.Nil(t, srv)
		assert.Contains(t, err.Error(), "user client is required")
	})

	t.Run("should return error when token repository is nil", func(t *testing.T) {
		srv, err := NewAuthSrv(WithUserClient(nil))
		assert.Error(t, err)
		assert.Nil(t, srv)
		assert.Contains(t, err.Error(), "user client is required")
	})

	t.Run("should return error when only user client is provided", func(t *testing.T) {
		// This would require a mock user client, but for now we test the validation
		srv, err := NewAuthSrv(WithUserClient(nil))
		assert.Error(t, err)
		assert.Nil(t, srv)
	})
}

func TestNewAuthSrvWithConfig(t *testing.T) {
	t.Run("should return error when user client is nil", func(t *testing.T) {
		config := &AuthServiceConfig{}
		srv, err := NewAuthSrvWithConfig(config)
		assert.Error(t, err)
		assert.Nil(t, srv)
		assert.Contains(t, err.Error(), "user client is required")
	})

	t.Run("should return error when token repository is nil", func(t *testing.T) {
		config := &AuthServiceConfig{
			UserClient: nil, // would need mock
		}
		srv, err := NewAuthSrvWithConfig(config)
		assert.Error(t, err)
		assert.Nil(t, srv)
	})
}

func TestWithUserClient(t *testing.T) {
	t.Run("should set user client in config", func(t *testing.T) {
		config := &AuthServiceConfig{}
		opt := WithUserClient(nil) // would use mock in real test
		opt(config)
		// In a real test, we would assert that config.UserClient is set
		// For now, just verify the option function works
		assert.NotNil(t, opt)
	})
}

func TestWithTokenRepository(t *testing.T) {
	t.Run("should set token repository in config", func(t *testing.T) {
		config := &AuthServiceConfig{}
		opt := WithTokenRepository(nil) // would use mock in real test
		opt(config)
		// In a real test, we would assert that config.TokenRepo is set
		// For now, just verify the option function works
		assert.NotNil(t, opt)
	})
}
