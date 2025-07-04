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

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/model"
)

// Login authenticates a user and returns access and refresh tokens
// @Summary User login
// @Description Authenticate a user with username and password, returns access and refresh tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Param login body model.LoginRequest true "User login credentials"
// @Success 200 {object} model.LoginResponse "Login successful"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 401 {object} model.ErrorResponse "Invalid credentials"
// @Router /auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var loginData model.LoginRequest

	if err := ctx.ShouldBindJSON(&loginData); err != nil {
		ctx.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	accessToken, refreshToken, err := c.authService.Login(ctx, loginData.Username, loginData.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
