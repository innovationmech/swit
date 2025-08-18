# Service Discovery

This guide covers the service discovery system in Swit, which provides Consul-based service registration and discovery with configurable failure modes, health check integration, and multi-endpoint support.

## Overview

The Swit framework includes integrated service discovery through Consul, allowing services to automatically register themselves and discover other services in the network. The system supports multiple failure modes, health check integration, and graceful handling of discovery failures.

## Core Components

### ServiceDiscoveryManager Interface

The main interface for service discovery operations:

```go
type ServiceDiscoveryManager interface {
    RegisterService(ctx context.Context, registration *ServiceRegistration) error
    DeregisterService(ctx context.Context, registration *ServiceRegistration) error
    RegisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
    DeregisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
    IsHealthy(ctx context.Context) bool
}
```

### ServiceRegistration Structure

Defines service registration parameters:

```go
type ServiceRegistration struct {
    ID              string            // Unique service instance ID
    Name            string            // Service name
    Tags            []string          // Service tags
    Address         string            // Service address
    Port            int               // Service port
    Check           *HealthCheck      // Health check configuration
    Meta            map[string]string // Additional metadata
}
```

### HealthCheck Configuration

Defines health check parameters for Consul:

```go
type HealthCheck struct {
    HTTP                           string        // HTTP health check URL
    GRPC                           string        // gRPC health check
    TCP                            string        // TCP health check
    Interval                       time.Duration // Check interval
    Timeout                        time.Duration // Check timeout
    DeregisterCriticalServiceAfter time.Duration // Auto-deregistration timeout
}
```

## Basic Configuration

### Discovery Configuration

```yaml
discovery:
  enabled: true
  address: "127.0.0.1:8500"  # Consul address
  service_name: "my-service"
  tags:
    - "v1"
    - "api"
    - "production"
  failure_mode: "graceful"  # "graceful", "fail_fast", "strict"
  health_check_required: false
  registration_timeout: "30s"
```

### Failure Modes

**Graceful Mode (default):**
- Server continues startup even if discovery registration fails
- Suitable for development environments
- Non-critical discovery failures don't prevent service operation

```yaml
discovery:
  failure_mode: "graceful"
```

**Fail-Fast Mode:**
- Server startup fails if discovery registration fails
- Recommended for production environments where discovery is critical
- Ensures services are properly registered before serving traffic

```yaml
discovery:
  failure_mode: "fail_fast"
```

**Strict Mode:**
- Requires discovery health check and fails fast on any discovery issues
- Highest level of reliability
- Ensures both registration and discovery health before proceeding

```yaml
discovery:
  failure_mode: "strict"
  health_check_required: true
```

## Service Registration

### Automatic Registration

The framework automatically registers services when enabled:

```go
config := &server.ServerConfig{
    ServiceName: "user-service",
    HTTP: server.HTTPConfig{
        Port:    "8080",
        Enabled: true,
    },
    GRPC: server.GRPCConfig{
        Port:    "9080", 
        Enabled: true,
    },
    Discovery: server.DiscoveryConfig{
        Enabled:     true,
        Address:     "127.0.0.1:8500",
        ServiceName: "user-service",
        Tags:        []string{"v1", "api"},
        FailureMode: server.DiscoveryFailureModeFailFast,
    },
}

srv, err := server.NewBusinessServerCore(config, registrar, deps)
if err != nil {
    return fmt.Errorf("failed to create server: %w", err)
}

// Services are automatically registered during startup
err = srv.Start(ctx)
```

### Manual Registration

For custom registration scenarios:

