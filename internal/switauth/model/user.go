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
)

// User is the user model.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	Username     string    `gorm:"unique;not null" json:"username"`
	Email        string    `gorm:"unique;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"default:'user'" json:"role"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName returns the database table name for the User model
func (User) TableName() string {
	return "users"
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"john_doe" validate:"required"`
	Password string `json:"password" binding:"required" example:"password123" validate:"required,min=6"`
}

// LoginResponse 登录响应体
type LoginResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// RefreshTokenRequest 刷新token请求体
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." validate:"required"`
}

// RefreshTokenResponse 刷新token响应体
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ValidateTokenResponse 验证token响应体
type ValidateTokenResponse struct {
	Message string    `json:"message" example:"Token is valid"`
	UserID  uuid.UUID `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// LogoutResponse 退出登录响应体
type LogoutResponse struct {
	Message string `json:"message" example:"Logged out successfully"`
}

// ErrorResponse 错误响应体
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid credentials"`
}

// HealthResponse 健康检查响应体
type HealthResponse struct {
	Message string `json:"message" example:"pong"`
}
