// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/model"
)

// ValidateToken validates an access token and returns token information
//
//	@Summary		Validate access token
//	@Description	Validate an access token and return token information including user ID
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	model.ValidateTokenResponse	"Token is valid"
//	@Failure		401	{object}	model.ErrorResponse			"Invalid or expired token"
//	@Router			/auth/validate [get]
func (c *AuthController) ValidateToken(ctx *gin.Context) {
	// Get Authorization header
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "Missing Authorization header"})
		return
	}

	// Split the Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		ctx.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "Invalid Authorization header format"})
		return
	}

	// Extract the actual token
	tokenString := parts[1]

	// Validate token
	token, err := c.authService.ValidateToken(ctx.Request.Context(), tokenString)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, model.ValidateTokenResponse{
		Message: "Token is valid",
		UserID:  token.UserID,
	})
}
