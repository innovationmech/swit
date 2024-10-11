package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (c *AuthController) ValidateToken(ctx *gin.Context) {
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
		return
	}

	token, err := c.authService.ValidateToken(tokenString)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Token is valid", "user_id": token.UserID})
}
