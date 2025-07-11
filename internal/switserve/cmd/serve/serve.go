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
	"github.com/innovationmech/swit/internal/switserve/config"
	"github.com/innovationmech/swit/internal/switserve/server"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

// cfg is the global configuration for the application.
var cfg *config.ServeConfig

// NewServeCmd creates a new serve command.
func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the SWIT server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := server.NewServer()
			if err != nil {
				return err
			}
			go func() {
				if err := srv.Run(":" + cfg.Server.Port); err != nil {
					logger.Logger.Error("Server exited", zap.Error(err))
				}
			}()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit

			logger.Logger.Info("Shutting down server...")
			srv.Shutdown()
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	return cmd
}

// initConfig initializes the global configuration for the application.
func initConfig() {
	logger.InitLogger()
	cfg = config.GetConfig()
}
