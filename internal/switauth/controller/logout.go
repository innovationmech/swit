package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (c *AuthController) Logout(ctx *gin.Context) {
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is missing"})
		return
	}

	if err := c.authService.Logout(tokenString); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
