# Community and Contributing

Welcome to the Swit framework community! We are an open, inclusive open-source community that welcomes all developers to participate in the project's development and improvement.

## 🤝 How to Get Involved

### Ways to Contribute

1. **Framework Core Development** - Improve `pkg/server/` and `pkg/transport/` components
2. **Example Services** - Add new examples in the `examples/` directory
3. **Documentation Improvements** - Enhance framework documentation and guides
4. **Test Coverage** - Add tests for framework components and examples
5. **Bug Reports** - Report framework functionality issues
6. **Feature Requests** - Suggest new framework capabilities

### Getting Started Steps

1. **Fork the repository** and clone your fork
2. **Setup development environment**: `make setup-dev`
3. **Run tests** to ensure everything works: `make test`
4. **Make changes** following existing patterns
5. **Add tests** for new functionality
6. **Submit a PR** with a clear description

## 📋 Contributing Guidelines

### Code Contributions

#### Framework Core Development
```bash
# Clone project
git clone https://github.com/innovationmech/swit.git
cd swit

# Setup development environment
make setup-dev

# Create feature branch
git checkout -b feature/my-new-feature

# Make changes
vim pkg/server/your-changes.go

# Run tests
make test

# Commit changes
git commit -m "feat: add new framework feature"
git push origin feature/my-new-feature
```

#### Example Service Development
```bash
# Create new example
mkdir examples/my-example
cd examples/my-example

# Implement example
vim main.go
vim README.md

# Test example
go run main.go

# Add to build system if needed
vim scripts/mk/build.mk
```

### Documentation Contributions

#### Improving Existing Documentation
```bash
# Edit documentation
vim docs/guide/your-topic.md

# Local preview (if documentation system is set up)
make docs-serve

# Commit changes
git add docs/
git commit -m "docs: improve topic documentation"
```

#### Adding New Documentation
```bash
# Create new documentation
mkdir docs/advanced/
vim docs/advanced/new-topic.md

# Update navigation
vim docs/.vitepress/config.ts
```

### Code Standards

#### Go Code Standards
- Use `gofmt` to format code
- Follow official Go code style
- Add documentation comments for public functions
- Use meaningful variable and function names

```go
// UserService provides user management functionality
type UserService struct {
    db     *gorm.DB
    logger *zap.Logger
}

// CreateUser creates a new user
// Parameters:
//   - ctx: request context
//   - user: user information
// Returns:
//   - *User: created user
//   - error: error information
func (s *UserService) CreateUser(ctx context.Context, user *User) (*User, error) {
    if err := s.validateUser(user); err != nil {
        return nil, fmt.Errorf("user validation failed: %w", err)
    }
    
    // Implementation logic...
    return user, nil
}
```

