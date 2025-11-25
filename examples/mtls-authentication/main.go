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

package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/pkg/logger"
	tlsconfig "github.com/innovationmech/swit/pkg/security/tls"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/transport"
)

// MTLSExampleService implements the ServiceRegistrar interface.
type MTLSExampleService struct {
	name string
}

// NewMTLSExampleService creates a new mTLS example service.
func NewMTLSExampleService(name string) *MTLSExampleService {
	return &MTLSExampleService{
		name: name,
	}
}

// RegisterServices registers the HTTP and gRPC handlers with the server.
func (s *MTLSExampleService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	httpHandler := &MTLSHTTPHandler{
		serviceName: s.name,
	}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register gRPC handler
	grpcHandler := &MTLSGRPCHandler{
		serviceName: s.name,
	}
	if err := registry.RegisterBusinessGRPCService(grpcHandler); err != nil {
		return fmt.Errorf("failed to register gRPC handler: %w", err)
	}

	// Register health check
	healthCheck := &MTLSHealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// MTLSHTTPHandler implements the HTTPHandler interface.
type MTLSHTTPHandler struct {
	serviceName string
}

// RegisterRoutes registers HTTP routes with the router.
func (h *MTLSHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Public endpoints (still require mTLS, but no specific client identity check)
	public := ginRouter.Group("/api/v1/public")
	{
		public.GET("/info", h.handlePublicInfo)
		public.GET("/health", h.handleHealth)
	}

	// Protected endpoints (require specific client certificate attributes)
	protected := ginRouter.Group("/api/v1/protected")
	{
		protected.GET("/profile", h.handleProfile)
		protected.GET("/data", h.handleData)
	}

	// Admin endpoints (require admin client certificate)
	admin := ginRouter.Group("/api/v1/admin")
	admin.Use(h.requireAdminCertMiddleware())
	{
		admin.GET("/dashboard", h.handleAdminDashboard)
		admin.GET("/certificates", h.handleListCertificates)
	}

	return nil
}

// GetServiceName returns the service name.
func (h *MTLSHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// handlePublicInfo handles the public info endpoint.
func (h *MTLSHTTPHandler) handlePublicInfo(c *gin.Context) {
	// Extract client certificate information
	certInfo := transport.GetClientCertificateInfo(c)

	response := gin.H{
		"message":   "Welcome to mTLS Authentication Example",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"tls_info": gin.H{
			"protocol": "mTLS",
			"version":  "TLS 1.2+",
		},
	}

	// Add client certificate info if available
	if certInfo != nil {
		response["client_certificate"] = gin.H{
			"common_name":  certInfo.CommonName,
			"organization": certInfo.Organization,
			"serial":       certInfo.SerialNumber,
			"not_before":   certInfo.NotBefore,
			"not_after":    certInfo.NotAfter,
			"issuer":       certInfo.Issuer,
		}
	}

	c.JSON(http.StatusOK, response)
}

// handleHealth handles the health check endpoint.
func (h *MTLSHTTPHandler) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
	})
}

// handleProfile handles the protected profile endpoint.
func (h *MTLSHTTPHandler) handleProfile(c *gin.Context) {
	certInfo := transport.GetClientCertificateInfo(c)
	if certInfo == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Client certificate required",
			"message": "This endpoint requires a valid client certificate",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Client profile retrieved successfully",
		"profile": gin.H{
			"identity":     certInfo.CommonName,
			"organization": certInfo.Organization,
			"ou":           certInfo.OrganizationalUnit,
			"country":      certInfo.Country,
			"email":        certInfo.EmailAddresses,
			"dns_names":    certInfo.DNSNames,
			"ip_addresses": certInfo.IPAddresses,
		},
		"certificate_details": gin.H{
			"serial":     certInfo.SerialNumber,
			"not_before": certInfo.NotBefore,
			"not_after":  certInfo.NotAfter,
			"issuer":     certInfo.Issuer,
			"subject":    certInfo.Subject,
		},
		"timestamp": time.Now().UTC(),
	})
}

// handleData handles the protected data endpoint.
func (h *MTLSHTTPHandler) handleData(c *gin.Context) {
	certInfo := transport.GetClientCertificateInfo(c)
	if certInfo == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Client certificate required",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Protected data retrieved successfully",
		"client":   certInfo.CommonName,
		"data":     generateSampleData(),
		"accessed": time.Now().UTC(),
	})
}

