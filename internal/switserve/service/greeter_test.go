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

package service

import (
	"context"
	"testing"

	"github.com/innovationmech/swit/api/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreeterService_SayHello(t *testing.T) {
	tests := []struct {
		name           string
		request        *pb.HelloRequest
		expectedResult *pb.HelloResponse
		expectError    bool
	}{
		{
			name:           "valid_name",
			request:        &pb.HelloRequest{Name: "World"},
			expectedResult: &pb.HelloResponse{Message: "Hello, World"},
			expectError:    false,
		},
		{
			name:           "empty_name",
			request:        &pb.HelloRequest{Name: ""},
			expectedResult: &pb.HelloResponse{Message: "Hello, "},
			expectError:    false,
		},
		{
			name:           "name_with_spaces",
			request:        &pb.HelloRequest{Name: "John Doe"},
			expectedResult: &pb.HelloResponse{Message: "Hello, John Doe"},
			expectError:    false,
		},
		{
			name:           "name_with_special_characters",
			request:        &pb.HelloRequest{Name: "Alice-123!"},
			expectedResult: &pb.HelloResponse{Message: "Hello, Alice-123!"},
			expectError:    false,
		},
		{
			name:           "unicode_name",
			request:        &pb.HelloRequest{Name: "世界"},
			expectedResult: &pb.HelloResponse{Message: "Hello, 世界"},
			expectError:    false,
		},
		{
			name:           "long_name",
			request:        &pb.HelloRequest{Name: "ThisIsAVeryLongNameThatShouldStillWork"},
			expectedResult: &pb.HelloResponse{Message: "Hello, ThisIsAVeryLongNameThatShouldStillWork"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &GreeterService{}
			ctx := context.Background()

			result, err := service.SayHello(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.Message, result.Message)
			}
		})
	}
}

func TestGreeterService_SayHello_NilRequest(t *testing.T) {
	service := &GreeterService{}
	ctx := context.Background()

	t.Run("nil_request_panics", func(t *testing.T) {
		assert.Panics(t, func() {
			service.SayHello(ctx, nil)
		})
	})
}

func TestGreeterService_SayHello_ContextVariations(t *testing.T) {
	service := &GreeterService{}
	request := &pb.HelloRequest{Name: "TestUser"}

	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "background_context",
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "todo_context",
			ctx:     context.TODO(),
			wantErr: false,
		},
		{
			name:    "context_with_value",
			ctx:     context.WithValue(context.Background(), "key", "value"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.SayHello(tt.ctx, request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "Hello, TestUser", result.Message)
			}
		})
	}
}

func TestGreeterService_SayHello_CancelledContext(t *testing.T) {
	service := &GreeterService{}
	request := &pb.HelloRequest{Name: "TestUser"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := service.SayHello(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Hello, TestUser", result.Message)
}

func TestGreeterService_Implements_Interface(t *testing.T) {
	service := &GreeterService{}

	_, ok := interface{}(service).(pb.GreeterServer)
	assert.True(t, ok, "GreeterService should implement pb.GreeterServer interface")
}

func TestGreeterService_ResponseNotNil(t *testing.T) {
	service := &GreeterService{}
	ctx := context.Background()
	request := &pb.HelloRequest{Name: "Test"}

	result, err := service.SayHello(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result, "Response should never be nil")
	assert.NotEmpty(t, result.Message, "Response message should not be empty")
}

func TestGreeterService_MessageFormat(t *testing.T) {
	service := &GreeterService{}
	ctx := context.Background()

	tests := []struct {
		inputName   string
		expectedMsg string
	}{
		{"", "Hello, "},
		{"a", "Hello, a"},
		{"Alice", "Hello, Alice"},
		{"   ", "Hello,    "},
		{"\n", "Hello, \n"},
		{"\t", "Hello, \t"},
	}

	for _, tt := range tests {
		t.Run("name_"+tt.inputName, func(t *testing.T) {
			request := &pb.HelloRequest{Name: tt.inputName}
			result, err := service.SayHello(ctx, request)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMsg, result.Message)
		})
	}
}

func TestGreeterService_Concurrent(t *testing.T) {
	service := &GreeterService{}
	ctx := context.Background()

	const numGoroutines = 100
	results := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()

			request := &pb.HelloRequest{Name: "User" + string(rune('A'+index%26))}
			result, err := service.SayHello(ctx, request)

			results[index] = ""
			if result != nil {
				results[index] = result.Message
			}
			errors[index] = err
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	for i, err := range errors {
		assert.NoError(t, err, "Goroutine %d should not return error", i)
	}

	for i, result := range results {
		assert.NotEmpty(t, result, "Goroutine %d should return non-empty result", i)
		assert.Contains(t, result, "Hello, ", "Goroutine %d result should contain greeting", i)
	}
}

func BenchmarkGreeterService_SayHello(b *testing.B) {
	service := &GreeterService{}
	ctx := context.Background()
	request := &pb.HelloRequest{Name: "BenchmarkUser"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.SayHello(ctx, request)
	}
}

func BenchmarkGreeterService_SayHello_EmptyName(b *testing.B) {
	service := &GreeterService{}
	ctx := context.Background()
	request := &pb.HelloRequest{Name: ""}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.SayHello(ctx, request)
	}
}

func BenchmarkGreeterService_SayHello_LongName(b *testing.B) {
	service := &GreeterService{}
	ctx := context.Background()
	request := &pb.HelloRequest{Name: "ThisIsAVeryLongUserNameThatMightBeUsedInSomeScenarios"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.SayHello(ctx, request)
	}
}

func BenchmarkGreeterService_SayHello_Parallel(b *testing.B) {
	service := &GreeterService{}
	ctx := context.Background()
	request := &pb.HelloRequest{Name: "ParallelUser"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = service.SayHello(ctx, request)
		}
	})
}
