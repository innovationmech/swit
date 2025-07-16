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

package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User is the user model.
//
//	@Description	User account information
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;" json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username     string    `gorm:"unique;not null" json:"username" example:"john_doe"`
	Email        string    `gorm:"unique;not null" json:"email" example:"john@example.com"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"default:'user'" json:"role" example:"user"`
	IsActive     bool      `gorm:"default:true" json:"is_active" example:"true"`
	CreatedAt    time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt    time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

// BeforeCreate is a GORM hook that runs before creating a new user
func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return
}

// CreateUserRequest represents the request body for creating a new user
//
//	@Description	Request body for creating a new user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required" example:"john_doe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
}
