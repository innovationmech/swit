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

package nats

import (
	"context"
	"fmt"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/adapters"
)

// Adapter implements messaging.MessageBrokerAdapter for NATS using nats-io/nats.go.
type Adapter struct {
	*adapters.BaseMessageBrokerAdapter
}

func newAdapter() *Adapter {
	info := &messaging.BrokerAdapterInfo{
		Name:                 "nats-go",
		Version:              "0.1.0",
		Description:          "NATS adapter built on nats-io/nats.go",
		SupportedBrokerTypes: []messaging.BrokerType{messaging.BrokerTypeNATS},
		Author:               "SWIT",
		License:              "MIT",
	}

	caps, _ := messaging.GetCapabilityProfile(messaging.BrokerTypeNATS)
	defaultCfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeNATS,
		Endpoints: []string{"nats://127.0.0.1:4222"},
	}

	return &Adapter{BaseMessageBrokerAdapter: adapters.NewBaseMessageBrokerAdapter(info, caps, defaultCfg)}
}

// CreateBroker implements MessageBrokerAdapter.CreateBroker.
func (a *Adapter) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	if config == nil {
		return nil, messaging.NewConfigError("nats adapter requires a configuration", nil)
	}

	if res := a.ValidateConfiguration(config); !res.Valid {
		return nil, messaging.NewConfigError("nats adapter configuration validation failed", nil)
	}

	if config.Type != messaging.BrokerTypeNATS {
		return nil, messaging.NewConfigError(
			fmt.Sprintf("unsupported broker type for nats adapter: %s", config.Type),
			nil,
		)
	}

	natsCfg, err := ParseConfig(config)
	if err != nil {
		return nil, err
	}

	broker := newNATSBroker(config, natsCfg)
	return broker, nil
}

// ValidateConfiguration augments base validation with NATS specifics.
func (a *Adapter) ValidateConfiguration(config *messaging.BrokerConfig) *messaging.AdapterValidationResult {
	result := a.BaseMessageBrokerAdapter.ValidateConfiguration(config)
	if !result.Valid {
		return result
	}

	// Parse Extra if present
	ncfg, err := ParseConfig(config)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, messaging.AdapterValidationError{
			Field:    "Extra.nats",
			Message:  err.Error(),
			Code:     "NATS_CONFIG_INVALID",
			Severity: messaging.AdapterValidationSeverityError,
		})
		return result
	}

	// Endpoint/TLS semantics
	if config != nil {
		if errs, warns, suggs := validateNATSConfiguration(config, ncfg); len(errs)+len(warns)+len(suggs) > 0 {
			if len(errs) > 0 {
				result.Valid = false
				result.Errors = append(result.Errors, errs...)
			}
			if len(warns) > 0 {
				result.Warnings = append(result.Warnings, warns...)
			}
			if len(suggs) > 0 {
				result.Suggestions = append(result.Suggestions, suggs...)
			}
		}
	}

	// JetStream semantic validation
	if config != nil && ncfg != nil && ncfg.JetStream != nil {
		if errs, warns, suggs := validateJetStream(ncfg.JetStream); len(errs)+len(warns)+len(suggs) > 0 {
			if len(errs) > 0 {
				result.Valid = false
				result.Errors = append(result.Errors, errs...)
			}
			if len(warns) > 0 {
				result.Warnings = append(result.Warnings, warns...)
			}
			if len(suggs) > 0 {
				result.Suggestions = append(result.Suggestions, suggs...)
			}
		}
	}

	return result
}

// HealthCheck validates adapter wiring (not broker connectivity).
func (a *Adapter) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	status, err := a.BaseMessageBrokerAdapter.HealthCheck(ctx)
	if status != nil {
		status.Details["library"] = "github.com/nats-io/nats.go"
	}
	return status, err
}

func init() {
	_ = adapters.RegisterGlobalAdapter(newAdapter())
}
