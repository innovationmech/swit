package natskv

import (
	"testing"
	"time"
)

func TestBucketConfigNormalizeAndTranslate(t *testing.T) {
	cfg := BucketConfig{
		Name:           "app.cfg",
		Description:    "application config",
		TTL:            -1,
		History:        200, // will be capped
		MaxValueSize:   1024,
		MaxBucketBytes: 10 * 1024,
		Storage:        "",
		Replicas:       0,
	}

	cfg.Normalize()
	if cfg.TTL != 0 {
		t.Fatalf("TTL should be normalized to 0 when negative, got %v", cfg.TTL)
	}
	if cfg.History > 64 {
		t.Fatalf("History should be capped <=64, got %d", cfg.History)
	}
	if cfg.Storage != "file" {
		t.Fatalf("default storage should be file, got %s", cfg.Storage)
	}
	if cfg.Replicas != 1 {
		t.Fatalf("replicas should default to 1, got %d", cfg.Replicas)
	}

	nkv, err := cfg.toKeyValueConfig()
	if err != nil {
		t.Fatalf("toKeyValueConfig error: %v", err)
	}
	if nkv.Bucket != "app.cfg" {
		t.Fatalf("bucket mismatch: %s", nkv.Bucket)
	}
	if nkv.Description != "application config" {
		t.Fatalf("desc mismatch: %s", nkv.Description)
	}
	if nkv.TTL != time.Duration(0) {
		t.Fatalf("ttl mismatch: %v", nkv.TTL)
	}
	if nkv.MaxValueSize != 1024 {
		t.Fatalf("max value size mismatch: %d", nkv.MaxValueSize)
	}
	if nkv.MaxBytes != 10*1024 {
		t.Fatalf("max bytes mismatch: %d", nkv.MaxBytes)
	}
	if nkv.Replicas != 1 {
		t.Fatalf("replicas mismatch: %d", nkv.Replicas)
	}
}
