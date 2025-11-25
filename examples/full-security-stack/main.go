// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main demonstrates a comprehensive security stack integration using the Swit framework.
// This example shows how to:
// - Configure OAuth2/OIDC authentication with Keycloak
// - Implement OPA-based authorization (RBAC and ABAC)
// - Enable mTLS for transport encryption
// - Collect security metrics with Prometheus
// - Generate audit logs for compliance
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/middleware"
	"github.com/innovationmech/swit/pkg/security"
	"github.com/innovationmech/swit/pkg/security/jwt"
	"github.com/innovationmech/swit/pkg/security/opa"
	switoauth2 "github.com/innovationmech/swit/pkg/security/oauth2"
	tlsconfig "github.com/innovationmech/swit/pkg/security/tls"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/transport"
)

// FullSecurityService implements the ServiceRegistrar interface for the full security stack example.
type FullSecurityService struct {
	name            string
	oauth2Client    *switoauth2.Client
	jwtValidator    *jwt.Validator
	opaClient       opa.Client
	securityMetrics *security.SecurityMetrics
	flowManager     *switoauth2.FlowManager
	config          *SecurityConfig
}

// SecurityConfig holds the security configuration for the example.
type SecurityConfig struct {
	OAuth2Enabled bool
	OPAEnabled    bool
	MTLSEnabled   bool
	AuditEnabled  bool
	MetricsEnabled bool
	PolicyType    string // "rbac" or "abac"
}

// NewFullSecurityService creates a new full security stack service.
func NewFullSecurityService(
	name string,
	oauth2Client *switoauth2.Client,
	jwtValidator *jwt.Validator,
	opaClient opa.Client,
	securityMetrics *security.SecurityMetrics,
	config *SecurityConfig,
) *FullSecurityService {
	return &FullSecurityService{
		name:            name,
		oauth2Client:    oauth2Client,
		jwtValidator:    jwtValidator,
		opaClient:       opaClient,
		securityMetrics: securityMetrics,
		flowManager:     switoauth2.NewFlowManager(),
		config:          config,
	}
}

