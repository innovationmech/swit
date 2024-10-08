package stop

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type StopHandler struct {
	shutdownFunc func()
}

func NewStopHandler(shutdownFunc func()) *StopHandler {
	return &StopHandler{
		shutdownFunc: shutdownFunc,
	}
}

func (h *StopHandler) Stop(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Server is stopping"})
	go func() {
		time.Sleep(time.Second) // Give some time for the response to be sent
		h.shutdownFunc()
	}()
}

func RegisterRoutes(r *gin.Engine, shutdownFunc func()) {
	handler := NewStopHandler(shutdownFunc)
	r.POST("/stop", handler.Stop)
}
