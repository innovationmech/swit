// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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
	"errors"
	"github.com/innovationmech/swit/internal/pkg/utils"
	"strings"

	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/innovationmech/swit/internal/switserve/repository"
)

// UserSrv is the service for the user model.
type UserSrv interface {
	CreateUser(user *model.User) error
	GetUserByUsername(username string) (*model.User, error)
	GetUserByEmail(email string) (*model.User, error)
	DeleteUser(id string) error
}

// userService is the implementation of the UserSrv interface.
type userService struct {
	repo repository.UserRepository
}

// NewUserSrv creates a new user service.
func NewUserSrv(repo repository.UserRepository) UserSrv {
	return &userService{
		repo: repo,
	}
}

// CreateUser creates a new user.
func (s *userService) CreateUser(user *model.User) error {
	// Add validation
	if user.Username == "" || user.Email == "" {
		return errors.New("username and email cannot be empty")
	}

	hashedPassword, err := utils.HashPassword(user.PasswordHash)
	if err != nil {
		return err
	}
	user.PasswordHash = hashedPassword

	// Attempt to create the user directly, letting the database handle uniqueness constraints
	err = s.repo.CreateUser(user)
	if err != nil {
		// Check if it's a uniqueness constraint error
		if strings.Contains(err.Error(), "Duplicate entry") {
			return errors.New("this email is already in use")
		}
		return err
	}

	return nil
}

// GetUserByUsername gets a user by username.
func (s *userService) GetUserByUsername(username string) (*model.User, error) {
	return s.repo.GetUserByUsername(username)
}

// GetUserByEmail gets a user by email.
func (s *userService) GetUserByEmail(email string) (*model.User, error) {
	return s.repo.GetUserByEmail(email)
}

// DeleteUser deletes a user by ID.
func (s *userService) DeleteUser(id string) error {
	return s.repo.DeleteUser(id)
}
