# Swit

Swit is a command-line tool designed to serve as a backend application and control system. It is built using Go and utilizes the Gin web framework for handling HTTP requests. The project includes features such as health checks, user management, and server control, with data persistence handled through a MySQL database using GORM.

## Key Features:
- Health check endpoint for monitoring application status
- User registration and retrieval functionality
- Server control (start, stop, health check) via CLI
- GitHub Actions workflow for automated building and testing
- Makefile for easy project management and building
- Docker support for containerized deployment

Swit is designed to be a robust and scalable backend service, suitable for various applications requiring user management, health monitoring, and remote control capabilities.

## Components

1. swit-serve: The main server application
2. switctl: Command-line tool for controlling the server

## Getting Started

### Prerequisites

- Go 1.22 or higher
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
The binaries will be created in the `_output` directory.

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
- `POST /users`: Register a new user
- `GET /users/:id`: Get user information
- `POST /stop`: Stop the server
- `GET /api/v1/users`: Get all users list
- `POST /api/v1/users`: Create a new user
- `GET /api/v1/users/:id`: Get specific user information
- `PUT /api/v1/users/:id`: Update user information
- `DELETE /api/v1/users/:id`: Delete a user

## Development

### Project Structure

- `cmd/swit-serve/`: Main application entry point
- `cmd/switctl/`: CLI tool entry point
- `internal/swit-serve/`: Server internal packages
  - `cmd/`: Command definitions
  - `config/`: Configuration management
  - `db/`: Database connection
  - `health/`: Health check handlers
  - `server/`: HTTP server setup
  - `stop/`: Server stop handler
  - `controller/`: API controllers
  - `model/`: Data models
  - `repository/`: Data access layer
  - `service/`: Business logic layer
- `internal/switctl/`: CLI tool internal packages
  - `cmd/`: CLI command definitions

### Makefile Commands

- `make tidy`: Run go mod tidy
- `make build`: Build the binaries
- `make clean`: Remove the output binaries
- `make test`: Run tests
- `make image-serve`: Build Docker image for swit-serve

### Docker Support

To build the Docker image:
```
make image-serve
```

To run the Docker container:
```
docker run -d -p 9000:9000 -v ./swit.yaml:/root/swit.yaml swit-serve:master
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.