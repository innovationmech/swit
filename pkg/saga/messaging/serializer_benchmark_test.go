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
)

// BenchmarkJSONSagaEventSerializer_Serialize benchmarks JSON serialization.
func BenchmarkJSONSagaEventSerializer_Serialize(b *testing.B) {
	serializer := NewJSONSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(ctx, event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONSagaEventSerializer_Deserialize benchmarks JSON deserialization.
func BenchmarkJSONSagaEventSerializer_Deserialize(b *testing.B) {
	serializer := NewJSONSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	data, err := serializer.Serialize(ctx, event)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Deserialize(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONSagaEventSerializer_RoundTrip benchmarks JSON serialization and deserialization.
func BenchmarkJSONSagaEventSerializer_RoundTrip(b *testing.B) {
	serializer := NewJSONSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := serializer.Serialize(ctx, event)
		if err != nil {
			b.Fatal(err)
		}
		_, err = serializer.Deserialize(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProtobufSagaEventSerializer_Serialize benchmarks Protobuf serialization.
func BenchmarkProtobufSagaEventSerializer_Serialize(b *testing.B) {
	serializer := NewProtobufSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(ctx, event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProtobufSagaEventSerializer_Deserialize benchmarks Protobuf deserialization.
func BenchmarkProtobufSagaEventSerializer_Deserialize(b *testing.B) {
	serializer := NewProtobufSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	data, err := serializer.Serialize(ctx, event)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Deserialize(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProtobufSagaEventSerializer_RoundTrip benchmarks Protobuf serialization and deserialization.
func BenchmarkProtobufSagaEventSerializer_RoundTrip(b *testing.B) {
	serializer := NewProtobufSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := serializer.Serialize(ctx, event)
		if err != nil {
			b.Fatal(err)
		}
		_, err = serializer.Deserialize(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONSagaEventSerializer_SerializeWithError benchmarks JSON serialization with error.
func BenchmarkJSONSagaEventSerializer_SerializeWithError(b *testing.B) {
	serializer := NewJSONSagaEventSerializer()
	event := createTestSagaEventWithError()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(ctx, event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProtobufSagaEventSerializer_SerializeWithError benchmarks Protobuf serialization with error.
func BenchmarkProtobufSagaEventSerializer_SerializeWithError(b *testing.B) {
	serializer := NewProtobufSagaEventSerializer()
	event := createTestSagaEventWithError()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(ctx, event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONSagaEventSerializer_Parallel benchmarks concurrent JSON serialization.
func BenchmarkJSONSagaEventSerializer_Parallel(b *testing.B) {
	serializer := NewJSONSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := serializer.Serialize(ctx, event)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkProtobufSagaEventSerializer_Parallel benchmarks concurrent Protobuf serialization.
func BenchmarkProtobufSagaEventSerializer_Parallel(b *testing.B) {
	serializer := NewProtobufSagaEventSerializer()
	event := createTestSagaEvent()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := serializer.Serialize(ctx, event)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

