package controller

import (
	"github.com/innovationmech/swit/internal/switauth/service"
)

type AuthController struct {
	authService service.AuthService
}

func NewAuthController(authService service.AuthService) *AuthController {
	return &AuthController{authService: authService}
}
