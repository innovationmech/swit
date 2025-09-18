package compose

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/segmentio/kafka-go"
	amqp "github.com/streadway/amqp"
)

const readinessInterval = 2 * time.Second

var errEmptyEndpoint = errors.New("endpoint must not be empty")

// WaitForKafka blocks until the Kafka endpoint accepts metadata requests.
func WaitForKafka(ctx context.Context, endpoint string) error {
	if strings.TrimSpace(endpoint) == "" {
		return errEmptyEndpoint
	}

	dialer := &kafka.Dialer{
		Timeout:   5 * time.Second,
		DualStack: true,
	}

	check := func(ctx context.Context) error {
		conn, err := dialer.DialContext(ctx, "tcp", endpoint)
		if err != nil {
			return err
		}
		defer conn.Close()

		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		partitions, err := conn.ReadPartitions()
		if err != nil {
			return err
		}
		if len(partitions) == 0 {
			return errors.New("no partitions reported")
		}
		return nil
	}

	return waitUntil(ctx, readinessInterval, check)
}

// WaitForRabbitMQ blocks until an AMQP connection can be established.
func WaitForRabbitMQ(ctx context.Context, uri string) error {
	if strings.TrimSpace(uri) == "" {
		return errEmptyEndpoint
	}

	check := func(context.Context) error {
		conn, err := amqp.DialConfig(uri, amqp.Config{
			Locale:    "en_US",
			Heartbeat: 5 * time.Second,
			Dial:      amqp.DefaultDial(5 * time.Second),
		})
		if err != nil {
			return err
		}
		return conn.Close()
	}

	return waitUntil(ctx, readinessInterval, check)
}

// WaitForNATS blocks until a NATS JetStream server accepts connections.
func WaitForNATS(ctx context.Context, uri string) error {
	if strings.TrimSpace(uri) == "" {
		return errEmptyEndpoint
	}

	check := func(context.Context) error {
		opts := []nats.Option{
			nats.Timeout(3 * time.Second),
			nats.RetryOnFailedConnect(false),
			nats.MaxReconnects(0),
		}
		nc, err := nats.Connect(uri, opts...)
		if err != nil {
			return err
		}
		defer nc.Close()
		return nc.FlushTimeout(2 * time.Second)
	}

	return waitUntil(ctx, readinessInterval, check)
}

func waitUntil(ctx context.Context, interval time.Duration, check func(context.Context) error) error {
	if interval <= 0 {
		interval = time.Second
	}

	var lastErr error
	if err := check(ctx); err == nil {
		return nil
	} else {
		lastErr = err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("readiness check timed out: %w", errors.Join(ctx.Err(), lastErr))
		case <-ticker.C:
			if err := check(ctx); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
	}
}

// WaitForPort is a lightweight helper used by tests to block until a TCP port is reachable.
func WaitForPort(ctx context.Context, address string) error {
	if strings.TrimSpace(address) == "" {
		return errEmptyEndpoint
	}

	check := func(context.Context) error {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return err
		}
		return conn.Close()
	}

	return waitUntil(ctx, readinessInterval, check)
}
