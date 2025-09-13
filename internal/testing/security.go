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
package testingx

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/innovationmech/swit/pkg/messaging"
	"go.uber.org/zap"
)

// NewTestSecurityManager 创建启用 AES-GCM 与 HMAC-SHA256 签名的默认安全管理器
func NewTestSecurityManager() (*messaging.SecurityManager, error) {
	policy := &messaging.SecurityPolicy{
		Name:    "default",
		Enabled: true,
		Encryption: &messaging.EncryptionConfig{
			Enabled:   true,
			Algorithm: messaging.EncryptionAlgorithmA256GCM,
			KeySize:   32,
		},
		Signing: &messaging.SigningConfig{
			Enabled:   true,
			Algorithm: messaging.SigningAlgorithmHS256,
		},
		Audit: &messaging.AuditConfig{
			Enabled:  true,
			LogLevel: "info",
		},
	}

	cfg := &messaging.SecurityManagerConfig{DefaultPolicy: policy, Logger: zap.NewNop()}
	return messaging.NewSecurityManager(cfg)
}

// NewTestAuthManagerWithJWT 创建仅启用 JWT Provider 的认证管理器
func NewTestAuthManagerWithJWT(secret string) (*messaging.AuthManager, error) {
	jwtProvider, err := messaging.NewJWTAuthProvider(&messaging.JWTAuthProviderConfig{Secret: secret})
	if err != nil {
		return nil, err
	}
	cfg := &messaging.AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]messaging.AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	}
	return messaging.NewAuthManager(cfg)
}

// NewTestAccessControlManager 创建内存提供者的访问控制管理器
func NewTestAccessControlManager(auth *messaging.AuthManager) (*messaging.AccessControlManager, error) {
	provider := messaging.NewMemoryAccessControlProvider(1 * time.Minute)
	cfg := &messaging.AccessControlManagerConfig{
		Provider:     provider,
		AuthManager:  auth,
		Enabled:      true,
		DefaultDeny:  false,
		CacheEnabled: true,
		CacheTTL:     1 * time.Minute,
		LogDecisions: true,
	}
	return messaging.NewAccessControlManager(cfg)
}

// MintTestJWT 生成用于测试的有效 JWT（user_id, scopes 可选）
func MintTestJWT(secret, userID string, scopes []string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(ttl).Unix(),
		"iat":     time.Now().Unix(),
	}
	if len(scopes) > 0 {
		claims["scopes"] = scopes
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// CreateAuthenticatedMessage 构造带 JWT 头的消息（便于通过中间件）
func CreateAuthenticatedMessage(messageID, topic, jwtToken string) *messaging.Message {
	return &messaging.Message{
		ID:        messageID,
		Topic:     topic,
		Payload:   []byte("hello"),
		Timestamp: time.Now(),
		Headers: map[string]string{
			"x-auth-type":   string(messaging.AuthTypeJWT),
			"authorization": "Bearer " + jwtToken,
		},
	}
}

// GrantPublishAll 为指定用户授予所有 topic 的发布权限
func GrantPublishAll(ctx context.Context, acm *messaging.AccessControlManager, userID string) error {
	entry := &messaging.AccessControlEntry{
		EntityType:  "user",
		EntityID:    userID,
		Resources:   []*messaging.Resource{messaging.ResourcePattern(messaging.AccessResourceTypeTopic, "*")},
		Permissions: []messaging.Permission{messaging.PermissionPublish},
		Effect:      "allow",
	}
	return acm.AddAccessControlEntry(ctx, entry)
}
