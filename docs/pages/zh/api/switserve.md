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

| 描述 | Type | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `credentials` | object | 必填 | User credentials |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Validation successful |
| `400` | Bad request |
| `401` | Invalid credentials |
| `500` | Internal server error |

**示例**

```bash
curl -X POST "/internal/validate-user" \
  -d '{"key": "value"}'
```

### users

#### POST /users/create

**Create a new user**

Create a new user with username, email and password

| 方法 | 路径 |
|---------|--------|
| `POST` | `/users/create` |

**参数**

| 描述 | Type | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `user` | object | 必填 | User information |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `201` | User created successfully |
| `400` | Bad request |
| `500` | Internal server error |

**示例**

```bash
curl -X POST "/users/create" \
  -d '{"key": "value"}'
```

#### GET /users/email/{email}

**Get user by email**

Get user details by email address

| 方法 | 路径 |
|---------|--------|
| `GET` | `/users/email/{email}` |

**参数**

| 描述 | Type | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `email` | string | 必填 | Email address |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**示例**

```bash
curl -X GET "/users/email/{email}"
```

#### GET /users/username/{username}

**Get user by username**

Get user details by username

| 方法 | 路径 |
|---------|--------|
| `GET` | `/users/username/{username}` |

**参数**

| 描述 | Type | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `username` | string | 必填 | Username |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | User details |
| `404` | User not found |
| `500` | Internal server error |

**示例**

```bash
curl -X GET "/users/username/{username}"
```

#### DELETE /users/{id}

**Delete a user**

Delete a user by ID

| 方法 | 路径 |
|---------|--------|
| `DELETE` | `/users/{id}` |

**参数**

| 描述 | Type | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `id` | string | 必填 | User ID |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | User deleted successfully |
| `400` | Invalid user ID |
| `500` | Internal server error |

**示例**

```bash
curl -X DELETE "/users/{id}"
```

