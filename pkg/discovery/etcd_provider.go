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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

const (
	defaultEtcdKeyPrefix   = "/swit/discovery"
	defaultEtcdLeaseTTL    = 30 * time.Second
	defaultEtcdDialTimeout = 5 * time.Second
	defaultEtcdOpTimeout   = 5 * time.Second
)

// EtcdConfig holds etcd-specific discovery configuration.
type EtcdConfig struct {
	// Endpoints lists the etcd cluster endpoints (e.g. "127.0.0.1:2379").
	Endpoints []string `yaml:"endpoints" json:"endpoints"`
	// KeyPrefix is the root key under which service instances are stored.
	// Defaults to "/swit/discovery".
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`
	// LeaseTTL controls how long a registration survives without keep-alive.
	// Defaults to 30s.
	LeaseTTL time.Duration `yaml:"lease_ttl" json:"lease_ttl"`
	// DialTimeout bounds the initial connection to etcd. Defaults to 5s.
	DialTimeout time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	// Username and Password enable etcd authentication when set.
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

func (c *EtcdConfig) setDefaults() {
	if c.KeyPrefix == "" {
		c.KeyPrefix = defaultEtcdKeyPrefix
	}
	if c.LeaseTTL <= 0 {
		c.LeaseTTL = defaultEtcdLeaseTTL
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = defaultEtcdDialTimeout
	}
}

// EtcdProvider implements Provider backed by an etcd cluster. Instances are
// stored as JSON values under "<prefix>/<service>/<instance-id>" keys bound
// to a keep-alive lease so crashed instances expire automatically.
type EtcdProvider struct {
	client *clientv3.Client
	config *EtcdConfig

	mu     sync.Mutex
	leases map[string]*etcdLease // instance key -> lease bookkeeping
}

type etcdLease struct {
	leaseID clientv3.LeaseID
	cancel  context.CancelFunc
}

// interface guard
var _ Provider = (*EtcdProvider)(nil)

// NewEtcdProvider creates an etcd-backed discovery provider.
func NewEtcdProvider(config *EtcdConfig) (*EtcdProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("etcd config cannot be nil")
	}
	if len(config.Endpoints) == 0 {
		return nil, fmt.Errorf("etcd endpoints are required")
	}
	config.setDefaults()

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.DialTimeout,
		Username:    config.Username,
		Password:    config.Password,
		Logger:      zap.NewNop(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	logger.GetLogger().Info("etcd discovery provider created",
		zap.Strings("endpoints", config.Endpoints),
		zap.String("key_prefix", config.KeyPrefix))

	return &EtcdProvider{
		client: client,
		config: config,
		leases: make(map[string]*etcdLease),
	}, nil
}

// Name returns the provider type name.
func (p *EtcdProvider) Name() string {
	return string(ProviderTypeEtcd)
}

func (p *EtcdProvider) serviceKey(serviceName string) string {
	return fmt.Sprintf("%s/%s/", strings.TrimSuffix(p.config.KeyPrefix, "/"), serviceName)
}

func (p *EtcdProvider) instanceKey(instance *ServiceInstance) string {
	return p.serviceKey(instance.Name) + instance.InstanceID()
}

// Register stores the instance under a lease and keeps the lease alive in the
// background until Deregister or Close is called.
func (p *EtcdProvider) Register(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("service instance cannot be nil")
	}

	value, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal service instance: %w", err)
	}

	grantCtx, cancelGrant := context.WithTimeout(ctx, defaultEtcdOpTimeout)
	defer cancelGrant()
	lease, err := p.client.Grant(grantCtx, int64(p.config.LeaseTTL.Seconds()))
	if err != nil {
		return fmt.Errorf("failed to grant etcd lease: %w", err)
	}

	key := p.instanceKey(instance)
	putCtx, cancelPut := context.WithTimeout(ctx, defaultEtcdOpTimeout)
	defer cancelPut()
	if _, err := p.client.Put(putCtx, key, string(value), clientv3.WithLease(lease.ID)); err != nil {
		return fmt.Errorf("failed to store service instance in etcd: %w", err)
	}

	// Keep the lease alive for as long as the provider lives.
	keepAliveCtx, cancelKeepAlive := context.WithCancel(context.Background())
	keepAliveCh, err := p.client.KeepAlive(keepAliveCtx, lease.ID)
	if err != nil {
		cancelKeepAlive()
		return fmt.Errorf("failed to start etcd lease keep-alive: %w", err)
	}
	go func() {
		for range keepAliveCh {
			// Drain keep-alive responses until the channel closes.
		}
	}()

	p.mu.Lock()
	if old, exists := p.leases[key]; exists {
		old.cancel()
	}
	p.leases[key] = &etcdLease{leaseID: lease.ID, cancel: cancelKeepAlive}
	p.mu.Unlock()

	logger.GetLogger().Info("Service registered with etcd",
		zap.String("key", key),
		zap.String("service_name", instance.Name))
	return nil
}

// Deregister removes the instance key and revokes its lease.
func (p *EtcdProvider) Deregister(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("service instance cannot be nil")
	}

	key := p.instanceKey(instance)

	p.mu.Lock()
	lease, exists := p.leases[key]
	if exists {
		lease.cancel()
		delete(p.leases, key)
	}
	p.mu.Unlock()

	opCtx, cancel := context.WithTimeout(ctx, defaultEtcdOpTimeout)
	defer cancel()

	if exists {
		if _, err := p.client.Revoke(opCtx, lease.leaseID); err != nil {
			logger.GetLogger().Warn("Failed to revoke etcd lease, deleting key directly",
				zap.String("key", key),
				zap.Error(err))
		} else {
			logger.GetLogger().Info("Service deregistered from etcd",
				zap.String("key", key))
			return nil
		}
	}

	if _, err := p.client.Delete(opCtx, key); err != nil {
		return fmt.Errorf("failed to delete service instance from etcd: %w", err)
	}

	logger.GetLogger().Info("Service deregistered from etcd",
		zap.String("key", key))
	return nil
}

// Discover lists all instances registered under the service prefix.
func (p *EtcdProvider) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	opCtx, cancel := context.WithTimeout(ctx, defaultEtcdOpTimeout)
	defer cancel()

	resp, err := p.client.Get(opCtx, p.serviceKey(serviceName), clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list service instances from etcd: %w", err)
	}

	instances := make([]*ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var instance ServiceInstance
		if err := json.Unmarshal(kv.Value, &instance); err != nil {
			logger.GetLogger().Warn("Skipping malformed service instance in etcd",
				zap.String("key", string(kv.Key)),
				zap.Error(err))
			continue
		}
		instances = append(instances, &instance)
	}
	return instances, nil
}

// IsHealthy reports whether at least one etcd endpoint responds to a status
// request.
func (p *EtcdProvider) IsHealthy(ctx context.Context) bool {
	opCtx, cancel := context.WithTimeout(ctx, defaultEtcdOpTimeout)
	defer cancel()

	for _, endpoint := range p.config.Endpoints {
		if _, err := p.client.Status(opCtx, endpoint); err == nil {
			return true
		}
	}
	return false
}

// Close cancels all keep-alive loops and closes the etcd client.
func (p *EtcdProvider) Close() error {
	p.mu.Lock()
	for key, lease := range p.leases {
		lease.cancel()
		delete(p.leases, key)
	}
	p.mu.Unlock()

	return p.client.Close()
}
