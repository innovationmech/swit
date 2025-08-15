# SWIT Auth API

This is the SWIT authentication service API documentation.

## API 概览

- **服务名称**: switauth
- **版本**: 1.0
- **基础URL**: http://localhost:8090/

## 接口列表

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

---

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

---

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

---

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

---

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

---

