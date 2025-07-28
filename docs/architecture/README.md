# Architecture Documentation

This directory contains architectural documentation for the SWIT project.

## Documents

## Architecture Overview

The SWIT project follows a microservice architecture with clean separation of concerns:

- **switserve** (port 9000) - Main user service
- **switauth** (port 8090) - Authentication service  
- **switctl** - Command-line control tool

### Key Principles

1. **Dependency Injection**: Manual DI following Go community best practices
2. **Clean Architecture**: Clear separation between layers (Handler → Service → Repository)
3. **Interface-based Design**: Loose coupling through interfaces
4. **Testability**: All components easily testable in isolation

For detailed implementation guidance, see the individual documentation files in this directory.