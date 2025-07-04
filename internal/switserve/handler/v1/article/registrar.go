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

package article

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/middleware"
)

// ArticleController 文章控制器（示例）
type ArticleController struct{}

// NewArticleController 创建文章控制器
func NewArticleController() *ArticleController {
	return &ArticleController{}
}

// GetArticles 获取文章列表
func (ac *ArticleController) GetArticles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get articles"})
}

// CreateArticle 创建文章
func (ac *ArticleController) CreateArticle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "create article"})
}

// GetArticleByID 根据ID获取文章
func (ac *ArticleController) GetArticleByID(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get article by id"})
}

// UpdateArticle 更新文章
func (ac *ArticleController) UpdateArticle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "update article"})
}

// DeleteArticle 删除文章
func (ac *ArticleController) DeleteArticle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "delete article"})
}

// ArticleRouteRegistrar 文章路由注册器
type ArticleRouteRegistrar struct {
	controller *ArticleController
}

// NewArticleRouteRegistrar 创建文章路由注册器
func NewArticleRouteRegistrar() *ArticleRouteRegistrar {
	return &ArticleRouteRegistrar{
		controller: NewArticleController(),
	}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (arr *ArticleRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	// 需要认证的文章API路由
	articleGroup := rg.Group("/articles")
	// 使用统一的认证中间件
	articleGroup.Use(middleware.AuthMiddleware())
	{
		articleGroup.GET("", arr.controller.GetArticles)
		articleGroup.POST("", arr.controller.CreateArticle)
		articleGroup.GET("/:id", arr.controller.GetArticleByID)
		articleGroup.PUT("/:id", arr.controller.UpdateArticle)
		articleGroup.DELETE("/:id", arr.controller.DeleteArticle)
	}

	return nil
}

// GetName 实现 RouteRegistrar 接口
func (arr *ArticleRouteRegistrar) GetName() string {
	return "article-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (arr *ArticleRouteRegistrar) GetVersion() string {
	return "v1"
}

// GetPrefix 实现 RouteRegistrar 接口
func (arr *ArticleRouteRegistrar) GetPrefix() string {
	return ""
}

// ArticlePublicRouteRegistrar 文章公开API路由注册器
type ArticlePublicRouteRegistrar struct {
	controller *ArticleController
}

// NewArticlePublicRouteRegistrar 创建文章公开API路由注册器
func NewArticlePublicRouteRegistrar() *ArticlePublicRouteRegistrar {
	return &ArticlePublicRouteRegistrar{
		controller: NewArticleController(),
	}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (aprr *ArticlePublicRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	// 公开的文章API路由，不需要认证
	publicGroup := rg.Group("/public/articles")
	{
		publicGroup.GET("", aprr.controller.GetArticles)
		publicGroup.GET("/:id", aprr.controller.GetArticleByID)
	}

	return nil
}

// GetName 实现 RouteRegistrar 接口
func (aprr *ArticlePublicRouteRegistrar) GetName() string {
	return "article-public-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (aprr *ArticlePublicRouteRegistrar) GetVersion() string {
	return "v1"
}

// GetPrefix 实现 RouteRegistrar 接口
func (aprr *ArticlePublicRouteRegistrar) GetPrefix() string {
	return ""
}
