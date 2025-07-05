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
)

type ServiceDiscovery struct {
	client          *api.Client
	mu              sync.Mutex
	roundRobinIndex int
}

func NewServiceDiscovery(address string) (*ServiceDiscovery, error) {
	config := api.DefaultConfig()
	config.Address = address
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ServiceDiscovery{client: client}, nil
}

// RegisterService registers a service with Consul's service registry.
func (sd *ServiceDiscovery) RegisterService(name, address string, port int) error {
	registration := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-%s-%d", name, address, port),
		Name:    name,
		Address: address,
		Port:    port,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", address, port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "1m",
		},
	}
	return sd.client.Agent().ServiceRegister(registration)
}

// DeregisterService removes a service from Consul's service registry.
func (sd *ServiceDiscovery) DeregisterService(name, address string, port int) error {
	return sd.client.Agent().ServiceDeregister(fmt.Sprintf("%s-%s-%d", name, address, port))
}

// GetInstanceRoundRobin retrieves a service instance using round-robin load balancing.
func (sd *ServiceDiscovery) GetInstanceRoundRobin(name string) (string, error) {
	services, _, err := sd.client.Health().Service(name, "", true, nil)
	if err != nil {
		return "", err
	}
	if len(services) == 0 {
		return "", fmt.Errorf("no healthy service instances found: %s", name)
	}

	sd.mu.Lock()
	idx := sd.roundRobinIndex % len(services)
	sd.roundRobinIndex++
	sd.mu.Unlock()

	service := services[idx].Service
	return fmt.Sprintf("%s:%d", service.Address, service.Port), nil
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

	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(len(services))
	service := services[idx].Service
	return fmt.Sprintf("%s:%d", service.Address, service.Port), nil
}
