package serve

import (
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/swit-serve/health"
	"github.com/innovationmech/swit/internal/swit-serve/logger"
	"github.com/innovationmech/swit/internal/swit-serve/user"
	"github.com/spf13/cobra"
)

func NewServeCmd() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "serve",
		Short: "server a controller application",
		Long:  `server a gin powerd controller application`,
		Run: func(cmd *cobra.Command, args []string) {
			logger.Init()
			gin.SetMode(gin.ReleaseMode)
			r := gin.Default()
			r.GET("/user/:name/:email", user.RegisterHandler)
			r.GET("/health", health.HealthHandler)
			r.GET("/user/:name", user.GetUserHandler)

			r.Run(":8080")
		},
	}
	return cmds
}
