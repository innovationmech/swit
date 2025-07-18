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

package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserByUsername gets a user by username.
//
//	@Summary		Get user by username
//	@Description	Get user details by username
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			username	path		string					true	"Username"
//	@Success		200			{object}	map[string]interface{}	"User details"
//	@Failure		404			{object}	map[string]interface{}	"User not found"
//	@Failure		500			{object}	map[string]interface{}	"Internal server error"
//	@Security		BearerAuth
//	@Router			/users/username/{username} [get]
func (uc *Controller) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	user, err := uc.userSrv.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUserByEmail gets a user by email.
//
//	@Summary		Get user by email
//	@Description	Get user details by email address
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			email	path		string					true	"Email address"
//	@Success		200		{object}	map[string]interface{}	"User details"
//	@Failure		404		{object}	map[string]interface{}	"User not found"
//	@Failure		500		{object}	map[string]interface{}	"Internal server error"
//	@Security		BearerAuth
//	@Router			/users/email/{email} [get]
func (uc *Controller) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")
	user, err := uc.userSrv.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}
