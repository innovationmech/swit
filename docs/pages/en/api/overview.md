# API Overview

The Swit framework provides a rich set of APIs for building microservice applications. This documentation provides a complete reference for the framework's core APIs.

## Core APIs

### Server API

The Server API is the heart of the framework, providing service lifecycle management, configuration, and dependency injection capabilities.

- [`BusinessServerCore`](/en/api/server#businessservercore) - Main server interface
- [`ServerConfig`](/en/api/server#serverconfig) - Server configuration
- [`BusinessServiceRegistrar`](/en/api/server#businessserviceregistrar) - Service registration interface

### Transport API

The Transport API manages HTTP and gRPC communication.

- [`TransportCoordinator`](/en/api/transport#transportcoordinator) - Transport coordinator
- [`NetworkTransport`](/en/api/transport#networktransport) - Network transport interface
- [`MultiTransportRegistry`](/en/api/transport#multitransportregistry) - Multi-transport registry

### Middleware API

The Middleware API provides extension points for request processing.

- [`Middleware`](/en/api/middleware#middleware) - Middleware interface
- [`AuthMiddleware`](/en/api/middleware#authmiddleware) - Authentication middleware
- [`LoggingMiddleware`](/en/api/middleware#loggingmiddleware) - Logging middleware

## Service APIs

### Swit Serve API

Complete REST API for user management service.

- [User Management](/en/api/swit-serve#users)
- [Role Permissions](/en/api/swit-serve#roles)
- [Organization Management](/en/api/swit-serve#organizations)

### Swit Auth API

Authentication and authorization service API.

- [JWT Authentication](/en/api/swit-auth#jwt)
- [OAuth 2.0](/en/api/swit-auth#oauth)
- [Permission Management](/en/api/swit-auth#permissions)

## API Conventions

### Request Format

All API requests should use JSON format:

```json
{
  "field1": "value1",
  "field2": "value2"
}
```

### Response Format

Standard response format:

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### Error Handling

Error response format:

```json
{
  "code": 400,
  "message": "Invalid request",
  "error": {
    "field": "username",
    "reason": "required"
  }
}
```

## Versioning

APIs use URL path versioning:

- `/api/v1/` - Version 1
- `/api/v2/` - Version 2 (future)

## Authentication

Most APIs require authentication. The following authentication methods are supported:

- **Bearer Token**: `Authorization: Bearer <token>`
- **API Key**: `X-API-Key: <key>`

## Rate Limiting

API requests are protected by rate limiting:

- Default limit: 100 requests per minute
- Authenticated users: 1000 requests per minute

## More Information

- [Complete API Reference](/en/api/server)
- [OpenAPI Specification](https://github.com/innovationmech/swit/blob/main/api/openapi.yaml)
- [Postman Collection](https://github.com/innovationmech/swit/blob/main/api/postman.json)