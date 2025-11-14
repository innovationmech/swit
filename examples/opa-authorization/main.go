// Copyright (c) 2024 Six-Thirty Labs, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/security/opa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/innovationmech/swit/api/gen/go/swit/common/v1"
)

var (
	port       = flag.Int("port", 8080, "HTTP server port")
	grpcPort   = flag.Int("grpc-port", 9090, "gRPC server port")
	opaMode    = flag.String("opa-mode", "embedded", "OPA mode: embedded or remote")
	opaURL     = flag.String("opa-url", "http://localhost:8181", "OPA server URL (for remote mode)")
	policyDir  = flag.String("policy-dir", "./policies", "Policy directory (for embedded mode)")
	policyType = flag.String("policy-type", "rbac", "Policy type: rbac or abac")
)

// Document 文档模型
type Document struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Owner   string `json:"owner"`
}

// DocumentService 文档服务
type DocumentService struct {
	documents map[string]*Document
	logger    *zap.Logger
	pb.UnimplementedHealthCheckServiceServer
}

// NewDocumentService 创建文档服务
func NewDocumentService(logger *zap.Logger) *DocumentService {
	return &DocumentService{
		documents: map[string]*Document{
			"doc-1": {
				ID:      "doc-1",
				Title:   "Public Document",
				Content: "This is a public document",
				Owner:   "system",
			},
			"doc-2": {
				ID:      "doc-2",
				Title:   "Alice's Document",
				Content: "This is Alice's private document",
				Owner:   "alice",
			},
			"doc-3": {
				ID:      "doc-3",
				Title:   "Bob's Document",
				Content: "This is Bob's private document",
				Owner:   "bob",
			},
		},
		logger: logger,
	}
}

// Check 实现 gRPC 健康检查
func (s *DocumentService) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status: pb.HealthCheckResponse_SERVING,
	}, nil
}

// Watch 实现 gRPC 健康检查流式方法
func (s *DocumentService) Watch(req *pb.HealthCheckRequest, stream pb.HealthCheckService_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watch is not implemented")
}

