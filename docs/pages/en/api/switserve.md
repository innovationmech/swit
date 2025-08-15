# SWIT Server API

This is the SWIT server API documentation.

## API Overview

- **Service Name**: switserve
- **Version**: 1.0
- **Base URL**: http://localhost:8080/

## Endpoints

### internal

#### POST /internal/validate-user

**Validate user credentials (Internal API)**

Internal API for validating user credentials, used by authentication service

| Method | Path |
|---------|--------|
| `POST` | `/internal/validate-user` |

**Parameters**

| Description | Type | Required/Optional | Description |
|------|------|----------|-------------|
| `credentials` | object | Required | User credentials |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Validation successful |
| `400` | Bad request |
| `401` | Invalid credentials |
| `500` | Internal server error |

**Examples**

```bash
curl -X POST "/internal/validate-user" \
  -d '{"key": "value"}'
```

### users

#### POST /users/create

**Create a new user**

Create a new user with username, email and password

| Method | Path |
|---------|--------|
| `POST` | `/users/create` |

**Parameters**

| Description | Type | Required/Optional | Description |
|------|------|----------|-------------|
| `user` | object | Required | User information |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `201` | User created successfully |
| `400` | Bad request |
| `500` | Internal server error |

**Examples**

```bash
curl -X POST "/users/create" \
  -d '{"key": "value"}'
```

#### GET /users/email/{email}

**Get user by email**

Get user details by email address

| Method | Path |
|---------|--------|
| `GET` | `/users/email/{email}` |

**Parameters**

| Description | Type | Required/Optional | Description |
|------|------|----------|-------------|
| `email` | string | Required | Email address |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**Examples**

```bash
curl -X GET "/users/email/{email}"
```

#### GET /users/username/{username}

**Get user by username**

Get user details by username

| Method | Path |
|---------|--------|
| `GET` | `/users/username/{username}` |

**Parameters**

| Description | Type | Required/Optional | Description |
|------|------|----------|-------------|
| `username` | string | Required | Username |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**Examples**

```bash
curl -X GET "/users/username/{username}"
```

#### DELETE /users/{id}

**Delete a user**

Delete a user by ID

| Method | Path |
|---------|--------|
| `DELETE` | `/users/{id}` |

**Parameters**

| Description | Type | Required/Optional | Description |
|------|------|----------|-------------|
| `id` | string | Required | User ID |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | User deleted successfully |
| `400` | Invalid user ID |
| `500` | Internal server error |

**Examples**

```bash
curl -X DELETE "/users/{id}"
```

