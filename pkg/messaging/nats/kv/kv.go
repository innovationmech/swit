// Copyright © 2025 SWIT
//
// NATS Key-Value helpers: thin utilities wrapping github.com/nats-io/nats.go
// KeyValue/JetStreamContext to provide a simple way to create buckets, and
// perform get/put/watch operations for distributed config/state.

package natskv

import (
	"context"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

// JetStream defines the minimal subset of nats.JetStreamContext used by helpers.
// This enables test doubles without requiring a running NATS server.
type JetStream interface {
	CreateKeyValue(cfg *nats.KeyValueConfig) (nats.KeyValue, error)
	KeyValue(bucket string) (nats.KeyValue, error)
}

// Watcher defines the minimal subset of nats.KeyWatcher used by helpers.
type Watcher interface {
	Updates() <-chan *nats.KeyValueEntry
	Stop() error
}

// BucketConfig contains the parameters to create or ensure a KV bucket exists.
type BucketConfig struct {
	// Name is required and maps to KeyValueConfig.Bucket.
	Name string

	// Description is an optional human-readable description.
	Description string

	// TTL controls time-to-live for entries in this bucket.
	TTL time.Duration

	// History controls the number of historical values to keep per key.
	History int

	// MaxValueSize limits value size in bytes (0 means unlimited as per server limits).
	MaxValueSize int32

	// MaxBucketBytes limits total bucket size in bytes (0 means unlimited).
	MaxBucketBytes int64

	// Storage controls storage backend: "file" (default) or "memory".
	Storage string

	// Replicas sets the replication factor for the underlying stream (JetStream only).
	Replicas int
}

// Normalize fills defaults and bounds for the configuration.
func (c *BucketConfig) Normalize() {
	if c.History < 0 {
		c.History = 0
	}
	if c.History > 64 {
		c.History = 64
	}
	if c.Replicas <= 0 {
		c.Replicas = 1
	}
	if c.Storage == "" {
		c.Storage = "file"
	}
	if c.TTL < 0 {
		c.TTL = 0
	}
}

// toKeyValueConfig translates BucketConfig to nats.KeyValueConfig.
func (c *BucketConfig) toKeyValueConfig() (*nats.KeyValueConfig, error) {
	if c == nil || c.Name == "" {
		return nil, messaging.NewConfigError("kv bucket name is required", nil)
	}
	c.Normalize()

	var storage nats.StorageType
	switch c.Storage {
	case "memory":
		storage = nats.MemoryStorage
	default:
		storage = nats.FileStorage
	}

	return &nats.KeyValueConfig{
		Bucket:       c.Name,
		Description:  c.Description,
		TTL:          c.TTL,
		MaxValueSize: c.MaxValueSize,
		History:      uint8(c.History),
		MaxBytes:     c.MaxBucketBytes,
		Storage:      storage,
		Replicas:     c.Replicas,
	}, nil
}

// EnsureBucket returns an existing bucket or creates it if missing.
// It is safe to call concurrently; the server enforces idempotency.
func EnsureBucket(ctx context.Context, js JetStream, cfg BucketConfig) (nats.KeyValue, error) {
	if js == nil {
		return nil, messaging.NewConfigError("nil JetStream context", nil)
	}

	// Try to open existing bucket first
	kv, err := js.KeyValue(cfg.Name)
	if err == nil && kv != nil {
		return kv, nil
	}

	// Create if not found
	nkvc, cerr := cfg.toKeyValueConfig()
	if cerr != nil {
		return nil, cerr
	}
	kv, err = js.CreateKeyValue(nkvc)
	if err != nil {
		return nil, err
	}
	return kv, nil
}

// OpenBucket opens an existing bucket by name.
func OpenBucket(_ context.Context, js JetStream, name string) (nats.KeyValue, error) {
	if js == nil {
		return nil, messaging.NewConfigError("nil JetStream context", nil)
	}
	if name == "" {
		return nil, messaging.NewConfigError("kv bucket name is required", nil)
	}
	return js.KeyValue(name)
}

// Put stores or replaces the value for the key. Returns the new revision.
func Put(_ context.Context, kv nats.KeyValue, key string, value []byte) (uint64, error) {
	if kv == nil {
		return 0, messaging.NewConfigError("nil KeyValue bucket", nil)
	}
	return kv.Put(key, value)
}

// Create inserts the value only if the key does not already exist. Returns revision.
func Create(_ context.Context, kv nats.KeyValue, key string, value []byte) (uint64, error) {
	if kv == nil {
		return 0, messaging.NewConfigError("nil KeyValue bucket", nil)
	}
	return kv.Create(key, value)
}

// Get returns the value and its revision for a given key.
func Get(_ context.Context, kv nats.KeyValue, key string) ([]byte, uint64, error) {
	if kv == nil {
		return nil, 0, messaging.NewConfigError("nil KeyValue bucket", nil)
	}
	entry, err := kv.Get(key)
	if err != nil {
		return nil, 0, err
	}
	return entry.Value(), entry.Revision(), nil
}

// EntryEvent is a simplified watch event.
type EntryEvent struct {
	Bucket    string
	Key       string
	Value     []byte
	Revision  uint64
	Operation string // "PUT" | "DEL" | "PURGE" | "UNKNOWN"
	Created   time.Time
}

// Watch starts a watcher on a key or pattern (e.g., ">" for all keys).
// includeHistory indicates whether to replay historical values on startup.
// It returns a channel of EntryEvent and a cancel function to stop the watcher.
func Watch(ctx context.Context, kv nats.KeyValue, keyPattern string, includeHistory bool) (<-chan EntryEvent, func() error, error) {
	if kv == nil {
		return nil, nil, messaging.NewConfigError("nil KeyValue bucket", nil)
	}

	var opts []nats.WatchOpt
	if includeHistory {
		opts = append(opts, nats.IncludeHistory())
	}
	// Drop delete notifications by default; callers can derive deletes via missing keys.
	opts = append(opts, nats.IgnoreDeletes())

	w, err := kv.Watch(keyPattern, opts...)
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan EntryEvent, 64)

	// Fan-out updates until canceled.
	go func() {
		defer close(ch)
		updates := w.Updates()
		for {
			select {
			case <-ctx.Done():
				_ = w.Stop()
				return
			case e, ok := <-updates:
				if !ok {
					return
				}
				if e == nil {
					// Initial snapshot barrier
					continue
				}
				ch <- EntryEvent{
					Bucket:    e.Bucket(),
					Key:       e.Key(),
					Value:     e.Value(),
					Revision:  e.Revision(),
					Operation: opToString(e.Operation()),
					Created:   e.Created(),
				}
			}
		}
	}()

	cancel := func() error { return w.Stop() }
	return ch, cancel, nil
}

func opToString(op nats.KeyValueOp) string {
	switch op {
	case nats.KeyValuePut:
		return "PUT"
	case nats.KeyValueDelete:
		return "DEL"
	case nats.KeyValuePurge:
		return "PURGE"
	default:
		return "UNKNOWN"
	}
}
