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

package start

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switauth"

	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// cfg is the global configuration for the application.
var cfg *config.AuthConfig

// NewStartCmd creates a new start command for the SWIT authentication service.
func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Short:   "Start the SWIT authentication service",
		Long:    "Start the SWIT authentication service with all necessary configurations and dependencies.",
		Version: "0.0.1",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := switauth.NewServer()
			if err != nil {
				return err
			}

			// Create a long-lived context for the server
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Channel to capture server startup errors
			startupErr := make(chan error, 1)
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := srv.Start(ctx); err != nil {
					startupErr <- err
				}
			}()

			// Wait for server startup with timeout
			select {
			case err := <-startupErr:
				logger.Logger.Error("Server startup failed", zap.Error(err))
				return err
			case <-time.After(5 * time.Second):
				// Server started successfully (no error within timeout)
				logger.Logger.Info("Server started successfully")
			}

			// Setup graceful shutdown
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt)
			<-quit

			logger.Logger.Info("Shutting down authentication server...")

			// Create context with timeout for graceful shutdown
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()

			if err := srv.Stop(shutdownCtx); err != nil {
				logger.Logger.Error("Error during server shutdown", zap.Error(err))
			}

			// Cancel the server context to ensure cleanup
			cancel()

			// Wait for server goroutine to finish
			wg.Wait()

			logger.Logger.Info("Server shut down successfully")
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	return cmd
}

// initConfig initializes the global configuration for the application.
func initConfig() {
	logger.InitLogger()

	// Initialize JWT secret from environment variable
	if err := config.InitJwtSecret(); err != nil {
		logger.Logger.Fatal("Failed to initialize JWT secret", zap.Error(err))
	}

	// Initialize encryption key from environment variable
	if err := utils.InitEncryptionKey(); err != nil {
		logger.Logger.Fatal("Failed to initialize encryption key", zap.Error(err))
	}

	cfg = config.GetConfig()
}
