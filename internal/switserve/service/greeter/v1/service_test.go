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
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/common/v1"
	greeterv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	"github.com/innovationmech/swit/pkg/logger"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

// setupTest initializes the logger for tests
func setupTest() {
	logger.InitLogger()
}

// Test the business logic service (GreeterService interface)
func TestGreeterService_GenerateGreeting(t *testing.T) {
	setupTest()
	tests := []struct {
		name           string
		inputName      string
		inputLanguage  string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "english_default",
			inputName:      "World",
			inputLanguage:  "",
			expectedResult: "Hello, World!",
			expectError:    false,
		},
		{
			name:           "english_explicit",
			inputName:      "World",
			inputLanguage:  "english",
			expectedResult: "Hello, World!",
			expectError:    false,
		},
		{
			name:           "chinese",
			inputName:      "世界",
			inputLanguage:  "chinese",
			expectedResult: "你好, 世界！",
			expectError:    false,
		},
		{
			name:           "chinese_zh",
			inputName:      "世界",
			inputLanguage:  "zh",
			expectedResult: "你好, 世界！",
			expectError:    false,
		},
		{
			name:           "chinese_native",
			inputName:      "世界",
			inputLanguage:  "中文",
			expectedResult: "你好, 世界！",
			expectError:    false,
		},
		{
			name:           "spanish",
			inputName:      "Mundo",
			inputLanguage:  "spanish",
			expectedResult: "¡Hola, Mundo!",
			expectError:    false,
		},
		{
			name:           "spanish_es",
			inputName:      "Mundo",
			inputLanguage:  "es",
			expectedResult: "¡Hola, Mundo!",
			expectError:    false,
		},
		{
			name:           "spanish_native",
			inputName:      "Mundo",
			inputLanguage:  "español",
			expectedResult: "¡Hola, Mundo!",
			expectError:    false,
		},
		{
			name:           "french",
			inputName:      "Monde",
			inputLanguage:  "french",
			expectedResult: "Bonjour, Monde!",
			expectError:    false,
		},
		{
			name:           "french_fr",
			inputName:      "Monde",
			inputLanguage:  "fr",
			expectedResult: "Bonjour, Monde!",
			expectError:    false,
		},
		{
			name:           "french_native",
			inputName:      "Monde",
			inputLanguage:  "français",
			expectedResult: "Bonjour, Monde!",
			expectError:    false,
		},
		{
			name:           "german",
			inputName:      "Welt",
			inputLanguage:  "german",
			expectedResult: "Hallo, Welt!",
			expectError:    false,
		},
		{
			name:           "german_de",
			inputName:      "Welt",
			inputLanguage:  "de",
			expectedResult: "Hallo, Welt!",
			expectError:    false,
		},
		{
			name:           "german_native",
			inputName:      "Welt",
			inputLanguage:  "deutsch",
			expectedResult: "Hallo, Welt!",
			expectError:    false,
		},
		{
			name:           "japanese",
			inputName:      "世界",
			inputLanguage:  "japanese",
			expectedResult: "こんにちは, 世界!",
			expectError:    false,
		},
		{
			name:           "japanese_ja",
			inputName:      "世界",
			inputLanguage:  "ja",
			expectedResult: "こんにちは, 世界!",
			expectError:    false,
		},
		{
			name:           "japanese_native",
			inputName:      "世界",
			inputLanguage:  "日本語",
			expectedResult: "こんにちは, 世界!",
			expectError:    false,
		},
		{
			name:           "korean",
			inputName:      "세계",
			inputLanguage:  "korean",
			expectedResult: "안녕하세요, 세계!",
			expectError:    false,
		},
		{
			name:           "korean_ko",
			inputName:      "세계",
			inputLanguage:  "ko",
			expectedResult: "안녕하세요, 세계!",
			expectError:    false,
		},
		{
			name:           "korean_native",
			inputName:      "세계",
			inputLanguage:  "한국어",
			expectedResult: "안녕하세요, 세계!",
			expectError:    false,
		},
		{
			name:          "empty_name",
			inputName:     "",
			inputLanguage: "english",
			expectError:   true,
		},
		{
			name:           "unknown_language",
			inputName:      "World",
			inputLanguage:  "unknown",
			expectedResult: "Hello, World!",
			expectError:    false,
		},
		{
			name:           "whitespace_name",
			inputName:      "  John Doe  ",
			inputLanguage:  "english",
			expectedResult: "Hello, John Doe!",
			expectError:    false,
		},
		{
			name:           "case_insensitive_language",
			inputName:      "World",
			inputLanguage:  "SPANISH",
			expectedResult: "¡Hola, World!",
			expectError:    false,
		},
		{
			name:           "mixed_case_language",
			inputName:      "World",
			inputLanguage:  "ChInEsE",
			expectedResult: "你好, World！",
			expectError:    false,
		},
		{
			name:           "language_with_whitespace",
			inputName:      "World",
			inputLanguage:  "  french  ",
			expectedResult: "Bonjour, World!",
			expectError:    false,
		},
		{
			name:           "unicode_name",
			inputName:      "👋🌍",
			inputLanguage:  "english",
			expectedResult: "Hello, 👋🌍!",
			expectError:    false,
		},
		{
			name:           "numbers_in_name",
			inputName:      "User123",
			inputLanguage:  "english",
			expectedResult: "Hello, User123!",
			expectError:    false,
		},
		{
			name:           "special_characters_in_name",
			inputName:      "Test@#$%",
			inputLanguage:  "english",
			expectedResult: "Hello, Test@#$%!",
			expectError:    false,
		},
		{
			name:           "very_long_name",
			inputName:      "VeryLongNameThatExceedsNormalLengthButShouldStillWorkProperly",
			inputLanguage:  "english",
			expectedResult: "Hello, VeryLongNameThatExceedsNormalLengthButShouldStillWorkProperly!",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService()
			ctx := context.Background()

			result, err := service.GenerateGreeting(ctx, tt.inputName, tt.inputLanguage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// Test the gRPC handler (GreeterGRPCHandler)
func TestGreeterGRPCHandler_SayHello(t *testing.T) {
	setupTest()
	tests := []struct {
		name            string
		request         *greeterv1.SayHelloRequest
		expectedMessage string
		expectError     bool
	}{
		{
			name:            "valid_name",
			request:         &greeterv1.SayHelloRequest{Name: "World"},
			expectedMessage: "Hello, World!",
			expectError:     false,
		},
		{
			name:            "valid_name_with_language",
			request:         &greeterv1.SayHelloRequest{Name: "世界", Language: "chinese"},
			expectedMessage: "你好, 世界！",
			expectError:     false,
		},
		{
			name:        "empty_name",
			request:     &greeterv1.SayHelloRequest{Name: ""},
			expectError: true,
		},
		{
			name:            "name_with_spaces",
			request:         &greeterv1.SayHelloRequest{Name: "John Doe"},
			expectedMessage: "Hello, John Doe!",
			expectError:     false,
		},
		{
			name:            "unicode_name",
			request:         &greeterv1.SayHelloRequest{Name: "世界"},
			expectedMessage: "Hello, 世界!",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService()
			handler := NewGRPCHandler(service)
			ctx := context.Background()

			result, err := handler.SayHello(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedMessage, result.Message)
				assert.NotNil(t, result.Metadata)
			}
		})
	}
}

func TestGreeterGRPCHandler_SayHello_NilRequest(t *testing.T) {
	setupTest()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	t.Run("nil_request_panics", func(t *testing.T) {
		assert.Panics(t, func() {
			handler.SayHello(ctx, nil)
		})
	})
}

func TestGreeterGRPCHandler_SayHello_WithMetadata(t *testing.T) {
	setupTest()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	request := &greeterv1.SayHelloRequest{
		Name: "TestUser",
		Metadata: &commonv1.RequestMetadata{
			RequestId: "test-request-123",
			UserAgent: "test-client",
			ClientIp:  "127.0.0.1",
		},
	}

	result, err := handler.SayHello(ctx, request)

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Hello, TestUser!", result.Message)
	assert.NotNil(t, result.Metadata)
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, "test-request-123", result.Metadata.RequestId)
	assert.Equal(t, "swit-serve-1", result.Metadata.ServerId)
}