// handleAdminDashboard handles the admin-only endpoint.
func (h *MTLSHTTPHandler) handleAdminDashboard(c *gin.Context) {
	certInfo := transport.GetClientCertificateInfo(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to the Admin Dashboard",
		"admin":   certInfo.CommonName,
		"dashboard": gin.H{
			"total_connections":    42,
			"active_certificates":  3,
			"revoked_certificates": 0,
			"last_audit":           time.Now().Add(-24 * time.Hour).UTC(),
		},
		"timestamp": time.Now().UTC(),
	})
}

// handleListCertificates handles the certificate listing endpoint.
func (h *MTLSHTTPHandler) handleListCertificates(c *gin.Context) {
	certInfo := transport.GetClientCertificateInfo(c)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Certificate inventory",
		"requested_by": certInfo.CommonName,
		"certificates": []gin.H{
			{
				"cn":      "swit-client",
				"type":    "client",
				"status":  "active",
				"issued":  "2025-01-01",
				"expires": "2026-01-01",
			},
			{
				"cn":      "swit-admin",
				"type":    "admin",
				"status":  "active",
				"issued":  "2025-01-01",
				"expires": "2026-01-01",
			},
		},
		"timestamp": time.Now().UTC(),
	})
}

// requireAdminCertMiddleware creates middleware that requires admin certificate.
func (h *MTLSHTTPHandler) requireAdminCertMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		certInfo := transport.GetClientCertificateInfo(c)
		if certInfo == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "Client certificate required",
				"message": "Admin endpoints require a valid client certificate",
			})
			return
		}

		// Check if the certificate is an admin certificate
		// In production, you might check specific attributes, OU, or maintain a whitelist
		if certInfo.CommonName != "swit-admin" && !containsOU(certInfo.OrganizationalUnit, "Admin") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "Admin certificate required",
				"message": "This endpoint requires an admin-level client certificate",
				"your_cn": certInfo.CommonName,
				"your_ou": certInfo.OrganizationalUnit,
			})
			return
		}

		c.Next()
	}
}

// containsOU checks if the organizational unit list contains a specific value.
func containsOU(ous []string, target string) bool {
	for _, ou := range ous {
		if ou == target {
			return true
		}
	}
	return false
}

// generateSampleData generates sample protected data.
func generateSampleData() []gin.H {
	return []gin.H{
		{"id": 1, "name": "Confidential Report A", "classification": "internal"},
		{"id": 2, "name": "Financial Summary Q4", "classification": "confidential"},
		{"id": 3, "name": "Strategic Plan 2025", "classification": "restricted"},
	}
}

// MTLSGRPCHandler implements a simple gRPC service for mTLS demonstration.
type MTLSGRPCHandler struct {
	serviceName string
}

// RegisterGRPC registers the gRPC service.
func (h *MTLSGRPCHandler) RegisterGRPC(server interface{}) error {
	// In a real application, you would register your protobuf service here
	// For this example, we just log that gRPC is available
	logger.GetLogger().Info("gRPC service registered for mTLS example",
		zap.String("service", h.serviceName))
	return nil
}

// GetServiceName returns the service name.
func (h *MTLSGRPCHandler) GetServiceName() string {
	return h.serviceName
}

// ExtractGRPCClientCertInfo extracts client certificate info from gRPC context.
func ExtractGRPCClientCertInfo(ctx context.Context) (*tlsconfig.CertificateInfo, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no peer information")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no TLS information")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no client certificate")
	}

	return tlsconfig.ExtractCertificateInfo(tlsInfo.State.PeerCertificates[0]), nil
}

// MTLSHealthCheck implements the HealthCheck interface.
type MTLSHealthCheck struct {
	serviceName string
}

// Check performs a health check.
func (h *MTLSHealthCheck) Check(ctx context.Context) error {
	// In a real service, check TLS certificate validity, CA availability, etc.
	return nil
}

// GetServiceName returns the service name.
func (h *MTLSHealthCheck) GetServiceName() string {
	return h.serviceName
}

