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
package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 集成测试：验证认证中间件提取与校验、访问控制中间件的允许/拒绝逻辑
func TestAuthAndAccessControl_Integration(t *testing.T) {
	// 准备 AuthManager 与 AccessControlManager
	const secret = "test-secret"
	authManager, err := newTestAuthManagerWithJWT(secret)
	require.NoError(t, err)

	accessManager, err := newTestAccessControlManager(authManager)
	require.NoError(t, err)

	// 中间件
	authMW := NewAuthMiddleware(&AuthMiddlewareConfig{AuthManager: authManager, Required: true})
	acMW := NewAccessControlMiddleware(&AccessControlMiddlewareConfig{AccessControlManager: accessManager, Required: true})

	// 业务处理器：验证能被调用
	invoked := false
	handler := MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		invoked = true
		return nil
	})

	// 包装处理器（先认证再鉴权）
	wrapped := authMW.Wrap(acMW.Wrap(handler))

	// 生成有效 JWT
	token, err := mintTestJWT(secret, "user-1", []string{"publish"}, time.Hour)
	require.NoError(t, err)

	// 构造消息（发布操作）
	msg := createAuthenticatedMessage("m-1", "orders.created", token)
	msg.Headers["x-operation"] = "publish"

	// 未授权前应拒绝（没有授予发布权限）
	ctx := context.Background()
	err = wrapped.Handle(ctx, msg)
	require.Error(t, err)
	assert.False(t, invoked)

	// 授权用户发布所有 topic
	err = grantPublishAll(ctx, accessManager, "user-1")
	require.NoError(t, err)

	// 重新处理，应当通过
	err = wrapped.Handle(ctx, msg)
	require.NoError(t, err)
	assert.True(t, invoked)
}

// --- 测试内联辅助 ---

func newTestAuthManagerWithJWT(secret string) (*AuthManager, error) {
	jwtProvider, err := NewJWTAuthProvider(&JWTAuthProviderConfig{Secret: secret})
	if err != nil {
		return nil, err
	}
	cfg := &AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	}
	return NewAuthManager(cfg)
}

func newTestAccessControlManager(auth *AuthManager) (*AccessControlManager, error) {
	provider := NewMemoryAccessControlProvider(1 * time.Minute)
	cfg := &AccessControlManagerConfig{
		Provider:     provider,
		AuthManager:  auth,
		Enabled:      true,
		DefaultDeny:  false,
		CacheEnabled: true,
		CacheTTL:     1 * time.Minute,
		LogDecisions: true,
	}
	return NewAccessControlManager(cfg)
}

func mintTestJWT(secret, userID string, scopes []string, ttl time.Duration) (string, error) {
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

func createAuthenticatedMessage(messageID, topic, jwtToken string) *Message {
	return &Message{
		ID:        messageID,
		Topic:     topic,
		Payload:   []byte("hello"),
		Timestamp: time.Now(),
		Headers: map[string]string{
			"x-auth-type":   string(AuthTypeJWT),
			"authorization": "Bearer " + jwtToken,
		},
	}
}

func grantPublishAll(ctx context.Context, acm *AccessControlManager, userID string) error {
	entry := &AccessControlEntry{
		EntityType:  "user",
		EntityID:    userID,
		Resources:   []*Resource{ResourcePattern(AccessResourceTypeTopic, "*")},
		Permissions: []Permission{PermissionPublish},
		Effect:      "allow",
	}
	return acm.AddAccessControlEntry(ctx, entry)
}
