package serve

import (
	"github.com/innovationmech/swit/internal/swit-serve/config"
	"github.com/innovationmech/swit/internal/swit-serve/server"
	"github.com/spf13/cobra"
)

var cfg *config.Config

func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the SWIT server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.NewServer()
			srv.SetupRoutes()
			return srv.Run(":" + cfg.Server.Port)
		},
	}

	cobra.OnInitialize(initConfig)

	return cmd
}

func initConfig() {
	config.InitLogger()
	cfg = config.GetConfig()
}