// MTLSDependencyContainer implements the DependencyContainer interface.
type MTLSDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewMTLSDependencyContainer creates a new dependency container.
func NewMTLSDependencyContainer() *MTLSDependencyContainer {
	return &MTLSDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

// GetService retrieves a service by name.
func (d *MTLSDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	service, exists := d.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// Close closes the dependency container.
func (d *MTLSDependencyContainer) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	return nil
}

// loadConfigFromFile loads configuration from YAML file.
func loadConfigFromFile(configPath string) (*server.ServerConfig, error) {
	config := &server.ServerConfig{}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvironmentOverrides(config)

	return config, nil
}

// createDefaultConfig creates default configuration with mTLS enabled.
func createDefaultConfig() *server.ServerConfig {
	return &server.ServerConfig{
		ServiceName: "mtls-authentication-example",
		HTTP: server.HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8443"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
			TLS: &tlsconfig.TLSConfig{
				Enabled:    true,
				CertFile:   getEnv("TLS_CERT_FILE", "certs/server.crt"),
				KeyFile:    getEnv("TLS_KEY_FILE", "certs/server.key"),
				CAFiles:    []string{getEnv("TLS_CA_FILE", "certs/ca.crt")},
				ClientAuth: getEnv("TLS_CLIENT_AUTH", "require_and_verify"),
				MinVersion: "TLS1.2",
				MaxVersion: "TLS1.3",
			},
		},
		GRPC: server.GRPCConfig{
			Port:                getEnv("GRPC_PORT", "50443"),
			Enabled:             getBoolEnv("GRPC_ENABLED", true),
			EnableReflection:    true,
			EnableHealthService: true,
			MaxRecvMsgSize:      4 * 1024 * 1024,
			MaxSendMsgSize:      4 * 1024 * 1024,
			TLS: &tlsconfig.TLSConfig{
				Enabled:    true,
				CertFile:   getEnv("TLS_CERT_FILE", "certs/server.crt"),
				KeyFile:    getEnv("TLS_KEY_FILE", "certs/server.key"),
				CAFiles:    []string{getEnv("TLS_CA_FILE", "certs/ca.crt")},
				ClientAuth: getEnv("TLS_CLIENT_AUTH", "require_and_verify"),
				MinVersion: "TLS1.2",
				MaxVersion: "TLS1.3",
			},
		},
		ShutdownTimeout: 30 * time.Second,
		Middleware: server.MiddlewareConfig{
			EnableCORS:    false,
			EnableLogging: true,
		},
	}
}

// applyEnvironmentOverrides applies environment variable overrides.
func applyEnvironmentOverrides(config *server.ServerConfig) {
	if httpPort := os.Getenv("HTTP_PORT"); httpPort != "" {
		config.HTTP.Port = httpPort
	}
	if grpcPort := os.Getenv("GRPC_PORT"); grpcPort != "" {
		config.GRPC.Port = grpcPort
	}
	if certFile := os.Getenv("TLS_CERT_FILE"); certFile != "" {
		if config.HTTP.TLS != nil {
			config.HTTP.TLS.CertFile = certFile
		}
		if config.GRPC.TLS != nil {
			config.GRPC.TLS.CertFile = certFile
		}
	}
	if keyFile := os.Getenv("TLS_KEY_FILE"); keyFile != "" {
		if config.HTTP.TLS != nil {
			config.HTTP.TLS.KeyFile = keyFile
		}
		if config.GRPC.TLS != nil {
			config.GRPC.TLS.KeyFile = keyFile
		}
	}
	if caFile := os.Getenv("TLS_CA_FILE"); caFile != "" {
		if config.HTTP.TLS != nil {
			config.HTTP.TLS.CAFiles = []string{caFile}
		}
		if config.GRPC.TLS != nil {
			config.GRPC.TLS.CAFiles = []string{caFile}
		}
	}
}

// printCertificateInfo prints certificate information for debugging.
func printCertificateInfo(cert *x509.Certificate) {
	if cert == nil {
		return
	}
	info := tlsconfig.ExtractCertificateInfo(cert)
	logger.GetLogger().Info("Certificate loaded",
		zap.String("subject", info.Subject),
		zap.String("issuer", info.Issuer),
		zap.String("not_before", info.NotBefore),
		zap.String("not_after", info.NotAfter))
}

