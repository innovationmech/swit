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
	"strings"

	"github.com/innovationmech/swit/internal/swit-serve/db"
	v1 "github.com/innovationmech/swit/internal/swit-serve/entity/v1"
	"gorm.io/gorm"
)

// UserSrv is the service for the user entity.
type UserSrv interface {
	CreateUser(user *v1.User) error
	GetUserByID(id string) (*v1.User, error)
	GetUserByUsername(username string) (*v1.User, error)
	GetUserByEmail(email string) (*v1.User, error)
	UpdateUser(id string, updates map[string]interface{}) error
	DeleteUser(id string) error
}

// userService is the implementation of the UserSrv interface.
type userService struct {
	db *gorm.DB
}

// NewUserSrv creates a new user service.
func NewUserSrv() UserSrv {
	return &userService{
		db: db.GetDB(),
	}
}

// CreateUser creates a new user.
func (s *userService) CreateUser(user *v1.User) error {
	// Add validation
	if user.Username == "" || user.Email == "" {
		return errors.New("username and email cannot be empty")
	}

	err := s.db.AutoMigrate(user)
	if err != nil {
		return err
	}

	// Attempt to create the user directly, letting the database handle uniqueness constraints
	err = s.db.Create(user).Error
	if err != nil {
		// Check if it's a uniqueness constraint error
		if strings.Contains(err.Error(), "Duplicate entry") {
			return errors.New("this email is already in use")
		}
		return err
	}

	return nil
}

// GetUserByID gets a user by ID.
func (s *userService) GetUserByID(id string) (*v1.User, error) {
	var user v1.User
	err := s.db.First(&user, "id = ?", id).Error
	return &user, err
}

// GetUserByUsername gets a user by username.
func (s *userService) GetUserByUsername(username string) (*v1.User, error) {
	var user v1.User
	err := s.db.First(&user, "username = ?", username).Error
	return &user, err
}

// GetUserByEmail gets a user by email.
func (s *userService) GetUserByEmail(email string) (*v1.User, error) {
	var user v1.User
	err := s.db.First(&user, "email = ?", email).Error
	return &user, err
}

// UpdateUser updates a user by ID.
func (s *userService) UpdateUser(id string, updates map[string]interface{}) error {
	// First, find the user
	var user v1.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return err
	}

	// Update user information
	return s.db.Model(&user).Updates(updates).Error
}

// DeleteUser deletes a user by ID.
func (s *userService) DeleteUser(id string) error {
	return s.db.Delete(&v1.User{}, "id = ?", id).Error
}
