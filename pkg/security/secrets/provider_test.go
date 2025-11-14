// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secrets

import (
	"testing"
	"time"
)

func TestSecret_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "no expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name: "not expired yet",
			expiresAt: func() *time.Time {
				t := time.Now().Add(1 * time.Hour)
				return &t
			}(),
			want: false,
		},
		{
			name: "already expired",
			expiresAt: func() *time.Time {
				t := time.Now().Add(-1 * time.Hour)
				return &t
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Secret{
				Key:       "test-key",
				Value:     "test-value",
				ExpiresAt: tt.expiresAt,
			}

			if got := s.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name: "valid env provider",
			config: &ProviderConfig{
				Type:    ProviderTypeEnv,
				Enabled: true,
				Env:     &EnvProviderConfig{},
			},
			wantErr: false,
		},
		{
			name: "valid file provider",
			config: &ProviderConfig{
				Type:    ProviderTypeFile,
				Enabled: true,
				File: &FileProviderConfig{
					Path:   "/tmp/secrets.json",
					Format: FileFormatJSON,
				},
			},
			wantErr: false,
		},
		{
			name: "valid vault provider",
			config: &ProviderConfig{
				Type:    ProviderTypeVault,
				Enabled: true,
				Vault: &VaultProviderConfig{
					Address: "https://vault.example.com:8200",
					Token:   "test-token",
					Path:    "secret/data/myapp",
				},
			},
			wantErr: false,
		},
		{
			name: "valid memory provider",
			config: &ProviderConfig{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "missing type",
			config: &ProviderConfig{
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "file provider without config",
			config: &ProviderConfig{
				Type:    ProviderTypeFile,
				Enabled: true,
				File:    nil,
			},
			wantErr: true,
		},
		{
			name: "vault provider without config",
			config: &ProviderConfig{
				Type:    ProviderTypeVault,
				Enabled: true,
				Vault:   nil,
			},
			wantErr: true,
		},
		{
			name: "unsupported provider type",
			config: &ProviderConfig{
				Type:    "unsupported",
				Enabled: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProviderType_Constants(t *testing.T) {
	// Test that provider type constants are defined correctly
	if ProviderTypeEnv != "env" {
		t.Errorf("ProviderTypeEnv = %v, want env", ProviderTypeEnv)
	}

	if ProviderTypeFile != "file" {
		t.Errorf("ProviderTypeFile = %v, want file", ProviderTypeFile)
	}

	if ProviderTypeVault != "vault" {
		t.Errorf("ProviderTypeVault = %v, want vault", ProviderTypeVault)
	}

	if ProviderTypeMemory != "memory" {
		t.Errorf("ProviderTypeMemory = %v, want memory", ProviderTypeMemory)
	}
}

func TestErrors(t *testing.T) {
	// Test that error constants are defined
	if ErrSecretNotFound == nil {
		t.Error("ErrSecretNotFound is nil")
	}

	if ErrProviderNotInitialized == nil {
		t.Error("ErrProviderNotInitialized is nil")
	}

	if ErrInvalidSecretKey == nil {
		t.Error("ErrInvalidSecretKey is nil")
	}

	if ErrOperationNotSupported == nil {
		t.Error("ErrOperationNotSupported is nil")
	}
}
