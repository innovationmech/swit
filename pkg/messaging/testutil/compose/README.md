# Messaging Integration Harness

The compose harness provides a reusable Docker Compose stack for messaging integration tests. It starts Kafka, RabbitMQ, and NATS (JetStream) instances with ports exposed on the host machine so Go tests can connect without additional setup.

## Usage

```go
h := compose.NewHarness()
ctx := context.Background()

if err := h.Start(ctx); err != nil {
    t.Fatalf("failed to start messaging stack: %v", err)
}
defer h.Stop(ctx)

endpoints := h.Endpoints()
// Use endpoints.Kafka, endpoints.Rabbit, endpoints.NATS in your tests.
```

`Start` automatically waits for each broker to become reachable using lightweight readiness probes. You can override the compose file, project name, or exposed endpoints with the supplied options.

```go
h := compose.NewHarness(
    compose.WithProjectName("messaging-ci"),
    compose.WithServices("kafka", "nats"),
)
```

To run the stack manually, execute:

```bash
docker compose -f pkg/messaging/testutil/compose/docker-compose.messaging.yml up -d
```

All services listen on the following host ports by default:

| Service  | Port |
|----------|------|
| Kafka    | 19092 |
| RabbitMQ | 5672 (AMQP) / 15672 (management) |
| NATS     | 4222 (client) / 8222 (monitoring) |

The harness requires Docker with Compose v2 or the legacy `docker-compose` binary available in `PATH`.
