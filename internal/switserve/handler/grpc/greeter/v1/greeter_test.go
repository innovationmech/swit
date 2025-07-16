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
	"errors"
	"testing"

	"github.com/google/uuid"
	greeterv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	greeterv1svc "github.com/innovationmech/swit/internal/switserve/service/greeter/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MockStream is a mock implementation of the gRPC stream
type MockStream struct {
	mock.Mock
	ctx context.Context
}

func (m *MockStream) Send(response *greeterv1.SayHelloStreamResponse) error {
	args := m.Called(response)
	return args.Error(0)
}

func (m *MockStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *MockStream) SetContext(ctx context.Context) {
	m.ctx = ctx
}

func (m *MockStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *MockStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *MockStream) SetTrailer(md metadata.MD) {
}

func (m *MockStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *MockStream) RecvMsg(msg interface{}) error {
	return nil
}

func TestNewGreeterHandler(t *testing.T) {
	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	assert.NotNil(t, handler)
	assert.Equal(t, service, handler.service)
}

func TestGreeterHandler_SayHello_Success(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	req := &greeterv1.SayHelloRequest{
		Name:     "John",
		Language: "english",
		Style:    greeterv1.GreetingStyle_GREETING_STYLE_CASUAL,
	}

	ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
	resp, err := handler.SayHello(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Hello, John!", resp.Message)
	assert.Equal(t, "english", resp.Language)
	assert.Equal(t, greeterv1.GreetingStyle_GREETING_STYLE_CASUAL, resp.Style)
	assert.NotNil(t, resp.Metadata)
	assert.Equal(t, "test-request-123", resp.Metadata.RequestId)
	assert.Equal(t, "swit-serve-1", resp.Metadata.ServerId)
}

func TestGreeterHandler_SayHello_EmptyName(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	req := &greeterv1.SayHelloRequest{
		Name:     "",
		Language: "english",
	}

	resp, err := handler.SayHello(context.Background(), req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "name is required")
}

func TestGreeterHandler_SayHello_GeneratesRequestID(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	req := &greeterv1.SayHelloRequest{
		Name: "John",
	}

	resp, err := handler.SayHello(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Metadata.RequestId)

	_, err = uuid.Parse(resp.Metadata.RequestId)
	assert.NoError(t, err)
}

func TestGreeterHandler_SayHello_DifferentLanguages(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	tests := []struct {
		name     string
		language string
		expected string
	}{
		{"John", "spanish", "¡Hola, John!"},
		{"John", "chinese", "你好, John！"},
		{"John", "french", "Bonjour, John!"},
		{"John", "", "Hello, John!"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			req := &greeterv1.SayHelloRequest{
				Name:     tt.name,
				Language: tt.language,
			}

			resp, err := handler.SayHello(context.Background(), req)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp.Message)
		})
	}
}

func TestGreeterHandler_SayHelloStream_Success(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	mockStream := &MockStream{}
	mockStream.On("Send", mock.AnythingOfType("*interactionv1.SayHelloStreamResponse")).Return(nil).Times(2)

	req := &greeterv1.SayHelloStreamRequest{
		Name:            "John",
		Language:        "english",
		Style:           greeterv1.GreetingStyle_GREETING_STYLE_FRIENDLY,
		Count:           2,
		IntervalSeconds: 0,
	}

	err := handler.SayHelloStream(req, mockStream)

	assert.NoError(t, err)
	mockStream.AssertExpectations(t)
}

func TestGreeterHandler_SayHelloStream_EmptyName(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	mockStream := &MockStream{}

	req := &greeterv1.SayHelloStreamRequest{
		Name:  "",
		Count: 1,
	}

	err := handler.SayHelloStream(req, mockStream)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "name is required")

	mockStream.AssertNotCalled(t, "Send")
}

func TestGreeterHandler_SayHelloStream_DefaultValues(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	mockStream := &MockStream{}
	mockStream.On("Send", mock.AnythingOfType("*interactionv1.SayHelloStreamResponse")).Return(nil).Once()

	req := &greeterv1.SayHelloStreamRequest{
		Name:            "John",
		Count:           0,
		IntervalSeconds: 0,
	}

	err := handler.SayHelloStream(req, mockStream)

	assert.NoError(t, err)
	mockStream.AssertExpectations(t)
}

func TestGreeterHandler_SayHelloStream_SendError(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	sendErr := errors.New("send error")
	mockStream := &MockStream{}
	mockStream.On("Send", mock.AnythingOfType("*interactionv1.SayHelloStreamResponse")).Return(sendErr)

	req := &greeterv1.SayHelloStreamRequest{
		Name:  "John",
		Count: 1,
	}

	err := handler.SayHelloStream(req, mockStream)

	assert.Error(t, err)
	assert.Equal(t, sendErr, err)

	mockStream.AssertExpectations(t)
}

func TestGreeterHandler_SayHelloStream_ContextCancelled(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockStream := &MockStream{}
	mockStream.SetContext(ctx)

	req := &greeterv1.SayHelloStreamRequest{
		Name:  "John",
		Count: 2,
	}

	err := handler.SayHelloStream(req, mockStream)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	mockStream.AssertNotCalled(t, "Send")
}

func TestGreeterHandler_SayHelloStream_ResponseStructure(t *testing.T) {
	logger.Logger = zap.NewNop()

	service := greeterv1svc.NewService()
	handler := NewGreeterHandler(service)

	var capturedResponse *greeterv1.SayHelloStreamResponse
	mockStream := &MockStream{}
	mockStream.On("Send", mock.AnythingOfType("*interactionv1.SayHelloStreamResponse")).Run(func(args mock.Arguments) {
		capturedResponse = args.Get(0).(*greeterv1.SayHelloStreamResponse)
	}).Return(nil).Once()

	req := &greeterv1.SayHelloStreamRequest{
		Name:     "John",
		Language: "english",
		Style:    greeterv1.GreetingStyle_GREETING_STYLE_PROFESSIONAL,
		Count:    1,
	}

	err := handler.SayHelloStream(req, mockStream)

	require.NoError(t, err)
	require.NotNil(t, capturedResponse)

	assert.Equal(t, "Hello, John!", capturedResponse.Message)
	assert.Equal(t, "english", capturedResponse.Language)
	assert.Equal(t, greeterv1.GreetingStyle_GREETING_STYLE_PROFESSIONAL, capturedResponse.Style)
	assert.Equal(t, int32(1), capturedResponse.Sequence)
	assert.NotNil(t, capturedResponse.Metadata)
	assert.NotEmpty(t, capturedResponse.Metadata.RequestId)
	assert.Equal(t, "swit-serve-1", capturedResponse.Metadata.ServerId)

	mockStream.AssertExpectations(t)
}
