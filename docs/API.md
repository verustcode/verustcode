# VerustCode API Reference

This document describes the REST API endpoints provided by VerustCode.

## Base URL

All API endpoints are prefixed with `/api/v1` unless otherwise specified.

## Authentication

Most API endpoints require JWT authentication. Include the token in the `Authorization` header:

```
Authorization: Bearer <token>
```

To obtain a token, use the login endpoint (see Authentication section below).

## Response Format

### Success Response

Successful responses return HTTP status codes in the 2xx range with JSON body:

```json
{
  "data": { ... },
  "message": "Success message"
}
```

### Error Response

Error responses follow this format:

```json
{
  "code": "E1001",
  "message": "Error message",
  "details": "Additional details (optional)"
}
```

### Error Codes

| Code | Description | HTTP Status |
|------|-------------|-------------|
| E1000 | Internal server error | 500 |
| E1001 | Validation error | 400 |
| E1002 | Not found | 404 |
| E1003 | Conflict | 409 |
| E1004 | Forbidden | 403 |
| E1005 | Unauthorized | 401 |
| E2001 | Git clone error | 500 |
| E2002 | Git authentication error | 401 |
| E2003 | Git repository not found | 404 |
| E2004 | Git webhook error | 400 |
| E3001 | Agent not found | 404 |
| E3002 | Agent unavailable | 503 |
| E3003 | Agent timeout | 504 |
| E3004 | Agent execution error | 500 |
| E4001 | Review not found | 404 |
| E4002 | Review failed | 500 |
| E4003 | Review pending | 202 |
| E5001 | Database connection error | 500 |
| E5002 | Database query error | 500 |
| E5003 | Database migration error | 500 |
| E6001 | Configuration not found | 404 |
| E6002 | Invalid configuration | 400 |
| E6003 | Configuration parse error | 400 |
| E6004 | Admin credentials empty | 400 |
| E6005 | Password complexity error | 400 |
| E6006 | JWT secret invalid | 400 |

## Public Endpoints

### Health Check

**GET** `/health`

Check if the server is running.

**Response:**
```json
{
  "status": "ok"
}
```

### Get Schema

**GET** `/api/v1/schemas/:name`

Get JSON schema for DSL configuration. Currently only `default` schema is supported.

**Parameters:**
- `name` (path): Schema name (`default` or `default.json`)

