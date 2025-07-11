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

package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switserve/model"
	"gorm.io/gorm"
)

// UserRepository is the interface for the user repository.
type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id string) error
}

// userRepository is the implementation of the UserRepository interface.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// CreateUser creates a new user.
func (u userRepository) CreateUser(ctx context.Context, user *model.User) error {
	// 确保 UUID 被设置
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	result := u.db.WithContext(ctx).Create(user)
	return result.Error
}

// GetUserByUsername gets a user by username.
func (u userRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	result := u.db.WithContext(ctx).Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByEmail gets a user by email.
func (u userRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	result := u.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// UpdateUser updates a user.
func (u userRepository) UpdateUser(ctx context.Context, user *model.User) error {
	result := u.db.WithContext(ctx).Save(user)
	return result.Error
}

// DeleteUser deletes a user.
func (u userRepository) DeleteUser(ctx context.Context, id string) error {
	result := u.db.WithContext(ctx).Delete(&model.User{}, id)
	return result.Error
}