func main() {
	flag.Parse()

	// 初始化日志
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	ctx := context.Background()

	// 创建 OPA 客户端
	opaClient, err := createOPAClient(ctx, logger)
	if err != nil {
		logger.Fatal("Failed to create OPA client", zap.Error(err))
	}
	defer opaClient.Close(ctx)

	logger.Info("OPA client initialized",
		zap.String("mode", string(*opaMode)),
		zap.String("policy_type", *policyType),
	)

	// 创建文档服务
	docService := NewDocumentService(logger)

	// 启动 HTTP 服务器
	httpServer := setupHTTPServer(opaClient, docService, logger)
	go func() {
		addr := fmt.Sprintf(":%d", *port)
		logger.Info("Starting HTTP server", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 启动 gRPC 服务器
	grpcServer := setupGRPCServer(opaClient, docService, logger)
	go func() {
		addr := fmt.Sprintf(":%d", *grpcPort)
		logger.Info("Starting gRPC server", zap.String("addr", addr))
		lis, err := grpc.NewServer().GetServiceInfo() // placeholder for actual listen
		_ = lis
		if err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}
	grpcServer.GracefulStop()

	logger.Info("Servers stopped")
}

// createOPAClient 创建 OPA 客户端
func createOPAClient(ctx context.Context, logger *zap.Logger) (opa.Client, error) {
	var config *opa.Config

	switch *opaMode {
	case "embedded":
		// 解析策略目录路径
		policyPath := *policyDir
		if !filepath.IsAbs(policyPath) {
			absPath, err := filepath.Abs(policyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve policy directory: %w", err)
			}
			policyPath = absPath
		}

		logger.Info("Using embedded OPA mode", zap.String("policy_dir", policyPath))

		config = &opa.Config{
			Mode: opa.ModeEmbedded,
			EmbeddedConfig: &opa.EmbeddedConfig{
				PolicyDir: policyPath,
			},
			CacheConfig: &opa.CacheConfig{
				Enabled: true,
				MaxSize: 1000,
				TTL:     5 * time.Minute,
			},
			DefaultDecisionPath: fmt.Sprintf("%s/allow", *policyType),
		}

	case "remote":
		logger.Info("Using remote OPA mode", zap.String("url", *opaURL))

		config = &opa.Config{
			Mode: opa.ModeRemote,
			RemoteConfig: &opa.RemoteConfig{
				URL:     *opaURL,
				Timeout: 5 * time.Second,
				HealthCheck: &opa.HealthCheckConfig{
					Enabled:          true,
					Interval:         30 * time.Second,
					Timeout:          5 * time.Second,
					FailureThreshold: 3,
					SuccessThreshold: 1,
				},
			},
			CacheConfig: &opa.CacheConfig{
				Enabled: true,
				MaxSize: 1000,
				TTL:     5 * time.Minute,
			},
			DefaultDecisionPath: fmt.Sprintf("%s/allow", *policyType),
		}

	default:
		return nil, fmt.Errorf("unsupported OPA mode: %s", *opaMode)
	}

	return opa.NewClient(ctx, config)
}

// setupHTTPServer 设置 HTTP 服务器
func setupHTTPServer(opaClient opa.Client, docService *DocumentService, logger *zap.Logger) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// 日志中间件
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	})

	// 认证中间件 (模拟从请求头获取用户信息)
	router.Use(func(c *gin.Context) {
		user := c.GetHeader("X-User")
		if user == "" {
			user = "anonymous"
		}
		roles := c.GetHeader("X-Roles")
		if roles == "" {
			roles = "viewer"
		}

		c.Set("user", user)
		c.Set("roles", roles)
		c.Next()
	})

	// OPA 策略中间件
	router.Use(middleware.OPAMiddleware(
		opaClient,
		middleware.WithOPALogger(logger),
		middleware.WithOPAInputBuilder(func(c *gin.Context) (*opa.PolicyInput, error) {
			user, _ := c.Get("user")
			roles, _ := c.Get("roles")

			return &opa.PolicyInput{
				Subject: opa.Subject{
					User:  user.(string),
					Roles: []string{roles.(string)},
				},
				Action: c.Request.Method,
				Resource: map[string]interface{}{
					"type": "document",
					"path": c.Request.URL.Path,
				},
				Context: map[string]interface{}{
					"time":      time.Now().Unix(),
					"client_ip": c.ClientIP(),
				},
			}, nil
		}),
		middleware.WithOPAAuditLog(true),
	))

	// API 路由
	api := router.Group("/api/v1")
	{
		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		// 文档操作
		api.GET("/documents", func(c *gin.Context) {
			docs := make([]*Document, 0, len(docService.documents))
			for _, doc := range docService.documents {
				docs = append(docs, doc)
			}
			c.JSON(http.StatusOK, docs)
		})

		api.GET("/documents/:id", func(c *gin.Context) {
			id := c.Param("id")
			doc, exists := docService.documents[id]
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
				return
			}
			c.JSON(http.StatusOK, doc)
		})

		api.POST("/documents", func(c *gin.Context) {
			var doc Document
			if err := c.ShouldBindJSON(&doc); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			user, _ := c.Get("user")
			doc.Owner = user.(string)
			docService.documents[doc.ID] = &doc

			c.JSON(http.StatusCreated, doc)
		})

		api.PUT("/documents/:id", func(c *gin.Context) {
			id := c.Param("id")
			doc, exists := docService.documents[id]
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
				return
			}

			var updates Document
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			doc.Title = updates.Title
			doc.Content = updates.Content

			c.JSON(http.StatusOK, doc)
		})

		api.DELETE("/documents/:id", func(c *gin.Context) {
			id := c.Param("id")
			if _, exists := docService.documents[id]; !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
				return
			}

			delete(docService.documents, id)
			c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
		})
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: router,
	}
}

// setupGRPCServer 设置 gRPC 服务器
func setupGRPCServer(opaClient opa.Client, docService *DocumentService, logger *zap.Logger) *grpc.Server {
	// 创建 gRPC 策略拦截器
	policyInterceptor := middleware.UnaryPolicyInterceptor(
		opaClient,
		middleware.WithGRPCPolicyLogger(logger),
		middleware.WithGRPCPolicyInputBuilder(func(ctx context.Context, fullMethod string, req interface{}) (*opa.PolicyInput, error) {
			// 从 context 提取用户信息（实际应用中应从 JWT 或其他认证机制获取）
			return &opa.PolicyInput{
				Subject: opa.Subject{
					User:  "grpc-user",
					Roles: []string{"admin"},
				},
				Action: fullMethod,
				Resource: map[string]interface{}{
					"type":   "grpc",
					"method": fullMethod,
				},
			}, nil
		}),
		middleware.WithGRPCPolicyAuditLog(true),
	)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(policyInterceptor),
	)

	// 注册服务
	pb.RegisterHealthCheckServiceServer(server, docService)

	return server
}

