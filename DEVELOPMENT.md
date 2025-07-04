# Development Guide

This document describes the development workflow and quality standards for the Swit project.

## Quick Start

### Setup Development Environment

```bash
# Install development tools and Git hooks
make setup-dev
```

This will install:
- golangci-lint for code linting
- Pre-commit hooks for automatic quality checks

### Build and Test

```bash
# Run full quality checks and build
make all

# Or run individual steps
make format    # Format code
make lint      # Run linter
make vet       # Run go vet
make test      # Run tests
make build     # Build binaries
```

## Development Workflow

### 1. Code Quality

All code must pass the following quality checks before being committed:

- **Format**: Code must be formatted with `gofmt`
- **Lint**: Code must pass `golangci-lint` checks
- **Vet**: Code must pass `go vet` analysis
- **Tests**: All tests must pass

### 2. Pre-commit Hooks

The pre-commit hook automatically runs:
- `go mod tidy`
- Code formatting with `gofmt`
- `go vet` analysis  
- `golangci-lint` checks
- Tests for affected packages

### 3. Continuous Integration

Our CI pipeline runs on every push and pull request:

1. **Quality Stage**: Format, lint, and vet checks
2. **Test Stage**: Unit tests with race detection and coverage
3. **Build Stage**: Build all binaries
4. **Security Stage**: Vulnerability scanning with Trivy

## Make Targets

| Target | Description |
|--------|-------------|
| `make all` | Run full build pipeline (quality + build) |
| `make quality` | Run all quality checks |
| `make format` | Format code with gofmt |
| `make lint` | Run golangci-lint |
| `make vet` | Run go vet |
| `make test` | Run unit tests |
| `make test-coverage` | Run tests with coverage report |
| `make test-race` | Run tests with race detection |
| `make build` | Build all binaries |
| `make ci` | Run full CI pipeline |
| `make setup-dev` | Setup development environment |

## Code Standards

### Go Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write clear, self-documenting code
- Include unit tests for new functionality
- Use structured logging with zap

### Git Commit Messages

- Use conventional commit format: `type(scope): description`
- Examples:
  - `feat(user): add user authentication`
  - `fix(api): resolve rate limiting issue`
  - `docs(readme): update installation instructions`

### Testing

- Write unit tests for all new functionality
- Aim for good test coverage (check with `make test-coverage`)
- Use table-driven tests where appropriate
- Mock external dependencies

## Project Structure

```
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── switserve/        # Main server application
│   └── switauth/         # Authentication service
├── pkg/                   # Public library code
├── api/                   # API definitions (protobuf, OpenAPI)
├── scripts/              # Build and utility scripts
├── build/                # Build configurations (Docker, etc.)
└── _output/              # Build artifacts (generated)
```

## Troubleshooting

### Lint Issues

If you encounter lint issues:

1. Run `make format` to fix formatting
2. Run `make lint` to see specific issues
3. Fix issues manually or use IDE suggestions
4. Some issues may require code refactoring

### Pre-commit Hook Issues

If the pre-commit hook is causing problems:

```bash
# Temporarily skip hooks for urgent fixes
git commit --no-verify -m "urgent fix"

# Or remove and reinstall hooks
rm .git/hooks/pre-commit
make install-hooks
```

### CI Pipeline Failures

1. Check the specific stage that failed
2. Run the same commands locally:
   ```bash
   make ci  # Run full CI pipeline locally
   ```
3. Fix issues and push again

## Getting Help

- Run `make help` to see all available targets
- Check existing code for examples and patterns
- Review CI logs for detailed error messages