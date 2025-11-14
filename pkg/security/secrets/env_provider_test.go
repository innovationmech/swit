// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secrets

import (
	"context"
	"os"
	"testing"
)

func TestEnvProvider_GetSecret(t *testing.T) {
	// Set up test environment variables
	_ = os.Setenv("TEST_SECRET", "test-value")
	_ = os.Setenv("APP_DB_PASSWORD", "db-password")
	defer func() {
		_ = os.Unsetenv("TEST_SECRET")
		_ = os.Unsetenv("APP_DB_PASSWORD")
	}()

	tests := []struct {
		name      string
		config    *EnvProviderConfig
		key       string
		wantValue string
		wantErr   error
	}{
		{
			name:      "get existing secret without prefix",
			config:    &EnvProviderConfig{},
			key:       "TEST_SECRET",
			wantValue: "test-value",
			wantErr:   nil,
		},
		{
			name:      "get existing secret with prefix",
			config:    &EnvProviderConfig{Prefix: "APP_"},
			key:       "DB_PASSWORD",
			wantValue: "db-password",
			wantErr:   nil,
		},
		{
			name:      "get non-existent secret",
			config:    &EnvProviderConfig{},
			key:       "NON_EXISTENT",
			wantValue: "",
			wantErr:   ErrSecretNotFound,
		},
		{
			name:      "get secret with empty key",
			config:    &EnvProviderConfig{},
			key:       "",
			wantValue: "",
			wantErr:   ErrInvalidSecretKey,
		},
		{
			name:      "get secret in allow list",
			config:    &EnvProviderConfig{AllowList: []string{"TEST_SECRET"}},
			key:       "TEST_SECRET",
			wantValue: "test-value",
			wantErr:   nil,
		},
		{
			name:      "get secret not in allow list",
			config:    &EnvProviderConfig{AllowList: []string{"OTHER_SECRET"}},
			key:       "TEST_SECRET",
			wantValue: "",
			wantErr:   ErrSecretNotFound,
		},
		{
			name:      "get secret in deny list",
			config:    &EnvProviderConfig{DenyList: []string{"TEST_SECRET"}},
			key:       "TEST_SECRET",
			wantValue: "",
			wantErr:   ErrSecretNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewEnvProvider(tt.config)
			if err != nil {
				t.Fatalf("NewEnvProvider() error = %v", err)
			}

			ctx := context.Background()
			secret, err := provider.GetSecret(ctx, tt.key)

			if err != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if secret.Value != tt.wantValue {
					t.Errorf("GetSecret() value = %v, want %v", secret.Value, tt.wantValue)
				}
				if secret.Key != tt.key {
					t.Errorf("GetSecret() key = %v, want %v", secret.Key, tt.key)
				}
			}
		})
	}
}

func TestEnvProvider_GetSecretValue(t *testing.T) {
	_ = os.Setenv("TEST_VALUE", "my-value")
	defer func() {
		_ = os.Unsetenv("TEST_VALUE")
	}()

	provider, err := NewEnvProvider(&EnvProviderConfig{})
	if err != nil {
		t.Fatalf("NewEnvProvider() error = %v", err)
	}

	ctx := context.Background()
	value, err := provider.GetSecretValue(ctx, "TEST_VALUE")
	if err != nil {
		t.Fatalf("GetSecretValue() error = %v", err)
	}

	if value != "my-value" {
		t.Errorf("GetSecretValue() = %v, want %v", value, "my-value")
	}
}

func TestEnvProvider_SetSecret(t *testing.T) {
	provider, err := NewEnvProvider(&EnvProviderConfig{})
	if err != nil {
		t.Fatalf("NewEnvProvider() error = %v", err)
	}

	ctx := context.Background()
	err = provider.SetSecret(ctx, "TEST_KEY", "test-value")
	if err != ErrOperationNotSupported {
		t.Errorf("SetSecret() error = %v, want %v", err, ErrOperationNotSupported)
	}
}

func TestEnvProvider_DeleteSecret(t *testing.T) {
	provider, err := NewEnvProvider(&EnvProviderConfig{})
	if err != nil {
		t.Fatalf("NewEnvProvider() error = %v", err)
	}

	ctx := context.Background()
	err = provider.DeleteSecret(ctx, "TEST_KEY")
	if err != ErrOperationNotSupported {
		t.Errorf("DeleteSecret() error = %v, want %v", err, ErrOperationNotSupported)
	}
}

func TestEnvProvider_ListSecrets(t *testing.T) {
	// Set up test environment variables
	_ = os.Setenv("APP_SECRET1", "value1")
	_ = os.Setenv("APP_SECRET2", "value2")
	_ = os.Setenv("OTHER_SECRET", "value3")
	defer func() {
		_ = os.Unsetenv("APP_SECRET1")
		_ = os.Unsetenv("APP_SECRET2")
		_ = os.Unsetenv("OTHER_SECRET")
	}()

	tests := []struct {
		name      string
		config    *EnvProviderConfig
		prefix    string
		wantCount int
		wantKeys  []string
	}{
		{
			name:      "list all secrets with prefix",
			config:    &EnvProviderConfig{Prefix: "APP_"},
			prefix:    "",
			wantCount: 2,
			wantKeys:  []string{"SECRET1", "SECRET2"},
		},
		{
			name:      "list secrets with allow list",
			config:    &EnvProviderConfig{AllowList: []string{"APP_SECRET1"}},
			prefix:    "APP_",
			wantCount: 1,
			wantKeys:  []string{"APP_SECRET1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewEnvProvider(tt.config)
			if err != nil {
				t.Fatalf("NewEnvProvider() error = %v", err)
			}

			ctx := context.Background()
			keys, err := provider.ListSecrets(ctx, tt.prefix)
			if err != nil {
				t.Fatalf("ListSecrets() error = %v", err)
			}

			if len(keys) != tt.wantCount {
				t.Errorf("ListSecrets() returned %d keys, want %d", len(keys), tt.wantCount)
			}
		})
	}
}

func TestEnvProvider_IsReadOnly(t *testing.T) {
	provider, err := NewEnvProvider(&EnvProviderConfig{})
	if err != nil {
		t.Fatalf("NewEnvProvider() error = %v", err)
	}

	if !provider.IsReadOnly() {
		t.Errorf("IsReadOnly() = false, want true")
	}
}

func TestEnvProvider_Name(t *testing.T) {
	provider, err := NewEnvProvider(&EnvProviderConfig{})
	if err != nil {
		t.Fatalf("NewEnvProvider() error = %v", err)
	}

	if provider.Name() != string(ProviderTypeEnv) {
		t.Errorf("Name() = %v, want %v", provider.Name(), ProviderTypeEnv)
	}
}
