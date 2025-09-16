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

package kafka

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/innovationmech/swit/pkg/messaging"
)

func TestSchemaSerDeAvroRoundTrip(t *testing.T) {
	schemaStr := `{"type":"record","name":"User","fields":[{"name":"id","type":"string"}]}`
	codec, err := goavro.NewCodec(schemaStr)
	if err != nil {
		t.Fatalf("failed to create codec: %v", err)
	}

	srSchema, err := srclient.NewSchema(10, schemaStr, srclient.Avro, 1, nil, codec, nil)
	if err != nil {
		t.Fatalf("NewSchema: %v", err)
	}

	descriptor := &schemaDescriptor{
		subject: "users-value",
		version: 1,
		id:      10,
		format:  messaging.FormatAvro,
		schema:  srSchema,
	}

	registry := &schemaRegistryClient{
		subjectCache: map[string]*schemaDescriptor{cacheKey("users-value", "latest"): descriptor},
		idCache:      map[int]*schemaDescriptor{10: descriptor},
	}

	serde := &schemaSerDe{
		format:   messaging.FormatAvro,
		registry: registry,
		config: &messaging.SerializationOptions{
			Format:         messaging.FormatAvro,
			SchemaRegistry: &messaging.SchemaRegistryConfig{Subject: "users-value"},
		},
	}

	msg := &messaging.Message{ID: "1", Payload: []byte(`{"id":"abc"}`)}
	encoded, err := serde.Encode(context.Background(), msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if !isConfluentWireFormat(encoded.Payload) {
		t.Fatalf("expected confluent wire format")
	}

	schemaID := binary.BigEndian.Uint32(encoded.Payload[1:5])
	if schemaID != 10 {
		t.Fatalf("unexpected schema id %d", schemaID)
	}

	decoded, err := serde.Decode(context.Background(), encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if string(decoded.Payload) != `{"id":"abc"}` {
		t.Fatalf("unexpected decoded payload: %s", decoded.Payload)
	}

	if decoded.Headers[schemaHeaderID] != "10" {
		t.Fatalf("missing schema header, got %v", decoded.Headers)
	}
}

func TestSchemaSerDeJSONRoundTrip(t *testing.T) {
	schemaStr := `{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}`
	validator, err := jsonschema.CompileString("schema.json", schemaStr)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	srSchema, err := srclient.NewSchema(5, schemaStr, srclient.Json, 2, nil, nil, validator)
	if err != nil {
		t.Fatalf("NewSchema: %v", err)
	}

	descriptor := &schemaDescriptor{
		subject: "users-json",
		version: 2,
		id:      5,
		format:  messaging.FormatJSON,
		schema:  srSchema,
	}

	registry := &schemaRegistryClient{
		subjectCache: map[string]*schemaDescriptor{cacheKey("users-json", "latest"): descriptor},
		idCache:      map[int]*schemaDescriptor{5: descriptor},
	}

	serde := &schemaSerDe{
		format:   messaging.FormatJSON,
		registry: registry,
		config: &messaging.SerializationOptions{
			Format:         messaging.FormatJSON,
			SchemaRegistry: &messaging.SchemaRegistryConfig{Subject: "users-json"},
		},
	}

	msg := &messaging.Message{ID: "1", Payload: []byte(`{"id":"42"}`)}
	encoded, err := serde.Encode(context.Background(), msg)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	if !isConfluentWireFormat(encoded.Payload) {
		t.Fatalf("expected wire format prefix")
	}

	decoded, err := serde.Decode(context.Background(), encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if string(decoded.Payload) != `{"id":"42"}` {
		t.Fatalf("unexpected decoded payload: %s", decoded.Payload)
	}

	if decoded.Headers[schemaHeaderVersion] != "2" {
		t.Fatalf("expected version header, got %v", decoded.Headers)
	}
}
