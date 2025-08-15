# SWIT Server API

This is the SWIT server API documentation.

## API 概览

- **服务名称**: switserve
- **版本**: 1.0
- **基础URL**: http://localhost:8080/

## 接口列表

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

---

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

---

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

---

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

---

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

---

