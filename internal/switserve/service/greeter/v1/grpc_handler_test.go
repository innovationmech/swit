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

package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	commonv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/common/v1"
	greeterv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewGRPCHandler(t *testing.T) {
	tests := []struct {
		name        string
		service     GreeterService
		description string
	}{
		{
			name:        "creates_handler_successfully",
			service:     NewService(),
			description: "Should create a new gRPC handler successfully",
		},
		{
			name:        "creates_handler_with_nil_service",
			service:     nil,
			description: "Should create handler even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewGRPCHandler(tt.service)

			assert.NotNil(t, handler)
			assert.Equal(t, tt.service, handler.service)
		})
	}
}

func TestGRPCHandler_SayHello(t *testing.T) {
	logger.InitLogger()

	tests := []struct {
		name            string
		request         *greeterv1.SayHelloRequest
		expectedMessage string
		expectedLang    string
		expectedStyle   greeterv1.GreetingStyle
		expectError     bool
		errorCode       codes.Code
		description     string
	}{
		{
			name: "success_default_language",
			request: &greeterv1.SayHelloRequest{
				Name: "World",
			},
			expectedMessage: "Hello, World!",
			expectedLang:    "",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should successfully greet with default language",
		},
		{
			name: "success_chinese_language",
			request: &greeterv1.SayHelloRequest{
				Name:     "世界",
				Language: "chinese",
			},
			expectedMessage: "你好, 世界！",
			expectedLang:    "chinese",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should successfully greet in Chinese",
		},
		{
			name: "success_spanish_language",
			request: &greeterv1.SayHelloRequest{
				Name:     "Mundo",
				Language: "spanish",
			},
			expectedMessage: "¡Hola, Mundo!",
			expectedLang:    "spanish",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should successfully greet in Spanish",
		},
		{
			name: "success_with_style",
			request: &greeterv1.SayHelloRequest{
				Name:     "Test",
				Language: "english",
				Style:    greeterv1.GreetingStyle_GREETING_STYLE_FORMAL,
			},
			expectedMessage: "Hello, Test!",
			expectedLang:    "english",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_FORMAL,
			expectError:     false,
			description:     "Should successfully greet with specific style",
		},
		{
			name: "success_with_metadata",
			request: &greeterv1.SayHelloRequest{
				Name:     "TestUser",
				Language: "english",
				Metadata: &commonv1.RequestMetadata{
					RequestId: "test-request-123",
					UserAgent: "test-client",
					ClientIp:  "127.0.0.1",
				},
			},
			expectedMessage: "Hello, TestUser!",
			expectedLang:    "english",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should successfully greet with metadata",
		},
		{
			name: "empty_name",
			request: &greeterv1.SayHelloRequest{
				Name: "",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when name is empty",
		},
		{
			name: "whitespace_name",
			request: &greeterv1.SayHelloRequest{
				Name: "   ",
			},
			expectedMessage: "Hello, !",
			expectedLang:    "",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should handle whitespace name (trimmed by service)",
		},
		{
			name: "unicode_name",
			request: &greeterv1.SayHelloRequest{
				Name: "👋🌍",
			},
			expectedMessage: "Hello, 👋🌍!",
			expectedLang:    "",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should handle unicode characters in name",
		},
		{
			name: "long_name",
			request: &greeterv1.SayHelloRequest{
				Name: "VeryLongNameThatExceedsNormalLengthButShouldStillWork",
			},
			expectedMessage: "Hello, VeryLongNameThatExceedsNormalLengthButShouldStillWork!",
			expectedLang:    "",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should handle long names",
		},
		{
			name: "unknown_language",
			request: &greeterv1.SayHelloRequest{
				Name:     "World",
				Language: "klingon",
			},
			expectedMessage: "Hello, World!",
			expectedLang:    "klingon",
			expectedStyle:   greeterv1.GreetingStyle_GREETING_STYLE_UNSPECIFIED,
			expectError:     false,
			description:     "Should fallback to English for unknown language",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService()
			handler := NewGRPCHandler(service)
			ctx := context.Background()

			response, err := handler.SayHello(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.expectedMessage, response.Message)
				assert.Equal(t, tt.expectedLang, response.Language)
				assert.Equal(t, tt.expectedStyle, response.Style)
				assert.NotNil(t, response.Metadata)
				assert.Equal(t, "swit-serve-1", response.Metadata.ServerId)
				assert.NotEmpty(t, response.Metadata.RequestId)
			}
		})
	}
}

