package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/swit-serve/config"
	"github.com/innovationmech/swit/internal/swit-serve/health"
	"github.com/innovationmech/swit/internal/swit-serve/stop"
	"github.com/innovationmech/swit/internal/swit-serve/user"
	"go.uber.org/zap"
)

type Server struct {
	router *gin.Engine
	srv    *http.Server
}

func NewServer() *Server {
	return &Server{
		router: gin.Default(),
	}
}

func (s *Server) SetupRoutes() {
	user.RegisterRoutes(s.router)
	health.RegisterRoutes(s.router)
	stop.RegisterRoutes(s.router, s.Shutdown)
}

func (s *Server) Run(addr string) error {
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		config.Logger.Error("Shutdown error", zap.Error(err))
	}
}
