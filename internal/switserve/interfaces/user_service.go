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

	"github.com/innovationmech/swit/internal/switserve/model"
)

// UserService defines the interface for user management operations.
// This interface abstracts the user business logic from the transport layer,
// allowing for easy testing and implementation swapping.
//
// All methods should handle context cancellation and return appropriate errors
// using the centralized error types from the types package.
type UserService interface {
	// CreateUser creates a new user in the system.
	// It validates the user data, hashes the password, and stores the user.
	// Returns an error if validation fails or if the user already exists.
	CreateUser(ctx context.Context, user *model.User) error

	// GetUserByUsername retrieves a user by their username.
	// Returns the user if found, or an error if not found or on database error.
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// GetUserByEmail retrieves a user by their email address.
	// Returns the user if found, or an error if not found or on database error.
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)

	// DeleteUser removes a user from the system by their ID.
	// Returns an error if the user is not found or on database error.
	DeleteUser(ctx context.Context, id string) error
}
