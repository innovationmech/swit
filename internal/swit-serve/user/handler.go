package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/swit-serve/logger"
	"go.uber.org/zap"
)

func RegisterHandler(c *gin.Context) {
	name := c.Param("name")
	email := c.Param("email")

	registerUser(name, email)
	c.JSON(http.StatusOK, gin.H{
		"message": "register success",
	})
}

func GetUserHandler(c *gin.Context) {
	name := c.Param("name")

	user := getUserByName(name)

	logger.Log.Info("get user", zap.String("name", user.Name), zap.String("email", user.Email))
	c.JSON(http.StatusOK, user)
}