```go
import "github.com/innovationmech/swit/pkg/discovery"

// Create discovery manager
manager, err := discovery.NewConsulServiceDiscoveryManager("127.0.0.1:8500")
if err != nil {
    return fmt.Errorf("failed to create discovery manager: %w", err)
}

// Create service registration
registration := &discovery.ServiceRegistration{
    ID:      "user-service-1",
    Name:    "user-service",
    Address: "192.168.1.100",
    Port:    8080,
    Tags:    []string{"v1", "http", "api"},
    Check: &discovery.HealthCheck{
        HTTP:     "http://192.168.1.100:8080/health",
        Interval: 10 * time.Second,
        Timeout:  3 * time.Second,
        DeregisterCriticalServiceAfter: 30 * time.Second,
    },
    Meta: map[string]string{
        "version":     "1.0.0",
        "environment": "production",
        "region":      "us-west-1",
    },
}

// Register service
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err = manager.RegisterService(ctx, registration)
if err != nil {
    return fmt.Errorf("failed to register service: %w", err)
}
```

### Multi-Transport Registration

Register both HTTP and gRPC endpoints:

```go
// HTTP service registration
httpRegistration := &discovery.ServiceRegistration{
    ID:      "user-service-http-1",
    Name:    "user-service",
    Address: "192.168.1.100",
    Port:    8080,
    Tags:    []string{"v1", "http", "api"},
    Check: &discovery.HealthCheck{
        HTTP:     "http://192.168.1.100:8080/health",
        Interval: 10 * time.Second,
        Timeout:  3 * time.Second,
    },
}

// gRPC service registration
grpcRegistration := &discovery.ServiceRegistration{
    ID:      "user-service-grpc-1", 
    Name:    "user-service-grpc",
    Address: "192.168.1.100",
    Port:    9080,
    Tags:    []string{"v1", "grpc", "api"},
    Check: &discovery.HealthCheck{
        GRPC:     "192.168.1.100:9080/grpc.health.v1.Health/Check",
        Interval: 10 * time.Second,
        Timeout:  3 * time.Second,
    },
}

// Register multiple endpoints
registrations := []*discovery.ServiceRegistration{
    httpRegistration,
    grpcRegistration,
}

err = manager.RegisterMultipleEndpoints(ctx, registrations)
if err != nil {
    return fmt.Errorf("failed to register endpoints: %w", err)
}
```

## Health Check Configuration

### HTTP Health Checks

```go
registration := &discovery.ServiceRegistration{
    ID:      "api-service-1",
    Name:    "api-service",
    Address: "10.0.1.100",
    Port:    8080,
    Check: &discovery.HealthCheck{
        HTTP:                           "http://10.0.1.100:8080/health",
        Interval:                       15 * time.Second,
        Timeout:                        5 * time.Second,
        DeregisterCriticalServiceAfter: 60 * time.Second,
    },
}
```

### gRPC Health Checks

```go
registration := &discovery.ServiceRegistration{
    ID:      "grpc-service-1",
    Name:    "grpc-service", 
    Address: "10.0.1.100",
    Port:    9080,
    Check: &discovery.HealthCheck{
        GRPC:                           "10.0.1.100:9080/grpc.health.v1.Health/Check",
        Interval:                       10 * time.Second,
        Timeout:                        3 * time.Second,
        DeregisterCriticalServiceAfter: 30 * time.Second,
    },
}
```

### TCP Health Checks

```go
registration := &discovery.ServiceRegistration{
    ID:      "tcp-service-1",
    Name:    "tcp-service",
    Address: "10.0.1.100", 
    Port:    5432,
    Check: &discovery.HealthCheck{
        TCP:                            "10.0.1.100:5432",
        Interval:                       30 * time.Second,
        Timeout:                        10 * time.Second,
        DeregisterCriticalServiceAfter: 90 * time.Second,
    },
}
```

### Custom Health Check Implementation

