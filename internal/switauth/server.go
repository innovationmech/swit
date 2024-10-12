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

package switauth

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/pkg/discovery"
	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/controller"
	"github.com/innovationmech/swit/internal/switauth/repository"
	"github.com/innovationmech/swit/internal/switauth/service"
	"github.com/innovationmech/swit/internal/switauth/store"
)

var cfg *config.AuthConfig

// Server is the server for SWIT authentication service.
type Server struct {
	router *gin.Engine
	sd     *discovery.ServiceDiscovery
}

// NewServer creates a new Server instance
func NewServer() (*Server, error) {
	s := &Server{
		router: gin.Default(),
	}

	initConfig()

	// Using GetServiceDiscovery function to get service discovery instance
	sd, err := config.GetServiceDiscovery()
	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery client: %v", err)
	}
	s.sd = sd

	userClient := client.NewUserClient(sd)
	tokenRepo := repository.NewTokenRepository(store.GetDB())
	authService := service.NewAuthService(userClient, tokenRepo)
	authController := controller.NewAuthController(authService)

	s.router = RegisterRoutes(authController)
	return s, nil
}

// Run starts the server
func (s *Server) Run() error {
	if s.sd == nil {
		return fmt.Errorf("service discovery instance not initialized")
	}

	port, _ := strconv.Atoi(cfg.Server.Port)
	address := fmt.Sprintf(":%d", port)

	// Register service
	err := s.sd.RegisterService("switauth", "http://localhost", port)
	if err != nil {
		return fmt.Errorf("failed to register service: %v", err)
	}
	defer s.sd.DeregisterService("switauth", "http://localhost", port)

	return s.router.Run(address)
}

func initConfig() {
	cfg = config.GetConfig()
}
