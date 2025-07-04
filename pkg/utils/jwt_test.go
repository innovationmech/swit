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

package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/innovationmech/swit/internal/switauth/config"
)

func TestGenerateAccessToken(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "正常用户ID",
			userID:  "user123",
			wantErr: false,
		},
		{
			name:    "空用户ID",
			userID:  "",
			wantErr: false,
		},
		{
			name:    "长用户ID",
			userID:  "very_long_user_id_with_special_characters_123456789",
			wantErr: false,
		},
		{
			name:    "特殊字符用户ID",
			userID:  "user@example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := GenerateAccessToken(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Errorf("GenerateAccessToken() 返回空令牌")
				}
				if expiresAt.Before(time.Now()) {
					t.Errorf("GenerateAccessToken() 过期时间应该在未来")
				}
				// 验证令牌过期时间大约是15分钟后
				expectedExpiry := time.Now().Add(time.Minute * 15)
				if expiresAt.Sub(expectedExpiry) > time.Minute || expectedExpiry.Sub(expiresAt) > time.Minute {
					t.Errorf("GenerateAccessToken() 过期时间不正确，期望大约15分钟后")
				}
			}
		})
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "正常用户ID",
			userID:  "user123",
			wantErr: false,
		},
		{
			name:    "空用户ID",
			userID:  "",
			wantErr: false,
		},
		{
			name:    "长用户ID",
			userID:  "very_long_user_id_with_special_characters_123456789",
			wantErr: false,
		},
		{
			name:    "特殊字符用户ID",
			userID:  "user@example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := GenerateRefreshToken(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Errorf("GenerateRefreshToken() 返回空令牌")
				}
				if expiresAt.Before(time.Now()) {
					t.Errorf("GenerateRefreshToken() 过期时间应该在未来")
				}
				// 验证令牌过期时间大约是72小时后
				expectedExpiry := time.Now().Add(time.Hour * 72)
				if expiresAt.Sub(expectedExpiry) > time.Minute || expectedExpiry.Sub(expiresAt) > time.Minute {
					t.Errorf("GenerateRefreshToken() 过期时间不正确，期望大约72小时后")
				}
			}
		})
	}
}

func TestValidateAccessToken(t *testing.T) {
	// 生成测试用的有效令牌
	validUserID := "test_user"
	validToken, _, err := GenerateAccessToken(validUserID)
	if err != nil {
		t.Fatalf("生成测试令牌失败: %v", err)
	}

	// 生成一个刷新令牌（类型不匹配）
	refreshToken, _, err := GenerateRefreshToken(validUserID)
	if err != nil {
		t.Fatalf("生成刷新令牌失败: %v", err)
	}

	// 生成一个过期的令牌
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": validUserID,
		"exp":     time.Now().Add(-time.Hour).Unix(), // 已过期
		"type":    "access",
	})
	expiredTokenString, _ := expiredToken.SignedString(config.JwtSecret)

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
		wantUserID  string
	}{
		{
			name:        "有效的访问令牌",
			tokenString: validToken,
			wantErr:     false,
			wantUserID:  validUserID,
		},
		{
			name:        "无效的令牌字符串",
			tokenString: "invalid.token.string",
			wantErr:     true,
		},
		{
			name:        "空令牌字符串",
			tokenString: "",
			wantErr:     true,
		},
		{
			name:        "过期的令牌",
			tokenString: expiredTokenString,
			wantErr:     true,
		},
		{
			name:        "错误的令牌类型（刷新令牌）",
			tokenString: refreshToken,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateAccessToken(tt.tokenString)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims == nil {
					t.Errorf("ValidateAccessToken() 返回空claims")
					return
				}
				if userID, ok := claims["user_id"].(string); !ok || userID != tt.wantUserID {
					t.Errorf("ValidateAccessToken() 用户ID = %v, want %v", userID, tt.wantUserID)
				}
				if tokenType, ok := claims["type"].(string); !ok || tokenType != "access" {
					t.Errorf("ValidateAccessToken() 令牌类型 = %v, want access", tokenType)
				}
			}
		})
	}
}