```go
type CustomHealthChecker struct {
    database *sql.DB
    redis    *redis.Client
    external *http.Client
}

func (c *CustomHealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    health := struct {
        Status      string            `json:"status"`
        Checks      map[string]string `json:"checks"`
        Timestamp   time.Time         `json:"timestamp"`
        Version     string            `json:"version"`
    }{
        Status:    "healthy",
        Checks:    make(map[string]string),
        Timestamp: time.Now(),
        Version:   "1.0.0",
    }
    
    // Check database
    if err := c.database.Ping(); err != nil {
        health.Status = "unhealthy"
        health.Checks["database"] = fmt.Sprintf("failed: %v", err)
    } else {
        health.Checks["database"] = "ok"
    }
    
    // Check Redis
    if err := c.redis.Ping(r.Context()).Err(); err != nil {
        health.Status = "unhealthy" 
        health.Checks["redis"] = fmt.Sprintf("failed: %v", err)
    } else {
        health.Checks["redis"] = "ok"
    }
    
    // Check external dependencies
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.external.com/health", nil)
    resp, err := c.external.Do(req)
    if err != nil || resp.StatusCode != 200 {
        health.Status = "unhealthy"
        health.Checks["external_api"] = "failed"
    } else {
        health.Checks["external_api"] = "ok"
        resp.Body.Close()
    }
    
    statusCode := 200
    if health.Status != "healthy" {
        statusCode = 503
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(health)
}

// Register health check endpoint
http.Handle("/health", &CustomHealthChecker{
    database: db,
    redis:    redisClient,
    external: &http.Client{Timeout: 5 * time.Second},
})
```

## Service Discovery Client

### Discovering Services

```go
import (
    "github.com/hashicorp/consul/api"
)

// Create Consul client
consulConfig := api.DefaultConfig()
consulConfig.Address = "127.0.0.1:8500"
client, err := api.NewClient(consulConfig)
if err != nil {
    return fmt.Errorf("failed to create consul client: %w", err)
}

// Discover healthy services
services, _, err := client.Health().Service("user-service", "", true, nil)
if err != nil {
    return fmt.Errorf("failed to discover services: %w", err)
}

// Use discovered services
for _, service := range services {
    endpoint := fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
    fmt.Printf("Discovered service: %s\n", endpoint)
    
    // Create client connection
    conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
    if err != nil {
        continue
    }
    defer conn.Close()
    
    // Use service...
}
```

### Service Discovery with Load Balancing

```go
type ServiceDiscoveryClient struct {
    consul   *api.Client
    services map[string][]*api.ServiceEntry
    mu       sync.RWMutex
}

func NewServiceDiscoveryClient(consulAddr string) (*ServiceDiscoveryClient, error) {
    config := api.DefaultConfig()
    config.Address = consulAddr
    
    client, err := api.NewClient(config)
    if err != nil {
        return nil, err
    }
    
    sdc := &ServiceDiscoveryClient{
        consul:   client,
        services: make(map[string][]*api.ServiceEntry),
    }
    
    // Start periodic service refresh
    go sdc.refreshServices()
    
    return sdc, nil
}

func (sdc *ServiceDiscoveryClient) GetService(serviceName string) (*api.ServiceEntry, error) {
    sdc.mu.RLock()
    services, exists := sdc.services[serviceName]
    sdc.mu.RUnlock()
    
    if !exists || len(services) == 0 {
        return nil, fmt.Errorf("no healthy instances found for service %s", serviceName)
    }
    
    // Simple round-robin load balancing
    index := rand.Intn(len(services))
    return services[index], nil
}

func (sdc *ServiceDiscoveryClient) refreshServices() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        // Discover all services
        services := []string{"user-service", "auth-service", "payment-service"}
        
        for _, serviceName := range services {
            healthyServices, _, err := sdc.consul.Health().Service(serviceName, "", true, nil)
            if err != nil {
                log.Printf("Failed to refresh service %s: %v", serviceName, err)
                continue
            }
            
            sdc.mu.Lock()
            sdc.services[serviceName] = healthyServices
            sdc.mu.Unlock()
        }
    }
}

// Usage
discoveryClient, err := NewServiceDiscoveryClient("127.0.0.1:8500")
if err != nil {
    return err
}

// Get a service instance
userService, err := discoveryClient.GetService("user-service")
if err != nil {
    return err
}

endpoint := fmt.Sprintf("%s:%d", userService.Service.Address, userService.Service.Port)
conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
```

## Advanced Service Discovery

### Service Mesh Integration

