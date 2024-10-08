/*
Copyright © 2023 jackelyj <dreamerlyj@gmail.com.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

*/
package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/swit-serve/config"
	"go.uber.org/zap"
)

func RegisterHandler(c *gin.Context) {
	name := c.Param("name")
	email := c.Param("email")

	registerUser(name, email)
	c.JSON(http.StatusOK, gin.H{
		"message": "register success",
	})
}

func GetUserHandler(c *gin.Context) {
	name := c.Param("name")

	user := getUserByName(name)
	config.Logger.Info("get user", zap.String("name", user.Name), zap.String("email", user.Email))
	c.JSON(http.StatusOK, user)
}
