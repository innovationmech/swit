package adapters_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
	adapters "github.com/innovationmech/swit/pkg/messaging/adapters"
	_ "github.com/innovationmech/swit/pkg/messaging/kafka"
	_ "github.com/innovationmech/swit/pkg/messaging/nats"
	_ "github.com/innovationmech/swit/pkg/messaging/rabbitmq"
)

type adapterFixture struct {
	info    *messaging.BrokerAdapterInfo
	adapter messaging.MessageBrokerAdapter
}

func loadAdapterFixtures(t *testing.T) []adapterFixture {
	t.Helper()

	infos := adapters.ListGlobalAdapters()
	if len(infos) == 0 {
		t.Fatalf("no adapters registered in global registry")
	}

	fixtures := make([]adapterFixture, 0, len(infos))
	for _, info := range infos {
		adapter, err := adapters.GetGlobalAdapter(info.Name)
		if err != nil {
			t.Fatalf("failed to resolve adapter %q: %v", info.Name, err)
		}
		fixtures = append(fixtures, adapterFixture{info: info, adapter: adapter})
	}

	return fixtures
}

func cloneBrokerConfig(cfg *messaging.BrokerConfig) *messaging.BrokerConfig {
	if cfg == nil {
		return nil
	}

	clone := *cfg

	if cfg.Endpoints != nil {
		clone.Endpoints = slices.Clone(cfg.Endpoints)
	}

	if cfg.Authentication != nil {
		authCopy := *cfg.Authentication
		if cfg.Authentication.Scopes != nil {
			authCopy.Scopes = slices.Clone(cfg.Authentication.Scopes)
		}
		clone.Authentication = &authCopy
	}

	if cfg.TLS != nil {
		tlsCopy := *cfg.TLS
		clone.TLS = &tlsCopy
	}

	if cfg.Audit != nil {
		auditCopy := *cfg.Audit
		if cfg.Audit.Events != nil {
			auditCopy.Events = slices.Clone(cfg.Audit.Events)
		}
		clone.Audit = &auditCopy
	}

	if cfg.Extra != nil {
		clone.Extra = maps.Clone(cfg.Extra)
	}

	return &clone
}

func TestRegisteredAdaptersContract(t *testing.T) {
	fixtures := loadAdapterFixtures(t)

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.info.Name, func(t *testing.T) {
			adapterInfo := fx.info
			adapter := fx.adapter

			if adapterInfo.Name == "" {
				t.Fatal("adapter info should include a name")
			}

			if len(adapterInfo.SupportedBrokerTypes) == 0 {
				t.Fatal("adapter should declare supported broker types")
			}

			capabilities := adapter.GetCapabilities()
			if capabilities == nil {
				t.Fatal("adapter capabilities must not be nil")
			}

			defaultCfg := adapter.GetDefaultConfiguration()
			if defaultCfg == nil {
				t.Fatal("adapter should provide default configuration")
			}

			validCfg := cloneBrokerConfig(defaultCfg)
			validCfg.Type = adapterInfo.SupportedBrokerTypes[0]
			if len(validCfg.Endpoints) == 0 {
				t.Fatal("default configuration must define at least one endpoint")
			}
			res := adapter.ValidateConfiguration(validCfg)
			if res == nil || !res.Valid {
				t.Fatalf("expected default configuration to be valid, got result: %#v", res)
			}

			broker, err := adapter.CreateBroker(validCfg)
			if err != nil {
				t.Fatalf("CreateBroker should succeed with default config: %v", err)
			}
			if broker == nil {
				t.Fatal("CreateBroker returned nil broker")
			}

			if _, err := adapter.CreateBroker(nil); err == nil {
				t.Fatal("CreateBroker should fail with nil configuration")
			}

			invalidCfg := cloneBrokerConfig(validCfg)
			invalidCfg.Type = messaging.BrokerType("unsupported")
			invalidResult := adapter.ValidateConfiguration(invalidCfg)
			if invalidResult == nil || invalidResult.Valid {
				t.Fatal("expected unsupported broker type to be rejected during validation")
			}

			nilValidation := adapter.ValidateConfiguration(nil)
			if nilValidation == nil || nilValidation.Valid {
				t.Fatal("ValidateConfiguration should mark nil config as invalid")
			}

			status, err := adapter.HealthCheck(context.Background())
			if err != nil {
				t.Fatalf("HealthCheck should not error: %v", err)
			}
			if status == nil {
				t.Fatal("HealthCheck must return a status")
			}
		})
	}
}
