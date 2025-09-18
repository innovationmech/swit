package compose

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWaitUntilSucceedsAfterRetries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	attempts := 0
	err := waitUntil(ctx, 10*time.Millisecond, func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errEmptyEndpoint
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestWaitUntilTimesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := waitUntil(ctx, 10*time.Millisecond, func(context.Context) error {
		return errEmptyEndpoint
	})
	if err == nil {
		t.Fatalf("expected error when context expires")
	}
}

func TestWaitForPortSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := WaitForPort(ctx, listener.Addr().String()); err != nil {
		t.Fatalf("expected success waiting for port, got %v", err)
	}
}

func TestWaitForPortDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := WaitForPort(ctx, "127.0.0.1:65535"); err == nil {
		t.Fatalf("expected error when port is unavailable")
	}
}

func TestWaitForKafkaUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := WaitForKafka(ctx, "127.0.0.1:65535"); err == nil {
		t.Fatalf("expected error when kafka endpoint is unreachable")
	}
}

func TestWaitForRabbitMQUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := WaitForRabbitMQ(ctx, "amqp://guest:guest@127.0.0.1:65535/"); err == nil {
		t.Fatalf("expected error when rabbitmq endpoint is unreachable")
	}
}

func TestWaitForNATSUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := WaitForNATS(ctx, "nats://127.0.0.1:65535"); err == nil {
		t.Fatalf("expected error when nats endpoint is unreachable")
	}
}
