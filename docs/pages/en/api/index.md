# API Documentation

Welcome to the Swit Framework API Documentation. Below are the available services and their API descriptions.

## Available Services

### [SWIT Server API](./switserve.md)

This is the SWIT server API documentation.

- **Endpoints**: 5
- **Version**: 1.0
- **Base URL**: http://localhost:8080/

### [SWIT Auth API](./switauth.md)

This is the SWIT authentication service API documentation.

- **Endpoints**: 5
- **Version**: 1.0
- **Base URL**: http://localhost:8090/

## General Information

### Authentication

All API endpoints require appropriate authentication. Please refer to the specific authentication requirements for each service.

### Error Handling

The API uses standard HTTP status codes to indicate the success or failure of requests.

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 500 | Internal Server Error |

