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

package serve

import (
	"context"
	"github.com/innovationmech/swit/internal/switserve"
	"os"
	"os/signal"
	"syscall"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewServeCmd creates a new serve command.
func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the SWIT server",
		Long: `Start the SWIT server using the improved architecture with:
- Unified gRPC and HTTP transport management
- Clean separation of business logic and protocol handling
- Enterprise-grade middleware support
- Multiple service registration (Greeter, Notification)`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize logger if not already done
			logger.InitLogger()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}

	return cmd
}

// runServer runs the SWIT server
func runServer() error {
	logger.Logger.Info("Starting SWIT server...")

	// Create server
	srv, err := switserve.NewServer()
	if err != nil {
		logger.Logger.Error("Failed to create server", zap.Error(err))
		return err
	}

	// Create context for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		// 获取传输层信息
		logger.Logger.Info("HTTP transport started", zap.String("address", srv.GetHTTPAddress()))
		logger.Logger.Info("gRPC transport started", zap.String("address", srv.GetGRPCAddress()))

		// 启动服务器
		if err := srv.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-quit:
		logger.Logger.Info("Shutdown signal received, stopping server...")
		cancel()

		if err := srv.Stop(); err != nil {
			logger.Logger.Error("Error during server shutdown", zap.Error(err))
			return err
		}

		logger.Logger.Info("Server shutdown complete")
	case err := <-serverErr:
		logger.Logger.Error("Server error", zap.Error(err))
		return err
	}

	return nil
}