func TestGreeterGRPCHandler_SayHello_ContextVariations(t *testing.T) {
	setupTest()
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
			ctx:     context.WithValue(context.Background(), contextKey("key"), "value"),
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

func TestGreeterGRPCHandler_SayHello_CancelledContext(t *testing.T) {
	setupTest()
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

func TestGreeterGRPCHandler_Implements_Interface(t *testing.T) {
	setupTest()
	service := NewService()
	handler := NewGRPCHandler(service)

	_, ok := interface{}(handler).(greeterv1.GreeterServiceServer)
	assert.True(t, ok, "GreeterGRPCHandler should implement greeterv1.GreeterServiceServer interface")
}

func TestGreeterGRPCHandler_ResponseNotNil(t *testing.T) {
	setupTest()
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	request := &greeterv1.SayHelloRequest{Name: "Test"}

	result, err := handler.SayHello(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result, "Response should never be nil")
	assert.NotEmpty(t, result.Message, "Response message should not be empty")
	assert.NotNil(t, result.Metadata, "Response metadata should not be nil")
}

func TestGreeterService_Concurrent(t *testing.T) {
	setupTest()
	service := NewService()
	ctx := context.Background()

	const numGoroutines = 100
	results := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()

			name := "User" + string(rune('A'+index%26))
			result, err := service.GenerateGreeting(ctx, name, "")

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
		assert.NotEmpty(t, result, "Goroutine %d should return non-empty result", i)
		assert.Contains(t, result, "Hello, ", "Goroutine %d result should contain greeting", i)
	}
}

// Note: ServiceRegistrar tests are now in the parent package

// Benchmarks
func BenchmarkGreeterService_GenerateGreeting(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateGreeting(ctx, "BenchmarkUser", "")
	}
}

