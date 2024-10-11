package utils

import (
	"errors"
	"github.com/innovationmech/swit/internal/switauth/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateAccessToken(userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Minute * 15) // 访问令牌有效期15分钟
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     expiresAt.Unix(),
		"type":    "access",
	})

	tokenString, err := token.SignedString(config.JwtSecret)
	return tokenString, expiresAt, err
}

func GenerateRefreshToken(userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Hour * 72) // 刷新令牌有效期72小时
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     expiresAt.Unix(),
		"type":    "refresh",
	})

	tokenString, err := token.SignedString(config.JwtSecret)
	return tokenString, expiresAt, err
}

func ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
	return validateToken(tokenString, "access")
}

func ValidateRefreshToken(tokenString string) (jwt.MapClaims, error) {
	return validateToken(tokenString, "refresh")
}

func validateToken(tokenString, tokenType string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return config.JwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("Invalid token claims")
	}

	if claims["type"] != tokenType {
		return nil, errors.New("Invalid token type")
	}

	return claims, nil
}
