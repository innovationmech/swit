// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
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

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/model"
)

// RefreshToken generates new access and refresh tokens using a valid refresh token
//
//	@Summary		Refresh access token
//	@Description	Generate new access and refresh tokens using a valid refresh token
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			refresh	body		model.RefreshTokenRequest	true	"Refresh token"
//	@Success		200		{object}	model.RefreshTokenResponse	"Token refresh successful"
//	@Failure		400		{object}	model.ErrorResponse			"Bad request"
//	@Failure		401		{object}	model.ErrorResponse			"Invalid or expired refresh token"
//	@Router			/auth/refresh [post]
func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var data model.RefreshTokenRequest

	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	newAccessToken, newRefreshToken, err := c.authService.RefreshToken(ctx.Request.Context(), data.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, model.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	})
}