func BenchmarkGreeterService_GenerateGreeting_Chinese(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateGreeting(ctx, "测试用户", "chinese")
	}
}

func BenchmarkGreeterGRPCHandler_SayHello(b *testing.B) {
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

func BenchmarkGreeterGRPCHandler_SayHello_WithMetadata(b *testing.B) {
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

func BenchmarkGreeterService_Parallel(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = service.GenerateGreeting(ctx, "ParallelUser", "")
		}
	})
}

// Additional comprehensive tests

func TestGreeterService_NewService(t *testing.T) {
	setupTest()

	service := NewService()

	assert.NotNil(t, service)
	// Service should be stateless, so we can't test much more
}

func TestGreeterService_GenerateGreeting_EdgeCases(t *testing.T) {
	setupTest()
	service := NewService()
	ctx := context.Background()

	t.Run("whitespace_only_name", func(t *testing.T) {
		result, err := service.GenerateGreeting(ctx, "   ", "")
		assert.NoError(t, err)
		assert.Equal(t, "Hello, !", result)
	})

	t.Run("newline_in_name", func(t *testing.T) {
		result, err := service.GenerateGreeting(ctx, "Line1\nLine2", "")
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Line1\nLine2!", result)
	})

	t.Run("tab_in_name", func(t *testing.T) {
		result, err := service.GenerateGreeting(ctx, "Tab\tName", "")
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Tab\tName!", result)
	})

	t.Run("null_byte_in_name", func(t *testing.T) {
		result, err := service.GenerateGreeting(ctx, "Null\x00Byte", "")
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Null\x00Byte!", result)
	})

	t.Run("rtl_text", func(t *testing.T) {
		result, err := service.GenerateGreeting(ctx, "العالم", "")
		assert.NoError(t, err)
		assert.Equal(t, "Hello, العالم!", result)
	})

	t.Run("mixed_scripts", func(t *testing.T) {
		result, err := service.GenerateGreeting(ctx, "Hello世界", "")
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Hello世界!", result)
	})
}

