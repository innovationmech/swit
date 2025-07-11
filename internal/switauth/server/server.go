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

package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/db"
	"github.com/innovationmech/swit/internal/switauth/handler"
	"github.com/innovationmech/swit/internal/switauth/repository"
	"github.com/innovationmech/swit/internal/switauth/service"
)

var cfg *config.AuthConfig

// Server is the server for SWIT authentication service.
type Server struct {
	router *gin.Engine
	sd     *discovery.ServiceDiscovery
	srv    *http.Server
}

// NewServer creates a new Server instance
func NewServer() (*Server, error) {
	s := &Server{
		router: gin.Default(),
	}

	initConfig()

	// Using unified service discovery manager
	cfg := config.GetConfig()
	sd, err := discovery.GetServiceDiscoveryByAddress(cfg.ServiceDiscovery.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery client: %v", err)
	}
	s.sd = sd

	userClient := client.NewUserClient(sd)
	tokenRepo := repository.NewTokenRepository(db.GetDB())
	authService := service.NewAuthService(userClient, tokenRepo)
	authController := handler.NewAuthController(authService)

	s.SetupRoutes(authController)
	return s, nil
}

// Run starts the server
func (s *Server) Run(addr string) error {
	var wg sync.WaitGroup
	readyGin := make(chan struct{})
	wg.Add(1)

	go s.runGinServer(addr, &wg, readyGin)

	<-readyGin

	port, _ := strconv.Atoi(config.GetConfig().Server.Port)
	// Register service with Consul
	err := s.sd.RegisterService("swit-auth", "localhost", port)
	if err != nil {
		logger.Logger.Error("failed to register swit-auth service", zap.Error(err))
		return err
	}
	wg.Wait()
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		logger.Logger.Error("Shutdown error", zap.Error(err))
	}
	port, _ := strconv.Atoi(config.GetConfig().Server.Port)
	// Deregister service from Consul
	if err := s.sd.DeregisterService("swit-auth", "localhost", port); err != nil {
		logger.Logger.Error("Deregister service error", zap.Error(err))
	} else {
		logger.Logger.Info("Service deregistered successfully")
	}
}

// runGinServer starts the Gin server on the specified address
func (s *Server) runGinServer(addr string, wg *sync.WaitGroup, ch chan struct{}) {
	defer wg.Done()
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	ln, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		fmt.Printf("listen gin server error: %v\n", err)
		return
	}

	close(ch)

	if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("listen gin server error: %v\n", err)
		return
	}
}

func initConfig() {
	cfg = config.GetConfig()
}
