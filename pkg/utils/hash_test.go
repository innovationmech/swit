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

package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "普通密码",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "空密码",
			password: "",
			wantErr:  false,
		},
		{
			name:     "长密码",
			password: "this_is_a_very_long_password_with_special_characters_!@#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "特殊字符密码",
			password: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr:  false,
		},
		{
			name:     "中文密码",
			password: "中文密码测试",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if hash == "" {
					t.Errorf("HashPassword() 返回空哈希值")
				}
				if hash == tt.password {
					t.Errorf("HashPassword() 返回的哈希值不应该等于原密码")
				}
			}
		})
	}
}

func TestCheckPasswordHash(t *testing.T) {
	// 预先生成一些测试用的哈希值
	password1 := "password123"
	hash1, _ := HashPassword(password1)

	password2 := "another_password"
	hash2, _ := HashPassword(password2)

	emptyPassword := ""
	emptyHash, _ := HashPassword(emptyPassword)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "正确密码匹配",
			password: password1,
			hash:     hash1,
			want:     true,
		},
		{
			name:     "错误密码不匹配",
			password: "wrong_password",
			hash:     hash1,
			want:     false,
		},
		{
			name:     "空密码匹配",
			password: emptyPassword,
			hash:     emptyHash,
			want:     true,
		},
		{
			name:     "空密码不匹配非空哈希",
			password: "",
			hash:     hash1,
			want:     false,
		},
		{
			name:     "非空密码不匹配空哈希",
			password: password1,
			hash:     emptyHash,
			want:     false,
		},
		{
			name:     "不同密码的哈希不匹配",
			password: password1,
			hash:     hash2,
			want:     false,
		},
		{
			name:     "无效哈希格式",
			password: password1,
			hash:     "invalid_hash",
			want:     false,
		},
		{
			name:     "空哈希",
			password: password1,
			hash:     "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPasswordHash(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("CheckPasswordHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	password := "test_password"

	// 多次哈希同一个密码，应该产生不同的哈希值（因为使用了随机盐）
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Errorf("HashPassword() 返回错误: err1=%v, err2=%v", err1, err2)
	}

	if hash1 == hash2 {
		t.Errorf("相同密码的多次哈希应该产生不同的哈希值")
	}

	// 但是两个哈希值都应该能够验证原密码
	if !CheckPasswordHash(password, hash1) {
		t.Errorf("第一个哈希值无法验证原密码")
	}

	if !CheckPasswordHash(password, hash2) {
		t.Errorf("第二个哈希值无法验证原密码")
	}
}

func TestHashPasswordAndCheckCombined(t *testing.T) {
	testPasswords := []string{
		"simple",
		"complex_password_123!@#",
		"",
		"中文密码",
		"emoji🔐password",
	}

	for _, password := range testPasswords {
		t.Run("密码: "+password, func(t *testing.T) {
			// 哈希密码
			hash, err := HashPassword(password)
			if err != nil {
				t.Errorf("HashPassword() 返回错误: %v", err)
				return
			}

			// 验证正确密码
			if !CheckPasswordHash(password, hash) {
				t.Errorf("正确密码验证失败")
			}

			// 验证错误密码
			wrongPassword := password + "wrong"
			if CheckPasswordHash(wrongPassword, hash) {
				t.Errorf("错误密码验证成功，应该失败")
			}
		})
	}
}
