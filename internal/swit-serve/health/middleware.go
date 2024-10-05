package health

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine) {
	router.GET("/health", HealthHandler)
}
