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

	"github.com/google/uuid"
	commonv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/common/v1"
	greeterv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/interaction/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler implements the gRPC server interface using business logic
type GRPCHandler struct {
	greeterv1.UnimplementedGreeterServiceServer
	service *Service
}

// NewGRPCHandler creates a new gRPC handler with business logic service
func NewGRPCHandler(service *Service) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// SayHello implements the SayHello gRPC method
func (h *GRPCHandler) SayHello(ctx context.Context, req *greeterv1.SayHelloRequest) (*greeterv1.SayHelloResponse, error) {
	// Input validation
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Call business logic
	greeting, err := h.service.GenerateGreeting(ctx, req.Name, req.Language)
	if err != nil {
		logger.Logger.Error("Failed to generate greeting", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate greeting")
	}

	// Build response
	response := &greeterv1.SayHelloResponse{
		Message:  greeting,
		Language: req.Language,
		Style:    req.Style,
		Metadata: &commonv1.ResponseMetadata{
			RequestId:        getRequestID(req.Metadata),
			ServerId:         "swit-serve-1",
			ProcessingTimeMs: 0, // TODO: Calculate actual processing time
		},
	}

	return response, nil
}

// getRequestID extracts request ID from metadata or generates new one
func getRequestID(metadata *commonv1.RequestMetadata) string {
	if metadata != nil && metadata.RequestId != "" {
		return metadata.RequestId
	}
	return uuid.New().String()
}

// SayHelloStream implements the SayHelloStream gRPC method
func (h *GRPCHandler) SayHelloStream(req *greeterv1.SayHelloStreamRequest, stream greeterv1.GreeterService_SayHelloStreamServer) error {
	// Input validation
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}

	if req.Count <= 0 {
		req.Count = 1
	}

	// TODO: Implement streaming logic using business service
	// For now, return unimplemented
	return status.Error(codes.Unimplemented, "streaming not yet implemented")
}
