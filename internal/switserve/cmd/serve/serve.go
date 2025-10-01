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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/innovationmech/swit/internal/switserve"

	cfg "github.com/innovationmech/swit/pkg/config"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
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

	// Initialize config manager and hot reloader for non-breaking updates
	manager := cfg.NewManager(cfg.DefaultOptions())
	if err := manager.Load(); err != nil {
		logger.Logger.Warn("Initial config load failed; continuing", zap.Error(err))
	}
	reloader := cfg.NewHotReloader(manager, 500*time.Millisecond)
	if err := reloader.Start(); err == nil {
		go func() {
			for change := range reloader.Events() {
				if change.Err != nil {
					logger.Logger.Warn("config reload error", zap.Error(change.Err))
					continue
				}
				if v, ok := change.Settings["logging"].(map[string]interface{}); ok {
					if level, ok2 := v["level"].(string); ok2 && level != "" {
						if err := logger.SetLevel(level); err != nil {
							logger.Logger.Warn("apply log level failed", zap.Error(err))
						} else {
							logger.Logger.Info("log level updated via hot-reload", zap.String("level", logger.GetLevel()))
						}
					}
				}
			}
		}()
	} else {
		logger.Logger.Debug("hot reloader not started", zap.Error(err))
	}

	// Create server
	srv, err := switserve.NewServer()
	if err != nil {
		logger.Logger.Error("Failed to create server", zap.Error(err))
		return err
	}

	// Initialize configuration monitoring and drift detection
	if pmc := srv.GetPrometheusCollector(); pmc != nil {
		var collector types.MetricsCollector = pmc
		monitor := cfg.NewConfigMonitor(manager, reloader, collector, cfg.ConfigMonitorOptions{
			DesiredConfigPath:    "", // optional, can be set by env SWIT_CONFIG_DESIRED_FILE
			EnableSectionMetrics: true,
			SectionLabelLimit:    12,
		})
		if err := monitor.Start(context.Background()); err != nil {
			logger.Logger.Warn("config monitor start failed", zap.Error(err))
		} else {
			logger.Logger.Info("configuration monitoring started")
		}
	} else {
		logger.Logger.Debug("Prometheus collector not available; config monitoring metrics disabled")
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		// Print registered transports
		transports := srv.GetTransports()
		logger.Logger.Info("Registered transports",
			zap.Int("count", len(transports)),
		)

		for _, transport := range transports {
			logger.Logger.Info("Transport registered",
				zap.String("name", transport.GetName()),
				zap.String("address", transport.GetAddress()),
			)
		}

		if err := srv.Start(ctx); err != nil {
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
