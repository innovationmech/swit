package messaging

import (
	"testing"
	"time"
)

func TestComplianceRetentionValidate(t *testing.T) {
	cfg := &ComplianceRetentionConfig{Enabled: true}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error when default period is not set")
	}

	cfg.DefaultPeriod = 24 * time.Hour
	cfg.TopicOverrides = map[string]time.Duration{
		"orders": 48 * time.Hour,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComplianceResidencyValidate(t *testing.T) {
	r := &ComplianceResidencyConfig{Enforce: true}
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error when enforce=true but no regions")
	}

	r.AllowedRegions = []string{"eu-west-1"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComplianceReportingValidate(t *testing.T) {
	r := &ComplianceReportingConfig{Enabled: true}
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error when enabled with zero interval")
	}
	r.Interval = time.Minute
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComplianceConfigValidateAndDefaults(t *testing.T) {
	c := &ComplianceConfig{Enabled: true,
		Retention: &ComplianceRetentionConfig{Enabled: true, DefaultPeriod: 24 * time.Hour},
		Redaction: &ComplianceRedactionConfig{Enabled: true, Fields: []string{"authorization"}},
		Residency: &ComplianceResidencyConfig{Enforce: true, AllowedRegions: []string{"us-east-1"}},
		Reporting: &ComplianceReportingConfig{Enabled: true, Interval: time.Minute},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.SetDefaults()
}

func TestBrokerConfig_ComplianceIntegration(t *testing.T) {
	b := &BrokerConfig{
		Type:      BrokerType("inmemory"),
		Endpoints: []string{"inmemory"},
		Compliance: &ComplianceConfig{Enabled: true,
			Retention: &ComplianceRetentionConfig{Enabled: true, DefaultPeriod: time.Hour},
			Reporting: &ComplianceReportingConfig{Enabled: true, Interval: time.Minute},
		},
		Connection: ConnectionConfig{Timeout: time.Second, KeepAlive: time.Second, MaxAttempts: 1, PoolSize: 1, IdleTimeout: time.Second},
		Retry:      RetryConfig{MaxAttempts: 1, InitialDelay: time.Millisecond * 10, MaxDelay: time.Millisecond * 20, Multiplier: 2, Jitter: 0.1},
		Monitoring: MonitoringConfig{Enabled: true, MetricsInterval: time.Second, HealthCheckInterval: time.Second, HealthCheckTimeout: time.Second},
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b.SetDefaults()
}
