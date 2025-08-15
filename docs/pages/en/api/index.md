---
title: API Documentation
description: Swit Framework API Documentation
---

# API Documentation

Welcome to the Swit Framework API Documentation. Below are the available services and their API descriptions.

## Available Services

<div class="service-grid">

<div class="service-card">

### [SWIT Server API](./switserve.md)

This is the SWIT server API documentation.

<div class="service-stats">

- **Endpoints**: 5
- **Version**: 1.0
- **Base URL**: `http://localhost:8080/`

</div>

[View Documentation →](./switserve.md)

</div>

<div class="service-card">

### [SWIT Auth API](./switauth.md)

This is the SWIT authentication service API documentation.

<div class="service-stats">

- **Endpoints**: 5
- **Version**: 1.0
- **Base URL**: `http://localhost:8090/`

</div>

[View Documentation →](./switauth.md)

</div>

</div>

## Quick Links

- [Complete API Reference](./complete.md)
- [Overview](./overview.md)
- [Authentication Service](./switauth.md)
- [User Management Service](./switserve.md)

## General Information

### Authentication

All API endpoints require appropriate authentication. Most endpoints use Bearer Token authentication:

```http
Authorization: Bearer <your_access_token>
```

### Request Format

- **Content-Type**: `application/json`
- **Accept**: `application/json`
- **Encoding**: UTF-8

### Error Handling

The API uses standard HTTP status codes to indicate the success or failure of requests:

| Status Code | Description | Example |
|-------------|-------------|----------|
| 200 | Success | Data retrieved successfully |
| 201 | Created | User created successfully |
| 400 | Bad Request | Invalid parameter format |
| 401 | Unauthorized | Invalid or expired token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | User does not exist |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | System error |

### Response Format

All responses follow a unified JSON format:

```json
{
  "status": "success|error",
  "data": {},
  "message": "Response message",
  "timestamp": "2023-12-01T12:00:00Z"
}
```

<style>
.service-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
  margin: 1rem 0;
}

.service-card {
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--vp-c-bg-soft);
}

.service-card h3 {
  margin-top: 0;
  color: var(--vp-c-brand-1);
}

.service-stats {
  margin: 1rem 0;
  font-size: 0.9em;
}

.service-card > p:last-child {
  margin-bottom: 0;
  text-align: right;
  font-weight: 500;
}

.service-card a {
  text-decoration: none;
}

.service-card a:hover {
  text-decoration: underline;
}
</style>

