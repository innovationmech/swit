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

---

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

---

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

---

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

---

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

---

