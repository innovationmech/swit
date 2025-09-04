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
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/model"
	"github.com/innovationmech/swit/internal/switserve/repository"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/tracing"
	"github.com/innovationmech/swit/pkg/utils"
)

// UserSrv is the legacy interface for backward compatibility
// Deprecated: Use interfaces.UserService instead
type UserSrv = interfaces.UserService

// UserServiceConfig user service config
type UserServiceConfig struct {
	UserRepo       repository.UserRepository
	TracingManager tracing.TracingManager
	// Logger      logger.Logger
	// Cache       cache.Cache
	// EventBus    eventbus.EventBus
	// Validator   validator.Validator
}

// UserServiceOption user service option function type
type UserServiceOption func(*UserServiceConfig)

// userService is the implementation of the UserSrv interface
type userService struct {
	config *UserServiceConfig
}

// WithUserRepository set the user repository dependency
func WithUserRepository(repo repository.UserRepository) UserServiceOption {
	return func(config *UserServiceConfig) {
		config.UserRepo = repo
	}
}

// WithTracingManager set the tracing manager dependency
func WithTracingManager(tracingManager tracing.TracingManager) UserServiceOption {
	return func(config *UserServiceConfig) {
		config.TracingManager = tracingManager
	}
}

// can add more options functions, such as:
// func WithLogger(logger logger.Logger) UserServiceOption {
//     return func(config *UserServiceConfig) {
//         config.Logger = logger
//     }
// }

// func WithCache(cache cache.Cache) UserServiceOption {
//     return func(config *UserServiceConfig) {
//         config.Cache = cache
//     }
// }

// NewUserSrv creates a new user service using options pattern.
func NewUserSrv(opts ...UserServiceOption) (interfaces.UserService, error) {
	config := &UserServiceConfig{}

	// apply options
	for _, opt := range opts {
		opt(config)
	}

	// check if the user repository is required
	if config.UserRepo == nil {
		return nil, types.ErrValidation("user repository is required")
	}

	return &userService{
		config: config,
	}, nil
}

// NewUserSrvWithConfig creates a new user service using config struct
func NewUserSrvWithConfig(config *UserServiceConfig) (interfaces.UserService, error) {
	if config.UserRepo == nil {
		return nil, types.ErrValidation("user repository is required")
	}

	return &userService{
		config: config,
	}, nil
}

// CreateUser creates a new user
func (s *userService) CreateUser(ctx context.Context, user *model.User) error {
	// Create tracing span
	var span tracing.Span
	if s.config.TracingManager != nil {
		ctx, span = s.config.TracingManager.StartSpan(ctx, "UserService.CreateUser",
			tracing.WithAttributes(
				attribute.String("user.email", user.Email),
				attribute.String("user.username", user.Username),
			),
		)
		defer span.End()
	}

	// Add validation span
	if s.config.TracingManager != nil {
		_, validationSpan := s.config.TracingManager.StartSpan(ctx, "validate_user_input")
		if user.Username == "" || user.Email == "" {
			validationSpan.SetStatus(codes.Error, "username and email cannot be empty")
			span.SetStatus(codes.Error, "validation failed")
			validationSpan.End()
			return types.ErrValidation("username and email cannot be empty")
		}
		validationSpan.End()
	} else {
		// Fallback validation if no tracing
		if user.Username == "" || user.Email == "" {
			return types.ErrValidation("username and email cannot be empty")
		}
	}

	// Hash password span
	if s.config.TracingManager != nil {
		_, hashSpan := s.config.TracingManager.StartSpan(ctx, "hash_password")
		hashedPassword, err := utils.HashPassword(user.Password)
		if err != nil {
			hashSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, "password hashing failed")
			hashSpan.End()
			return types.ErrInternalWithCause("failed to hash password", err)
		}
		user.PasswordHash = hashedPassword
		// Clear the plain text password for security
		user.Password = ""
		hashSpan.End()
	} else {
		// Fallback password hashing if no tracing
		hashedPassword, err := utils.HashPassword(user.Password)
		if err != nil {
			return types.ErrInternalWithCause("failed to hash password", err)
		}
		user.PasswordHash = hashedPassword
		user.Password = ""
	}

	// Database operation span (GORM will automatically trace this if DB tracing is enabled)
	err := s.config.UserRepo.CreateUser(ctx, user)
	if err != nil {
		// check if it's a uniqueness constraint error
		if strings.Contains(err.Error(), "Duplicate entry") {
			if span != nil {
				span.SetStatus(codes.Error, "duplicate email")
				span.SetAttribute("error.type", "conflict")
			}
			return types.ErrConflict("this email is already in use")
		}
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
			span.SetAttribute("error.type", "database")
		}
		return types.ErrInternalWithCause("failed to create user", err)
	}

	// Record success
	if span != nil {
		span.SetAttribute("user.id", user.ID.String())
		span.SetAttribute("operation.success", true)
	}

	return nil
}

