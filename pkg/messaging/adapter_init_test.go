package messaging_test

// Test-only side-effect imports to ensure adapter registry is connected and
// concrete adapters are registered with the default registry used by the factory.

import (
	_ "github.com/innovationmech/swit/pkg/messaging/adapters"
	_ "github.com/innovationmech/swit/pkg/messaging/kafka"
	_ "github.com/innovationmech/swit/pkg/messaging/nats"
	_ "github.com/innovationmech/swit/pkg/messaging/rabbitmq"
)
