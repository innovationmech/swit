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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// KubernetesMode selects how the Kubernetes provider resolves services.
type KubernetesMode string

const (
	// KubernetesModeEndpoints resolves instances through the Kubernetes
	// Endpoints API (requires in-cluster credentials or explicit token).
	KubernetesModeEndpoints KubernetesMode = "endpoints"
	// KubernetesModeDNS resolves instances through cluster DNS
	// (<service>.<namespace>.svc.<domain>).
	KubernetesModeDNS KubernetesMode = "dns"
)

const (
	defaultKubernetesAPIServer  = "https://kubernetes.default.svc"
	defaultKubernetesNamespace  = "default"
	defaultKubernetesDNSDomain  = "cluster.local"
	defaultKubernetesTokenFile  = "/var/run/secrets/kubernetes.io/serviceaccount/token" // #nosec G101 -- well-known in-cluster path, not a credential
	defaultKubernetesCACertFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	defaultKubernetesOpTimeout  = 5 * time.Second
)

// KubernetesConfig holds Kubernetes-specific discovery configuration.
type KubernetesConfig struct {
	// Mode selects Endpoints API or DNS resolution. Defaults to "endpoints".
	Mode KubernetesMode `yaml:"mode" json:"mode"`
	// Namespace scopes lookups. Defaults to "default" (or the pod namespace
	// when running in-cluster and the namespace file is readable).
	Namespace string `yaml:"namespace" json:"namespace"`
	// APIServer is the Kubernetes API base URL for endpoints mode. Defaults
	// to the in-cluster address "https://kubernetes.default.svc".
	APIServer string `yaml:"api_server" json:"api_server"`
	// TokenFile is the bearer token path for endpoints mode. Defaults to the
	// in-cluster service account token path.
	TokenFile string `yaml:"token_file" json:"token_file"`
	// CACertFile is the CA bundle used to verify the API server. Defaults to
	// the in-cluster service account CA path.
	CACertFile string `yaml:"ca_cert_file" json:"ca_cert_file"`
	// InsecureSkipTLSVerify disables TLS verification (testing only).
	InsecureSkipTLSVerify bool `yaml:"insecure_skip_tls_verify" json:"insecure_skip_tls_verify"`
	// DNSDomain is the cluster DNS domain for dns mode. Defaults to "cluster.local".
	DNSDomain string `yaml:"dns_domain" json:"dns_domain"`
	// Port is the fallback port used in dns mode when SRV records are not
	// available (e.g. non-headless services).
	Port int `yaml:"port" json:"port"`
	// PortName filters endpoints mode to a named port when a service exposes
	// multiple ports. Empty selects the first port.
	PortName string `yaml:"port_name" json:"port_name"`
}

func (c *KubernetesConfig) setDefaults() {
	if c.Mode == "" {
		c.Mode = KubernetesModeEndpoints
	}
	if c.Namespace == "" {
		c.Namespace = defaultKubernetesNamespace
	}
	if c.APIServer == "" {
		c.APIServer = defaultKubernetesAPIServer
	}
	if c.TokenFile == "" {
		c.TokenFile = defaultKubernetesTokenFile
	}
	if c.CACertFile == "" {
		c.CACertFile = defaultKubernetesCACertFile
	}
	if c.DNSDomain == "" {
		c.DNSDomain = defaultKubernetesDNSDomain
	}
}

// kubernetesResolver abstracts DNS lookups for testability.
type kubernetesResolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
	LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error)
}

// KubernetesProvider implements Provider on top of the Kubernetes Endpoints
// API or cluster DNS. Registration is a no-op because Kubernetes manages
// instance membership through pod lifecycle and readiness probes.
type KubernetesProvider struct {
	config     *KubernetesConfig
	httpClient *http.Client
	resolver   kubernetesResolver
	token      string
}

// interface guard
var _ Provider = (*KubernetesProvider)(nil)

// NewKubernetesProvider creates a Kubernetes-backed discovery provider.
func NewKubernetesProvider(config *KubernetesConfig) (*KubernetesProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("kubernetes config cannot be nil")
	}
	config.setDefaults()

	switch config.Mode {
	case KubernetesModeEndpoints, KubernetesModeDNS:
	default:
		return nil, fmt.Errorf("unsupported kubernetes discovery mode %q (supported: endpoints, dns)", config.Mode)
	}

	provider := &KubernetesProvider{
		config:   config,
		resolver: net.DefaultResolver,
	}

	if config.Mode == KubernetesModeEndpoints {
		httpClient, token, err := buildKubernetesHTTPClient(config)
		if err != nil {
			return nil, err
		}
		provider.httpClient = httpClient
		provider.token = token
	}

	logger.GetLogger().Info("Kubernetes discovery provider created",
		zap.String("mode", string(config.Mode)),
		zap.String("namespace", config.Namespace))

	return provider, nil
}

