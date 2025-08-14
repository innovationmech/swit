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

package interfaces

import (
	"context"

	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/types"
)

// AuthService defines the interface for authentication business logic
// including login, logout, token refresh, and token validation operations
type AuthService interface {
	// Login authenticates a user and returns access and refresh tokens
	Login(ctx context.Context, username, password string) (*types.AuthResponse, error)

	// RefreshToken generates new tokens using a refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*types.AuthResponse, error)

	// ValidateToken validates an access token and returns token details
	ValidateToken(ctx context.Context, tokenString string) (*model.Token, error)

	// Logout invalidates a token and logs out the user
	Logout(ctx context.Context, tokenString string) error
}
