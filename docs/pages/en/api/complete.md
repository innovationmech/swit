---
title: Complete API Reference
description: Unified API documentation for all services
outline: deep
---

# Complete API Reference

This is a comprehensive reference for all API endpoints in the Swit framework, organized for easy lookup and usage.

## Services Overview

| Service | Description | Endpoints | Version |
|------------|-------------|------------|----------|
| [SWIT Server API](#switserve) | This is the SWIT server API documentation. | 5 | 1.0 |
| [SWIT Auth API](#switauth) | This is the SWIT authentication service API documentation. | 5 | 1.0 |


## SWIT Server API {#switserve}

This is the SWIT server API documentation.

- **Base URL**: `http://localhost:8080/`
- **Version**: `1.0`
- **Endpoints**: 5

### internal

#### POST /internal/validate-user

**Validate user credentials (Internal API)**

Internal API for validating user credentials, used by authentication service

| Method | Path |
|---------|--------|
| `POST` | `/internal/validate-user` |

**Parameters**

| 参数名 | 类型 | Required/Optional | Description |
|------|------|----------|-------------|
| `credentials` | `object` | Required | User credentials |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Validation successful |
| `400` | Bad request |
| `401` | Invalid credentials |
| `500` | Internal server error |

**Examples**

:::tabs

== cURL

```bash
curl -X POST \
  "http://localhost:8080/internal/validate-user" \
  -d '{"key": "value"}'
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/internal/validate-user', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
  },
  body: JSON.stringify({ key: 'value' }),
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.post(
    'http://localhost:8080/internal/validate-user',
    json={'key': 'value'},
    headers=headers
)
data = response.json()
print(data)
```

:::

### users

#### POST /users/create

**Create a new user**

Create a new user with username, email and password

| Method | Path |
|---------|--------|
| `POST` | `/users/create` |

**Parameters**

| 参数名 | 类型 | Required/Optional | Description |
|------|------|----------|-------------|
| `user` | `object` | Required | User information |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `201` | User created successfully |
| `400` | Bad request |
| `500` | Internal server error |

**Examples**

:::tabs

== cURL

```bash
curl -X POST \
  "http://localhost:8080/users/create" \
  -d '{"key": "value"}'
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/users/create', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
  },
  body: JSON.stringify({ key: 'value' }),
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.post(
    'http://localhost:8080/users/create',
    json={'key': 'value'},
    headers=headers
)
data = response.json()
print(data)
```

:::

#### GET /users/email/{email}

**Get user by email**

Get user details by email address

| Method | Path |
|---------|--------|
| `GET` | `/users/email/{email}` |

**Parameters**

| 参数名 | 类型 | Required/Optional | Description |
|------|------|----------|-------------|
| `email` | `string` | Required | Email address |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**Examples**

:::tabs

== cURL

```bash
curl -X GET \
  "http://localhost:8080/users/email/{email}
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/users/email/{email}', {
  headers: {
    'Authorization': 'Bearer <your_token>',
  },
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.get(
    'http://localhost:8080/users/email/{email}',
    headers=headers
)
data = response.json()
print(data)
```

:::

#### GET /users/username/{username}

**Get user by username**

Get user details by username

| Method | Path |
|---------|--------|
| `GET` | `/users/username/{username}` |

**Parameters**

| 参数名 | 类型 | Required/Optional | Description |
|------|------|----------|-------------|
| `username` | `string` | Required | Username |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**Examples**

:::tabs

== cURL

```bash
curl -X GET \
  "http://localhost:8080/users/username/{username}
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/users/username/{username}', {
  headers: {
    'Authorization': 'Bearer <your_token>',
  },
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.get(
    'http://localhost:8080/users/username/{username}',
    headers=headers
)
data = response.json()
print(data)
```

:::

#### DELETE /users/{id}

**Delete a user**

Delete a user by ID

| Method | Path |
|---------|--------|
| `DELETE` | `/users/{id}` |

**Parameters**

| 参数名 | 类型 | Required/Optional | Description |
|------|------|----------|-------------|
| `id` | `string` | Required | User ID |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | User deleted successfully |
| `400` | Invalid user ID |
| `500` | Internal server error |

**Examples**

:::tabs

== cURL

```bash
curl -X DELETE \
  "http://localhost:8080/users/{id}
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/users/{id}', {
  method: 'DELETE',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
  },
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.delete(
    'http://localhost:8080/users/{id}',
    headers=headers
)
data = response.json()
print(data)
```

:::


## SWIT Auth API {#switauth}

This is the SWIT authentication service API documentation.

- **Base URL**: `http://localhost:8090/`
- **Version**: `1.0`
- **Endpoints**: 5

### authentication

#### POST /auth/login

**User login**

Authenticate a user with username and password, returns access and refresh tokens

| Method | Path |
|---------|--------|
| `POST` | `/auth/login` |

**Parameters**

| 参数名 | 类型 | Required/Optional | Description |
|------|------|----------|-------------|
| `login` | `object` | Required | User login credentials |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Login successful |
| `400` | Bad request |
| `401` | Invalid credentials |

**Examples**

:::tabs

== cURL

```bash
curl -X POST \
  "http://localhost:8080/auth/login" \
  -d '{"key": "value"}'
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
  },
  body: JSON.stringify({ key: 'value' }),
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.post(
    'http://localhost:8080/auth/login',
    json={'key': 'value'},
    headers=headers
)
data = response.json()
print(data)
```

:::

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

:::tabs

== cURL

```bash
curl -X POST \
  "http://localhost:8080/auth/logout
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/auth/logout', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
  },
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.post(
    'http://localhost:8080/auth/logout',
    headers=headers
)
data = response.json()
print(data)
```

:::

#### POST /auth/refresh

**Refresh access token**

Generate new access and refresh tokens using a valid refresh token

| Method | Path |
|---------|--------|
| `POST` | `/auth/refresh` |

**Parameters**

| 参数名 | 类型 | Required/Optional | Description |
|------|------|----------|-------------|
| `refresh` | `object` | Required | Refresh token |

**Responses**

| Status Code | Description |
|-------------|-------------|
| `200` | Token refresh successful |
| `400` | Bad request |
| `401` | Invalid or expired refresh token |

**Examples**

:::tabs

== cURL

```bash
curl -X POST \
  "http://localhost:8080/auth/refresh" \
  -d '{"key": "value"}'
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/auth/refresh', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
  },
  body: JSON.stringify({ key: 'value' }),
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.post(
    'http://localhost:8080/auth/refresh',
    json={'key': 'value'},
    headers=headers
)
data = response.json()
print(data)
```

:::

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

:::tabs

== cURL

```bash
curl -X GET \
  "http://localhost:8080/auth/validate
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/auth/validate', {
  headers: {
    'Authorization': 'Bearer <your_token>',
  },
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.get(
    'http://localhost:8080/auth/validate',
    headers=headers
)
data = response.json()
print(data)
```

:::

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

:::tabs

== cURL

```bash
curl -X GET \
  "http://localhost:8080/health
```

== JavaScript

```javascript
const response = await fetch('http://localhost:8080/health', {
  headers: {
    'Authorization': 'Bearer <your_token>',
  },
});
const data = await response.json();
console.log(data);
```

== Python

```python
import requests

headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <your_token>',
}

response = requests.get(
    'http://localhost:8080/health',
    headers=headers
)
data = response.json()
print(data)
```

:::

