package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func WebServe() {
	r := gin.Default()
	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run()
}
