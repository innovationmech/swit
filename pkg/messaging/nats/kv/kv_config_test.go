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
