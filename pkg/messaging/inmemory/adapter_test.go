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

package inmemory

import (
	"context"
	"testing"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapterInfo(t *testing.T) {
	a := newAdapter()
	info := a.GetAdapterInfo()
	require.NotNil(t, info)
	assert.Equal(t, "inmemory", info.Name)
	assert.Contains(t, info.SupportedBrokerTypes, messaging.BrokerTypeInMemory)
}

func TestAdapterValidateConfiguration(t *testing.T) {
	a := newAdapter()

	result := a.ValidateConfiguration(nil)
	assert.False(t, result.Valid)

	// Endpoints are not required for the in-memory broker.
	result = a.ValidateConfiguration(&messaging.BrokerConfig{Type: messaging.BrokerTypeInMemory})
	assert.True(t, result.Valid)

	// Endpoints are tolerated but flagged as ignored.
	result = a.ValidateConfiguration(&messaging.BrokerConfig{
		Type:      messaging.BrokerTypeInMemory,
		Endpoints: []string{"localhost:9092"},
	})
	assert.True(t, result.Valid)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, "ENDPOINTS_IGNORED", result.Warnings[0].Code)

	result = a.ValidateConfiguration(&messaging.BrokerConfig{Type: messaging.BrokerTypeKafka})
	assert.False(t, result.Valid)
}

func TestAdapterCreateBroker(t *testing.T) {
	a := newAdapter()

	_, err := a.CreateBroker(nil)
	assert.Error(t, err)

	_, err = a.CreateBroker(&messaging.BrokerConfig{Type: messaging.BrokerTypeNATS})
	assert.Error(t, err)

	broker, err := a.CreateBroker(&messaging.BrokerConfig{Type: messaging.BrokerTypeInMemory})
	require.NoError(t, err)
	require.NotNil(t, broker)
	require.NoError(t, broker.Connect(context.Background()))
	assert.True(t, broker.IsConnected())
	require.NoError(t, broker.Close())
}

func TestAdapterHealthCheck(t *testing.T) {
	a := newAdapter()
	status, err := a.HealthCheck(context.Background())
	require.NoError(t, err)
	assert.Equal(t, messaging.HealthStatusHealthy, status.Status)
	assert.Equal(t, "in-process", status.Details["library"])
}

func TestGlobalRegistration(t *testing.T) {
	// init() must have registered the adapter with the global registry.
	adapter, err := adapters.GetGlobalAdapter("inmemory")
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// The default messaging factory resolves inmemory through the registry.
	broker, err := messaging.NewMessageBroker(&messaging.BrokerConfig{Type: messaging.BrokerTypeInMemory})
	require.NoError(t, err)
	require.NotNil(t, broker)

	types := messaging.GetSupportedBrokerTypes()
	assert.Contains(t, types, messaging.BrokerTypeInMemory)
}
