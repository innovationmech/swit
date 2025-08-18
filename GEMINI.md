# Gemini Code Assistant Context

This document provides context for the Gemini code assistant to understand the `swit` project.

## Project Overview

`swit` is a comprehensive microservice framework for Go. It provides a unified, production-ready foundation for building scalable microservices. The framework includes a base server, a unified transport layer for HTTP and gRPC, a dependency injection system, and various tools for rapid development.

The project is structured as follows:

-   `api/`: Contains Protocol Buffer (Protobuf) definitions for the services.
-   `cmd/`: Contains the main applications for the services (`swit-serve`, `swit-auth`, `switctl`).
-   `internal/`: Contains the internal code for the services, including handlers, services, and repositories.
-   `pkg/`: Contains the core framework packages, such as the server, transport, and discovery mechanisms.
-   `docs/`: Contains project documentation.
-   `Makefile`: Provides a set of commands for building, testing, and running the project.

The key technologies used in this project are:

-   **Go**: The primary programming language.
-   **Gin**: The HTTP web framework.
-   **gRPC**: The RPC framework.
-   **Consul**: For service discovery.
-   **Viper**: For configuration management.
-   **Cobra**: For creating powerful CLI applications.
-   **GORM**: The ORM library for Go.
-   **Docker**: For containerization.

## Building and Running

The project uses a `Makefile` to streamline the development process. Here are some of the most common commands:

-   `make all`: Builds all services and generates all necessary code (proto, swagger, etc.). This is the default goal.
-   `make build`: Builds all services.
-   `make test`: Runs all tests.
-   `make docker`: Builds Docker images for all services.
-   `make run-serve`: Runs the `swit-serve` service.
-   `make run-auth`: Runs the `swit-auth` service.
-   `make clean`: Removes all generated files and build artifacts.

To run the services, you can use the `make` commands above, or you can run the individual services directly:

```bash
# Run the user management service
go run ./cmd/swit-serve

# Run the authentication service
go run ./cmd/swit-auth
```

## Development Conventions

The project follows standard Go development conventions. Some specific conventions to note are:

-   **Dependency Injection**: The project uses a dependency injection container to manage dependencies. Services are defined as interfaces and injected into the handlers.
-   **Interface-based Design**: Services and repositories are defined as interfaces to allow for easy mocking and testing.
-   **Configuration**: Configuration is managed through a `swit.yaml` file and environment variables, using the Viper library.
-   **API Development**: The project uses a "API-first" approach, with API definitions stored in Protobuf files in the `api/` directory. The `buf` tool is used to manage the Protobuf lifecycle.
-   **Testing**: The project has a comprehensive test suite, including unit tests, integration tests, and end-to-end tests. The `testify` library is used for assertions and mocking.
-   **Documentation**: The project has extensive documentation in the `docs/` directory, including a service development guide and architecture overviews.
