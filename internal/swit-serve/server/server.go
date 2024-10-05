package server

import (
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/swit-serve/health"
	"github.com/innovationmech/swit/internal/swit-serve/user"
)

type Server struct {
	router *gin.Engine
}

func NewServer() *Server {
	return &Server{
		router: gin.Default(),
	}
}

func (s *Server) SetupRoutes() {
	user.RegisterRoutes(s.router)
	health.RegisterRoutes(s.router)
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
