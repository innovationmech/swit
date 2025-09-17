# Adapter Switching Example

This example shows how to drive broker selection through configuration and how to use
`messaging.PlanBrokerSwitch` to evaluate the semantic impact before flipping the switch.

## Usage

```bash
cd examples/messaging/adapter-switch
# Optional: inspect the existing switch plan (RabbitMQ -> Kafka)
go run .

# Evaluate a different target without editing the file
SWIT_TARGET_BROKER=rabbitmq go run .
```

The program loads `config.yaml`, determines the current and target adapters, and prints a
migration checklist plus feature deltas so that you can update your services confidently.

To switch adapters permanently, edit `config.yaml` and update the `current` property or
provide the `SWIT_TARGET_BROKER` environment variable in your deployment manifests.
