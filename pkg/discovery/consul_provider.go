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
	"context"
	"fmt"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// ConsulProvider adapts the Consul-backed ServiceDiscovery client to the
// backend-agnostic Provider interface.
type ConsulProvider struct {
	sd *ServiceDiscovery
}

// interface guard
var _ Provider = (*ConsulProvider)(nil)

// NewConsulProvider creates a Consul-backed discovery provider.
func NewConsulProvider(address string) (*ConsulProvider, error) {
	sd, err := NewServiceDiscovery(address)
	if err != nil {
		return nil, err
	}
	return &ConsulProvider{sd: sd}, nil
}

// NewConsulProviderFromServiceDiscovery wraps an existing ServiceDiscovery
// client as a Provider.
func NewConsulProviderFromServiceDiscovery(sd *ServiceDiscovery) *ConsulProvider {
	return &ConsulProvider{sd: sd}
}

// Name returns the provider type name.
func (p *ConsulProvider) Name() string {
	return string(ProviderTypeConsul)
}

// Register registers a service instance with the Consul agent.
func (p *ConsulProvider) Register(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("service instance cannot be nil")
	}

	registration := &api.AgentServiceRegistration{
		ID:      instance.InstanceID(),
		Name:    instance.Name,
		Address: instance.Address,
		Port:    instance.Port,
		Tags:    instance.Tags,
		Meta:    instance.Meta,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", instance.Address, instance.Port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "1m",
		},
	}

	if err := p.sd.client.Agent().ServiceRegister(registration); err != nil {
		logger.GetLogger().Error("Failed to register service with Consul",
			zap.String("service_id", registration.ID),
			zap.String("service_name", instance.Name),
			zap.Error(err))
		return err
	}

	logger.GetLogger().Info("Service registered with Consul",
		zap.String("service_id", registration.ID),
		zap.String("service_name", instance.Name))
	return nil
}

// Deregister removes a service instance from the Consul agent.
func (p *ConsulProvider) Deregister(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("service instance cannot be nil")
	}
	return p.sd.DeregisterService(instance.Name, instance.Address, instance.Port)
}

// Discover returns the healthy instances of a service known to Consul.
func (p *ConsulProvider) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	services, _, err := p.sd.client.Health().Service(serviceName, "", true, (&api.QueryOptions{}).WithContext(ctx))
	if err != nil {
		return nil, err
	}

	instances := make([]*ServiceInstance, 0, len(services))
	for _, entry := range services {
		svc := entry.Service
		address := svc.Address
		if address == "" && entry.Node != nil {
			address = entry.Node.Address
		}
		instances = append(instances, &ServiceInstance{
			ID:      svc.ID,
			Name:    serviceName,
			Address: address,
			Port:    svc.Port,
			Tags:    svc.Tags,
			Meta:    svc.Meta,
		})
	}
	return instances, nil
}

// IsHealthy reports whether the Consul agent is reachable.
func (p *ConsulProvider) IsHealthy(ctx context.Context) bool {
	_, err := p.sd.client.Status().Leader()
	return err == nil
}

// Close releases the provider. The underlying Consul client has no
// long-lived connections to close.
func (p *ConsulProvider) Close() error {
	return nil
}
