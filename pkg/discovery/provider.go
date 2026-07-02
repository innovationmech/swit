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
	"math/rand"
	"strings"
	"sync"
)

// ProviderType identifies a service discovery backend implementation.
type ProviderType string

const (
	// ProviderTypeConsul selects the HashiCorp Consul backend.
	ProviderTypeConsul ProviderType = "consul"
	// ProviderTypeEtcd selects the etcd backend.
	ProviderTypeEtcd ProviderType = "etcd"
	// ProviderTypeKubernetes selects the Kubernetes backend (Endpoints API or DNS).
	ProviderTypeKubernetes ProviderType = "kubernetes"
)

// SupportedProviderTypes returns all provider types supported by the factory.
func SupportedProviderTypes() []ProviderType {
	return []ProviderType{ProviderTypeConsul, ProviderTypeEtcd, ProviderTypeKubernetes}
}

// ServiceInstance describes a single instance of a service as seen by a
// discovery backend.
type ServiceInstance struct {
	// ID uniquely identifies the instance within the service. When empty a
	// deterministic ID is derived from name, address and port.
	ID string `json:"id"`
	// Name is the logical service name.
	Name string `json:"name"`
	// Address is the host or IP the instance listens on.
	Address string `json:"address"`
	// Port is the port the instance listens on.
	Port int `json:"port"`
	// Tags carries backend-specific labels (e.g. Consul tags).
	Tags []string `json:"tags,omitempty"`
	// Meta carries arbitrary key/value metadata.
	Meta map[string]string `json:"meta,omitempty"`
}

// InstanceID returns the explicit instance ID or a deterministic default.
func (si *ServiceInstance) InstanceID() string {
	if si.ID != "" {
		return si.ID
	}
	return fmt.Sprintf("%s-%s-%d", si.Name, si.Address, si.Port)
}

// Endpoint returns the "host:port" representation of the instance.
func (si *ServiceInstance) Endpoint() string {
	return fmt.Sprintf("%s:%d", si.Address, si.Port)
}

// Provider is the backend-agnostic service discovery abstraction. All
// discovery backends (Consul, etcd, Kubernetes, ...) implement this interface
// so callers can switch between them purely through configuration.
type Provider interface {
	// Name returns the provider type name (e.g. "consul").
	Name() string
	// Register registers a service instance with the backend. Backends that
	// do not support explicit registration (e.g. Kubernetes) treat this as a
	// no-op.
	Register(ctx context.Context, instance *ServiceInstance) error
	// Deregister removes a service instance from the backend.
	Deregister(ctx context.Context, instance *ServiceInstance) error
	// Discover returns the currently known healthy instances of a service.
	Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	// IsHealthy reports whether the discovery backend itself is reachable.
	IsHealthy(ctx context.Context) bool
	// Close releases resources held by the provider.
	Close() error
}

// ProviderConfig selects and configures a discovery backend.
type ProviderConfig struct {
	// Type selects the backend. Defaults to "consul" when empty for backward
	// compatibility.
	Type ProviderType `yaml:"type" json:"type"`
	// Address is the generic backend address. It is used directly by the
	// Consul backend and as a fallback single endpoint for etcd.
	Address string `yaml:"address" json:"address"`
	// Etcd holds etcd-specific settings (used when Type == "etcd").
	Etcd EtcdConfig `yaml:"etcd" json:"etcd"`
	// Kubernetes holds Kubernetes-specific settings (used when Type == "kubernetes").
	Kubernetes KubernetesConfig `yaml:"kubernetes" json:"kubernetes"`
}

// normalizedType returns the effective provider type, defaulting to Consul.
func (c *ProviderConfig) normalizedType() ProviderType {
	t := ProviderType(strings.ToLower(strings.TrimSpace(string(c.Type))))
	if t == "" {
		return ProviderTypeConsul
	}
	return t
}

// cacheKey returns a stable key used to cache provider instances.
func (c *ProviderConfig) cacheKey() string {
	switch c.normalizedType() {
	case ProviderTypeEtcd:
		endpoints := c.Etcd.Endpoints
		if len(endpoints) == 0 && c.Address != "" {
			endpoints = []string{c.Address}
		}
		return fmt.Sprintf("etcd|%s|%s", strings.Join(endpoints, ","), c.Etcd.KeyPrefix)
	case ProviderTypeKubernetes:
		return fmt.Sprintf("kubernetes|%s|%s|%s", c.Kubernetes.Namespace, c.Kubernetes.Mode, c.Kubernetes.APIServer)
	default:
		return fmt.Sprintf("consul|%s", c.Address)
	}
}

// NewProvider creates a discovery provider for the configured backend type.
func NewProvider(config *ProviderConfig) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("discovery provider config cannot be nil")
	}

	switch config.normalizedType() {
	case ProviderTypeConsul:
		return NewConsulProvider(config.Address)
	case ProviderTypeEtcd:
		etcdConfig := config.Etcd
		if len(etcdConfig.Endpoints) == 0 && config.Address != "" {
			etcdConfig.Endpoints = []string{config.Address}
		}
		return NewEtcdProvider(&etcdConfig)
	case ProviderTypeKubernetes:
		k8sConfig := config.Kubernetes
		return NewKubernetesProvider(&k8sConfig)
	default:
		return nil, fmt.Errorf("unsupported discovery provider type %q (supported: consul, etcd, kubernetes)", config.Type)
	}
}

// LoadBalancedResolver provides round-robin and random instance selection on
// top of any Provider, mirroring the historical ServiceDiscovery resolution
// helpers for all backends.
type LoadBalancedResolver struct {
	provider Provider

	mu              sync.Mutex
	roundRobinIndex int
}

// NewLoadBalancedResolver creates a resolver backed by the given provider.
func NewLoadBalancedResolver(provider Provider) *LoadBalancedResolver {
	return &LoadBalancedResolver{provider: provider}
}

// GetInstanceRoundRobin resolves a service instance using round-robin selection.
func (r *LoadBalancedResolver) GetInstanceRoundRobin(ctx context.Context, serviceName string) (string, error) {
	instances, err := r.provider.Discover(ctx, serviceName)
	if err != nil {
		return "", err
	}
	if len(instances) == 0 {
		return "", fmt.Errorf("no healthy service instances found: %s", serviceName)
	}

	r.mu.Lock()
	idx := r.roundRobinIndex % len(instances)
	r.roundRobinIndex++
	r.mu.Unlock()

	return instances[idx].Endpoint(), nil
}

// GetInstanceRandom resolves a random service instance.
func (r *LoadBalancedResolver) GetInstanceRandom(ctx context.Context, serviceName string) (string, error) {
	instances, err := r.provider.Discover(ctx, serviceName)
	if err != nil {
		return "", err
	}
	if len(instances) == 0 {
		return "", fmt.Errorf("no healthy service instances found: %s", serviceName)
	}

	return instances[rand.Intn(len(instances))].Endpoint(), nil
}
