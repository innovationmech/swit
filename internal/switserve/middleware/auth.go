package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/config"
	"net/http"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.GetHeader("Authorization")
		if tokenString == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}
		cfg := config.GetConfig()
		resp, err := http.Get(cfg.AuthServer + "/validate")
		if err != nil || resp.StatusCode != http.StatusOK {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		ctx.Next()
	}
}
