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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server is the server for the application.
type Server struct {
	router     *gin.Engine
	sd         *discovery.ServiceDiscovery
	srv        *http.Server
	grpcServer *grpc.Server
}

// NewServer creates a new server for the application.
func NewServer() (*Server, error) {
	s := &Server{
		router: gin.Default(),
		// grpcServer will be initialized in runGRPCServer()
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
	var wg sync.WaitGroup
	readyGin := make(chan struct{})
	wg.Add(2)

	go s.runGinServer(addr, &wg, readyGin)
	go s.runGRPCServer(&wg)

	<-readyGin

	// Register service
	port, _ := strconv.Atoi(config.GetConfig().Server.Port)
	err := s.sd.RegisterService("swit-serve", "localhost", port)
	if err != nil {
		logger.Logger.Error("failed to register swit-serve service", zap.Error(err))
		return err
	}

	wg.Wait()
	return nil
}

// Shutdown shuts down the server and deregisters it from the service discovery.
func (s *Server) Shutdown() {
	shutdownTimeout := 5 * time.Second

	// Gracefully shutdown gRPC server
	s.GracefulShutdownGRPC(shutdownTimeout)

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		logger.Logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.Logger.Info("HTTP server shut down successfully")
	}

	// Deregister service from service discovery
	port, _ := strconv.Atoi(config.GetConfig().Server.Port)
	if err := s.sd.DeregisterService("swit-serve", "localhost", port); err != nil {
		logger.Logger.Error("Deregister service error", zap.Error(err))
	} else {
		logger.Logger.Info("Service deregistered successfully", zap.String("service", "swit-serve"))
	}
}
