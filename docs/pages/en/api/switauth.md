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

---

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

---

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

---

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

---

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

---

