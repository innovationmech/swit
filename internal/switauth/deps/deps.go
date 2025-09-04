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
	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/db"
	"github.com/innovationmech/swit/internal/switauth/interfaces"
	"github.com/innovationmech/swit/internal/switauth/repository"
	authv1 "github.com/innovationmech/swit/internal/switauth/service/auth/v1"
	"github.com/innovationmech/swit/internal/switauth/service/health"
	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/tracing"
	"gorm.io/gorm"
)

// Dependencies holds all the service dependencies for switauth
type Dependencies struct {
	// Core dependencies
	DB     *gorm.DB
	Config *config.AuthConfig
	SD     *discovery.ServiceDiscovery

	// Repository layer
	TokenRepo repository.TokenRepository

	// Client layer
	UserClient client.UserClient

	// Service layer
	AuthSrv   interfaces.AuthService
	HealthSrv interfaces.HealthService

	// Tracing
	TracingManager tracing.TracingManager
}

// NewDependencies creates and initializes all service dependencies
func NewDependencies() (*Dependencies, error) {
	// Initialize configuration
	cfg := config.GetConfig()

	// Initialize tracing manager
	tracingManager := tracing.NewTracingManager()

	// Initialize database
	database := db.GetDB()

	// Setup database tracing
	if err := db.SetupTracing(database, tracingManager); err != nil {
		// Don't fail if tracing setup fails - continue without tracing
		// Note: proper logging would be added here in production
	}

	// Initialize service discovery
	sd, err := discovery.GetServiceDiscoveryByAddress(cfg.ServiceDiscovery.Address)
	if err != nil {
		return nil, err
	}

	// Initialize repository layer
	tokenRepo := repository.NewTokenRepository(database)

	// Initialize client layer
	userClient := client.NewUserClient(sd)

	// Initialize service layer
	authSrv, err := authv1.NewAuthSrv(
		authv1.WithUserClient(userClient),
		authv1.WithTokenRepository(tokenRepo),
		authv1.WithTracingManager(tracingManager),
	)
	if err != nil {
		return nil, err
	}

	healthSrv := health.NewHealthService()

	return &Dependencies{
		DB:             database,
		Config:         cfg,
		SD:             sd,
		TokenRepo:      tokenRepo,
		UserClient:     userClient,
		AuthSrv:        authSrv,
		HealthSrv:      healthSrv,
		TracingManager: tracingManager,
	}, nil
}
