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

package deps

import (
	"fmt"

	"github.com/innovationmech/swit/internal/switserve/db"
	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/repository"
	greeterv1 "github.com/innovationmech/swit/internal/switserve/service/greeter/v1"
	"github.com/innovationmech/swit/internal/switserve/service/health"
	notificationv1 "github.com/innovationmech/swit/internal/switserve/service/notification/v1"
	"github.com/innovationmech/swit/internal/switserve/service/stop"
	userv1 "github.com/innovationmech/swit/internal/switserve/service/user/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Dependencies manages all service dependencies using interface types
type Dependencies struct {
	// Infrastructure
	DB *gorm.DB

	// Repository layer
	UserRepo repository.UserRepository

	// Service layer - using interface types for better dependency injection
	UserSrv         interfaces.UserService
	GreeterSrv      interfaces.GreeterService
	NotificationSrv notificationv1.NotificationService
	HealthSrv       interfaces.HealthService
	StopSrv         interfaces.StopService
}

// NewDependencies creates and initializes all service dependencies
func NewDependencies(shutdownFunc func()) (*Dependencies, error) {
	// 1. Initialize infrastructure
	database := db.GetDB()
	if database == nil {
		logger.Logger.Error("failed to get database connection")
		return nil, ErrDatabaseConnection
	}

	// 2. Initialize Repository layer
	userRepo := repository.NewUserRepository(database)

	// 3. Initialize Service layer with interface types
	userSrv, err := userv1.NewUserSrv(
		userv1.WithUserRepository(userRepo),
	)
	if err != nil {
		logger.Logger.Error("failed to create user service", zap.Error(err))
		return nil, fmt.Errorf("%w: user service - %v", ErrServiceInitialization, err)
	}

	// Initialize other services
	greeterSrv := greeterv1.NewService()
	notificationSrv := notificationv1.NewService()
	healthSrv := health.NewHealthSrv()
	stopSrv := stop.NewService(shutdownFunc)

	// Create dependencies struct
	deps := &Dependencies{
		DB:              database,
		UserRepo:        userRepo,
		UserSrv:         userSrv, // userv1.UserSrv implements interfaces.UserService
		GreeterSrv:      greeterSrv,
		NotificationSrv: notificationSrv,
		HealthSrv:       healthSrv,
		StopSrv:         stopSrv,
	}

	// Validate dependencies
	if err := deps.Validate(); err != nil {
		logger.Logger.Error("dependency validation failed", zap.Error(err))
		return nil, fmt.Errorf("%w: validation failed - %v", ErrServiceInitialization, err)
	}

	logger.Logger.Info("successfully initialized all dependencies")
	return deps, nil
}

// Validate checks that all required dependencies are properly initialized
func (d *Dependencies) Validate() error {
	if d == nil {
		return fmt.Errorf("dependencies struct is nil")
	}

	// Validate infrastructure dependencies
	if d.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Validate repository layer
	if d.UserRepo == nil {
		return fmt.Errorf("user repository is nil")
	}

	// Validate service layer
	if d.UserSrv == nil {
		return fmt.Errorf("user service is nil")
	}
	if d.GreeterSrv == nil {
		return fmt.Errorf("greeter service is nil")
	}
	if d.NotificationSrv == nil {
		return fmt.Errorf("notification service is nil")
	}
	if d.HealthSrv == nil {
		return fmt.Errorf("health service is nil")
	}
	if d.StopSrv == nil {
		return fmt.Errorf("stop service is nil")
	}

	return nil
}

// Close gracefully closes all dependencies
func (d *Dependencies) Close() error {
	// Check if database is nil
	if d.DB == nil {
		return nil // Nothing to close
	}

	// Close database connections and other resources if needed
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err // Return error from d.DB.DB()
	}

	return sqlDB.Close()
}
