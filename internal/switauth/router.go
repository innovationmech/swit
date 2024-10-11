package switauth

import (
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/controller"
)

func RegisterRoutes(authController *controller.AuthController) *gin.Engine {
	r := gin.Default()

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", authController.Login)
		authGroup.POST("/logout", authController.Logout)
		authGroup.POST("/refresh", authController.RefreshToken)
		authGroup.GET("/validate", authController.ValidateToken)
	}

	return r
}