func main() {
	// Initialize logger
	logger.InitLogger()

	// Determine working directory
	execDir, err := os.Executable()
	if err != nil {
		execDir = "."
	} else {
		execDir = filepath.Dir(execDir)
	}

	// Load server configuration
	configPath := getEnv("CONFIG_PATH", "swit.yaml")
	if !filepath.IsAbs(configPath) {
		// Try current directory first, then executable directory
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = filepath.Join(execDir, configPath)
		}
	}

	serverConfig, err := loadConfigFromFile(configPath)
	if err != nil {
		logger.GetLogger().Fatal("Failed to load server configuration", zap.Error(err))
	}

	if serverConfig == nil {
		logger.GetLogger().Warn("Using default server configuration")
		serverConfig = createDefaultConfig()
	}

	// Adjust certificate paths if they are relative
	adjustCertPaths(serverConfig, execDir)

	// Validate server configuration
	if err := serverConfig.Validate(); err != nil {
		logger.GetLogger().Fatal("Invalid server configuration", zap.Error(err))
	}

	// Create service and dependencies
	service := NewMTLSExampleService("mtls-authentication-example")
	deps := NewMTLSDependencyContainer()

	// Create base server
	baseServer, err := server.NewBusinessServerCore(serverConfig, service, deps)
	if err != nil {
		logger.GetLogger().Fatal("Failed to create server", zap.Error(err))
	}

	// Start server
	startCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(startCtx); err != nil {
		logger.GetLogger().Fatal("Failed to start server", zap.Error(err))
	}

	// Log startup information
	logger.GetLogger().Info("mTLS Authentication Example Service Started",
		zap.String("https_address", baseServer.GetHTTPAddress()),
		zap.String("grpc_address", baseServer.GetGRPCAddress()),
		zap.String("service_name", "mtls-authentication-example"))

	logger.GetLogger().Info("mTLS Configuration",
		zap.String("client_auth", serverConfig.HTTP.TLS.ClientAuth),
		zap.String("min_version", serverConfig.HTTP.TLS.MinVersion),
		zap.String("max_version", serverConfig.HTTP.TLS.MaxVersion))

	logger.GetLogger().Info("Example endpoints (use curl with client certificate)",
		zap.String("public_info", fmt.Sprintf("https://localhost:%s/api/v1/public/info", serverConfig.HTTP.Port)),
		zap.String("profile", fmt.Sprintf("https://localhost:%s/api/v1/protected/profile", serverConfig.HTTP.Port)),
		zap.String("admin", fmt.Sprintf("https://localhost:%s/api/v1/admin/dashboard", serverConfig.HTTP.Port)))

	logger.GetLogger().Info("Test with curl",
		zap.String("command", fmt.Sprintf(
			"curl --cacert certs/ca.crt --cert certs/client.crt --key certs/client.key https://localhost:%s/api/v1/public/info",
			serverConfig.HTTP.Port)))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.GetLogger().Info("Shutdown signal received, stopping server")

	// Graceful shutdown
	if err := baseServer.Shutdown(); err != nil {
		logger.GetLogger().Error("Error during shutdown", zap.Error(err))
	} else {
		logger.GetLogger().Info("Server stopped gracefully")
	}
}

// adjustCertPaths adjusts certificate paths to be relative to the working directory.
func adjustCertPaths(config *server.ServerConfig, baseDir string) {
	// Check if we're running from the example directory
	if _, err := os.Stat("certs/server.crt"); err == nil {
		return // Paths are already correct
	}

	// Try to find certs relative to executable
	if config.HTTP.TLS != nil {
		if !filepath.IsAbs(config.HTTP.TLS.CertFile) {
			config.HTTP.TLS.CertFile = filepath.Join(baseDir, config.HTTP.TLS.CertFile)
		}
		if !filepath.IsAbs(config.HTTP.TLS.KeyFile) {
			config.HTTP.TLS.KeyFile = filepath.Join(baseDir, config.HTTP.TLS.KeyFile)
		}
		for i, caFile := range config.HTTP.TLS.CAFiles {
			if !filepath.IsAbs(caFile) {
				config.HTTP.TLS.CAFiles[i] = filepath.Join(baseDir, caFile)
			}
		}
	}

	if config.GRPC.TLS != nil {
		if !filepath.IsAbs(config.GRPC.TLS.CertFile) {
			config.GRPC.TLS.CertFile = filepath.Join(baseDir, config.GRPC.TLS.CertFile)
		}
		if !filepath.IsAbs(config.GRPC.TLS.KeyFile) {
			config.GRPC.TLS.KeyFile = filepath.Join(baseDir, config.GRPC.TLS.KeyFile)
		}
		for i, caFile := range config.GRPC.TLS.CAFiles {
			if !filepath.IsAbs(caFile) {
				config.GRPC.TLS.CAFiles[i] = filepath.Join(baseDir, caFile)
			}
		}
	}
}

// getEnv gets an environment variable with a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable with a default value.
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// Compile-time interface checks
var (
	_ server.BusinessServiceRegistrar    = (*MTLSExampleService)(nil)
	_ server.BusinessHTTPHandler         = (*MTLSHTTPHandler)(nil)
	_ server.BusinessGRPCService         = (*MTLSGRPCHandler)(nil)
	_ server.BusinessHealthCheck         = (*MTLSHealthCheck)(nil)
	_ server.BusinessDependencyContainer = (*MTLSDependencyContainer)(nil)
)
