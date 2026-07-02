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

package inmemory

import (
	"context"
	"fmt"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/adapters"
)

// Adapter implements messaging.MessageBrokerAdapter for the in-memory broker.
// Unlike network-backed adapters it does not require endpoints, making it
// suitable as the default broker for local development and CI.
type Adapter struct {
	*adapters.BaseMessageBrokerAdapter
}

func newAdapter() *Adapter {
	info := &messaging.BrokerAdapterInfo{
		Name:                 "inmemory",
		Version:              "0.1.0",
		Description:          "In-process broker for local development and testing",
		SupportedBrokerTypes: []messaging.BrokerType{messaging.BrokerTypeInMemory},
		Author:               "SWIT",
		License:              "MIT",
	}

	caps, _ := messaging.GetCapabilityProfile(messaging.BrokerTypeInMemory)
	defaultCfg := &messaging.BrokerConfig{
		Type: messaging.BrokerTypeInMemory,
	}

	return &Adapter{BaseMessageBrokerAdapter: adapters.NewBaseMessageBrokerAdapter(info, caps, defaultCfg)}
}

// CreateBroker implements MessageBrokerAdapter.CreateBroker.
func (a *Adapter) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	if config == nil {
		return nil, messaging.NewConfigError("inmemory adapter requires a configuration", nil)
	}

	if res := a.ValidateConfiguration(config); !res.Valid {
		return nil, messaging.NewConfigError("inmemory adapter configuration validation failed", nil)
	}

	return New(config), nil
}

// ValidateConfiguration implements MessageBrokerAdapter.ValidateConfiguration.
// The in-memory broker requires no endpoints, so the base endpoint check is
// intentionally replaced with a type-only validation.
func (a *Adapter) ValidateConfiguration(config *messaging.BrokerConfig) *messaging.AdapterValidationResult {
	result := &messaging.AdapterValidationResult{
		Valid:       true,
		Errors:      []messaging.AdapterValidationError{},
		Warnings:    []messaging.AdapterValidationWarning{},
		Suggestions: []messaging.AdapterValidationSuggestion{},
	}

	if config == nil {
		result.Valid = false
		result.Errors = append(result.Errors, messaging.AdapterValidationError{
			Message:  "configuration cannot be nil",
			Code:     "CONFIG_NULL",
			Severity: messaging.AdapterValidationSeverityError,
		})
		return result
	}

	if config.Type != messaging.BrokerTypeInMemory {
		result.Valid = false
		result.Errors = append(result.Errors, messaging.AdapterValidationError{
			Field:    "Type",
			Message:  fmt.Sprintf("unsupported broker type for inmemory adapter: %s", config.Type),
			Code:     "TYPE_UNSUPPORTED",
			Severity: messaging.AdapterValidationSeverityError,
		})
	}

	if len(config.Endpoints) > 0 {
		result.Warnings = append(result.Warnings, messaging.AdapterValidationWarning{
			Field:   "Endpoints",
			Message: "endpoints are ignored by the in-memory broker",
			Code:    "ENDPOINTS_IGNORED",
		})
	}

	return result
}

// HealthCheck implements MessageBrokerAdapter.HealthCheck.
func (a *Adapter) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	status, err := a.BaseMessageBrokerAdapter.HealthCheck(ctx)
	if status != nil {
		status.Details["library"] = "in-process"
	}
	return status, err
}

func init() {
	_ = adapters.RegisterGlobalAdapter(newAdapter())
}