#### Commit Message Standards
We use [Conventional Commits](https://www.conventionalcommits.org/) specification:

```bash
# Feature addition
git commit -m "feat: add user management API"

# Bug fix
git commit -m "fix: resolve database connection leak issue"

# Documentation update
git commit -m "docs: update quick start guide"

# Performance optimization
git commit -m "perf: optimize database query performance"

# Code refactoring
git commit -m "refactor: restructure transport layer architecture"

# Test addition
git commit -m "test: add user service unit tests"
```

### Testing Requirements

#### Unit Testing
```go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name        string
        user        *User
        wantErr     bool
        expectedErr string
    }{
        {
            name: "valid user",
            user: &User{
                Name:  "John Doe",
                Email: "john@example.com",
            },
            wantErr: false,
        },
        {
            name: "invalid email",
            user: &User{
                Name:  "Jane Smith",
                Email: "invalid-email",
            },
            wantErr:     true,
            expectedErr: "invalid email format",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := setupTestService()
            
            result, err := service.CreateUser(context.Background(), tt.user)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErr)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.NotZero(t, result.ID)
            }
        })
    }
}
```

#### Integration Testing
```go
func TestHTTPIntegration(t *testing.T) {
    // Setup test server
    config := &server.ServerConfig{
        HTTP: server.HTTPConfig{
            Port:     "0", // Random port
            Enabled:  true,
            TestMode: true,
        },
    }
    
    service := &TestService{}
    srv, err := server.NewBusinessServerCore(config, service, nil)
    require.NoError(t, err)
    
    go srv.Start(context.Background())
    defer srv.Shutdown()
    
    // Wait for server startup
    time.Sleep(100 * time.Millisecond)
    
    // Test endpoints
    baseURL := fmt.Sprintf("http://%s", srv.GetHTTPAddress())
    
    t.Run("health check", func(t *testing.T) {
        resp, err := http.Get(baseURL + "/health")
        require.NoError(t, err)
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    })
    
    t.Run("user creation", func(t *testing.T) {
        user := map[string]interface{}{
            "name":  "Test User",
            "email": "test@example.com",
        }
        
        body, _ := json.Marshal(user)
        resp, err := http.Post(baseURL+"/api/v1/users", "application/json", bytes.NewReader(body))
        
        require.NoError(t, err)
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
    })
}
```

## 🐛 Issue Reporting

### How to Report Bugs

1. **Search existing issues** - Check if similar issues already exist
2. **Use issue template** - Fill out the complete issue report template
3. **Provide detailed information** - Include reproduction steps, environment info, error logs
4. **Add labels** - Choose appropriate labels (bug, enhancement, question, etc.)

### Bug Report Template

```markdown
## Issue Description
Brief description of the problem encountered

## Reproduction Steps
1. Run `make build`
2. Execute `./bin/swit-serve`
3. Access `http://localhost:9000/api/v1/users`
4. See error...

## Expected Behavior
Description of what you expected to happen

## Actual Behavior
Description of what actually happened

## Environment Information
- Operating System: macOS 13.0
- Go Version: 1.23.12
- Swit Version: v1.0.0
- Database: MySQL 8.0

## Error Logs
```
Error log content...
```

## Additional Information
Any other information that helps understand the issue
```

### Feature Request Template

```markdown
## Feature Description
Clear and concise description of the feature you want

## Use Case
Description of the use case and necessity for this feature

## Proposed Solution
Description of how you'd like this feature to be implemented

## Alternative Solutions
Description of other solutions you've considered

## Additional Context
Any other relevant information or screenshots
```

## 🎯 Development Roadmap

### Current Version (v1.0)
- ✅ Core server framework
- ✅ HTTP and gRPC transport layer
- ✅ Dependency injection system
- ✅ Basic middleware support
- ✅ Service discovery integration

### Next Version (v1.1)
- 🔄 Enhanced monitoring and metrics
- 🔄 More middleware options
- 🔄 Configuration hot reloading
- 🔄 Distributed tracing support

### Future Plans (v2.0)
- 📋 Service mesh integration
- 📋 Automated testing tools
- 📋 Performance analysis tools
- 📋 Cloud-native deployment support

## 📢 Community Resources

### Official Resources
- **GitHub Repository**: [innovationmech/swit](https://github.com/innovationmech/swit)
- **Issue Tracking**: [GitHub Issues](https://github.com/innovationmech/swit/issues)
- **Release Notes**: [GitHub Releases](https://github.com/innovationmech/swit/releases)
- **Contributing Guide**: [CONTRIBUTING.md](https://github.com/innovationmech/swit/blob/master/CONTRIBUTING.md)

### Community Communication
- **Discussion Forum**: [GitHub Discussions](https://github.com/innovationmech/swit/discussions)
- **Technical Support**: Get help through GitHub Issues
- **Feature Suggestions**: Submit via GitHub Issues or Discussions

### Community Guidelines
We are committed to creating an open, inclusive community environment. Please read and follow our [Code of Conduct](https://github.com/innovationmech/swit/blob/master/CODE_OF_CONDUCT.md).

#### Core Values
- **Inclusivity** - Welcome developers from all backgrounds
- **Respect** - Respect different perspectives and experiences
- **Collaboration** - Work together to improve the project
- **Learning** - Learn from and grow with each other
- **Professionalism** - Maintain professional and constructive discussions

### Contributor Recognition

We appreciate all developers who contribute to the Swit framework. Contributors will be recognized in:

- **README.md** contributors section
- **CONTRIBUTORS.md** detailed contribution records
- **Release notes** special thanks
- **GitHub contribution graph** showing contribution activity

## 🎁 Rewards and Incentives

### Contribution Rewards
- **First-time contributors** - Special badge and welcome gift
- **Core contributors** - Project maintainer privileges
- **Documentation experts** - Documentation team certification
- **Test champions** - Quality assurance team certification

### Growth Opportunities
- **Skill improvement** - Practice Go programming in real projects
- **Open source experience** - Build open source contribution record
- **Network building** - Connect with other developers
- **Career development** - Enhance resume and professional skills

## 📚 Learning Resources

### Framework Learning
- **Official Documentation** - Complete framework documentation
- **Example Code** - Practical code examples
- **Video Tutorials** - Framework usage tutorials (planned)
- **Blog Articles** - In-depth technical articles

### Go Language Learning
- [Go Official Documentation](https://golang.org/doc/)
- [Go by Example](https://gobyexample.com/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [The Go Programming Language](https://www.gopl.io/)

### Microservices Learning
- [Microservices.io](https://microservices.io/)
- [12-Factor App](https://12factor.net/)
- [Cloud Native Computing Foundation](https://www.cncf.io/)
- [gRPC Official Documentation](https://grpc.io/docs/)

Join us in building a better microservice framework!