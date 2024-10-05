package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {
	userGroup := r.Group("/users")
	{
		userGroup.POST("/", RegisterHandler)
		userGroup.GET("/:id", GetUserHandler)
	}
}
