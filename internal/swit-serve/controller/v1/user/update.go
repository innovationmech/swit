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

package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (uc *UserController) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	// Create a map to store fields that need to be updated
	updates := make(map[string]interface{})

	// Check and add fields that need to be updated
	if username := c.PostForm("username"); username != "" {
		updates["username"] = username
	}
	if email := c.PostForm("email"); email != "" {
		updates["email"] = email
	}

	// If there are no fields to update, return an error
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields provided for update"})
		return
	}

	// Call the UpdateUser method of the service layer
	err := uc.userSrv.UpdateUser(id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User information updated successfully"})
}
