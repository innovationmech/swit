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

package messaging

import (
	"context"
	"testing"

	"google.golang.org/protobuf/types/known/emptypb"
)

func TestRegisterProtobufType(t *testing.T) {
	t.Run("nil message returns error", func(t *testing.T) {
		err := RegisterProtobufType(nil)
		if err == nil {
			t.Fatal("RegisterProtobufType(nil) expected error, got nil")
		}
	})

	t.Run("already registered type is idempotent", func(t *testing.T) {
		// Well-known types self-register with the global registry, so
		// registering again must succeed without error.
		if err := RegisterProtobufType(&emptypb.Empty{}); err != nil {
			t.Fatalf("RegisterProtobufType() unexpected error: %v", err)
		}
	})
}

func TestGetRegisteredProtobufTypes(t *testing.T) {
	types := GetRegisteredProtobufTypes()
	if len(types) == 0 {
		t.Fatal("GetRegisteredProtobufTypes() returned empty list; expected globally registered types")
	}

	found := false
	for _, name := range types {
		if name == "google.protobuf.Empty" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetRegisteredProtobufTypes() missing well-known type google.protobuf.Empty")
	}
}

func TestProtobufSerializerGetSchemaDesignBoundary(t *testing.T) {
	serializer := NewProtobufSerializer(nil)
	defer serializer.Close()

	schema, err := serializer.GetSchema(context.Background())
	if err == nil {
		t.Fatal("GetSchema() expected SchemaError by design, got nil")
	}
	if schema != nil {
		t.Errorf("GetSchema() expected nil schema, got %v", schema)
	}
	if _, ok := err.(*SchemaError); !ok {
		t.Errorf("GetSchema() error type = %T, want *SchemaError", err)
	}
}
