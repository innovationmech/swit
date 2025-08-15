---
title: 完整 API 参考
description: 所有服务的统一 API 文档
outline: deep
---

# 完整 API 参考

这里汇总了 Swit 框架中所有服务的 API 接口文档，方便您快速查找和使用。

## 服务概览

| 服务名称 | 描述 | 端点数量 | 版本 |
|------------|-------------|------------|----------|
| [SWIT Server API](#switserve) | This is the SWIT server API documentation. | 5 | 1.0 |
| [SWIT Auth API](#switauth) | This is the SWIT authentication service API documentation. | 5 | 1.0 |


## SWIT Server API {#switserve}

This is the SWIT server API documentation.

- **基础 URL**: `http://localhost:8080/`
- **版本**: `1.0`
- **端点数量**: 5

### internal

#### POST /internal/validate-user

**Validate user credentials (Internal API)**

Internal API for validating user credentials, used by authentication service

| 方法 | 路径 |
|---------|--------|
| `POST` | `/internal/validate-user` |

**参数**

| 参数名 | 类型 | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `credentials` | `object` | 必填 | User credentials |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Validation successful |
| `400` | Bad request |
| `401` | Invalid credentials |
| `500` | Internal server error |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `POST` | `/users/create` |

**参数**

| 参数名 | 类型 | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `user` | `object` | 必填 | User information |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `201` | User created successfully |
| `400` | Bad request |
| `500` | Internal server error |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `GET` | `/users/email/{email}` |

**参数**

| 参数名 | 类型 | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `email` | `string` | 必填 | Email address |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `GET` | `/users/username/{username}` |

**参数**

| 参数名 | 类型 | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `username` | `string` | 必填 | Username |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `DELETE` | `/users/{id}` |

**参数**

| 参数名 | 类型 | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `id` | `string` | 必填 | User ID |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | User deleted successfully |
| `400` | Invalid user ID |
| `500` | Internal server error |

**示例**

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

- **基础 URL**: `http://localhost:8090/`
- **版本**: `1.0`
- **端点数量**: 5

### authentication

#### POST /auth/login

**User login**

Authenticate a user with username and password, returns access and refresh tokens

| 方法 | 路径 |
|---------|--------|
| `POST` | `/auth/login` |

**参数**

| 参数名 | 类型 | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `login` | `object` | 必填 | User login credentials |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Login successful |
| `400` | Bad request |
| `401` | Invalid credentials |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `POST` | `/auth/logout` |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Logout successful |
| `400` | Authorization header is missing |
| `500` | Internal server error |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `POST` | `/auth/refresh` |

**参数**

| 参数名 | 类型 | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `refresh` | `object` | 必填 | Refresh token |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Token refresh successful |
| `400` | Bad request |
| `401` | Invalid or expired refresh token |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `GET` | `/auth/validate` |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Token is valid |
| `401` | Invalid or expired token |

**示例**

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

| 方法 | 路径 |
|---------|--------|
| `GET` | `/health` |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Service is healthy |

**示例**

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

