# 快速开始示例：添加产品管理 API

本示例展示如何使用新的路由注册系统快速添加一个产品管理API模块。

## 步骤 1: 创建产品控制器

```bash
mkdir -p internal/switserve/controller/v1/product
```

创建 `internal/switserve/controller/v1/product/controller.go`:

```go
package product

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type ProductController struct {
    // 这里可以添加服务依赖
}

func NewProductController() *ProductController {
    return &ProductController{}
}

// GetProducts 获取产品列表
func (pc *ProductController) GetProducts(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "get products",
        "data": []gin.H{
            {"id": 1, "name": "Product 1", "price": 99.99},
            {"id": 2, "name": "Product 2", "price": 199.99},
        },
    })
}

// GetProductByID 根据ID获取产品
func (pc *ProductController) GetProductByID(c *gin.Context) {
    id := c.Param("id")
    c.JSON(http.StatusOK, gin.H{
        "message": "get product by id",
        "id": id,
        "data": gin.H{"id": id, "name": "Product " + id, "price": 99.99},
    })
}

// CreateProduct 创建产品
func (pc *ProductController) CreateProduct(c *gin.Context) {
    c.JSON(http.StatusCreated, gin.H{
        "message": "product created",
        "data": gin.H{"id": 3, "name": "New Product", "price": 299.99},
    })
}

// UpdateProduct 更新产品
func (pc *ProductController) UpdateProduct(c *gin.Context) {
    id := c.Param("id")
    c.JSON(http.StatusOK, gin.H{
        "message": "product updated",
        "id": id,
    })
}

// DeleteProduct 删除产品
func (pc *ProductController) DeleteProduct(c *gin.Context) {
    id := c.Param("id")
    c.JSON(http.StatusOK, gin.H{
        "message": "product deleted",
        "id": id,
    })
}
```

## 步骤 2: 实现路由注册器

创建 `internal/switserve/controller/v1/product/registrar.go`:

```go
package product

import (
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/internal/switserve/middleware"
)

// ProductRouteRegistrar 产品路由注册器
type ProductRouteRegistrar struct {
    controller *ProductController
}

// NewProductRouteRegistrar 创建产品路由注册器
func NewProductRouteRegistrar() *ProductRouteRegistrar {
    return &ProductRouteRegistrar{
        controller: NewProductController(),
    }
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (prr *ProductRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
    // 需要认证的产品API路由
    productGroup := rg.Group("/products")
    productGroup.Use(middleware.AuthMiddleware()) // 需要认证
    {
        productGroup.GET("", prr.controller.GetProducts)
        productGroup.POST("", prr.controller.CreateProduct)
        productGroup.GET("/:id", prr.controller.GetProductByID)
        productGroup.PUT("/:id", prr.controller.UpdateProduct)
        productGroup.DELETE("/:id", prr.controller.DeleteProduct)
    }

    return nil
}

// GetName 实现 RouteRegistrar 接口
func (prr *ProductRouteRegistrar) GetName() string {
    return "product-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (prr *ProductRouteRegistrar) GetVersion() string {
    return "v1"
}

// GetPrefix 实现 RouteRegistrar 接口
func (prr *ProductRouteRegistrar) GetPrefix() string {
    return ""
}

// ProductPublicRouteRegistrar 产品公开API路由注册器（无需认证）
type ProductPublicRouteRegistrar struct {
    controller *ProductController
}

// NewProductPublicRouteRegistrar 创建产品公开API路由注册器
func NewProductPublicRouteRegistrar() *ProductPublicRouteRegistrar {
    return &ProductPublicRouteRegistrar{
        controller: NewProductController(),
    }
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (pprr *ProductPublicRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
    // 公开的产品API路由，不需要认证
    publicGroup := rg.Group("/public/products")
    {
        publicGroup.GET("", pprr.controller.GetProducts)
        publicGroup.GET("/:id", pprr.controller.GetProductByID)
    }

    return nil
}

// GetName 实现 RouteRegistrar 接口
func (pprr *ProductPublicRouteRegistrar) GetName() string {
    return "product-public-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (pprr *ProductPublicRouteRegistrar) GetVersion() string {
    return "v1"
}

// GetPrefix 实现 RouteRegistrar 接口
func (pprr *ProductPublicRouteRegistrar) GetPrefix() string {
    return ""
}
```

## 步骤 3: 注册到主路由

在 `internal/switserve/server/router.go` 中添加：

```go
import (
    // ... 其他导入 ...
    "github.com/innovationmech/swit/internal/switserve/controller/v1/product"
)

func (s *Server) SetupRoutes() {
    registry := NewRouteRegistry()

    // ... 其他注册 ...

    // 注册产品路由
    registry.RegisterRoute(product.NewProductRouteRegistrar())
    registry.RegisterRoute(product.NewProductPublicRouteRegistrar())

    // ... 设置路由 ...
}
```

## 步骤 4: 测试新的API

启动服务器后，你将拥有以下新的API端点：

### 需要认证的API:
- `GET /v1/products` - 获取产品列表
- `POST /v1/products` - 创建产品
- `GET /v1/products/:id` - 获取指定产品
- `PUT /v1/products/:id` - 更新产品
- `DELETE /v1/products/:id` - 删除产品

### 公开的API（无需认证）:
- `GET /v1/public/products` - 获取产品列表
- `GET /v1/public/products/:id` - 获取指定产品

### 调试API:
- `GET /debug/routes` - 查看所有注册的路由
- `GET /debug/middlewares` - 查看所有注册的中间件

## 优势展示

1. **只需要3个文件**：controller.go、registrar.go 和更新router.go
2. **自动版本管理**：路由自动加上`/v1`前缀
3. **中间件支持**：可以轻松添加认证、日志等中间件
4. **易于测试**：每个组件都可以独立测试
5. **调试友好**：内置调试端点查看路由信息

## 扩展示例

如果需要添加 v2 版本的产品API，只需要：

1. 创建 `internal/switserve/controller/v2/product/` 目录
2. 实现新的控制器和注册器
3. 在注册器的 `GetVersion()` 方法中返回 `"v2"`
4. 注册到路由表

这样就会自动生成 `/v2/products` 等路径，与v1版本并存。

## 总结

这个新的路由注册系统让添加新API变得非常简单和标准化，符合企业级开发的最佳实践。