// RegisterServices registers the HTTP handlers with the server.
func (s *FullSecurityService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	httpHandler := &FullSecurityHTTPHandler{
		serviceName:     s.name,
		oauth2Client:    s.oauth2Client,
		jwtValidator:    s.jwtValidator,
		opaClient:       s.opaClient,
		securityMetrics: s.securityMetrics,
		flowManager:     s.flowManager,
		config:          s.config,
		documents:       initializeDocuments(),
	}
	if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register health check
	healthCheck := &FullSecurityHealthCheck{
		serviceName:  s.name,
		oauth2Client: s.oauth2Client,
		opaClient:    s.opaClient,
	}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// Document represents a document in the system.
type Document struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Owner          string    `json:"owner"`
	Classification string    `json:"classification"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// initializeDocuments creates sample documents for demonstration.
func initializeDocuments() map[string]*Document {
	return map[string]*Document{
		"doc-1": {
			ID:             "doc-1",
			Title:          "Public Announcement",
			Content:        "This is a public document accessible to all authenticated users.",
			Owner:          "system",
			Classification: "public",
			CreatedAt:      time.Now().Add(-72 * time.Hour),
			UpdatedAt:      time.Now().Add(-24 * time.Hour),
		},
		"doc-2": {
			ID:             "doc-2",
			Title:          "Engineering Specs",
			Content:        "Internal engineering specifications and technical details.",
			Owner:          "alice",
			Classification: "internal",
			CreatedAt:      time.Now().Add(-48 * time.Hour),
			UpdatedAt:      time.Now().Add(-12 * time.Hour),
		},
		"doc-3": {
			ID:             "doc-3",
			Title:          "Financial Report Q4",
			Content:        "Confidential financial data for Q4 2024.",
			Owner:          "bob",
			Classification: "confidential",
			CreatedAt:      time.Now().Add(-24 * time.Hour),
			UpdatedAt:      time.Now(),
		},
		"doc-4": {
			ID:             "doc-4",
			Title:          "Strategic Plan 2025",
			Content:        "Restricted strategic planning document.",
			Owner:          "admin",
			Classification: "restricted",
			CreatedAt:      time.Now().Add(-96 * time.Hour),
			UpdatedAt:      time.Now().Add(-48 * time.Hour),
		},
	}
}

// FullSecurityHTTPHandler implements the HTTPHandler interface with full security features.
type FullSecurityHTTPHandler struct {
	serviceName     string
	oauth2Client    *switoauth2.Client
	jwtValidator    *jwt.Validator
	opaClient       opa.Client
	securityMetrics *security.SecurityMetrics
	flowManager     *switoauth2.FlowManager
	config          *SecurityConfig
	documents       map[string]*Document
	mu              sync.RWMutex
}

// RegisterRoutes registers HTTP routes with the router.
func (h *FullSecurityHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Public endpoints (no authentication required)
	public := ginRouter.Group("/api/v1/public")
	{
		public.GET("/info", h.handlePublicInfo)
		public.GET("/login", h.handleLogin)
		public.GET("/callback", h.handleCallback)
		public.POST("/refresh", h.handleRefresh)
		public.POST("/logout", h.handleLogout)
		public.GET("/security-status", h.handleSecurityStatus)
	}

	// Protected endpoints (OAuth2 authentication required)
	protected := ginRouter.Group("/api/v1/protected")
	if h.config.OAuth2Enabled && h.oauth2Client != nil && h.jwtValidator != nil {
		protected.Use(middleware.OAuth2Middleware(h.oauth2Client, h.jwtValidator))
	}
	// Apply OPA authorization if enabled
	if h.config.OPAEnabled && h.opaClient != nil {
		protected.Use(h.opaMiddleware())
	}
	{
		protected.GET("/profile", h.handleProfile)
		protected.GET("/documents", h.handleListDocuments)
		protected.GET("/documents/:id", h.handleGetDocument)
		protected.POST("/documents", h.handleCreateDocument)
		protected.PUT("/documents/:id", h.handleUpdateDocument)
		protected.DELETE("/documents/:id", h.handleDeleteDocument)
	}

	// Admin endpoints (require admin role)
	admin := ginRouter.Group("/api/v1/admin")
	if h.config.OAuth2Enabled && h.oauth2Client != nil && h.jwtValidator != nil {
		admin.Use(middleware.OAuth2Middleware(h.oauth2Client, h.jwtValidator))
		admin.Use(middleware.RequireRoles("admin"))
	}
	if h.config.OPAEnabled && h.opaClient != nil {
		admin.Use(h.opaMiddleware())
	}
	{
		admin.GET("/dashboard", h.handleAdminDashboard)
		admin.GET("/audit-logs", h.handleAuditLogs)
		admin.GET("/security-metrics", h.handleSecurityMetrics)
		admin.GET("/certificates", h.handleListCertificates)
	}

	// mTLS-specific endpoints (certificate-based access)
	mtls := ginRouter.Group("/api/v1/mtls")
	mtls.Use(h.requireClientCertMiddleware())
	{
		mtls.GET("/verify", h.handleMTLSVerify)
		mtls.GET("/certificate-info", h.handleCertificateInfo)
	}

	return nil
}

// GetServiceName returns the service name.
func (h *FullSecurityHTTPHandler) GetServiceName() string {
	return h.serviceName
}

// opaMiddleware creates OPA authorization middleware.
func (h *FullSecurityHTTPHandler) opaMiddleware() gin.HandlerFunc {
	return middleware.OPAMiddleware(
		h.opaClient,
		middleware.WithDecisionPath(fmt.Sprintf("%s/allow", h.config.PolicyType)),
		middleware.WithLogger(logger.GetLogger()),
		middleware.WithInputBuilder(func(c *gin.Context) (*opa.PolicyInput, error) {
			builder := opa.NewPolicyInputBuilder().FromHTTPRequest(c)
			user := opa.ExtractUserFromContext(c)
			builder.WithUser(user)
			builder.WithResourceType("document")
			return builder.Build(), nil
		}),
		middleware.WithAuditLog(func(auditLog *middleware.AuditLog) {
			if h.config.AuditEnabled {
				h.recordAuditLog(auditLog)
			}
			if h.securityMetrics != nil {
				decision := "allow"
				if !auditLog.Allowed {
					decision = "deny"
					h.securityMetrics.RecordAccessDenied("policy_denied", auditLog.Request.Path)
				}
				h.securityMetrics.RecordPolicyEvaluation(h.config.PolicyType, decision)
				h.securityMetrics.RecordPolicyEvaluationDuration(h.config.PolicyType, float64(auditLog.Duration)/1000.0)
			}
		}),
	)
}

// requireClientCertMiddleware creates middleware that requires a client certificate.
func (h *FullSecurityHTTPHandler) requireClientCertMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		certInfo := transport.GetClientCertificateInfo(c)
		if certInfo == nil {
			if h.securityMetrics != nil {
				h.securityMetrics.RecordSecurityEvent("missing_client_cert", "high")
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "Client certificate required",
				"message": "This endpoint requires mTLS authentication with a valid client certificate",
			})
			return
		}

		// Record successful mTLS connection
		if h.securityMetrics != nil {
			h.securityMetrics.RecordTLSConnection("1.3", "mutual_tls")
		}

		c.Next()
	}
}

// recordAuditLog records an audit log entry.
func (h *FullSecurityHTTPHandler) recordAuditLog(auditLog *middleware.AuditLog) {
	logger.GetLogger().Info("Security audit log",
		zap.Bool("allowed", auditLog.Allowed),
		zap.String("path", sanitizeLogValue(auditLog.Request.Path)),
		zap.String("method", auditLog.Request.Method),
		zap.String("client_ip", auditLog.Request.ClientIP),
		zap.Int64("duration_ms", auditLog.Duration),
		zap.String("decision_id", auditLog.DecisionID))
}

// handlePublicInfo handles the public info endpoint.
func (h *FullSecurityHTTPHandler) handlePublicInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "Welcome to Full Security Stack Example",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
		"security_features": gin.H{
			"oauth2_enabled":  h.config.OAuth2Enabled,
			"opa_enabled":     h.config.OPAEnabled,
			"mtls_enabled":    h.config.MTLSEnabled,
			"audit_enabled":   h.config.AuditEnabled,
			"metrics_enabled": h.config.MetricsEnabled,
			"policy_type":     h.config.PolicyType,
		},
		"endpoints": gin.H{
			"login":           "/api/v1/public/login",
			"profile":         "/api/v1/protected/profile",
			"documents":       "/api/v1/protected/documents",
			"admin_dashboard": "/api/v1/admin/dashboard",
			"mtls_verify":     "/api/v1/mtls/verify",
		},
	})
}

// handleSecurityStatus handles the security status endpoint.
func (h *FullSecurityHTTPHandler) handleSecurityStatus(c *gin.Context) {
	status := gin.H{
		"timestamp": time.Now().UTC(),
		"components": gin.H{
			"oauth2": gin.H{
				"enabled": h.config.OAuth2Enabled,
				"status":  "unknown",
			},
			"opa": gin.H{
				"enabled":     h.config.OPAEnabled,
				"policy_type": h.config.PolicyType,
				"status":      "unknown",
			},
			"mtls": gin.H{
				"enabled": h.config.MTLSEnabled,
				"status":  "unknown",
			},
			"audit": gin.H{
				"enabled": h.config.AuditEnabled,
			},
			"metrics": gin.H{
				"enabled": h.config.MetricsEnabled,
			},
		},
	}

	// Check OAuth2 status
	if h.config.OAuth2Enabled && h.oauth2Client != nil {
		status["components"].(gin.H)["oauth2"].(gin.H)["status"] = "healthy"
	}

	// Check OPA status by performing a simple evaluation
	if h.config.OPAEnabled && h.opaClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		// Perform a simple query to check OPA health
		_, err := h.opaClient.Evaluate(ctx, fmt.Sprintf("%s/allow", h.config.PolicyType), map[string]interface{}{
			"user":    map[string]interface{}{"roles": []string{}},
			"request": map[string]interface{}{"method": "GET", "path": "/health"},
		})
		if err == nil {
			status["components"].(gin.H)["opa"].(gin.H)["status"] = "healthy"
		} else {
			status["components"].(gin.H)["opa"].(gin.H)["status"] = "unhealthy"
		}
	}

	// Check mTLS status
	if h.config.MTLSEnabled {
		status["components"].(gin.H)["mtls"].(gin.H)["status"] = "enabled"
	}

	c.JSON(http.StatusOK, status)
}

// handleLogin initiates the OAuth2 authorization code flow.
func (h *FullSecurityHTTPHandler) handleLogin(c *gin.Context) {
	if !h.config.OAuth2Enabled || h.oauth2Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "OAuth2 not enabled",
			"message": "OAuth2 authentication is not configured for this service",
		})
		return
	}

	// Record auth attempt
	if h.securityMetrics != nil {
		h.securityMetrics.RecordAuthAttempt("oauth2", "initiated")
	}

	// Start OAuth2 flow with state and PKCE
	flowState, err := h.flowManager.StartFlow(
		getEnv("OAUTH2_REDIRECT_URL", "http://localhost:8080/api/v1/public/callback"),
		[]string{"openid", "profile", "email"},
		true, // Use PKCE
	)
	if err != nil {
		if h.securityMetrics != nil {
			h.securityMetrics.RecordAuthAttempt("oauth2", "error")
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start OAuth2 flow",
			"details": err.Error(),
		})
		return
	}

	// Generate authorization URL with flow state
	authURL := h.oauth2Client.AuthCodeURLWithFlow(flowState)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Redirect to this URL to authenticate",
		"authorization_url": authURL,
		"state":             flowState.State,
		"hint":              "Open the authorization_url in your browser",
	})
}

// handleCallback handles the OAuth2 callback.
func (h *FullSecurityHTTPHandler) handleCallback(c *gin.Context) {
	if !h.config.OAuth2Enabled || h.oauth2Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OAuth2 not enabled",
		})
		return
	}

	start := time.Now()

	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		if h.securityMetrics != nil {
			h.securityMetrics.RecordAuthAttempt("oauth2", "failed")
			h.securityMetrics.RecordSecurityEvent("oauth2_error", "medium")
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             errorParam,
			"error_description": c.Query("error_description"),
		})
		return
	}

	if code == "" {
		if h.securityMetrics != nil {
			h.securityMetrics.RecordAuthAttempt("oauth2", "failed")
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing authorization code",
		})
		return
	}

	// Validate state parameter
	flowState, err := h.flowManager.ValidateState(state)
	if err != nil {
		if h.securityMetrics != nil {
			h.securityMetrics.RecordAuthAttempt("oauth2", "failed")
			h.securityMetrics.RecordSecurityEvent("invalid_state", "high")
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid state parameter",
			"details": err.Error(),
		})
		return
	}

	// Exchange authorization code for tokens
	ctx := context.Background()
	token, err := h.oauth2Client.ExchangeWithFlow(ctx, code, flowState)
	if err != nil {
		if h.securityMetrics != nil {
			h.securityMetrics.RecordAuthAttempt("oauth2", "failed")
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to exchange authorization code",
			"details": err.Error(),
		})
		return
	}

	// Record successful authentication
	if h.securityMetrics != nil {
		h.securityMetrics.RecordAuthAttempt("oauth2", "success")
		h.securityMetrics.RecordAuthDuration("oauth2", time.Since(start).Seconds())
	}

	// Extract and verify ID token
	var idTokenClaims map[string]interface{}
	if rawIDToken, ok := token.Extra("id_token").(string); ok {
		idToken, err := h.oauth2Client.VerifyIDToken(ctx, rawIDToken)
		if err != nil {
			logger.GetLogger().Warn("Failed to verify ID token", zap.Error(err))
		} else {
			if err := idToken.Claims(&idTokenClaims); err == nil {
				// Verify nonce if present
				if nonce, ok := idTokenClaims["nonce"].(string); ok && flowState.Nonce != "" {
					if err := switoauth2.ValidateNonce(flowState.Nonce, nonce); err != nil {
						if h.securityMetrics != nil {
							h.securityMetrics.RecordSecurityEvent("nonce_mismatch", "high")
						}
						c.JSON(http.StatusBadRequest, gin.H{
							"error":   "Nonce validation failed",
							"details": err.Error(),
						})
						return
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Authentication successful",
		"access_token":    token.AccessToken,
		"token_type":      token.TokenType,
		"expires_in":      int(time.Until(token.Expiry).Seconds()),
		"refresh_token":   token.RefreshToken,
		"id_token_claims": idTokenClaims,
		"hint":            "Use the access_token in the Authorization header for protected endpoints",
	})
}

// handleRefresh handles token refresh requests.
func (h *FullSecurityHTTPHandler) handleRefresh(c *gin.Context) {
	if !h.config.OAuth2Enabled || h.oauth2Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OAuth2 not enabled",
		})
		return
	}

	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	ctx := context.Background()
	newToken, err := h.oauth2Client.RefreshToken(ctx, &oauth2.Token{
		RefreshToken: request.RefreshToken,
	})
	if err != nil {
		if h.securityMetrics != nil {
			h.securityMetrics.RecordSecurityEvent("token_refresh_failed", "medium")
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh token",
			"details": err.Error(),
		})
		return
	}

	if h.securityMetrics != nil {
		h.securityMetrics.RecordTokenRefresh("success")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Token refreshed successfully",
		"access_token":  newToken.AccessToken,
		"token_type":    newToken.TokenType,
		"expires_in":    int(time.Until(newToken.Expiry).Seconds()),
		"refresh_token": newToken.RefreshToken,
	})
}

// handleLogout handles logout requests (token revocation).
func (h *FullSecurityHTTPHandler) handleLogout(c *gin.Context) {
	if !h.config.OAuth2Enabled || h.oauth2Client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OAuth2 not enabled",
		})
		return
	}

	var request struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	ctx := context.Background()
	if err := h.oauth2Client.RevokeToken(ctx, request.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to revoke token",
			"details": err.Error(),
		})
		return
	}

	if h.securityMetrics != nil {
		h.securityMetrics.RecordSessionRevoked("user_initiated")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// handleProfile handles the protected profile endpoint.
func (h *FullSecurityHTTPHandler) handleProfile(c *gin.Context) {
	userInfo, ok := middleware.GetUserInfo(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User info not found",
		})
		return
	}

	// Add certificate info if available (mTLS)
	var certInfo interface{}
	if h.config.MTLSEnabled {
		if cert := transport.GetClientCertificateInfo(c); cert != nil {
			certInfo = gin.H{
				"common_name":  cert.CommonName,
				"organization": cert.Organization,
				"serial":       cert.SerialNumber,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Profile retrieved successfully",
		"user_info":        userInfo,
		"certificate_info": certInfo,
		"timestamp":        time.Now().UTC(),
	})
}

// handleListDocuments handles listing all documents.
func (h *FullSecurityHTTPHandler) handleListDocuments(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	docs := make([]*Document, 0, len(h.documents))
	for _, doc := range h.documents {
		docs = append(docs, doc)
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": docs,
		"count":     len(docs),
		"timestamp": time.Now().UTC(),
	})
}

// handleGetDocument handles getting a specific document.
func (h *FullSecurityHTTPHandler) handleGetDocument(c *gin.Context) {
	id := c.Param("id")

	h.mu.RLock()
	doc, exists := h.documents[id]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Document not found",
		})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// handleCreateDocument handles creating a new document.
func (h *FullSecurityHTTPHandler) handleCreateDocument(c *gin.Context) {
	var doc Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Set owner from user info
	if userInfo, ok := middleware.GetUserInfo(c); ok {
		doc.Owner = userInfo.Subject
	}

	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	h.mu.Lock()
	h.documents[doc.ID] = &doc
	h.mu.Unlock()

	c.JSON(http.StatusCreated, doc)
}

// handleUpdateDocument handles updating a document.
func (h *FullSecurityHTTPHandler) handleUpdateDocument(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	doc, exists := h.documents[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Document not found",
		})
		return
	}

	var updates Document
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	doc.Title = updates.Title
	doc.Content = updates.Content
	doc.Classification = updates.Classification
	doc.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, doc)
}

// handleDeleteDocument handles deleting a document.
func (h *FullSecurityHTTPHandler) handleDeleteDocument(c *gin.Context) {
	id := c.Param("id")

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.documents[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Document not found",
		})
		return
	}

	delete(h.documents, id)
	c.JSON(http.StatusOK, gin.H{
		"message": "Document deleted successfully",
	})
}

// handleAdminDashboard handles the admin dashboard endpoint.
func (h *FullSecurityHTTPHandler) handleAdminDashboard(c *gin.Context) {
	userInfo, _ := middleware.GetUserInfo(c)

	h.mu.RLock()
	docCount := len(h.documents)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to the Admin Dashboard",
		"admin":   userInfo,
		"dashboard": gin.H{
			"total_documents":       docCount,
			"security_events_today": 42,
			"active_sessions":       15,
			"policy_evaluations":    1234,
		},
		"timestamp": time.Now().UTC(),
	})
}

// handleAuditLogs handles the audit logs endpoint.
func (h *FullSecurityHTTPHandler) handleAuditLogs(c *gin.Context) {
	// In a real application, this would query an audit log store
	c.JSON(http.StatusOK, gin.H{
		"message": "Audit logs retrieved",
		"logs": []gin.H{
			{
				"timestamp": time.Now().Add(-1 * time.Hour).UTC(),
				"action":    "document_access",
				"user":      "alice",
				"resource":  "doc-1",
				"result":    "allowed",
			},
			{
				"timestamp": time.Now().Add(-30 * time.Minute).UTC(),
				"action":    "document_update",
				"user":      "bob",
				"resource":  "doc-2",
				"result":    "allowed",
			},
			{
				"timestamp": time.Now().Add(-15 * time.Minute).UTC(),
				"action":    "admin_access",
				"user":      "charlie",
				"resource":  "dashboard",
				"result":    "denied",
			},
		},
		"timestamp": time.Now().UTC(),
	})
}

// handleSecurityMetrics handles the security metrics endpoint.
func (h *FullSecurityHTTPHandler) handleSecurityMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Security metrics",
		"metrics": gin.H{
			"authentication": gin.H{
				"total_attempts":     1000,
				"successful":         950,
				"failed":             50,
				"avg_duration_ms":    125,
			},
			"authorization": gin.H{
				"policy_evaluations": 5000,
				"allowed":            4800,
				"denied":             200,
				"avg_duration_ms":    15,
			},
			"security_events": gin.H{
				"total":    100,
				"critical": 2,
				"high":     10,
				"medium":   30,
				"low":      58,
			},
			"tls": gin.H{
				"total_connections": 2500,
				"mtls_connections":  500,
			},
		},
		"timestamp": time.Now().UTC(),
	})
}

// handleListCertificates handles the certificate listing endpoint.
func (h *FullSecurityHTTPHandler) handleListCertificates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Certificate inventory",
		"certificates": []gin.H{
			{
				"cn":      "swit-server",
				"type":    "server",
				"status":  "active",
				"issued":  "2025-01-01",
				"expires": "2026-01-01",
			},
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

// handleMTLSVerify handles mTLS verification endpoint.
func (h *FullSecurityHTTPHandler) handleMTLSVerify(c *gin.Context) {
	certInfo := transport.GetClientCertificateInfo(c)

	c.JSON(http.StatusOK, gin.H{
		"message":     "mTLS verification successful",
		"certificate": gin.H{
			"common_name":  certInfo.CommonName,
			"organization": certInfo.Organization,
			"serial":       certInfo.SerialNumber,
			"not_before":   certInfo.NotBefore,
			"not_after":    certInfo.NotAfter,
			"issuer":       certInfo.Issuer,
		},
		"verified":  true,
		"timestamp": time.Now().UTC(),
	})
}

// handleCertificateInfo handles the certificate info endpoint.
func (h *FullSecurityHTTPHandler) handleCertificateInfo(c *gin.Context) {
	certInfo := transport.GetClientCertificateInfo(c)

	c.JSON(http.StatusOK, gin.H{
		"certificate": gin.H{
			"subject":             certInfo.Subject,
			"issuer":              certInfo.Issuer,
			"common_name":         certInfo.CommonName,
			"organization":        certInfo.Organization,
			"organizational_unit": certInfo.OrganizationalUnit,
			"country":             certInfo.Country,
			"serial_number":       certInfo.SerialNumber,
			"not_before":          certInfo.NotBefore,
			"not_after":           certInfo.NotAfter,
			"dns_names":           certInfo.DNSNames,
			"ip_addresses":        certInfo.IPAddresses,
			"email_addresses":     certInfo.EmailAddresses,
		},
		"timestamp": time.Now().UTC(),
	})
}

// FullSecurityHealthCheck implements the HealthCheck interface.
type FullSecurityHealthCheck struct {
	serviceName  string
	oauth2Client *switoauth2.Client
	opaClient    opa.Client
}

// Check performs a health check.
func (h *FullSecurityHealthCheck) Check(ctx context.Context) error {
	// Check OPA health by performing a simple evaluation
	if h.opaClient != nil {
		_, err := h.opaClient.Evaluate(ctx, "rbac/allow", map[string]interface{}{
			"user":    map[string]interface{}{"roles": []string{}},
			"request": map[string]interface{}{"method": "GET", "path": "/health"},
		})
		if err != nil {
			return fmt.Errorf("OPA health check failed: %w", err)
		}
	}
	return nil
}

// GetServiceName returns the service name.
func (h *FullSecurityHealthCheck) GetServiceName() string {
	return h.serviceName
}

// FullSecurityDependencyContainer implements the DependencyContainer interface.
type FullSecurityDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

// NewFullSecurityDependencyContainer creates a new dependency container.
func NewFullSecurityDependencyContainer(
	oauth2Client *switoauth2.Client,
	jwtValidator *jwt.Validator,
	opaClient opa.Client,
	securityMetrics *security.SecurityMetrics,
) *FullSecurityDependencyContainer {
	container := &FullSecurityDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}

	if oauth2Client != nil {
		container.services["oauth2_client"] = oauth2Client
	}
	if jwtValidator != nil {
		container.services["jwt_validator"] = jwtValidator
	}
	if opaClient != nil {
		container.services["opa_client"] = opaClient
	}
	if securityMetrics != nil {
		container.services["security_metrics"] = securityMetrics
	}

	return container
}

// GetService retrieves a service by name.
func (d *FullSecurityDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	service, exists := d.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// Close closes the dependency container and cleans up resources.
func (d *FullSecurityDependencyContainer) Close() error {
	if d.closed {
		return nil
	}

	// Close OAuth2 client
	if oauth2Client, ok := d.services["oauth2_client"].(*switoauth2.Client); ok {
		if err := oauth2Client.Close(); err != nil {
			logger.GetLogger().Warn("Failed to close OAuth2 client", zap.Error(err))
		}
	}

	// Close OPA client
	if opaClient, ok := d.services["opa_client"].(opa.Client); ok {
		if err := opaClient.Close(context.Background()); err != nil {
			logger.GetLogger().Warn("Failed to close OPA client", zap.Error(err))
		}
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

	applyEnvironmentOverrides(config)

	return config, nil
}

// createDefaultConfig creates default configuration.
func createDefaultConfig() *server.ServerConfig {
	mtlsEnabled := getBoolEnv("MTLS_ENABLED", false)

	config := &server.ServerConfig{
		ServiceName: "full-security-stack-example",
		HTTP: server.HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Enabled: false,
		},
		ShutdownTimeout: 30 * time.Second,
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	// Configure mTLS if enabled
	if mtlsEnabled {
		config.HTTP.Port = getEnv("HTTP_PORT", "8443")
		config.HTTP.TLS = &tlsconfig.TLSConfig{
			Enabled:    true,
			CertFile:   getEnv("TLS_CERT_FILE", "certs/server.crt"),
			KeyFile:    getEnv("TLS_KEY_FILE", "certs/server.key"),
			CAFiles:    []string{getEnv("TLS_CA_FILE", "certs/ca.crt")},
			ClientAuth: getEnv("TLS_CLIENT_AUTH", "require_and_verify"),
			MinVersion: "TLS1.2",
			MaxVersion: "TLS1.3",
		}
	}

	return config
}

// applyEnvironmentOverrides applies environment variable overrides.
func applyEnvironmentOverrides(config *server.ServerConfig) {
	if httpPort := os.Getenv("HTTP_PORT"); httpPort != "" {
		config.HTTP.Port = httpPort
	}
}

func main() {
	// Initialize logger
	logger.InitLogger()

	// Load server configuration
	configPath := getEnv("CONFIG_PATH", "swit.yaml")
	if !filepath.IsAbs(configPath) {
		if execDir, err := os.Executable(); err == nil {
			configPath = filepath.Join(filepath.Dir(execDir), configPath)
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

	// Validate server configuration
	if err := serverConfig.Validate(); err != nil {
		logger.GetLogger().Fatal("Invalid server configuration", zap.Error(err))
	}

	// Security configuration
	securityConfig := &SecurityConfig{
		OAuth2Enabled:  getBoolEnv("OAUTH2_ENABLED", true),
		OPAEnabled:     getBoolEnv("OPA_ENABLED", true),
		MTLSEnabled:    getBoolEnv("MTLS_ENABLED", false),
		AuditEnabled:   getBoolEnv("AUDIT_ENABLED", true),
		MetricsEnabled: getBoolEnv("METRICS_ENABLED", true),
		PolicyType:     getEnv("POLICY_TYPE", "rbac"),
	}

	ctx := context.Background()

	// Initialize OAuth2 client
	var oauth2Client *switoauth2.Client
	var jwtValidator *jwt.Validator
	if securityConfig.OAuth2Enabled {
		oauth2Config := &switoauth2.Config{
			Enabled:      true,
			Provider:     getEnv("OAUTH2_PROVIDER", "keycloak"),
			ClientID:     getEnv("OAUTH2_CLIENT_ID", "swit-example"),
			ClientSecret: getEnv("OAUTH2_CLIENT_SECRET", "swit-example-secret"),
			RedirectURL:  getEnv("OAUTH2_REDIRECT_URL", "http://localhost:8080/api/v1/public/callback"),
			Scopes:       []string{"openid", "profile", "email"},
			IssuerURL:    getEnv("OAUTH2_ISSUER_URL", "http://localhost:8081/realms/swit"),
			UseDiscovery: getBoolEnv("OAUTH2_USE_DISCOVERY", true),
			HTTPTimeout:  30 * time.Second,
		}
		oauth2Config.LoadFromEnv("OAUTH2_")

		oauth2Client, err = switoauth2.NewClient(ctx, oauth2Config)
		if err != nil {
			logger.GetLogger().Warn("Failed to create OAuth2 client, OAuth2 will be disabled", zap.Error(err))
			securityConfig.OAuth2Enabled = false
		} else {
			jwtConfig := &jwt.Config{
				Issuer:         oauth2Client.GetIssuerURL(),
				Audience:       oauth2Config.ClientID,
				LeewayDuration: oauth2Config.JWTConfig.ClockSkew,
				JWKSConfig: &jwt.JWKSCacheConfig{
					URL:            oauth2Client.GetJWKSURL(),
					RefreshTTL:     15 * time.Minute,
					AutoRefresh:    true,
					MinRefreshWait: 1 * time.Minute,
				},
			}

			jwtValidator, err = jwt.NewValidator(jwtConfig)
			if err != nil {
				logger.GetLogger().Warn("Failed to create JWT validator", zap.Error(err))
			}
		}
	}

	// Initialize OPA client
	var opaClient opa.Client
	if securityConfig.OPAEnabled {
		opaMode := getEnv("OPA_MODE", "embedded")
		var opaConfig *opa.Config

		if opaMode == "embedded" {
			policyPath := getEnv("OPA_POLICY_DIR", "./policies")
			if !filepath.IsAbs(policyPath) {
				absPath, err := filepath.Abs(policyPath)
				if err == nil {
					policyPath = absPath
				}
			}

			opaConfig = &opa.Config{
				Mode: opa.ModeEmbedded,
				EmbeddedConfig: &opa.EmbeddedConfig{
					PolicyDir: policyPath,
				},
				CacheConfig: &opa.CacheConfig{
					Enabled: true,
					MaxSize: 1000,
					TTL:     5 * time.Minute,
				},
				DefaultDecisionPath: fmt.Sprintf("%s/allow", securityConfig.PolicyType),
			}
		} else {
			opaConfig = &opa.Config{
				Mode: opa.ModeRemote,
				RemoteConfig: &opa.RemoteConfig{
					URL:     getEnv("OPA_URL", "http://localhost:8181"),
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
				DefaultDecisionPath: fmt.Sprintf("%s/allow", securityConfig.PolicyType),
			}
		}

		opaClient, err = opa.NewClient(ctx, opaConfig)
		if err != nil {
			logger.GetLogger().Warn("Failed to create OPA client, OPA will be disabled", zap.Error(err))
			securityConfig.OPAEnabled = false
		}
	}

	// Initialize security metrics
	var securityMetrics *security.SecurityMetrics
	if securityConfig.MetricsEnabled {
		metricsConfig := &security.SecurityMetricsConfig{
			Namespace: "swit",
			Subsystem: "security",
			Registry:  prometheus.DefaultRegisterer.(*prometheus.Registry),
		}

		securityMetrics, err = security.NewSecurityMetrics(metricsConfig)
		if err != nil {
			logger.GetLogger().Warn("Failed to create security metrics", zap.Error(err))
		}
	}

	// Create service and dependencies
	service := NewFullSecurityService(
		"full-security-stack-example",
		oauth2Client,
		jwtValidator,
		opaClient,
		securityMetrics,
		securityConfig,
	)

	deps := NewFullSecurityDependencyContainer(oauth2Client, jwtValidator, opaClient, securityMetrics)

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
	logger.GetLogger().Info("Full Security Stack Example Service Started",
		zap.String("http_address", baseServer.GetHTTPAddress()),
		zap.String("service_name", "full-security-stack-example"))

	logger.GetLogger().Info("Security Features",
		zap.Bool("oauth2_enabled", securityConfig.OAuth2Enabled),
		zap.Bool("opa_enabled", securityConfig.OPAEnabled),
		zap.Bool("mtls_enabled", securityConfig.MTLSEnabled),
		zap.Bool("audit_enabled", securityConfig.AuditEnabled),
		zap.Bool("metrics_enabled", securityConfig.MetricsEnabled),
		zap.String("policy_type", securityConfig.PolicyType))

	logger.GetLogger().Info("Example endpoints",
		zap.String("public_info", fmt.Sprintf("http://%s/api/v1/public/info", baseServer.GetHTTPAddress())),
		zap.String("security_status", fmt.Sprintf("http://%s/api/v1/public/security-status", baseServer.GetHTTPAddress())),
		zap.String("login", fmt.Sprintf("http://%s/api/v1/public/login", baseServer.GetHTTPAddress())))

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

// sanitizeLogValue sanitizes a log value to prevent log injection.
func sanitizeLogValue(value string) string {
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\t", " ")
	return value
}

// Compile-time interface checks
var (
	_ server.BusinessServiceRegistrar    = (*FullSecurityService)(nil)
	_ server.BusinessHTTPHandler         = (*FullSecurityHTTPHandler)(nil)
	_ server.BusinessHealthCheck         = (*FullSecurityHealthCheck)(nil)
	_ server.BusinessDependencyContainer = (*FullSecurityDependencyContainer)(nil)
)

// Ensure json package is used
var _ = json.Marshal

