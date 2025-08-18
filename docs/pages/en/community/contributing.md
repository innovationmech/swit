# Contributing to Swit

Thank you for your interest in contributing to the Swit microservice framework! This guide will help you get started with contributing code, documentation, or reporting issues.

## Table of Contents

- [Getting Started](#getting-started)
- [Ways to Contribute](#ways-to-contribute)
- [Development Setup](#development-setup)
- [Code Contribution Process](#code-contribution-process)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation Contributions](#documentation-contributions)
- [Issue Reporting](#issue-reporting)
- [Pull Request Process](#pull-request-process)
- [Community Guidelines](#community-guidelines)

## Getting Started

Before you start contributing, please:

1. **Read the [Code of Conduct](./code_of_conduct.md)** - All contributors must follow our community guidelines
2. **Check existing issues** - Look for existing issues or feature requests that interest you
3. **Join discussions** - Participate in issue discussions to understand the project direction
4. **Start small** - Begin with small bug fixes or documentation improvements

## Ways to Contribute

### Code Contributions
- **Bug fixes** - Fix reported issues and improve stability
- **New features** - Implement new framework capabilities or enhancements
- **Performance improvements** - Optimize existing code for better performance
- **Example services** - Create new examples demonstrating framework usage
- **Framework components** - Add new transport layers, middleware, or utilities

### Documentation
- **API documentation** - Improve code documentation and API references
- **Guides and tutorials** - Write comprehensive guides for framework features
- **Example documentation** - Document example services and use cases
- **Translation** - Help translate documentation to other languages

### Testing
- **Unit tests** - Write tests for new features or improve existing test coverage
- **Integration tests** - Create end-to-end tests for framework components
- **Performance tests** - Develop benchmarks and performance regression tests

### Community Support
- **Issue triage** - Help categorize and reproduce reported issues
- **Code review** - Review pull requests from other contributors
- **Community support** - Help answer questions from other users

## Development Setup

### Prerequisites

- **Go 1.24+** - Latest Go version with generics support
- **Git** - Version control system
- **Make** - Build automation tool
- **Docker** (optional) - For building container images

### Environment Setup

1. **Fork and clone the repository:**
   ```bash
   # Fork the repository on GitHub first, then clone your fork
   git clone https://github.com/YOUR_USERNAME/swit.git
   cd swit
   
   # Add upstream remote
   git remote add upstream https://github.com/innovationmech/swit.git
   ```

2. **Set up development environment:**
   ```bash
   # Install development tools and pre-commit hooks
   make setup-dev
   ```

3. **Verify setup:**
   ```bash
   # Run full build pipeline to ensure everything works
   make all
   ```

## Code Contribution Process

### 1. Find or Create an Issue

- **Check existing issues** - Look for issues labeled `good first issue` or `help wanted`
- **Create new issues** - For bugs or feature requests, create a detailed issue first
- **Discuss approach** - For major changes, discuss your approach in the issue

### 2. Create a Feature Branch

```bash
# Ensure you're on the main branch and up to date
git checkout master
git pull upstream master

# Create a new feature branch
git checkout -b feature/your-feature-name
# or for bug fixes
git checkout -b fix/issue-description
```

### 3. Make Your Changes

- **Write clean code** - Follow Go best practices and existing code patterns
- **Add tests** - Include unit tests for new functionality
- **Update documentation** - Update relevant documentation and examples
- **Test thoroughly** - Run the full test suite and verify your changes

### 4. Commit Your Changes

```bash
# Stage your changes
git add .

# Commit with a descriptive message (follows conventional commits)
git commit -m "feat(transport): add HTTP/2 support for HTTP transport

- Implement HTTP/2 server configuration options  
- Add HTTP/2 compatibility for existing HTTP handlers
- Update documentation with HTTP/2 examples
- Include unit tests for HTTP/2 functionality

Closes #123"
```

### 5. Push and Create Pull Request

```bash
# Push to your fork
git push origin feature/your-feature-name

# Create pull request through GitHub UI
```

## Code Standards

### Go Code Style

- **Follow standard Go conventions** - Use `gofmt`, `goimports`, and `go vet`
- **Write self-documenting code** - Use clear variable names and function signatures
- **Add code comments** - Document exported functions and complex logic
- **Handle errors properly** - Always check and handle errors appropriately
- **Use structured logging** - Use zap logger throughout the codebase

### Commit Message Format

Use conventional commit format for clear history:

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat` - New features
- `fix` - Bug fixes
- `docs` - Documentation changes
- `style` - Code style changes (formatting, etc.)
- `refactor` - Code refactoring
- `test` - Adding or fixing tests
- `chore` - Maintenance tasks

**Examples:**
```
feat(server): add graceful shutdown support
fix(transport): resolve gRPC connection leak
docs(guide): update installation instructions
test(pkg/server): add unit tests for server lifecycle
```

### Code Organization

- **Package structure** - Follow existing package organization patterns
- **Interfaces** - Define clear interfaces for extensibility
- **Error types** - Create custom error types for different failure modes
- **Configuration** - Use structured configuration with validation
- **Dependency injection** - Follow the established DI patterns

## Testing Guidelines

### Unit Tests

```go
// Example unit test structure
func TestServiceRegistration(t *testing.T) {
    tests := []struct {
        name    string
        setup   func() (*Service, error)
        want    error
        wantErr bool
    }{
        {
            name: "successful registration",
            setup: func() (*Service, error) {
                return NewService("test"), nil
            },
            want:    nil,
            wantErr: false,
        },
        // Add more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service, err := tt.setup()
            require.NoError(t, err)
            
            // Test the functionality
            err = service.Register()
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests

- **Use test containers** - For database and external service dependencies  
- **Test real scenarios** - Test complete request/response cycles
- **Clean up resources** - Always clean up test resources properly
- **Test timeouts** - Verify timeout and cancellation behavior

### Test Coverage

```bash
# Generate coverage report
make test-coverage

# View coverage in browser
go tool cover -html=coverage.out
```

Aim for:
- **80%+ coverage** for core framework components
- **100% coverage** for critical path code
- **Meaningful tests** - Focus on behavior, not just line coverage

## Documentation Contributions

### API Documentation

- **Godoc comments** - Document all exported functions and types
- **Examples** - Include code examples in documentation
- **Error documentation** - Document possible errors and their meanings

```go
// ProcessRequest handles incoming service requests with validation and error handling.
// It validates the request format, processes the business logic, and returns
// a properly formatted response.
//
// Parameters:
//   - ctx: Request context for cancellation and timeouts
//   - req: The service request to process
//
// Returns:
//   - *Response: Processed response data
//   - error: Processing error, wrapped with context information
//
// Errors:
//   - ErrInvalidRequest: When request validation fails
//   - ErrServiceUnavailable: When required dependencies are unavailable
//   - context.DeadlineExceeded: When processing timeout is reached
//
// Example:
//   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//   defer cancel()
//   
//   response, err := service.ProcessRequest(ctx, &Request{Data: "example"})
//   if err != nil {
//       return fmt.Errorf("request processing failed: %w", err)
//   }
func ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
    // Implementation...
}
```

### Guide Documentation

- **Clear structure** - Use consistent headings and organization
- **Working examples** - Provide complete, runnable examples
- **Step-by-step instructions** - Break complex procedures into clear steps
- **Screenshots/diagrams** - Include visual aids where helpful

### Documentation Standards

- **Markdown format** - Use consistent markdown formatting
- **Code highlighting** - Use appropriate language tags for code blocks
- **Links** - Include relevant cross-references and external links
- **Version compatibility** - Indicate version requirements where relevant

## Issue Reporting

### Bug Reports

When reporting bugs, include:

1. **Clear description** - Describe what happened vs. what was expected
2. **Reproduction steps** - Provide minimal steps to reproduce the issue
3. **Environment details** - Go version, OS, framework version
4. **Code samples** - Include minimal code that demonstrates the issue
5. **Error messages** - Include complete error messages and stack traces

**Bug Report Template:**
```markdown
## Bug Description
Brief description of the bug.

## Steps to Reproduce
1. Step one
2. Step two
3. Step three

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened.

## Environment
- Go version: 1.24.0
- OS: Ubuntu 22.04
- Framework version: v1.0.0

## Code Sample
```go
// Minimal code that reproduces the issue
```

## Error Messages
```
Complete error message and stack trace
```
```

### Feature Requests

For feature requests, include:

1. **Use case description** - Explain the problem you're trying to solve
2. **Proposed solution** - Describe your suggested approach
3. **Alternatives considered** - Mention other approaches you've considered
4. **Implementation ideas** - Share ideas about how it might be implemented

## Pull Request Process

### Before Creating a Pull Request

1. **Sync with upstream** - Ensure your branch is up to date with the main branch
2. **Run full test suite** - Execute `make ci` to run all quality checks
3. **Update documentation** - Include relevant documentation updates
4. **Write meaningful tests** - Add tests for new functionality

### Pull Request Description

Include in your PR description:

- **Summary** - Brief description of changes
- **Related issues** - Reference related issues with `Closes #123`
- **Changes made** - List the main changes
- **Testing** - Describe how you tested the changes
- **Breaking changes** - Highlight any breaking changes

**PR Template:**
```markdown
## Summary
Brief description of the changes in this PR.

## Related Issues
Closes #123
Relates to #456

## Changes Made
- Added new HTTP/2 support for transport layer
- Updated configuration options for HTTP/2
- Added unit tests for HTTP/2 functionality
- Updated documentation with HTTP/2 examples

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass  
- [ ] Manual testing completed
- [ ] Performance impact assessed

## Breaking Changes
- [ ] This PR contains breaking changes
- [ ] This PR is backward compatible

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review of code completed
- [ ] Code is commented, particularly in hard-to-understand areas
- [ ] Corresponding changes to documentation made
- [ ] Tests added that prove the fix is effective or feature works
- [ ] New and existing unit tests pass locally
```

### Review Process

1. **Automated checks** - CI pipeline must pass
2. **Code review** - At least one maintainer review required
3. **Address feedback** - Respond to review comments promptly
4. **Update as needed** - Make requested changes
5. **Final approval** - Maintainer approval for merge

## Community Guidelines

### Communication

- **Be respectful** - Treat all community members with respect
- **Be constructive** - Provide helpful feedback and suggestions
- **Be patient** - Reviews and responses may take time
- **Ask questions** - Don't hesitate to ask for help or clarification

### Collaboration

- **Share knowledge** - Help other contributors learn and improve
- **Give credit** - Acknowledge others' contributions
- **Be inclusive** - Welcome contributors from all backgrounds
- **Follow up** - Respond to feedback and continue conversations

### Quality Standards

- **Test your changes** - Ensure your code works before submitting
- **Document your work** - Help others understand your contributions
- **Follow conventions** - Use established patterns and styles
- **Review thoroughly** - Take time to review your own changes

## Getting Help

If you need help contributing:

1. **Check existing documentation** - Review guides and examples
2. **Search issues** - Look for similar questions or problems
3. **Ask in discussions** - Use GitHub discussions for general questions
4. **Join community channels** - Participate in community forums
5. **Tag maintainers** - For urgent issues, respectfully tag maintainers

## Recognition

We appreciate all contributions to the Swit project! Contributors will be:

- **Listed in contributors** - Recognized in project documentation
- **Credited in releases** - Major contributions noted in release notes
- **Invited to community** - Access to contributor discussions and events

Thank you for helping make Swit better for everyone! 🚀