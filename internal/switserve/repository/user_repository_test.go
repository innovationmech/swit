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
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	cleanup := func() {
		sqlDB.Close()
	}

	return gormDB, mock, cleanup
}

func TestNewUserRepository(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	assert.NotNil(t, repo)
	assert.Implements(t, (*UserRepository)(nil), repo)
}

func TestUserRepository_CreateUser(t *testing.T) {
	tests := []struct {
		name        string
		user        *model.User
		setupMock   func(sqlmock.Sqlmock, *model.User)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success_create_user_with_new_uuid",
			user: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Role:         "user",
				IsActive:     true,
			},
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `users`")).
					WithArgs(
						sqlmock.AnyArg(), // ID (UUID)
						user.Username,
						user.Email,
						user.PasswordHash,
						user.Role,
						user.IsActive,
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "success_create_user_with_existing_uuid",
			user: &model.User{
				ID:           uuid.New(),
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Role:         "user",
				IsActive:     true,
			},
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `users`")).
					WithArgs(
						user.ID.String(),
						user.Username,
						user.Email,
						user.PasswordHash,
						user.Role,
						user.IsActive,
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "error_database_failure",
			user: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
			},
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `users`")).
					WillReturnError(errors.New("database connection failed"))
				mock.ExpectRollback()
			},
			expectError: true,
			errorMsg:    "database connection failed",
		},
		{
			name: "error_duplicate_entry",
			user: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
			},
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `users`")).
					WillReturnError(errors.New("Error 1062: Duplicate entry"))
				mock.ExpectRollback()
			},
			expectError: true,
			errorMsg:    "Duplicate entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.setupMock(mock, tt.user)

			repo := NewUserRepository(db)
			ctx := context.Background()

			err := repo.CreateUser(ctx, tt.user)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				// Verify UUID was set if it was initially nil
				assert.NotEqual(t, uuid.Nil, tt.user.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByUsername(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		setupMock   func(sqlmock.Sqlmock)
		expectUser  *model.User
		expectError bool
		errorMsg    string
	}{
		{
			name:     "success_get_user",
			username: "testuser",
			setupMock: func(mock sqlmock.Sqlmock) {
				userID := uuid.New()
				now := time.Now()
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "is_active", "created_at", "updated_at"}).
					AddRow(userID.String(), "testuser", "test@example.com", "hashed_password", "user", true, now, now)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE username = ? ORDER BY `users`.`id` LIMIT ?")).
					WithArgs("testuser", 1).
					WillReturnRows(rows)
			},
			expectUser: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Role:         "user",
				IsActive:     true,
			},
			expectError: false,
		},
		{
			name:     "error_user_not_found",
			username: "nonexistent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE username = ? ORDER BY `users`.`id` LIMIT ?")).
					WithArgs("nonexistent", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectUser:  nil,
			expectError: true,
			errorMsg:    "record not found",
		},
		{
			name:     "error_database_failure",
			username: "testuser",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE username = ? ORDER BY `users`.`id` LIMIT ?")).
					WithArgs("testuser", 1).
					WillReturnError(errors.New("database connection failed"))
			},
			expectUser:  nil,
			expectError: true,
			errorMsg:    "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.setupMock(mock)

			repo := NewUserRepository(db)
			ctx := context.Background()

			user, err := repo.GetUserByUsername(ctx, tt.username)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectUser.Username, user.Username)
				assert.Equal(t, tt.expectUser.Email, user.Email)
				assert.Equal(t, tt.expectUser.PasswordHash, user.PasswordHash)
				assert.Equal(t, tt.expectUser.Role, user.Role)
				assert.Equal(t, tt.expectUser.IsActive, user.IsActive)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetUserByEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		setupMock   func(sqlmock.Sqlmock)
		expectUser  *model.User
		expectError bool
		errorMsg    string
	}{
		{
			name:  "success_get_user",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				userID := uuid.New()
				now := time.Now()
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "is_active", "created_at", "updated_at"}).
					AddRow(userID.String(), "testuser", "test@example.com", "hashed_password", "user", true, now, now)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE email = ? ORDER BY `users`.`id` LIMIT ?")).
					WithArgs("test@example.com", 1).
					WillReturnRows(rows)
			},
			expectUser: &model.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Role:         "user",
				IsActive:     true,
			},
			expectError: false,
		},
		{
			name:  "error_user_not_found",
			email: "nonexistent@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE email = ? ORDER BY `users`.`id` LIMIT ?")).
					WithArgs("nonexistent@example.com", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectUser:  nil,
			expectError: true,
			errorMsg:    "record not found",
		},
		{
			name:  "error_database_failure",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE email = ? ORDER BY `users`.`id` LIMIT ?")).
					WithArgs("test@example.com", 1).
					WillReturnError(errors.New("database connection failed"))
			},
			expectUser:  nil,
			expectError: true,
			errorMsg:    "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.setupMock(mock)

			repo := NewUserRepository(db)
			ctx := context.Background()

			user, err := repo.GetUserByEmail(ctx, tt.email)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectUser.Username, user.Username)
				assert.Equal(t, tt.expectUser.Email, user.Email)
				assert.Equal(t, tt.expectUser.PasswordHash, user.PasswordHash)
				assert.Equal(t, tt.expectUser.Role, user.Role)
				assert.Equal(t, tt.expectUser.IsActive, user.IsActive)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_UpdateUser(t *testing.T) {
	tests := []struct {
		name        string
		user        *model.User
		setupMock   func(sqlmock.Sqlmock, *model.User)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success_update_user",
			user: &model.User{
				ID:           uuid.New(),
				Username:     "updateduser",
				Email:        "updated@example.com",
				PasswordHash: "updated_password",
				Role:         "admin",
				IsActive:     false,
			},
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `users`")).
					WithArgs(
						user.Username,
						user.Email,
						user.PasswordHash,
						user.Role,
						user.IsActive,
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
						user.ID.String(),
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "error_database_failure",
			user: &model.User{
				ID:           uuid.New(),
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
			},
			setupMock: func(mock sqlmock.Sqlmock, user *model.User) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `users`")).
					WillReturnError(errors.New("database connection failed"))
				mock.ExpectRollback()
			},
			expectError: true,
			errorMsg:    "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.setupMock(mock, tt.user)

			repo := NewUserRepository(db)
			ctx := context.Background()

			err := repo.UpdateUser(ctx, tt.user)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_DeleteUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		setupMock   func(sqlmock.Sqlmock, string)
		expectError bool
		errorMsg    string
	}{
		{
			name:   "success_delete_user",
			userID: uuid.New().String(),
			setupMock: func(mock sqlmock.Sqlmock, userID string) {
				mock.ExpectBegin()
				mock.ExpectExec("DELETE FROM `users` WHERE").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "error_database_failure",
			userID: uuid.New().String(),
			setupMock: func(mock sqlmock.Sqlmock, userID string) {
				mock.ExpectBegin()
				mock.ExpectExec("DELETE FROM `users` WHERE").
					WillReturnError(errors.New("database connection failed"))
				mock.ExpectRollback()
			},
			expectError: true,
			errorMsg:    "database connection failed",
		},
		{
			name:   "success_delete_nonexistent_user",
			userID: uuid.New().String(),
			setupMock: func(mock sqlmock.Sqlmock, userID string) {
				mock.ExpectBegin()
				mock.ExpectExec("DELETE FROM `users` WHERE").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectError: false, // GORM doesn't return error for 0 affected rows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.setupMock(mock, tt.userID)

			repo := NewUserRepository(db)
			ctx := context.Background()

			err := repo.DeleteUser(ctx, tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_CreateUser_WithContext(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	user := &model.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}

	// Test context behavior - GORM will check context and return error before SQL execution
	repo := NewUserRepository(db)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.CreateUser(ctx, user)
	// Context should be cancelled before SQL execution
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// No SQL expectations since context is cancelled before execution
	assert.NoError(t, mock.ExpectationsWereMet())
}
