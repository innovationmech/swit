package rabbitmq

import (
	"context"
	"fmt"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/adapters"
)

// Adapter implements messaging.MessageBrokerAdapter for RabbitMQ using streadway/amqp.
type Adapter struct {
	*adapters.BaseMessageBrokerAdapter
}

func newAdapter() *Adapter {
	info := &messaging.BrokerAdapterInfo{
		Name:                 "streadway-amqp",
		Version:              "0.1.0",
		Description:          "RabbitMQ adapter built on streadway/amqp",
		SupportedBrokerTypes: []messaging.BrokerType{messaging.BrokerTypeRabbitMQ},
		Author:               "SWIT",
		License:              "MIT",
	}

	caps, _ := messaging.GetCapabilityProfile(messaging.BrokerTypeRabbitMQ)
	defaultCfg := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://guest:guest@localhost:5672/"},
	}

	return &Adapter{BaseMessageBrokerAdapter: adapters.NewBaseMessageBrokerAdapter(info, caps, defaultCfg)}
}

func (a *Adapter) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	if config == nil {
		return nil, messaging.NewConfigError("rabbitmq adapter requires a configuration", nil)
	}

	if res := a.ValidateConfiguration(config); !res.Valid {
		return nil, messaging.NewConfigError("rabbitmq adapter configuration validation failed", nil)
	}

	if config.Type != messaging.BrokerTypeRabbitMQ {
		return nil, messaging.NewConfigError(
			fmt.Sprintf("unsupported broker type for rabbitmq adapter: %s", config.Type),
			nil,
		)
	}

	rabbitCfg, err := ParseConfig(config)
	if err != nil {
		return nil, err
	}

	broker := newRabbitBroker(config, rabbitCfg)
	return broker, nil
}

func (a *Adapter) ValidateConfiguration(config *messaging.BrokerConfig) *messaging.AdapterValidationResult {
	result := a.BaseMessageBrokerAdapter.ValidateConfiguration(config)
	if !result.Valid {
		return result
	}

	if config.Connection.PoolSize <= 0 {
		result.Suggestions = append(result.Suggestions, messaging.AdapterValidationSuggestion{
			Field:          "Connection.PoolSize",
			Message:        "use a positive pool size for RabbitMQ connections",
			SuggestedValue: 2,
		})
	}

	if _, err := ParseConfig(config); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, messaging.AdapterValidationError{
			Field:    "Extra.rabbitmq",
			Message:  err.Error(),
			Code:     "RABBITMQ_CONFIG_INVALID",
			Severity: messaging.AdapterValidationSeverityError,
		})
	}

	return result
}

func (a *Adapter) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	status, err := a.BaseMessageBrokerAdapter.HealthCheck(ctx)
	if status != nil {
		status.Details["library"] = "github.com/streadway/amqp"
	}
	return status, err
}

func init() {
	_ = adapters.RegisterGlobalAdapter(newAdapter())
}