func buildKubernetesHTTPClient(config *KubernetesConfig) (*http.Client, string, error) {
	transport := &http.Transport{}

	if strings.HasPrefix(config.APIServer, "https://") {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		if config.InsecureSkipTLSVerify {
			tlsConfig.InsecureSkipVerify = true // #nosec G402 -- explicit opt-in for testing
		} else if caData, err := os.ReadFile(config.CACertFile); err == nil {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caData) {
				return nil, "", fmt.Errorf("failed to parse kubernetes CA certificate from %s", config.CACertFile)
			}
			tlsConfig.RootCAs = pool
		}
		transport.TLSClientConfig = tlsConfig
	}

	token := ""
	if data, err := os.ReadFile(config.TokenFile); err == nil {
		token = strings.TrimSpace(string(data))
	}

	return &http.Client{Transport: transport, Timeout: defaultKubernetesOpTimeout}, token, nil
}

// Name returns the provider type name.
func (p *KubernetesProvider) Name() string {
	return string(ProviderTypeKubernetes)
}

// Register is a no-op: Kubernetes tracks instance membership through pod
// lifecycle, readiness probes and Endpoints controllers.
func (p *KubernetesProvider) Register(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("service instance cannot be nil")
	}
	logger.GetLogger().Debug("Kubernetes provider ignores explicit registration",
		zap.String("service_name", instance.Name))
	return nil
}

// Deregister is a no-op for the same reason as Register.
func (p *KubernetesProvider) Deregister(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("service instance cannot be nil")
	}
	logger.GetLogger().Debug("Kubernetes provider ignores explicit deregistration",
		zap.String("service_name", instance.Name))
	return nil
}

// Discover resolves ready instances of a service.
func (p *KubernetesProvider) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	switch p.config.Mode {
	case KubernetesModeDNS:
		return p.discoverViaDNS(ctx, serviceName)
	default:
		return p.discoverViaEndpoints(ctx, serviceName)
	}
}

// kubernetesEndpoints mirrors the subset of the core/v1 Endpoints object we
// need, avoiding a dependency on k8s.io/api.
type kubernetesEndpoints struct {
	Subsets []struct {
		Addresses []struct {
			IP        string `json:"ip"`
			TargetRef *struct {
				Name string `json:"name"`
			} `json:"targetRef,omitempty"`
		} `json:"addresses"`
		Ports []struct {
			Name string `json:"name"`
			Port int    `json:"port"`
		} `json:"ports"`
	} `json:"subsets"`
}

func (p *KubernetesProvider) discoverViaEndpoints(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/endpoints/%s",
		strings.TrimSuffix(p.config.APIServer, "/"), p.config.Namespace, serviceName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes endpoints request: %w", err)
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query kubernetes endpoints API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no healthy service instances found: %s", serviceName)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("kubernetes endpoints API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var endpoints kubernetesEndpoints
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("failed to decode kubernetes endpoints response: %w", err)
	}

	var instances []*ServiceInstance
	for _, subset := range endpoints.Subsets {
		port := 0
		for _, subsetPort := range subset.Ports {
			if p.config.PortName == "" || subsetPort.Name == p.config.PortName {
				port = subsetPort.Port
				break
			}
		}
		if port == 0 {
			continue
		}

		for _, address := range subset.Addresses {
			instance := &ServiceInstance{
				Name:    serviceName,
				Address: address.IP,
				Port:    port,
			}
			if address.TargetRef != nil && address.TargetRef.Name != "" {
				instance.ID = address.TargetRef.Name
			}
			instances = append(instances, instance)
		}
	}
	return instances, nil
}

func (p *KubernetesProvider) discoverViaDNS(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	host := fmt.Sprintf("%s.%s.svc.%s", serviceName, p.config.Namespace, p.config.DNSDomain)

	opCtx, cancel := context.WithTimeout(ctx, defaultKubernetesOpTimeout)
	defer cancel()

	// SRV lookup works for headless services and carries port information.
	if _, srvs, err := p.resolver.LookupSRV(opCtx, "", "", host); err == nil && len(srvs) > 0 {
		instances := make([]*ServiceInstance, 0, len(srvs))
		for _, srv := range srvs {
			instances = append(instances, &ServiceInstance{
				Name:    serviceName,
				Address: strings.TrimSuffix(srv.Target, "."),
				Port:    int(srv.Port),
			})
		}
		return instances, nil
	}

	// Fall back to A/AAAA records combined with the configured port.
	ips, err := p.resolver.LookupIP(opCtx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve kubernetes service %s: %w", host, err)
	}
	if p.config.Port <= 0 {
		return nil, fmt.Errorf("kubernetes dns mode requires a configured port when SRV records are unavailable (service %s)", serviceName)
	}

	instances := make([]*ServiceInstance, 0, len(ips))
	for _, ip := range ips {
		instances = append(instances, &ServiceInstance{
			Name:    serviceName,
			Address: ip.String(),
			Port:    p.config.Port,
		})
	}
	return instances, nil
}

// IsHealthy reports whether the resolution backend is reachable.
func (p *KubernetesProvider) IsHealthy(ctx context.Context) bool {
	switch p.config.Mode {
	case KubernetesModeDNS:
		// DNS mode has no meaningful backend health probe beyond resolution
		// itself, which happens per-request.
		return true
	default:
		url := strings.TrimSuffix(p.config.APIServer, "/") + "/version"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false
		}
		if p.token != "" {
			req.Header.Set("Authorization", "Bearer "+p.token)
		}
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}
}

// Close releases provider resources.
func (p *KubernetesProvider) Close() error {
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}
	return nil
}
