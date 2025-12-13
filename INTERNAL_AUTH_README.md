# Internal Authentication Setup

This document explains how to use the internal authentication system as an alternative to SSO.

## Configuration

### Switching to Internal Authentication

Edit your `config.yml` and set:

```yaml
is_sso_enable: false
```

## Database Setup

### 1. Run Database Migration

```bash
go run main.go -PushDB
```

### 2. Generate GORM Models

```bash
go run main.go -PullDB
```

## Creating Users

### Using the Helper Script

```bash
go run tools/create_user.go <username> <password> <firstname> <lastname>
```

**Example:**

```bash
go run tools/create_user.go admin admin123 Admin User
```

## API Usage

### Login Endpoint

**POST** `/auth/login`

**Request Body:**

```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Login Successfull",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "firstname": "Admin",
    "lastname": "User",
    "username": "admin"
  }
}
```

The `token` is a JWT refresh token valid for 7 days.

### Verify Endpoint

**GET** `/auth/verify`

**Headers:**

```
Authorization: Bearer <refresh-token>
```

The verify endpoint returns a new refresh token in the `X-Refresh-Token` header.

## Example Test User

```
Username: admin
Password: admin123
Firstname: Admin
Lastname: User
```