func TestGRPCHandler_SayHello_NilRequest(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	t.Run("nil_request_panics", func(t *testing.T) {
		assert.Panics(t, func() {
			handler.SayHello(ctx, nil)
		})
	})
}

func TestGRPCHandler_SayHello_ServiceError(t *testing.T) {
	logger.InitLogger()

	// Create a mock service that returns an error
	service := &Service{}
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	request := &greeterv1.SayHelloRequest{
		Name: "", // This will cause service to return an error
	}

	response, err := handler.SayHello(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGRPCHandler_SayHello_ContextVariations(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	request := &greeterv1.SayHelloRequest{Name: "TestUser"}

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
			ctx:     context.WithValue(context.Background(), struct{ string }{"key"}, "value"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.SayHello(tt.ctx, request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "Hello, TestUser!", result.Message)
			}
		})
	}
}

func TestGRPCHandler_SayHello_CancelledContext(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	request := &greeterv1.SayHelloRequest{Name: "TestUser"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Note: Our simple implementation doesn't check for context cancellation
	// In a real implementation, you might want to check ctx.Err() in the business logic
	result, err := handler.SayHello(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Hello, TestUser!", result.Message)
}

func TestGRPCHandler_SayHelloStream(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)

	tests := []struct {
		name        string
		request     *greeterv1.SayHelloStreamRequest
		expectError bool
		errorCode   codes.Code
		description string
	}{
		{
			name: "empty_name",
			request: &greeterv1.SayHelloStreamRequest{
				Name: "",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when name is empty",
		},
		{
			name: "valid_name",
			request: &greeterv1.SayHelloStreamRequest{
				Name:  "TestUser",
				Count: 5,
			},
			expectError: true,
			errorCode:   codes.Unimplemented,
			description: "Should return Unimplemented as streaming is not implemented",
		},
		{
			name: "zero_count",
			request: &greeterv1.SayHelloStreamRequest{
				Name:  "TestUser",
				Count: 0,
			},
			expectError: true,
			errorCode:   codes.Unimplemented,
			description: "Should return Unimplemented even with zero count",
		},
		{
			name: "negative_count",
			request: &greeterv1.SayHelloStreamRequest{
				Name:  "TestUser",
				Count: -1,
			},
			expectError: true,
			errorCode:   codes.Unimplemented,
			description: "Should return Unimplemented even with negative count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.SayHelloStream(tt.request, nil)

			if tt.expectError {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRPCHandler_GetRequestID(t *testing.T) {
	tests := []struct {
		name            string
		metadata        *commonv1.RequestMetadata
		expectGenerated bool
		description     string
	}{
		{
			name: "with_request_id",
			metadata: &commonv1.RequestMetadata{
				RequestId: "test-request-id",
			},
			expectGenerated: false,
			description:     "Should use provided request ID",
		},
		{
			name: "without_request_id",
			metadata: &commonv1.RequestMetadata{
				RequestId: "",
			},
			expectGenerated: true,
			description:     "Should generate new request ID when empty",
		},
		{
			name:            "nil_metadata",
			metadata:        nil,
			expectGenerated: true,
			description:     "Should generate new request ID when metadata is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestID := getRequestID(tt.metadata)

			assert.NotEmpty(t, requestID)

			if tt.expectGenerated {
				// Check that generated ID is a valid UUID
				_, err := uuid.Parse(requestID)
				assert.NoError(t, err, "Generated request ID should be a valid UUID")
				// Ensure it's not the hardcoded value
				assert.NotEqual(t, "generated-request-id", requestID)
			} else {
				assert.Equal(t, tt.metadata.RequestId, requestID)
			}
		})
	}
}

func TestGRPCHandler_Interface_Compliance(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)

	t.Run("implements_greeter_service_server", func(t *testing.T) {
		_, ok := interface{}(handler).(greeterv1.GreeterServiceServer)
		assert.True(t, ok, "GRPCHandler should implement greeterv1.GreeterServiceServer interface")
	})
}

func TestGRPCHandler_Response_Structure(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	tests := []struct {
		name    string
		request *greeterv1.SayHelloRequest
	}{
		{
			name: "basic_request",
			request: &greeterv1.SayHelloRequest{
				Name: "Test",
			},
		},
		{
			name: "request_with_language",
			request: &greeterv1.SayHelloRequest{
				Name:     "Test",
				Language: "spanish",
			},
		},
		{
			name: "request_with_style",
			request: &greeterv1.SayHelloRequest{
				Name:  "Test",
				Style: greeterv1.GreetingStyle_GREETING_STYLE_CASUAL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.SayHello(ctx, tt.request)

			assert.NoError(t, err)
			assert.NotNil(t, response, "Response should never be nil")
			assert.NotEmpty(t, response.Message, "Response message should not be empty")
			assert.NotNil(t, response.Metadata, "Response metadata should not be nil")
			assert.NotEmpty(t, response.Metadata.RequestId, "Response metadata request ID should not be empty")
			assert.Equal(t, "swit-serve-1", response.Metadata.ServerId, "Response metadata server ID should be set")
			assert.Equal(t, tt.request.Language, response.Language, "Response language should match request")
			assert.Equal(t, tt.request.Style, response.Style, "Response style should match request")
		})
	}
}

func TestGRPCHandler_Concurrent_Access(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	const numGoroutines = 100
	results := make([]*greeterv1.SayHelloResponse, numGoroutines)
	errors := make([]error, numGoroutines)
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()

			request := &greeterv1.SayHelloRequest{
				Name: "ConcurrentUser" + string(rune('A'+index%26)),
			}

			result, err := handler.SayHello(ctx, request)
			results[index] = result
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
		assert.NotNil(t, result, "Goroutine %d should return non-nil result", i)
		assert.NotEmpty(t, result.Message, "Goroutine %d should return non-empty message", i)
		assert.Contains(t, result.Message, "Hello, ", "Goroutine %d result should contain greeting", i)
	}
}

func TestGRPCHandler_Error_Handling(t *testing.T) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	t.Run("service_error_handling", func(t *testing.T) {
		// Test that service errors are properly converted to gRPC errors
		request := &greeterv1.SayHelloRequest{
			Name: "", // This will cause service to return an error
		}

		response, err := handler.SayHello(ctx, request)

		assert.Error(t, err)
		assert.Nil(t, response)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// Benchmark tests
func BenchmarkGRPCHandler_SayHello(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	request := &greeterv1.SayHelloRequest{Name: "BenchmarkUser"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.SayHello(ctx, request)
	}
}

func BenchmarkGRPCHandler_SayHello_WithMetadata(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	request := &greeterv1.SayHelloRequest{
		Name: "BenchmarkUser",
		Metadata: &commonv1.RequestMetadata{
			RequestId: "bench-request-123",
			UserAgent: "bench-client",
			ClientIp:  "127.0.0.1",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.SayHello(ctx, request)
	}
}

func BenchmarkGRPCHandler_SayHello_DifferentLanguages(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	languages := []string{"english", "chinese", "spanish", "french", "german", "japanese", "korean"}
	requests := make([]*greeterv1.SayHelloRequest, len(languages))
	for i, lang := range languages {
		requests[i] = &greeterv1.SayHelloRequest{
			Name:     "BenchmarkUser",
			Language: lang,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := requests[i%len(requests)]
		_, _ = handler.SayHello(ctx, request)
	}
}

func BenchmarkGRPCHandler_Concurrent(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	request := &greeterv1.SayHelloRequest{Name: "ConcurrentUser"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = handler.SayHello(ctx, request)
		}
	})
}
