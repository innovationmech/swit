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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/tracing"

	"github.com/innovationmech/swit/internal/switauth/model"
)

// UserClient defines the interface for user-related operations
// including user credential validation through service discovery
type UserClient interface {
	ValidateUserCredentials(ctx context.Context, username, password string) (*model.User, error)
}

type userClient struct {
	sd             *discovery.ServiceDiscovery
	httpClient     *http.Client
	tracingManager tracing.TracingManager
}

// NewUserClient creates a new UserClient instance with service discovery
// for communicating with the user service
func NewUserClient(sd *discovery.ServiceDiscovery) UserClient {
	// Create a shared HTTP client with optimized settings
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &userClient{
		sd:         sd,
		httpClient: httpClient,
	}
}

// NewUserClientWithTracing creates a new UserClient instance with tracing support
func NewUserClientWithTracing(sd *discovery.ServiceDiscovery, tracingManager tracing.TracingManager) UserClient {
	// Create a shared HTTP client with optimized settings
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &userClient{
		sd:             sd,
		httpClient:     httpClient,
		tracingManager: tracingManager,
	}
}

func (c *userClient) ValidateUserCredentials(ctx context.Context, username, password string) (*model.User, error) {
	// Create tracing span for cross-service HTTP call
	var span tracing.Span
	if c.tracingManager != nil {
		ctx, span = c.tracingManager.StartSpan(ctx, "HTTP_POST /internal/validate-user",
			tracing.WithSpanKind(oteltrace.SpanKindClient),
			tracing.WithAttributes(
				attribute.String("http.method", "POST"),
				attribute.String("service.name", "swit-serve"),
				attribute.String("operation.type", "validate_user_credentials"),
				attribute.String("user.username", username),
			),
		)
		defer span.End()
	}

	serviceURL, err := c.sd.GetInstanceRoundRobin("swit-serve")
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "service discovery failed")
			span.SetAttribute("error.type", "service_discovery")
			span.RecordError(err)
		}
		return nil, fmt.Errorf("unable to discover swit-serve service: %v", err)
	}

	url := fmt.Sprintf("%s/internal/validate-user", serviceURL)

	if span != nil {
		span.SetAttribute("http.url", url)
		span.SetAttribute("net.peer.name", serviceURL)
	}

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(credentials)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "request serialization failed")
			span.RecordError(err)
		}
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "request creation failed")
			span.RecordError(err)
		}
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Inject tracing context into HTTP headers for distributed tracing
	if c.tracingManager != nil {
		c.tracingManager.InjectHTTPHeaders(ctx, req.Header)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "HTTP request failed")
			span.SetAttribute("error.type", "network")
			span.RecordError(err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if span != nil {
		span.SetAttribute("http.status_code", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
			span.SetAttribute("error.type", "http_error")
		}
		return nil, fmt.Errorf("user credential validation failed: %s", resp.Status)
	}

	var user model.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "response deserialization failed")
			span.RecordError(err)
		}
		return nil, err
	}

	// Record successful operation
	if span != nil {
		span.SetAttribute("operation.success", true)
		span.SetAttribute("user.id", user.ID.String())
		span.SetStatus(codes.Ok, "user validation successful")
	}

	return &user, nil
}
