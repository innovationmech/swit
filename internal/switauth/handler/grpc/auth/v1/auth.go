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
	"github.com/innovationmech/swit/internal/switauth/service"
)

// Server handles gRPC requests for authentication operations
// This is a placeholder for future gRPC authentication service implementation
type Server struct {
	// UnimplementedAuthServiceServer should be embedded when protobuf is available
	authService service.AuthService
}

// NewAuthServer creates a new gRPC auth server instance
// with the provided authentication service
func NewAuthServer(authService service.AuthService) *Server {
	return &Server{authService: authService}
}

// TODO: Implement gRPC methods when protobuf definitions are available
// Example methods that would be implemented:
// - Login(context.Context, *LoginRequest) (*LoginResponse, error)
// - Logout(context.Context, *LogoutRequest) (*LogoutResponse, error)
// - RefreshToken(context.Context, *RefreshTokenRequest) (*RefreshTokenResponse, error)
// - ValidateToken(context.Context, *ValidateTokenRequest) (*ValidateTokenResponse, error)
