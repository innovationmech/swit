package switauth

import (
	"github.com/gin-gonic/gin"
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
}

// NewServer creates a new Server instance
func NewServer() (*Server, error) {
	s := &Server{
		router: gin.Default(),
	}

	initConfig()

	userServiceUrl := cfg.BaseUrl
	userClient := client.NewUserClient(userServiceUrl)
	tokenRepo := repository.NewTokenRepository(store.GetDB())
	authService := service.NewAuthService(userClient, tokenRepo)
	authController := controller.NewAuthController(authService)

	s.router = RegisterRoutes(authController)
	return s, nil
}

// Run starts the server
func (s *Server) Run() error {
	return s.router.Run(cfg.Server.Port)
}

func initConfig() {
	cfg = config.GetConfig()
}
