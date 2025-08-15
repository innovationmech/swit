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

| 描述 | Type | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `login` | object | 必填 | User login credentials |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Login successful |
| `400` | Bad request |
| `401` | Invalid credentials |

**示例**

```bash
curl -X POST "/auth/login" \
  -d '{"key": "value"}'
```

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

```bash
curl -X POST "/auth/logout"
```

#### POST /auth/refresh

**Refresh access token**

Generate new access and refresh tokens using a valid refresh token

| 方法 | 路径 |
|---------|--------|
| `POST` | `/auth/refresh` |

**参数**

| 描述 | Type | 必填/可选 | 描述 |
|------|------|----------|-------------|
| `refresh` | object | 必填 | Refresh token |

**响应**

| 状态码 | 描述 |
|-------------|-------------|
| `200` | Token refresh successful |
| `400` | Bad request |
| `401` | Invalid or expired refresh token |

**示例**

```bash
curl -X POST "/auth/refresh" \
  -d '{"key": "value"}'
```

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

```bash
curl -X GET "/auth/validate"
```

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

```bash
curl -X GET "/health"
```

