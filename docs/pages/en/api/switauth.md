# SWIT Auth API

This is the SWIT authentication service API documentation.

## API Overview

- **Service Name**: switauth
- **Version**: 1.0
- **Base URL**: http://localhost:8090/

## Endpoints

### authentication

#### POST /auth/login

**User login**

Authenticate a user with username and password, returns access and refresh tokens

| Method | Path |
|---------|--------|
| `POST` | `/auth/login` |

**Parameters**

| Description | Type | Required/Optional | Description |
|------|------|----------|-------------|
| `login` | object | Required | User login credentials |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Login successful |
| `400` | Bad request |
| `401` | Invalid credentials |

**Examples**

```bash
curl -X POST "/auth/login" \
  -d '{"key": "value"}'
```

#### POST /auth/logout

**User logout**

Invalidate the user's access token and log them out

| Method | Path |
|---------|--------|
| `POST` | `/auth/logout` |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Logout successful |
| `400` | Authorization header is missing |
| `500` | Internal server error |

**Examples**

```bash
curl -X POST "/auth/logout"
```

#### POST /auth/refresh

**Refresh access token**

Generate new access and refresh tokens using a valid refresh token

| Method | Path |
|---------|--------|
| `POST` | `/auth/refresh` |

**Parameters**

| Description | Type | Required/Optional | Description |
|------|------|----------|-------------|
| `refresh` | object | Required | Refresh token |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Token refresh successful |
| `400` | Bad request |
| `401` | Invalid or expired refresh token |

**Examples**

```bash
curl -X POST "/auth/refresh" \
  -d '{"key": "value"}'
```

#### GET /auth/validate

**Validate access token**

Validate an access token and return token information including user ID

| Method | Path |
|---------|--------|
| `GET` | `/auth/validate` |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Token is valid |
| `401` | Invalid or expired token |

**Examples**

```bash
curl -X GET "/auth/validate"
```

### health

#### GET /health

**Health check**

Check if the authentication service is healthy

| Method | Path |
|---------|--------|
| `GET` | `/health` |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Service is healthy |

**Examples**

```bash
curl -X GET "/health"
```