**Response:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  ...
}
```

### Get Report Types

**GET** `/api/v1/report-types`

Get list of available report types.

**Response:**
```json
{
  "data": [
    {
      "id": "wiki",
      "name": "Wiki Report",
      "description": "Generate wiki-style documentation"
    }
  ]
}
```

## Authentication Endpoints

### Login

**POST** `/api/v1/auth/login`

Authenticate and receive JWT token.

**Request Body:**
```json
{
  "username": "admin",
  "password": "password",
  "remember_me": false
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-01-01T12:00:00Z"
}
```

**Note:** If `remember_me` is `true`, token expires in 7 days. Otherwise, it expires in 24 hours (or as configured).

### Get Setup Status

**GET** `/api/v1/auth/setup/status`

Check if admin password setup is required.

**Response:**
```json
{
  "needs_setup": true
}
```

**Note:** Returns 404 if password is already set.

### Setup Password

**POST** `/api/v1/auth/setup`

Set admin password on first launch.

**Request Body:**
```json
{
  "password": "SecurePassword123!",
  "confirm_password": "SecurePassword123!"
}
```

**Response:**
```json
{
  "message": "Password set successfully"
}
```

**Password Requirements:**
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one digit
- At least one special character

### Get Current User

**GET** `/api/v1/auth/me`

Get current authenticated user information.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Response:**
```json
{
  "username": "admin"
}
```

## Review Endpoints

All review endpoints require authentication.

### Create Review

**POST** `/api/v1/reviews`

Create a new code review task.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Request Body:**
```json
{
  "repository": "owner/repo",
  "ref": "main",
  "provider": "github",
  "agent": "cursor",
  "pr_number": 123
}
```

**Response:**
```json
{
  "id": "review-id",
  "status": "pending",
  "repo_url": "https://github.com/owner/repo",
  "ref": "main",
  "created_at": "2024-01-01T12:00:00Z"
}
```

### List Reviews

**GET** `/api/v1/reviews`

List all reviews with pagination.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `page` (int, default: 1): Page number
- `page_size` (int, default: 20): Items per page (1-100)
- `status` (string, optional): Filter by status (`pending`, `running`, `completed`, `failed`, `cancelled`)

**Response:**
```json
{
  "data": [
    {
      "id": "review-id",
      "status": "completed",
      "repo_url": "https://github.com/owner/repo",
      "ref": "main",
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

### Get Review

**GET** `/api/v1/reviews/:id`

Get details of a specific review.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Review ID

**Response:**
```json
{
  "id": "review-id",
  "status": "completed",
  "repo_url": "https://github.com/owner/repo",
  "ref": "main",
  "commit_sha": "abc123...",
  "rules": [
    {
      "id": "rule-id",
      "rule_index": 0,
      "status": "completed",
      "result": { ... }
    }
  ],
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:05:00Z"
}
```

### Cancel Review

**POST** `/api/v1/reviews/:id/cancel`

Cancel a pending or running review.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Review ID

**Response:**
```json
{
  "message": "Review cancelled"
}
```

### Retry Review

**POST** `/api/v1/reviews/:id/retry`

Retry a failed or cancelled review.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Review ID

**Response:**
```json
{
  "message": "Review queued for retry"
}
```

### Retry Review Rule

**POST** `/api/v1/reviews/:id/rules/:rule_id/retry`

Retry a specific rule within a review.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Review ID
- `rule_id` (path): Rule ID

**Response:**
```json
{
  "message": "Rule queued for retry"
}
```

### Get Review Logs

**GET** `/api/v1/reviews/:id/logs`

Get logs for a specific review.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Review ID

**Query Parameters:**
- `page` (int, default: 1): Page number
- `page_size` (int, default: 100): Items per page
- `level` (string, optional): Filter by log level (`debug`, `info`, `warn`, `error`, `fatal`)

**Response:**
```json
{
  "data": [
    {
      "id": "log-id",
      "level": "info",
      "message": "Review started",
      "timestamp": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 100
}
```

## Report Endpoints

All report endpoints require authentication.

### Create Report

**POST** `/api/v1/reports`

Create a new report generation task.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Request Body:**
```json
{
  "repo_url": "https://github.com/owner/repo",
  "ref": "main",
  "report_type": "wiki",
  "title": "Custom Title"
}
```

**Response:**
```json
{
  "id": "report-id",
  "status": "pending",
  "repo_url": "https://github.com/owner/repo",
  "ref": "main",
  "report_type": "wiki",
  "created_at": "2024-01-01T12:00:00Z"
}
```

### List Reports

**GET** `/api/v1/reports`

List all reports with pagination.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `page` (int, default: 1): Page number
- `page_size` (int, default: 20): Items per page
- `status` (string, optional): Filter by status
- `report_type` (string, optional): Filter by report type

**Response:**
```json
{
  "data": [
    {
      "id": "report-id",
      "status": "completed",
      "repo_url": "https://github.com/owner/repo",
      "ref": "main",
      "report_type": "wiki"
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 20
}
```

### Get Report

**GET** `/api/v1/reports/:id`

Get details of a specific report.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Report ID

**Response:**
```json
{
  "id": "report-id",
  "status": "completed",
  "repo_url": "https://github.com/owner/repo",
  "ref": "main",
  "report_type": "wiki",
  "result": { ... },
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:10:00Z"
}
```

### Get Report Progress

**GET** `/api/v1/reports/:id/progress`

Get progress information for a running report.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Report ID

**Response:**
```json
{
  "status": "running",
  "progress": 45,
  "current_step": "Generating sections",
  "total_steps": 10
}
```

### Cancel Report

**POST** `/api/v1/reports/:id/cancel`

Cancel a pending or running report.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Report ID

**Response:**
```json
{
  "message": "Report cancelled"
}
```

### Retry Report

**POST** `/api/v1/reports/:id/retry`

Retry a failed or cancelled report.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Report ID

**Response:**
```json
{
  "message": "Report queued for retry"
}
```

### Export Report

**GET** `/api/v1/reports/:id/export`

Export report in various formats.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Report ID

**Query Parameters:**
- `format` (string, default: `pdf`): Export format (`pdf`, `html`, `markdown`)

**Response:**
File download (Content-Type depends on format)

### Get Report Logs

**GET** `/api/v1/reports/:id/logs`

Get logs for a specific report.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Parameters:**
- `id` (path): Report ID

**Query Parameters:**
- `page` (int, default: 1): Page number
- `page_size` (int, default: 100): Items per page
- `level` (string, optional): Filter by log level

**Response:**
```json
{
  "data": [
    {
      "id": "log-id",
      "level": "info",
      "message": "Report generation started",
      "timestamp": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 30,
  "page": 1,
  "page_size": 100
}
```

### Get Repositories

**GET** `/api/v1/reports/repositories`

Get list of repositories available for report generation.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Response:**
```json
{
  "data": [
    {
      "repo_url": "https://github.com/owner/repo",
      "last_report_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

### Get Branches

**GET** `/api/v1/reports/branches`

Get list of branches for a repository.

**Headers:**
- `Authorization: Bearer <token>` (required)

**Query Parameters:**
- `repo_url` (string, required): Repository URL

**Response:**
```json
{
  "data": [
    {
      "name": "main",
      "sha": "abc123...",
      "is_default": true
    }
  ]
}
```

## Admin Endpoints

All admin endpoints require authentication.

### Get Current User (Admin)

**GET** `/api/v1/admin/me`

Same as `/api/v1/auth/me`.

### Get App Meta

**GET** `/api/v1/admin/meta`

Get application metadata.

**Response:**
```json
{
  "name": "VerustCode",
  "subtitle": "DSL-Driven AI Code Review",
  "version": "1.0.0"
}
```

### Get Server Status

**GET** `/api/v1/admin/status`

Get server status information.

**Response:**
```json
{
  "status": "running",
  "uptime": "2h30m",
  "version": "1.0.0"
}
```

### Get Stats

**GET** `/api/v1/admin/stats`

Get overall statistics.

**Response:**
```json
{
  "total_reviews": 100,
  "total_reports": 50,
  "total_repositories": 10,
  "reviews_today": 5,
  "reports_today": 2
}
```

### Get Repository Stats

**GET** `/api/v1/admin/stats/repo`

Get repository-level statistics.

**Query Parameters:**
- `page` (int, default: 1): Page number
- `page_size` (int, default: 20): Items per page
- `sort_by` (string, default: `review_count`): Sort field
- `sort_order` (string, default: `desc`): Sort order (`asc` or `desc`)

**Response:**
```json
{
  "data": [
    {
      "repo_url": "https://github.com/owner/repo",
      "review_count": 50,
      "report_count": 10,
      "last_review_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20
}
```

### List Findings

**GET** `/api/v1/admin/findings`

Get aggregated findings across all repositories.

**Query Parameters:**
- `page` (int, default: 1): Page number
- `page_size` (int, default: 20): Items per page
- `severity` (string, optional): Filter by severity
- `repo_url` (string, optional): Filter by repository

**Response:**
```json
{
  "data": [
    {
      "id": "finding-id",
      "severity": "high",
      "message": "Security vulnerability found",
      "repo_url": "https://github.com/owner/repo",
      "file": "src/main.go",
      "line": 42
    }
  ],
  "total": 200,
  "page": 1,
  "page_size": 20
}
```

### Queue Status

**GET** `/api/v1/queue/status`

Get review queue status.

**Response:**
```json
{
  "total_pending": 5,
  "total_running": 2,
  "repo_count": 3,
  "repos": [
    {
      "repo_url": "https://github.com/owner/repo",
      "pending": 2,
      "running": 1
    }
  ]
}
```

## Rules Management

### List Rules

**GET** `/api/v1/admin/rules`

List all review rule files.

**Response:**
```json
{
  "data": [
    {
      "name": "default.yaml",
      "size": 1024,
      "modified_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

### Get Rule

**GET** `/api/v1/admin/rules/:name`

Get content of a specific rule file.

**Parameters:**
- `name` (path): Rule file name

**Response:**
```json
{
  "name": "default.yaml",
  "content": "version: \"1.0\"\n..."
}
```

### Create Rule File

**POST** `/api/v1/admin/rules`

Create a new rule file.

**Request Body:**
```json
{
  "name": "custom.yaml",
  "content": "version: \"1.0\"\n..."
}
```

**Response:**
```json
{
  "message": "Rule file created"
}
```

### Save Rule

**PUT** `/api/v1/admin/rules/:name`

Update an existing rule file.

**Parameters:**
- `name` (path): Rule file name

**Request Body:**
```json
{
  "content": "version: \"1.0\"\n..."
}
```

**Response:**
```json
{
  "message": "Rule file saved"
}
```

### Validate Rule

**POST** `/api/v1/admin/rules/validate`

Validate a rule file content.

**Request Body:**
```json
{
  "content": "version: \"1.0\"\n..."
}
```

**Response:**
```json
{
  "valid": true,
  "errors": []
}
```

### List Review Files

**GET** `/api/v1/admin/review-files`

List available review configuration files.

**Response:**
```json
{
  "data": [
    "default.yaml",
    "security.yaml"
  ]
}
```

## Report Types Management

### List Report Types

**GET** `/api/v1/admin/report-types`

List all report type configuration files.

**Response:**
```json
{
  "data": [
    {
      "name": "wiki.yaml",
      "size": 2048,
      "modified_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

### Get Report Type

**GET** `/api/v1/admin/report-types/:name`

Get content of a specific report type file.

**Parameters:**
- `name` (path): Report type file name

**Response:**
```json
{
  "name": "wiki.yaml",
  "content": "version: \"1.0\"\n..."
}
```

### Create Report Type

**POST** `/api/v1/admin/report-types`

Create a new report type file.

**Request Body:**
```json
{
  "name": "custom.yaml",
  "content": "version: \"1.0\"\n..."
}
```

**Response:**
```json
{
  "message": "Report type created"
}
```

### Save Report Type

**PUT** `/api/v1/admin/report-types/:name`

Update an existing report type file.

**Parameters:**
- `name` (path): Report type file name

**Request Body:**
```json
{
  "content": "version: \"1.0\"\n..."
}
```

**Response:**
```json
{
  "message": "Report type saved"
}
```

### Validate Report Type

**POST** `/api/v1/admin/report-types/validate`

Validate a report type file content.

**Request Body:**
```json
{
  "content": "version: \"1.0\"\n..."
}
```

**Response:**
```json
{
  "valid": true,
  "errors": []
}
```

## Repository Configuration

### List Repositories

**GET** `/api/v1/admin/repositories`

List all repository configurations.

**Query Parameters:**
- `page` (int, default: 1): Page number
- `page_size` (int, default: 20): Items per page
- `search` (string, optional): Search by repository URL
- `sort_by` (string, default: `repo_url`): Sort field (`repo_url`, `review_count`, `last_review_at`)
- `sort_order` (string, default: `asc`): Sort order (`asc` or `desc`)

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "repo_url": "https://github.com/owner/repo",
      "review_file": "default.yaml",
      "description": "Main repository",
      "review_count": 50,
      "last_review_at": "2024-01-01T12:00:00Z",
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20
}
```

### Create Repository Config

**POST** `/api/v1/admin/repositories`

Create a new repository configuration.

**Request Body:**
```json
{
  "repo_url": "https://github.com/owner/repo",
  "review_file": "default.yaml",
  "description": "Repository description"
}
```

**Response:**
```json
{
  "id": 1,
  "repo_url": "https://github.com/owner/repo",
  "review_file": "default.yaml",
  "description": "Repository description"
}
```

### Update Repository Config

**PUT** `/api/v1/admin/repositories/:id`

Update an existing repository configuration.

**Parameters:**
- `id` (path): Repository configuration ID

**Request Body:**
```json
{
  "review_file": "security.yaml",
  "description": "Updated description"
}
```

**Response:**
```json
{
  "message": "Repository configuration updated"
}
```

### Delete Repository Config

**DELETE** `/api/v1/admin/repositories/:id`

Delete a repository configuration.

**Parameters:**
- `id` (path): Repository configuration ID

**Response:**
```json
{
  "message": "Repository configuration deleted"
}
```

### Parse Repository URL

**POST** `/api/v1/admin/parse-repo-url`

Parse a repository URL to extract provider, owner, and repo name.

**Request Body:**
```json
{
  "url": "https://github.com/owner/repo"
}
```

**Response:**
```json
{
  "provider": "github",
  "owner": "owner",
  "repo": "repo"
}
```

## Settings Management

### Get All Settings

**GET** `/api/v1/admin/settings`

Get all runtime settings.

**Response:**
```json
{
  "git": { ... },
  "agents": { ... },
  "review": { ... },
  "report": { ... },
  "notifications": { ... }
}
```

### Get Settings by Category

**GET** `/api/v1/admin/settings/:category`

Get settings for a specific category.

**Parameters:**
- `category` (path): Settings category (`git`, `agents`, `review`, `report`, `notifications`)

**Response:**
```json
{
  "providers": [ ... ]
}
```

### Update Settings by Category

**PUT** `/api/v1/admin/settings/:category`

Update settings for a specific category.

**Parameters:**
- `category` (path): Settings category

**Request Body:**
```json
{
  "providers": [ ... ]
}
```

**Response:**
```json
{
  "message": "Settings updated"
}
```

### Apply Settings

**POST** `/api/v1/admin/settings/apply`

Apply all settings at once.

**Request Body:**
```json
{
  "git": { ... },
  "agents": { ... },
  "review": { ... },
  "report": { ... },
  "notifications": { ... }
}
```

**Response:**
```json
{
  "message": "Settings applied successfully"
}
```

### Test Git Provider

**POST** `/api/v1/admin/settings/git/test`

Test a Git provider configuration.

**Request Body:**
```json
{
  "type": "github",
  "token": "token",
  "url": "https://api.github.com"
}
```

**Response:**
```json
{
  "valid": true,
  "message": "Connection successful"
}
```

### Test Agent

**POST** `/api/v1/admin/settings/agents/test`

Test an AI agent configuration.

**Request Body:**
```json
{
  "name": "cursor",
  "api_key": "key",
  "model": "composer-1"
}
```

**Response:**
```json
{
  "valid": true,
  "message": "Agent test successful"
}
```

### Test Notification Config

**POST** `/api/v1/admin/settings/notifications/test`

Test a notification configuration.

**Request Body:**
```json
{
  "type": "webhook",
  "url": "https://example.com/webhook",
  "events": ["review.completed"]
}
```

**Response:**
```json
{
  "valid": true,
  "message": "Notification test successful"
}
```

## Notification Endpoints

### Get Notification Status

**GET** `/api/v1/admin/notifications/status`

Get notification system status.

**Response:**
```json
{
  "enabled": true,
  "channels": [
    {
      "type": "webhook",
      "enabled": true
    }
  ]
}
```

### Test Notification

**POST** `/api/v1/admin/notifications/test`

Send a test notification.

**Request Body:**
```json
{
  "type": "webhook",
  "message": "Test notification"
}
```

**Response:**
```json
{
  "message": "Test notification sent"
}
```

## Webhook Endpoints

### Handle Webhook

**POST** `/api/v1/webhooks/:provider`

Handle incoming webhooks from Git providers.

**Parameters:**
- `provider` (path): Git provider (`github`, `gitlab`, `gitea`)

**Headers:**
- `X-GitHub-Event` (GitHub): Event type
- `X-GitHub-Delivery` (GitHub): Delivery ID
- `X-Hub-Signature-256` (GitHub): HMAC signature
- `X-Gitlab-Event` (GitLab): Event type
- `X-Gitlab-Token` (GitLab): Token for validation
- `X-Gitea-Event` (Gitea): Event type
- `X-Gitea-Signature` (Gitea): HMAC signature

**Request Body:**
Provider-specific webhook payload (JSON)

**Response:**
```json
{
  "message": "Webhook processed",
  "review_id": "review-id"
}
```

**Note:** This endpoint is public but requires valid webhook signature/token for security.

## Rate Limiting

Currently, there is no rate limiting implemented. Consider implementing rate limiting for production deployments.

## Versioning

The API is versioned using the `/api/v1` prefix. Future breaking changes will be introduced in `/api/v2`, etc.

## Support

For API support, please:
- Open an issue on GitHub
- Check the documentation
- Review the [Contributing Guide](../CONTRIBUTING.md)
