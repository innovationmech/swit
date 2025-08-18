# Installation

This guide will help you install the Swit Go Microservice Framework and set up your development environment.

## Requirements

Before installing Swit, make sure you have the following requirements:

- **Go 1.24+** - Modern Go version with generics support
- **Git** - For framework and example code management
- **Make** - For building and development commands (optional but recommended)

### Verify Go Installation

Check your Go version:

```bash
go version
```

You should see Go version 1.24 or higher. If you don't have Go installed, visit [https://golang.org/doc/install](https://golang.org/doc/install).

## Install Swit Framework

### Option 1: Using go get (Recommended)

Install the framework directly in your project:

```bash
go get github.com/innovationmech/swit
```

### Option 2: Clone and Build from Source

For development or customization:

```bash
# Clone the repository
git clone https://github.com/innovationmech/swit.git
cd swit

# Build the framework
make build

# Run tests to verify installation
make test
```

## Verify Installation

Create a simple test to verify the installation:

```bash
mkdir test-swit
cd test-swit
go mod init test-swit
```

Create a `main.go` file:

```go
package main

import (
    "fmt"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := &server.ServerConfig{
        Name:    "test-service",
        Version: "1.0.0",
    }
    fmt.Printf("Swit framework loaded successfully!\n")
    fmt.Printf("Service: %s v%s\n", config.Name, config.Version)
}
```

Run the test:

```bash
go run main.go
```

You should see:
```
Swit framework loaded successfully!
Service: test-service v1.0.0
```

## Development Environment Setup

For framework development, set up the complete development environment:

```bash
# Clone the repository
git clone https://github.com/innovationmech/swit.git
cd swit

# Set up development environment
make setup-dev

# Verify development setup
make ci
```

This will:
- Install all development dependencies
- Generate protocol buffer code
- Run all tests
- Verify code quality

## IDE Setup

### VS Code

For the best development experience with VS Code:

1. Install the Go extension
2. Install the Protocol Buffer extension (for API development)
3. Add these settings to your workspace:

```json
{
    "go.toolsEnvVars": {
        "GOPROXY": "https://proxy.golang.org,direct"
    },
    "go.useLanguageServer": true,
    "go.lintTool": "golint",
    "go.formatTool": "gofmt"
}
```

### GoLand/IntelliJ

1. Ensure Go plugin is installed and enabled
2. Set GOROOT to your Go installation path
3. Set GOPATH if using Go modules outside module mode

## Troubleshooting

### Common Issues

**Go version too old:**
```
go get: module github.com/innovationmech/swit requires Go 1.24
```
Solution: Upgrade Go to version 1.24 or higher.

**Module not found:**
```
go: github.com/innovationmech/swit@latest: module github.com/innovationmech/swit: Get "https://proxy.golang.org/github.com/innovationmech/swit/@v/list": dial tcp: lookup proxy.golang.org: no such host
```
Solution: Check your network connection or configure GOPROXY.

**Build failures:**
```
make: *** No rule to make target 'build'. Stop.
```
Solution: Make sure you're in the correct directory with a Makefile.

### Getting Help

- Check the [troubleshooting guide](./troubleshooting.md)
- Review [examples](/en/examples/)
- Visit our [community forums](/en/community/)

## Next Steps

Now that you have Swit installed:

1. [Get started](./getting-started.md) with your first service
2. Explore [examples](/en/examples/) for different use cases
3. Read the [API documentation](/en/api/) for detailed reference

## System Requirements

### Minimum Requirements
- **CPU:** 1 core
- **RAM:** 512MB
- **Disk:** 100MB free space
- **Network:** Internet connection for dependencies

### Recommended Requirements
- **CPU:** 2+ cores
- **RAM:** 2GB+
- **Disk:** 1GB+ free space
- **Network:** High-speed internet for faster builds