func TestValidateRefreshToken(t *testing.T) {
	// 生成测试用的有效刷新令牌
	validUserID := "test_user"
	validToken, _, err := GenerateRefreshToken(validUserID)
	if err != nil {
		t.Fatalf("生成测试令牌失败: %v", err)
	}

	// 生成一个访问令牌（类型不匹配）
	accessToken, _, err := GenerateAccessToken(validUserID)
	if err != nil {
		t.Fatalf("生成访问令牌失败: %v", err)
	}

	// 生成一个过期的刷新令牌
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": validUserID,
		"exp":     time.Now().Add(-time.Hour).Unix(), // 已过期
		"type":    "refresh",
	})
	expiredTokenString, _ := expiredToken.SignedString(config.JwtSecret)

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
		wantUserID  string
	}{
		{
			name:        "有效的刷新令牌",
			tokenString: validToken,
			wantErr:     false,
			wantUserID:  validUserID,
		},
		{
			name:        "无效的令牌字符串",
			tokenString: "invalid.token.string",
			wantErr:     true,
		},
		{
			name:        "空令牌字符串",
			tokenString: "",
			wantErr:     true,
		},
		{
			name:        "过期的令牌",
			tokenString: expiredTokenString,
			wantErr:     true,
		},
		{
			name:        "错误的令牌类型（访问令牌）",
			tokenString: accessToken,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateRefreshToken(tt.tokenString)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims == nil {
					t.Errorf("ValidateRefreshToken() 返回空claims")
					return
				}
				if userID, ok := claims["user_id"].(string); !ok || userID != tt.wantUserID {
					t.Errorf("ValidateRefreshToken() 用户ID = %v, want %v", userID, tt.wantUserID)
				}
				if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
					t.Errorf("ValidateRefreshToken() 令牌类型 = %v, want refresh", tokenType)
				}
			}
		})
	}
}

func TestTokenGenerateAndValidateCombined(t *testing.T) {
	testUserIDs := []string{
		"user123",
		"",
		"user@example.com",
		"很长的用户ID_with_special_characters_123456789",
	}

	for _, userID := range testUserIDs {
		t.Run("用户ID: "+userID, func(t *testing.T) {
			// 测试访问令牌
			accessToken, _, err := GenerateAccessToken(userID)
			if err != nil {
				t.Errorf("GenerateAccessToken() 失败: %v", err)
				return
			}

			claims, err := ValidateAccessToken(accessToken)
			if err != nil {
				t.Errorf("ValidateAccessToken() 失败: %v", err)
				return
			}

			if claims["user_id"] != userID {
				t.Errorf("访问令牌用户ID不匹配: got %v, want %v", claims["user_id"], userID)
			}

			// 测试刷新令牌
			refreshToken, _, err := GenerateRefreshToken(userID)
			if err != nil {
				t.Errorf("GenerateRefreshToken() 失败: %v", err)
				return
			}

			claims, err = ValidateRefreshToken(refreshToken)
			if err != nil {
				t.Errorf("ValidateRefreshToken() 失败: %v", err)
				return
			}

			if claims["user_id"] != userID {
				t.Errorf("刷新令牌用户ID不匹配: got %v, want %v", claims["user_id"], userID)
			}
		})
	}
}

func TestTokenWithDifferentSecret(t *testing.T) {
	// 使用不同的密钥生成令牌
	differentSecret := []byte("different-secret-key")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test_user",
		"exp":     time.Now().Add(time.Hour).Unix(),
		"type":    "access",
	})
	tokenString, err := token.SignedString(differentSecret)
	if err != nil {
		t.Fatalf("生成令牌失败: %v", err)
	}

	// 使用默认密钥验证应该失败
	_, err = ValidateAccessToken(tokenString)
	if err == nil {
		t.Errorf("ValidateAccessToken() 应该失败，因为使用了不同的密钥")
	}
}

func TestTokenWithoutRequiredClaims(t *testing.T) {
	// 生成没有type字段的令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test_user",
		"exp":     time.Now().Add(time.Hour).Unix(),
		// 缺少 type 字段
	})
	tokenString, err := token.SignedString(config.JwtSecret)
	if err != nil {
		t.Fatalf("生成令牌失败: %v", err)
	}

	// 验证应该失败
	_, err = ValidateAccessToken(tokenString)
	if err == nil {
		t.Errorf("ValidateAccessToken() 应该失败，因为缺少type字段")
	}
}
