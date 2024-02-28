package health

import "github.com/gin-gonic/gin"

func HealthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
