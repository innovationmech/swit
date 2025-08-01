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

	"github.com/innovationmech/swit/pkg/discovery"

	"github.com/innovationmech/swit/internal/switauth/model"
)

// UserClient defines the interface for user-related operations
// including user credential validation through service discovery
type UserClient interface {
	ValidateUserCredentials(ctx context.Context, username, password string) (*model.User, error)
}

type userClient struct {
	sd         *discovery.ServiceDiscovery
	httpClient *http.Client
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

func (c *userClient) ValidateUserCredentials(ctx context.Context, username, password string) (*model.User, error) {
	serviceURL, err := c.sd.GetInstanceRoundRobin("swit-serve")
	if err != nil {
		return nil, fmt.Errorf("unable to discover swit-serve service: %v", err)
	}

	url := fmt.Sprintf("%s/internal/validate-user", serviceURL)

	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user credential validation failed: %s", resp.Status)
	}

	var user model.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
