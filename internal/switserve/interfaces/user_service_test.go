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
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/innovationmech/swit/internal/switserve/model"
)

// MockUserService is a mock implementation of UserService for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestUserService_Interface(t *testing.T) {
	// Test that MockUserService implements UserService interface
	var _ UserService = (*MockUserService)(nil)
}

func TestMockUserService_CreateUser(t *testing.T) {
	mockService := new(MockUserService)
	ctx := context.Background()
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Test successful creation
	mockService.On("CreateUser", ctx, user).Return(nil)
	err := mockService.CreateUser(ctx, user)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test creation failure
	mockService = new(MockUserService)
	expectedErr := errors.New("user already exists")
	mockService.On("CreateUser", ctx, user).Return(expectedErr)
	err = mockService.CreateUser(ctx, user)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockUserService_GetUserByUsername(t *testing.T) {
	mockService := new(MockUserService)
	ctx := context.Background()
	username := "testuser"
	expectedUser := &model.User{
		ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Username: username,
		Email:    "test@example.com",
	}

	// Test successful retrieval
	mockService.On("GetUserByUsername", ctx, username).Return(expectedUser, nil)
	user, err := mockService.GetUserByUsername(ctx, username)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockService.AssertExpectations(t)

	// Test user not found
	mockService = new(MockUserService)
	expectedErr := errors.New("user not found")
	mockService.On("GetUserByUsername", ctx, username).Return(nil, expectedErr)
	user, err = mockService.GetUserByUsername(ctx, username)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockUserService_GetUserByEmail(t *testing.T) {
	mockService := new(MockUserService)
	ctx := context.Background()
	email := "test@example.com"
	expectedUser := &model.User{
		ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Username: "testuser",
		Email:    email,
	}

	// Test successful retrieval
	mockService.On("GetUserByEmail", ctx, email).Return(expectedUser, nil)
	user, err := mockService.GetUserByEmail(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockService.AssertExpectations(t)

	// Test user not found
	mockService = new(MockUserService)
	expectedErr := errors.New("user not found")
	mockService.On("GetUserByEmail", ctx, email).Return(nil, expectedErr)
	user, err = mockService.GetUserByEmail(ctx, email)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockUserService_DeleteUser(t *testing.T) {
	mockService := new(MockUserService)
	ctx := context.Background()
	userID := "123"

	// Test successful deletion
	mockService.On("DeleteUser", ctx, userID).Return(nil)
	err := mockService.DeleteUser(ctx, userID)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test deletion failure
	mockService = new(MockUserService)
	expectedErr := errors.New("user not found")
	mockService.On("DeleteUser", ctx, userID).Return(expectedErr)
	err = mockService.DeleteUser(ctx, userID)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}