func TestGreeterService_GenerateGreeting_LanguageAliases(t *testing.T) {
	setupTest()
	service := NewService()
	ctx := context.Background()

	tests := []struct {
		name           string
		language       string
		expectedResult string
	}{
		// Test all language aliases are working
		{"chinese_full", "chinese", "你好, Test！"},
		{"chinese_short", "zh", "你好, Test！"},
		{"chinese_native", "中文", "你好, Test！"},
		{"spanish_full", "spanish", "¡Hola, Test!"},
		{"spanish_short", "es", "¡Hola, Test!"},
		{"spanish_native", "español", "¡Hola, Test!"},
		{"french_full", "french", "Bonjour, Test!"},
		{"french_short", "fr", "Bonjour, Test!"},
		{"french_native", "français", "Bonjour, Test!"},
		{"german_full", "german", "Hallo, Test!"},
		{"german_short", "de", "Hallo, Test!"},
		{"german_native", "deutsch", "Hallo, Test!"},
		{"japanese_full", "japanese", "こんにちは, Test!"},
		{"japanese_short", "ja", "こんにちは, Test!"},
		{"japanese_native", "日本語", "こんにちは, Test!"},
		{"korean_full", "korean", "안녕하세요, Test!"},
		{"korean_short", "ko", "안녕하세요, Test!"},
		{"korean_native", "한국어", "안녕하세요, Test!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GenerateGreeting(ctx, "Test", tt.language)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGreeterService_GenerateGreeting_NilContext(t *testing.T) {
	setupTest()
	service := NewService()

	// Test that service handles nil context gracefully
	result, err := service.GenerateGreeting(nil, "Test", "")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, Test!", result)
}

func TestGreeterService_GenerateGreeting_ContextWithDeadline(t *testing.T) {
	setupTest()
	service := NewService()

	// Create context with very short deadline
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()

	// Sleep to ensure context is expired
	time.Sleep(time.Millisecond)

	// Service should still work (it doesn't check context cancellation)
	result, err := service.GenerateGreeting(ctx, "Test", "")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, Test!", result)
}

func TestGreeterService_GenerateGreeting_ContextWithValues(t *testing.T) {
	setupTest()
	service := NewService()

	// Create context with values
	type contextKey string
	const (
		userIDKey  contextKey = "user_id"
		traceIDKey contextKey = "trace_id"
	)
	ctx := context.WithValue(context.Background(), userIDKey, "123")
	ctx = context.WithValue(ctx, traceIDKey, "abc-def-ghi")

	result, err := service.GenerateGreeting(ctx, "Test", "")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, Test!", result)
}

func TestGreeterService_GenerateGreeting_Stress(t *testing.T) {
	setupTest()
	service := NewService()
	ctx := context.Background()

	// Stress test with many operations
	const numOperations = 1000

	for i := 0; i < numOperations; i++ {
		name := fmt.Sprintf("User%d", i)
		language := []string{"", "chinese", "spanish", "french", "german", "japanese", "korean"}[i%7]

		result, err := service.GenerateGreeting(ctx, name, language)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, name)
	}
}

func TestGreeterService_GenerateGreeting_MemoryConsumption(t *testing.T) {
	setupTest()
	service := NewService()
	ctx := context.Background()

	// Test that service doesn't leak memory with large inputs
	largeName := strings.Repeat("A", 10000)

	result, err := service.GenerateGreeting(ctx, largeName, "")
	assert.NoError(t, err)
	assert.Contains(t, result, largeName)
	assert.Equal(t, "Hello, "+largeName+"!", result)
}

func TestGreeterService_GenerateGreeting_ThreadSafety(t *testing.T) {
	setupTest()
	service := NewService()
	ctx := context.Background()

	const numGoroutines = 100
	const numIterations = 10

	var wg sync.WaitGroup
	results := make([][]string, numGoroutines)
	errors := make([][]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		results[i] = make([]string, numIterations)
		errors[i] = make([]error, numIterations)

		wg.Add(1)
		go func(goroutineIndex int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				name := fmt.Sprintf("User%d-%d", goroutineIndex, j)
				language := []string{"", "chinese", "spanish"}[j%3]

				result, err := service.GenerateGreeting(ctx, name, language)
				results[goroutineIndex][j] = result
				errors[goroutineIndex][j] = err
			}
		}(i)
	}

	wg.Wait()

	// Verify all results
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numIterations; j++ {
			assert.NoError(t, errors[i][j], "Goroutine %d iteration %d should not error", i, j)
			assert.NotEmpty(t, results[i][j], "Goroutine %d iteration %d should have result", i, j)
		}
	}
}

func TestGreeterService_GenerateGreeting_Internationalization(t *testing.T) {
	setupTest()
	service := NewService()
	ctx := context.Background()

	// Test with names from different cultures
	tests := []struct {
		name     string
		userName string
		language string
		expected string
	}{
		{"arabic_name", "محمد", "english", "Hello, محمد!"},
		{"cyrillic_name", "Владимир", "english", "Hello, Владимир!"},
		{"thai_name", "สมชาย", "english", "Hello, สมชาย!"},
		{"hebrew_name", "יוסף", "english", "Hello, יוסף!"},
		{"hindi_name", "राम", "english", "Hello, राम!"},
		{"chinese_name_chinese_greeting", "李明", "chinese", "你好, 李明！"},
		{"japanese_name_japanese_greeting", "田中", "japanese", "こんにちは, 田中!"},
		{"korean_name_korean_greeting", "김철수", "korean", "안녕하세요, 김철수!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GenerateGreeting(ctx, tt.userName, tt.language)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Additional benchmarks for comprehensive performance testing
func BenchmarkGreeterService_GenerateGreeting_AllLanguages(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	ctx := context.Background()

	languages := []string{"", "chinese", "spanish", "french", "german", "japanese", "korean"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lang := languages[i%len(languages)]
		_, _ = service.GenerateGreeting(ctx, "BenchUser", lang)
	}
}

func BenchmarkGreeterService_GenerateGreeting_LongName(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	ctx := context.Background()

	longName := strings.Repeat("VeryLongUserName", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateGreeting(ctx, longName, "")
	}
}

func BenchmarkGreeterService_GenerateGreeting_UnicodeName(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	ctx := context.Background()

	unicodeName := "用户名👋🌍测试"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateGreeting(ctx, unicodeName, "chinese")
	}
}

func BenchmarkGreeterService_GenerateGreeting_ConcurrentDifferentLanguages(b *testing.B) {
	logger.InitLogger()
	service := NewService()
	ctx := context.Background()

	languages := []string{"", "chinese", "spanish", "french", "german", "japanese", "korean"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			lang := languages[i%len(languages)]
			_, _ = service.GenerateGreeting(ctx, "ConcurrentUser", lang)
			i++
		}
	})
}