```go
type ServiceMeshConfig struct {
    EnableTLS           bool
    CertificatePath     string
    PrivateKeyPath      string
    TrustedCAPath       string
    EnableMutualTLS     bool
    ServiceIdentity     string
    UpstreamServices    []string
}

func (sdc *ServiceDiscoveryClient) GetServiceWithTLS(serviceName string, meshConfig *ServiceMeshConfig) (*grpc.ClientConn, error) {
    service, err := sdc.GetService(serviceName)
    if err != nil {
        return nil, err
    }
    
    endpoint := fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
    
    if !meshConfig.EnableTLS {
        return grpc.Dial(endpoint, grpc.WithInsecure())
    }
    
    // Load TLS certificates
    cert, err := tls.LoadX509KeyPair(meshConfig.CertificatePath, meshConfig.PrivateKeyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load client certificates: %w", err)
    }
    
    // Load CA certificate
    caCert, err := ioutil.ReadFile(meshConfig.TrustedCAPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read CA certificate: %w", err)
    }
    
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)
    
    // Create TLS config
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        ServerName:   serviceName, // Use service name for SNI
    }
    
    if meshConfig.EnableMutualTLS {
        tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
    }
    
    // Create secure gRPC connection
    creds := credentials.NewTLS(tlsConfig)
    return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
}
```

### Circuit Breaker Integration

```go
import "github.com/sony/gobreaker"

type CircuitBreakerServiceClient struct {
    discoveryClient *ServiceDiscoveryClient
    breakers        map[string]*gobreaker.CircuitBreaker
    mu              sync.RWMutex
}

func NewCircuitBreakerServiceClient(discoveryClient *ServiceDiscoveryClient) *CircuitBreakerServiceClient {
    return &CircuitBreakerServiceClient{
        discoveryClient: discoveryClient,
        breakers:        make(map[string]*gobreaker.CircuitBreaker),
    }
}

func (cbsc *CircuitBreakerServiceClient) GetServiceWithCircuitBreaker(serviceName string) (*grpc.ClientConn, error) {
    breaker := cbsc.getOrCreateBreaker(serviceName)
    
    conn, err := breaker.Execute(func() (interface{}, error) {
        return cbsc.discoveryClient.GetServiceConnection(serviceName)
    })
    
    if err != nil {
        return nil, err
    }
    
    return conn.(*grpc.ClientConn), nil
}

func (cbsc *CircuitBreakerServiceClient) getOrCreateBreaker(serviceName string) *gobreaker.CircuitBreaker {
    cbsc.mu.RLock()
    breaker, exists := cbsc.breakers[serviceName]
    cbsc.mu.RUnlock()
    
    if exists {
        return breaker
    }
    
    cbsc.mu.Lock()
    defer cbsc.mu.Unlock()
    
    // Double-check after acquiring write lock
    if breaker, exists := cbsc.breakers[serviceName]; exists {
        return breaker
    }
    
    settings := gobreaker.Settings{
        Name:        fmt.Sprintf("%s-breaker", serviceName),
        MaxRequests: 3,
        Interval:    10 * time.Second,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= 3
        },
        OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
            log.Printf("Circuit breaker %s changed state from %s to %s", name, from, to)
        },
    }
    
    breaker = gobreaker.NewCircuitBreaker(settings)
    cbsc.breakers[serviceName] = breaker
    
    return breaker
}
```

## Testing Service Discovery

### Unit Testing Discovery Manager

