# Swit

Swit is a command-line tool designed to serve as a backend application and control system. It is built using Go and utilizes the Gin web framework for handling HTTP requests. The project includes features such as health checks, user management, and server control, with data persistence handled through a MySQL database using GORM.

## Key features:
- Health check endpoint for monitoring application status
- User registration and retrieval functionality
- Server control (start, stop, health check) via CLI
- GitHub Actions workflow for automated building and testing
- Makefile for easy project management and building

Swit is designed to be a robust and scalable backend service, suitable for various applications requiring user management, health monitoring, and remote control capabilities.

## Components

1. swit-serve: The main server application
2. switctl: Command-line tool for controlling the server

## Getting Started

### Prerequisites

- Go 1.20 or higher
- MySQL database

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/innovationmech/swit.git
   cd swit
   ```

2. Configure the application:
   Edit the `swit.yaml` file to set your database and server configurations.

### Building

To build both the server and CLI applications, run:
```
make
```
The binary will be created in the `_output` directory.

### Running

To start the server:
```
./_output/swit-serve/swit-serve serve
```
## Usage

Swit provides the following commands:

- `serve`: Start the HTTP server
- `version`: Print the version information

For more details, run:
```
./_output/swit-serve/swit-serve --help
```

## API Endpoints

- `GET /health`: Health check endpoint
- `POST /users/`: Register a new user
- `GET /users/:id`: Get user information

## Development

### Project Structure

- `cmd/swit-serve/`: Main application entry point
- `internal/swit-serve/`: Internal packages
  - `cmd/`: Command definitions
  - `config/`: Configuration management
  - `db/`: Database connection
  - `health/`: Health check handlers
  - `server/`: HTTP server setup
  - `user/`: User-related handlers and models

### Makefile Commands

- `make tidy`: Run go mod tidy
- `make build`: Build the binary
- `make clean`: Remove the output binary
- `make test`: Run tests (to be implemented)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.