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

package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/pkg/discovery"
	"github.com/innovationmech/swit/internal/pkg/logger"
	"github.com/innovationmech/swit/internal/switserve/config"
	"go.uber.org/zap"
)

// Server is the server for the application.
type Server struct {
	router *gin.Engine
	sd     *discovery.ServiceDiscovery
	srv    *http.Server
}

// NewServer creates a new server for the application.
func NewServer() (*Server, error) {
	s := &Server{
		router: gin.Default(),
	}

	// Get service discovery address from configuration
	sdAddress := config.GetConfig().ServiceDiscovery.Address
	sd, err := discovery.NewServiceDiscovery(sdAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery client: %v", err)
	}
	s.sd = sd

	s.SetupRoutes()
	return s, nil
}

// Run runs the server on the given address.
func (s *Server) Run(addr string) error {
	// Register service
	port, _ := strconv.Atoi(config.GetConfig().Server.Port)
	err := s.sd.RegisterService("swit-serve", "http://localhost", port)
	if err != nil {
		return fmt.Errorf("failed to register service: %v", err)
	}
	defer s.sd.DeregisterService("swit-serve", "http://localhost", port)

	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return s.srv.ListenAndServe()
}

// Shutdown shuts down the server and deregisters it from the service discovery.
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		logger.Logger.Error("Shutdown error", zap.Error(err))
	}
	port, _ := strconv.Atoi(config.GetConfig().Server.Port)
	if err := s.sd.DeregisterService("swit-serve", "http://localhost", port); err != nil {
		logger.Logger.Error("Deregister service error", zap.Error(err))
	}
}