```go
func TestServiceRegistration(t *testing.T) {
    // Create test Consul container using testcontainers
    ctx := context.Background()
    consulContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "consul:1.15",
            ExposedPorts: []string{"8500/tcp"},
            Cmd:          []string{"consul", "agent", "-dev", "-client", "0.0.0.0"},
        },
        Started: true,
    })
    require.NoError(t, err)
    defer consulContainer.Terminate(ctx)
    
    // Get Consul address
    consulAddr, err := consulContainer.MappedPort(ctx, "8500")
    require.NoError(t, err)
    
    // Create discovery manager
    manager, err := discovery.NewConsulServiceDiscoveryManager(fmt.Sprintf("127.0.0.1:%s", consulAddr.Port()))
    require.NoError(t, err)
    
    // Test service registration
    registration := &discovery.ServiceRegistration{
        ID:      "test-service-1",
        Name:    "test-service",
        Address: "127.0.0.1",
        Port:    8080,
        Tags:    []string{"test", "v1"},
    }
    
    err = manager.RegisterService(ctx, registration)
    assert.NoError(t, err)
    
    // Verify registration
    consulClient, _ := api.NewClient(&api.Config{Address: fmt.Sprintf("127.0.0.1:%s", consulAddr.Port())})
    services, _, err := consulClient.Catalog().Service("test-service", "", nil)
    require.NoError(t, err)
    assert.Len(t, services, 1)
    assert.Equal(t, "test-service-1", services[0].ServiceID)
    
    // Test deregistration
    err = manager.DeregisterService(ctx, registration)
    assert.NoError(t, err)
    
    // Verify deregistration
    services, _, err = consulClient.Catalog().Service("test-service", "", nil)
    require.NoError(t, err)
    assert.Len(t, services, 0)
}
```

### Integration Testing with Server

```go
func TestServerWithServiceDiscovery(t *testing.T) {
    // Start test Consul
    consulContainer := startTestConsul(t)
    defer consulContainer.Terminate(context.Background())
    
    consulAddr := getConsulAddress(t, consulContainer)
    
    // Create server config with discovery
    config := server.NewServerConfig()
    config.HTTP.Port = "0" // Dynamic port
    config.GRPC.Port = "0" // Dynamic port
    config.Discovery.Enabled = true
    config.Discovery.Address = consulAddr
    config.Discovery.ServiceName = "test-service"
    config.Discovery.FailureMode = server.DiscoveryFailureModeFailFast
    
    // Create and start server
    srv, err := server.NewBusinessServerCore(config, &TestServiceRegistrar{}, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Wait for registration to complete
    time.Sleep(2 * time.Second)
    
    // Verify service is registered
    consulClient, _ := api.NewClient(&api.Config{Address: consulAddr})
    services, _, err := consulClient.Health().Service("test-service", "", true, nil)
    require.NoError(t, err)
    assert.Len(t, services, 2) // HTTP and gRPC endpoints
    
    // Verify health checks are passing
    for _, service := range services {
        for _, check := range service.Checks {
            assert.Equal(t, "passing", check.Status)
        }
    }
}
```

## Best Practices

### Service Registration

1. **Unique Service IDs** - Use unique service instance IDs to avoid conflicts
2. **Meaningful Tags** - Use descriptive tags for service filtering and discovery
3. **Health Check Configuration** - Configure appropriate health check intervals and timeouts
4. **Graceful Deregistration** - Always deregister services during shutdown
5. **Metadata Usage** - Use service metadata for additional service information

### Health Checks

1. **Lightweight Checks** - Keep health checks fast and lightweight
2. **Dependency Checking** - Check critical dependencies in health checks
3. **Timeout Configuration** - Set reasonable timeouts for health checks
4. **Auto-deregistration** - Configure auto-deregistration for failed services
5. **Status Endpoints** - Provide detailed status information in health endpoints

### Production Deployment

1. **High Availability** - Deploy Consul in HA mode for production
2. **Security Configuration** - Enable ACLs and TLS for Consul security
3. **Monitoring** - Monitor Consul health and service registration status
4. **Backup Strategy** - Implement backup strategies for Consul data
5. **Network Segmentation** - Use appropriate network segmentation for Consul traffic

### Failure Handling

1. **Failure Mode Selection** - Choose appropriate failure modes based on requirements
2. **Retry Logic** - Implement retry logic for transient discovery failures
3. **Fallback Mechanisms** - Provide fallback mechanisms when discovery is unavailable
4. **Circuit Breakers** - Use circuit breakers for service-to-service communication
5. **Graceful Degradation** - Design services to degrade gracefully when dependencies are unavailable

This service discovery guide covers all aspects of the integrated Consul-based service discovery system, from basic configuration to advanced patterns, testing strategies, and production deployment considerations.