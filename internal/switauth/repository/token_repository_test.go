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
	"github.com/innovationmech/swit/internal/switauth/model"
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

func TestNewTokenRepository(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTokenRepository(db)
	assert.NotNil(t, repo)
	assert.Implements(t, (*TokenRepository)(nil), repo)
}

func TestTokenRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		token       *model.Token
		setupMock   func(sqlmock.Sqlmock, *model.Token)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success_create_token_with_new_uuid",
			token: &model.Token{
				UserID:           uuid.New(),
				AccessToken:      "access_token_123",
				RefreshToken:     "refresh_token_123",
				AccessExpiresAt:  time.Now().Add(time.Hour),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				IsValid:          true,
			},
			setupMock: func(mock sqlmock.Sqlmock, token *model.Token) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `tokens`")).
					WithArgs(
						sqlmock.AnyArg(), // ID (UUID)
						token.UserID.String(),
						token.AccessToken,
						token.RefreshToken,
						sqlmock.AnyArg(), // AccessExpiresAt
						sqlmock.AnyArg(), // RefreshExpiresAt
						token.IsValid,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "success_create_token_with_existing_uuid",
			token: &model.Token{
				ID:               uuid.New(),
				UserID:           uuid.New(),
				AccessToken:      "access_token_456",
				RefreshToken:     "refresh_token_456",
				AccessExpiresAt:  time.Now().Add(time.Hour),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				IsValid:          true,
			},
			setupMock: func(mock sqlmock.Sqlmock, token *model.Token) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `tokens`")).
					WithArgs(
						token.ID.String(),
						token.UserID.String(),
						token.AccessToken,
						token.RefreshToken,
						sqlmock.AnyArg(), // AccessExpiresAt
						sqlmock.AnyArg(), // RefreshExpiresAt
						token.IsValid,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "error_database_failure",
			token: &model.Token{
				UserID:           uuid.New(),
				AccessToken:      "access_token_789",
				RefreshToken:     "refresh_token_789",
				AccessExpiresAt:  time.Now().Add(time.Hour),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				IsValid:          true,
			},
			setupMock: func(mock sqlmock.Sqlmock, token *model.Token) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `tokens`")).
					WillReturnError(errors.New("database connection failed"))
				mock.ExpectRollback()
			},
			expectError: true,
			errorMsg:    "database connection failed",
		},
		{
			name: "error_duplicate_token",
			token: &model.Token{
				UserID:           uuid.New(),
				AccessToken:      "duplicate_token",
				RefreshToken:     "refresh_token_123",
				AccessExpiresAt:  time.Now().Add(time.Hour),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				IsValid:          true,
			},
			setupMock: func(mock sqlmock.Sqlmock, token *model.Token) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `tokens`")).
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

			tt.setupMock(mock, tt.token)

			repo := NewTokenRepository(db)
			ctx := context.Background()

			err := repo.Create(ctx, tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				// Verify UUID was set if it was initially nil
				assert.NotEqual(t, uuid.Nil, tt.token.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTokenRepository_GetByAccessToken(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		setupMock   func(sqlmock.Sqlmock)
		expectToken *model.Token
		expectError bool
		errorMsg    string
	}{
		{
			name:        "success_get_valid_token",
			tokenString: "valid_access_token",
			setupMock: func(mock sqlmock.Sqlmock) {
				tokenID := uuid.New()
				userID := uuid.New()
				now := time.Now()
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "access_token", "refresh_token",
					"access_expires_at", "refresh_expires_at", "is_valid",
					"created_at", "updated_at",
				}).AddRow(
					tokenID.String(), userID.String(), "valid_access_token", "refresh_token_123",
					now.Add(time.Hour), now.Add(24*time.Hour), true, now, now,
				)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `tokens` WHERE access_token = ? AND is_valid = ? ORDER BY `tokens`.`id` LIMIT ?")).
					WithArgs("valid_access_token", true, 1).
					WillReturnRows(rows)
			},
			expectToken: &model.Token{
				AccessToken:  "valid_access_token",
				RefreshToken: "refresh_token_123",
				IsValid:      true,
			},
			expectError: false,
		},
		{
			name:        "error_token_not_found",
			tokenString: "nonexistent_token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `tokens` WHERE access_token = ? AND is_valid = ? ORDER BY `tokens`.`id` LIMIT ?")).
					WithArgs("nonexistent_token", true, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectToken: nil,
			expectError: true,
			errorMsg:    "invalid or expired token",
		},
		{
			name:        "error_database_failure",
			tokenString: "test_token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `tokens` WHERE access_token = ? AND is_valid = ? ORDER BY `tokens`.`id` LIMIT ?")).
					WithArgs("test_token", true, 1).
					WillReturnError(errors.New("database connection failed"))
			},
			expectToken: nil,
			expectError: true,
			errorMsg:    "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.setupMock(mock)

			repo := NewTokenRepository(db)
			ctx := context.Background()

			token, err := repo.GetByAccessToken(ctx, tt.tokenString)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tt.expectToken.AccessToken, token.AccessToken)
				assert.Equal(t, tt.expectToken.RefreshToken, token.RefreshToken)
				assert.Equal(t, tt.expectToken.IsValid, token.IsValid)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTokenRepository_GetByRefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		refreshToken string
		setupMock    func(sqlmock.Sqlmock)
		expectToken  *model.Token
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "success_get_valid_refresh_token",
			refreshToken: "valid_refresh_token",
			setupMock: func(mock sqlmock.Sqlmock) {
				tokenID := uuid.New()
				userID := uuid.New()
				now := time.Now()
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "access_token", "refresh_token",
					"access_expires_at", "refresh_expires_at", "is_valid",
					"created_at", "updated_at",
				}).AddRow(
					tokenID.String(), userID.String(), "access_token_123", "valid_refresh_token",
					now.Add(time.Hour), now.Add(24*time.Hour), true, now, now,
				)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `tokens` WHERE refresh_token = ? AND is_valid = ? ORDER BY `tokens`.`id` LIMIT ?")).
					WithArgs("valid_refresh_token", true, 1).
					WillReturnRows(rows)
			},
			expectToken: &model.Token{
				AccessToken:  "access_token_123",
				RefreshToken: "valid_refresh_token",
				IsValid:      true,
			},
			expectError: false,
		},
		{
			name:         "error_refresh_token_not_found",
			refreshToken: "nonexistent_refresh_token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `tokens` WHERE refresh_token = ? AND is_valid = ? ORDER BY `tokens`.`id` LIMIT ?")).
					WithArgs("nonexistent_refresh_token", true, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectToken: nil,
			expectError: true,
			errorMsg:    "invalid or expired refresh token",
		},
		{
			name:         "error_database_failure",
			refreshToken: "test_refresh_token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `tokens` WHERE refresh_token = ? AND is_valid = ? ORDER BY `tokens`.`id` LIMIT ?")).
					WithArgs("test_refresh_token", true, 1).
					WillReturnError(errors.New("database connection failed"))
			},
			expectToken: nil,
			expectError: true,
			errorMsg:    "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.setupMock(mock)

			repo := NewTokenRepository(db)
			ctx := context.Background()

			token, err := repo.GetByRefreshToken(ctx, tt.refreshToken)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tt.expectToken.AccessToken, token.AccessToken)
				assert.Equal(t, tt.expectToken.RefreshToken, token.RefreshToken)
				assert.Equal(t, tt.expectToken.IsValid, token.IsValid)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTokenRepository_Update(t *testing.T) {
	tests := []struct {
		name        string
		token       *model.Token
		setupMock   func(sqlmock.Sqlmock, *model.Token)
		expectError bool
		errorMsg    string
	}{
		{
			name: "success_update_token",
			token: &model.Token{
				ID:               uuid.New(),
				UserID:           uuid.New(),
				AccessToken:      "updated_access_token",
				RefreshToken:     "updated_refresh_token",
				AccessExpiresAt:  time.Now().Add(2 * time.Hour),
				RefreshExpiresAt: time.Now().Add(48 * time.Hour),
				IsValid:          true,
			},
			setupMock: func(mock sqlmock.Sqlmock, token *model.Token) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `tokens`")).
					WithArgs(
						sqlmock.AnyArg(), // user_id
						sqlmock.AnyArg(), // access_token
						sqlmock.AnyArg(), // refresh_token
						sqlmock.AnyArg(), // access_expires_at
						sqlmock.AnyArg(), // refresh_expires_at
						sqlmock.AnyArg(), // is_valid
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
						sqlmock.AnyArg(), // id (WHERE clause)
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "error_database_failure",
			token: &model.Token{
				ID:               uuid.New(),
				UserID:           uuid.New(),
				AccessToken:      "test_token",
				RefreshToken:     "test_refresh_token",
				AccessExpiresAt:  time.Now().Add(time.Hour),
				RefreshExpiresAt: time.Now().Add(24 * time.Hour),
				IsValid:          false,
			},
			setupMock: func(mock sqlmock.Sqlmock, token *model.Token) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `tokens`")).
					WithArgs(
						sqlmock.AnyArg(), // user_id
						sqlmock.AnyArg(), // access_token
						sqlmock.AnyArg(), // refresh_token
						sqlmock.AnyArg(), // access_expires_at
						sqlmock.AnyArg(), // refresh_expires_at
						sqlmock.AnyArg(), // is_valid
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
						sqlmock.AnyArg(), // id (WHERE clause)
					).
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

			tt.setupMock(mock, tt.token)

			repo := NewTokenRepository(db)
			ctx := context.Background()

			err := repo.Update(ctx, tt.token)

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

func TestTokenRepository_InvalidateToken(t *testing.T) {
	tests := []struct {
		name        string
		tokenString string
		setupMock   func(sqlmock.Sqlmock, string)
		expectError bool
		errorMsg    string
	}{
		{
			name:        "success_invalidate_token",
			tokenString: "valid_token_to_invalidate",
			setupMock: func(mock sqlmock.Sqlmock, tokenString string) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `tokens` SET `is_valid`=?,`updated_at`=? WHERE access_token = ?")).
					WithArgs(false, sqlmock.AnyArg(), tokenString).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:        "success_invalidate_nonexistent_token",
			tokenString: "nonexistent_token",
			setupMock: func(mock sqlmock.Sqlmock, tokenString string) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `tokens` SET `is_valid`=?,`updated_at`=? WHERE access_token = ?")).
					WithArgs(false, sqlmock.AnyArg(), tokenString).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectError: false, // GORM doesn't return error for 0 affected rows
		},
		{
			name:        "error_database_failure",
			tokenString: "test_token",
			setupMock: func(mock sqlmock.Sqlmock, tokenString string) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `tokens` SET `is_valid`=?,`updated_at`=? WHERE access_token = ?")).
					WithArgs(false, sqlmock.AnyArg(), tokenString).
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

			tt.setupMock(mock, tt.tokenString)

			repo := NewTokenRepository(db)
			ctx := context.Background()

			err := repo.InvalidateToken(ctx, tt.tokenString)

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

func TestTokenRepository_WithContext(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	token := &model.Token{
		UserID:           uuid.New(),
		AccessToken:      "test_token",
		RefreshToken:     "test_refresh_token",
		AccessExpiresAt:  time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:          true,
	}

	repo := NewTokenRepository(db)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.Create(ctx, token)
	// Context should be cancelled before SQL execution
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// No SQL expectations since context is cancelled before execution
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenRepository_CreateTokenWithExpiredTimes(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	// Test creating token with past expiration times
	token := &model.Token{
		UserID:           uuid.New(),
		AccessToken:      "expired_token",
		RefreshToken:     "expired_refresh_token",
		AccessExpiresAt:  time.Now().Add(-time.Hour), // Past time
		RefreshExpiresAt: time.Now().Add(-time.Hour), // Past time
		IsValid:          true,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `tokens`")).
		WithArgs(
			sqlmock.AnyArg(), // ID
			token.UserID.String(),
			token.AccessToken,
			token.RefreshToken,
			sqlmock.AnyArg(), // AccessExpiresAt
			sqlmock.AnyArg(), // RefreshExpiresAt
			token.IsValid,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	repo := NewTokenRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, token)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, token.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}
