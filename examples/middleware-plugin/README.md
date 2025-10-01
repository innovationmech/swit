# Middleware Plugin System Example

This example demonstrates the custom middleware plugin system for Swit framework.

## Overview

The plugin system allows you to:
- Create custom middleware as standalone plugins
- Load plugins dynamically at runtime
- Manage plugin lifecycle (Initialize, Start, Stop, Shutdown)
- Hot-reload plugins without restarting the application

## Architecture

### Core Components

1. **Plugin Interface**: Defines the contract for all plugins
   - `Initialize()`: Prepare plugin with configuration
   - `Start()`: Begin plugin operation
   - `Stop()`: Halt plugin gracefully
   - `Shutdown()`: Final cleanup
   - `CreateMiddleware()`: Create middleware instances
   - `HealthCheck()`: Verify plugin health

2. **PluginLoader**: Load plugins from various sources
   - File system (.so files)
   - Factories (statically compiled)
   - Directories (batch loading)

3. **PluginRegistry**: Manage registered plugins
   - Registration and discovery
   - Lifecycle coordination
   - Health monitoring

## Example: Audit Plugin

The audit plugin provides compliance-ready audit logging for message processing.

### Features

- Logs all message processing events
- Captures success/failure status
- Records processing duration
- Includes message metadata
- Writes to configurable audit file

### Plugin Structure

```go
type AuditPlugin struct {
    *messaging.BasePlugin
    auditFile *os.File
    logger    *log.Logger
}

func NewPlugin() (messaging.Plugin, error) {
    // Plugin factory function
}
```

### Middleware Implementation

```go
type AuditMiddleware struct {
    plugin *AuditPlugin
}

func (am *AuditMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
    // Wrap handler with audit logging
}
```

## Usage

### 1. Static Loading (Factory-based)

```go
// Create plugin loader
loader := messaging.NewPluginLoader(messaging.GlobalPluginRegistry)

// Load audit plugin from factory
factory := func() (messaging.Plugin, error) {
    return NewPlugin()
}

config := map[string]interface{}{
    "audit_file": "/var/log/swit/audit.log",
}

err := loader.LoadFromFactory(ctx, "audit", factory, config)
```

### 2. Dynamic Loading (File-based)

```bash
# Build plugin as shared object
go build -buildmode=plugin -o audit.so audit_plugin.go

# Load at runtime
loader.LoadFromFile(ctx, "./audit.so", config)
```

### 3. Directory Discovery

```go
// Create discovery instance
discovery := messaging.NewPluginDiscovery(
    "/etc/swit/plugins",
    "/usr/local/lib/swit/plugins",
)

// Discover and load all plugins
configs := map[string]map[string]interface{}{
    "audit": {
        "audit_file": "/var/log/swit/audit.log",
    },
}

err := discovery.DiscoverAndLoad(ctx, loader, configs)
```

### 4. Using Plugin Middleware

```go
// Get plugin from registry
plugin, err := messaging.GlobalPluginRegistry.GetPlugin("audit")

// Create middleware instance
middleware, err := plugin.CreateMiddleware()

// Add to middleware chain
chain := messaging.NewMiddlewareChain()
chain.Add(middleware)

// Build handler with middleware
handler := chain.Build(yourHandler)
```

## Configuration

### Audit Plugin Configuration

| Key | Type | Description | Default |
|-----|------|-------------|---------|
| `audit_file` | string | Path to audit log file | `/tmp/audit.log` |

## Lifecycle Management

### Start All Plugins

```go
ctx := context.Background()
err := messaging.GlobalPluginRegistry.StartAll(ctx)
```

### Stop All Plugins

```go
err := messaging.GlobalPluginRegistry.StopAll(ctx)
```

### Health Check

```go
results := messaging.GlobalPluginRegistry.HealthCheckAll(ctx)
for name, err := range results {
    if err != nil {
        log.Printf("Plugin %s unhealthy: %v", name, err)
    }
}
```

## Creating Your Own Plugin

### Step 1: Define Plugin Structure

```go
type MyPlugin struct {
    *messaging.BasePlugin
    // Add your fields
}
```

### Step 2: Implement Plugin Interface

```go
func NewPlugin() (messaging.Plugin, error) {
    metadata := messaging.PluginMetadata{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My custom plugin",
        Author:      "Your Name",
    }
    
    return &MyPlugin{
        BasePlugin: messaging.NewBasePlugin(metadata),
    }, nil
}

func (mp *MyPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
}

func (mp *MyPlugin) CreateMiddleware() (messaging.Middleware, error) {
    return &MyMiddleware{plugin: mp}, nil
}
```

### Step 3: Implement Middleware

```go
type MyMiddleware struct {
    plugin *MyPlugin
}

func (mm *MyMiddleware) Name() string {
    return "my-middleware"
}

func (mm *MyMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
    return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
        // Pre-processing
        
        err := next.Handle(ctx, message)
        
        // Post-processing
        
        return err
    })
}
```

### Step 4: Build and Use

#### Static (built-in):
```go
loader.LoadFromFactory(ctx, "my-plugin", NewPlugin, config)
```

#### Dynamic (.so file):
```bash
go build -buildmode=plugin -o my-plugin.so my_plugin.go
```

## Best Practices

1. **Lifecycle Management**
   - Always implement proper cleanup in `Shutdown()`
   - Use context for cancellation signals
   - Handle errors gracefully

2. **Configuration**
   - Validate configuration in `Initialize()`
   - Use sensible defaults
   - Document all configuration options

3. **Thread Safety**
   - Protect shared state with mutexes
   - Use async operations for non-blocking work
   - Avoid blocking in middleware

4. **Error Handling**
   - Set plugin state to `PluginStateFailed` on critical errors
   - Return descriptive errors
   - Log errors appropriately

5. **Health Checks**
   - Implement meaningful health checks
   - Check external dependencies
   - Return quickly to avoid timeouts

## Testing

Run the example:

```bash
cd examples/middleware-plugin
go run main.go
```

Run tests:

```bash
go test -v
```

## Security Considerations

1. **Plugin Loading**
   - Only load plugins from trusted sources
   - Verify plugin signatures if possible
   - Use file permissions to restrict plugin directories

2. **Configuration**
   - Sanitize configuration input
   - Avoid storing secrets in config
   - Use environment variables for sensitive data

3. **Isolation**
   - Plugins run in the same process space
   - A malicious plugin can access all memory
   - Consider sandboxing for untrusted plugins

## Performance

- Plugin loading has overhead (one-time cost)
- Middleware execution should be fast
- Use async operations for I/O
- Benchmark your plugins under load

## Troubleshooting

### Plugin fails to load

- Check plugin exports `NewPlugin` function
- Verify function signature matches
- Check for initialization errors

### Middleware not executing

- Verify plugin is started
- Check middleware is added to chain
- Ensure chain is built correctly

### Health check fails

- Check plugin state
- Verify external dependencies
- Review plugin logs

## References

- [Plugin Design Specification](../../docs/architecture/plugin-system.md)
- [Middleware Architecture](../../docs/architecture/middleware.md)
- [API Documentation](../../docs/api/messaging.md)

