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

package discovery

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// Initialize random seed once at package level
func init() {
	rand.Seed(time.Now().UnixNano())
}

// ServiceDiscovery handles service discovery using Consul
type ServiceDiscovery struct {
	client          *api.Client
	mu              sync.Mutex
	roundRobinIndex int
}

// NewServiceDiscovery creates a new service discovery client
func NewServiceDiscovery(address string) (*ServiceDiscovery, error) {
	config := api.DefaultConfig()
	// If no address is provided use the default address from the Consul
	// client configuration. Previously we overwrote the default with an
	// empty string which caused client creation to fail when tests passed an
	// empty address expecting the default to be used.
	if address != "" {
		config.Address = address
	}

	logger.GetLogger().Debug("Creating Consul client",
		zap.String("address", config.Address))

	client, err := api.NewClient(config)
	if err != nil {
		logger.GetLogger().Error("Failed to create Consul client",
			zap.String("address", config.Address),
			zap.Error(err))
		return nil, err
	}

	logger.GetLogger().Info("Consul client created successfully",
		zap.String("address", config.Address))

	return &ServiceDiscovery{client: client}, nil
}

// RegisterService registers a service with Consul's service registry.
func (sd *ServiceDiscovery) RegisterService(name, address string, port int) error {
	serviceID := fmt.Sprintf("%s-%s-%d", name, address, port)
	healthCheckURL := fmt.Sprintf("http://%s:%d/health", address, port)

	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    name,
		Address: address,
		Port:    port,
		Check: &api.AgentServiceCheck{
			HTTP:                           healthCheckURL,
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "1m",
		},
	}

	logger.GetLogger().Info("Registering service with Consul",
		zap.String("service_id", serviceID),
		zap.String("service_name", name),
		zap.String("address", address),
		zap.Int("port", port),
		zap.String("health_check_url", healthCheckURL))

	err := sd.client.Agent().ServiceRegister(registration)
	if err != nil {
		logger.GetLogger().Error("Failed to register service with Consul",
			zap.String("service_id", serviceID),
			zap.String("service_name", name),
			zap.Error(err))
		return err
	}

	logger.GetLogger().Info("Service registered successfully with Consul",
		zap.String("service_id", serviceID),
		zap.String("service_name", name))

	return nil
}

// DeregisterService removes a service from Consul's service registry.
func (sd *ServiceDiscovery) DeregisterService(name, address string, port int) error {
	serviceID := fmt.Sprintf("%s-%s-%d", name, address, port)

	logger.GetLogger().Info("Deregistering service from Consul",
		zap.String("service_id", serviceID),
		zap.String("service_name", name))

	err := sd.client.Agent().ServiceDeregister(serviceID)
	if err != nil {
		logger.GetLogger().Error("Failed to deregister service from Consul",
			zap.String("service_id", serviceID),
			zap.String("service_name", name),
			zap.Error(err))
		return err
	}

	logger.GetLogger().Info("Service deregistered successfully from Consul",
		zap.String("service_id", serviceID),
		zap.String("service_name", name))

	return nil
}

// GetInstanceRoundRobin retrieves a service instance using round-robin load balancing.
func (sd *ServiceDiscovery) GetInstanceRoundRobin(name string) (string, error) {
	logger.GetLogger().Debug("Discovering service instances",
		zap.String("service_name", name))

	services, _, err := sd.client.Health().Service(name, "", true, nil)
	if err != nil {
		logger.GetLogger().Error("Failed to discover service instances",
			zap.String("service_name", name),
			zap.Error(err))
		return "", err
	}

	if len(services) == 0 {
		logger.GetLogger().Warn("No healthy service instances found",
			zap.String("service_name", name))
		return "", fmt.Errorf("no healthy service instances found: %s", name)
	}

	sd.mu.Lock()
	idx := sd.roundRobinIndex % len(services)
	sd.roundRobinIndex++
	sd.mu.Unlock()

	service := services[idx].Service
	instance := fmt.Sprintf("%s:%d", service.Address, service.Port)

	logger.GetLogger().Debug("Selected service instance using round-robin",
		zap.String("service_name", name),
		zap.String("instance", instance),
		zap.Int("index", idx),
		zap.Int("total_instances", len(services)))

	return instance, nil
}

// GetInstanceRandom retrieves a random service instance.
func (sd *ServiceDiscovery) GetInstanceRandom(name string) (string, error) {
	services, _, err := sd.client.Health().Service(name, "", true, nil)
	if err != nil {
		return "", err
	}
	if len(services) == 0 {
		return "", fmt.Errorf("no healthy service instances found: %s", name)
	}

	idx := rand.Intn(len(services))
	service := services[idx].Service
	return fmt.Sprintf("%s:%d", service.Address, service.Port), nil
}
