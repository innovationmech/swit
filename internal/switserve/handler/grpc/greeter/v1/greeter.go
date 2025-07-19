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
	"time"

	"github.com/google/uuid"
	commonv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/common/v1"
	greeterv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	greeterv1svc "github.com/innovationmech/swit/internal/switserve/service/greeter/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GreeterHandler handles gRPC requests for Greeter service
type GreeterHandler struct {
	greeterv1.UnimplementedGreeterServiceServer
	service greeterv1svc.GreeterService
}

// NewGreeterHandler creates a new GreeterHandler
func NewGreeterHandler(svc greeterv1svc.GreeterService) *GreeterHandler {
	return &GreeterHandler{
		service: svc,
	}
}

// SayHello implements the SayHello RPC
func (h *GreeterHandler) SayHello(ctx context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	// Input validation
	if req.Name == "" {
		logger.Logger.Warn("SayHello called with empty name")
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	logger.Logger.Info("Processing SayHello request",
		zap.String("name", req.Name),
		zap.String("language", req.Language),
	)

	// Call business logic
	greeting, err := h.service.GenerateGreeting(ctx, req.Name, req.Language)
	if err != nil {
		logger.Logger.Error("Failed to generate greeting", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate greeting")
	}

	// Extract request ID from context
	requestID, _ := ctx.Value("request_id").(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Build response
	response := &greeterv1.SayHelloResponse{
		Message:  greeting,
		Language: req.Language,
		Style:    req.Style,
		Metadata: &commonv1.ResponseMetadata{
			RequestId:        requestID,
			ServerId:         "swit-serve-1", // This could be dynamic
			ProcessingTimeMs: 0,              // Calculate actual processing time
		},
	}

	logger.Logger.Info("SayHello request completed successfully",
		zap.String("name", req.Name),
		zap.String("request_id", requestID),
	)

	return response, nil
}

// SayHelloStream implements the streaming SayHello RPC
func (h *GreeterHandler) SayHelloStream(req *greeterv1.SayHelloStreamRequest, stream greeterv1.GreeterService_SayHelloStreamServer) error {
	// Input validation
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}

	if req.Count <= 0 {
		req.Count = 1
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 1
	}

	logger.Logger.Info("Processing SayHelloStream request",
		zap.String("name", req.Name),
		zap.Int32("count", req.Count),
		zap.Int32("interval", req.IntervalSeconds),
	)

	// Generate streaming responses
	for i := int32(0); i < req.Count; i++ {
		// Check if context is cancelled
		if stream.Context().Err() != nil {
			logger.Logger.Info("Stream cancelled by client")
			return stream.Context().Err()
		}

		// Generate greeting for this iteration
		greeting, err := h.service.GenerateGreeting(stream.Context(), req.Name, "")
		if err != nil {
			logger.Logger.Error("Failed to generate greeting in stream", zap.Error(err))
			return status.Error(codes.Internal, "failed to generate greeting")
		}

		response := &greeterv1.SayHelloStreamResponse{
			Message:  greeting,
			Language: req.Language,
			Style:    req.Style,
			Metadata: &commonv1.ResponseMetadata{
				RequestId:        uuid.New().String(),
				ServerId:         "swit-serve-1",
				ProcessingTimeMs: 0,
			},
			Sequence: i + 1,
		}

		if err := stream.Send(response); err != nil {
			logger.Logger.Error("Failed to send stream response", zap.Error(err))
			return err
		}

		// Wait for interval (except for the last message)
		if i < req.Count-1 {
			time.Sleep(time.Duration(req.IntervalSeconds) * time.Second)
		}
	}

	logger.Logger.Info("SayHelloStream completed successfully",
		zap.String("name", req.Name),
		zap.Int32("count", req.Count),
	)

	return nil
}
