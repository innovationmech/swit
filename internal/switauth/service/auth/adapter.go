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

package auth

import (
	"context"

	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/service"
	"github.com/innovationmech/swit/internal/switauth/service/auth/v1"
)

// AuthServiceAdapter adapts v1.AuthSrv to service.AuthService interface
// This provides backward compatibility for existing handlers
type AuthServiceAdapter struct {
	v1Service v1.AuthSrv
}

// NewAuthServiceAdapter creates a new adapter for v1.AuthSrv
func NewAuthServiceAdapter(v1Service v1.AuthSrv) service.AuthService {
	return &AuthServiceAdapter{
		v1Service: v1Service,
	}
}

// Login adapts v1.AuthSrv.Login to service.AuthService.Login
func (a *AuthServiceAdapter) Login(ctx context.Context, username, password string) (string, string, error) {
	resp, err := a.v1Service.Login(ctx, username, password)
	if err != nil {
		return "", "", err
	}
	return resp.AccessToken, resp.RefreshToken, nil
}

// RefreshToken adapts v1.AuthSrv.RefreshToken to service.AuthService.RefreshToken
func (a *AuthServiceAdapter) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	resp, err := a.v1Service.RefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", err
	}
	return resp.AccessToken, resp.RefreshToken, nil
}

// ValidateToken delegates to v1.AuthSrv.ValidateToken
func (a *AuthServiceAdapter) ValidateToken(ctx context.Context, tokenString string) (*model.Token, error) {
	return a.v1Service.ValidateToken(ctx, tokenString)
}

// Logout delegates to v1.AuthSrv.Logout
func (a *AuthServiceAdapter) Logout(ctx context.Context, tokenString string) error {
	return a.v1Service.Logout(ctx, tokenString)
}