// GetUserByUsername gets a user by username
func (s *userService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	// Create tracing span
	var span tracing.Span
	if s.config.TracingManager != nil {
		ctx, span = s.config.TracingManager.StartSpan(ctx, "UserService.GetUserByUsername",
			tracing.WithAttributes(
				attribute.String("user.username", username),
			),
		)
		defer span.End()
	}

	if username == "" {
		if span != nil {
			span.SetStatus(codes.Error, "username cannot be empty")
		}
		return nil, types.ErrValidation("username cannot be empty")
	}

	user, err := s.config.UserRepo.GetUserByUsername(ctx, username)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if span != nil {
				span.SetStatus(codes.Error, "user not found")
				span.SetAttribute("error.type", "not_found")
			}
			return nil, types.ErrNotFound("user")
		}
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
			span.SetAttribute("error.type", "database")
		}
		return nil, types.ErrInternalWithCause("failed to get user by username", err)
	}
	if user == nil {
		if span != nil {
			span.SetStatus(codes.Error, "user not found")
			span.SetAttribute("error.type", "not_found")
		}
		return nil, types.ErrNotFound("user")
	}

	if span != nil {
		span.SetAttribute("user.id", user.ID.String())
		span.SetAttribute("operation.success", true)
	}

	return user, nil
}

// GetUserByEmail gets a user by email
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	// Create tracing span
	var span tracing.Span
	if s.config.TracingManager != nil {
		ctx, span = s.config.TracingManager.StartSpan(ctx, "UserService.GetUserByEmail",
			tracing.WithAttributes(
				attribute.String("user.email", email),
			),
		)
		defer span.End()
	}

	if email == "" {
		if span != nil {
			span.SetStatus(codes.Error, "email cannot be empty")
		}
		return nil, types.ErrValidation("email cannot be empty")
	}

	user, err := s.config.UserRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if span != nil {
				span.SetStatus(codes.Error, "user not found")
				span.SetAttribute("error.type", "not_found")
			}
			return nil, types.ErrNotFound("user")
		}
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
			span.SetAttribute("error.type", "database")
		}
		return nil, types.ErrInternalWithCause("failed to get user by email", err)
	}
	if user == nil {
		if span != nil {
			span.SetStatus(codes.Error, "user not found")
			span.SetAttribute("error.type", "not_found")
		}
		return nil, types.ErrNotFound("user")
	}

	if span != nil {
		span.SetAttribute("user.id", user.ID.String())
		span.SetAttribute("operation.success", true)
	}

	return user, nil
}

// DeleteUser deletes a user by ID
func (s *userService) DeleteUser(ctx context.Context, id string) error {
	// Create tracing span
	var span tracing.Span
	if s.config.TracingManager != nil {
		ctx, span = s.config.TracingManager.StartSpan(ctx, "UserService.DeleteUser",
			tracing.WithAttributes(
				attribute.String("user.id", id),
			),
		)
		defer span.End()
	}

	if id == "" {
		if span != nil {
			span.SetStatus(codes.Error, "user ID cannot be empty")
		}
		return types.ErrValidation("user ID cannot be empty")
	}

	err := s.config.UserRepo.DeleteUser(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if span != nil {
				span.SetStatus(codes.Error, "user not found")
				span.SetAttribute("error.type", "not_found")
			}
			return types.ErrNotFound("user")
		}
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
			span.SetAttribute("error.type", "database")
		}
		return types.ErrInternalWithCause("failed to delete user", err)
	}

	if span != nil {
		span.SetAttribute("operation.success", true)
	}

	return nil
}
