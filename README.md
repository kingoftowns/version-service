# Version Service

A production-ready semantic versioning microservice in Go. This service provides a centralized, consistent way to generate orderable version numbers for CI/CD pipelines, replacing the need to use git SHAs.

## Features

- **Semantic Versioning**: Strict adherence to semantic versioning (major.minor.patch) format
- **Automatic Version Management**: Auto-increment patch versions on main branch, with support for minor and major increments
- **Dev Branch Support**: Generate development versions with SHA suffixes for non-production branches
- **Dual Storage**: Redis for fast caching and Git repository as the source of truth
- **RESTful API**: Simple HTTP API using the Gin framework
- **High Availability**: Supports multiple replicas with concurrent request handling

## Quick Start

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Redis (for local development)
- Git repository for version storage
- GitLab/GitHub access token

### DevContainer Development

The easiest way to get started is using VS Code DevContainers:

1. Open the project in VS Code
2. Click "Reopen in Container" when prompted (or use Command Palette: "Dev Containers: Reopen in Container")
3. Wait for the container to build and Redis to start
4. Press `F5` to start debugging or use `Ctrl+Shift+P` and type "Debug: Start Debugging"

The devcontainer automatically:
- Sets up Go 1.24 development environment
- Starts Redis service
- Installs development tools (air, golangci-lint, delve)
- Configures VS Code with Go extensions and settings
- Maps ports 8080 (app) and 6379 (Redis)

## API Documentation

### Health Check
Check service health and dependencies status.

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "checks": {
    "redis": "healthy",
    "git": "healthy"
  }
}
```

### Get Version
Get current and next version for an application.

```http
GET /version/{app-id}
```

**Parameters:**
- `app-id`: Application identifier in format `{project-id}-{app-name}` (e.g., "1234-user-service")

**Response:**
```json
{
  "current": "1.2.3",
  "next": "1.2.4",
  "project_id": "1234",
  "app_name": "user-service",
  "last_updated": "2025-01-15T10:30:00Z"
}
```

### Increment Version
Increment the version of an application.

```http
POST /version/{app-id}/increment[?type=minor|major]
```

**Parameters:**
- `app-id`: Application identifier
- `type` (optional): Increment type - "patch" (default), "minor", or "major"

**Response:**
```json
{
  "version": "1.2.4"
}
```

### Get Dev Version
Get a development version for a feature branch.

```http
POST /version/{app-id}/dev
```

**Request Body:**
```json
{
  "sha": "abc1234567890",
  "branch": "feature/new-feature"
}
```

**Response:**
```json
{
  "version": "1.2.4-dev-abc1234"
}
```

### List All Versions
List all application versions.

```http
GET /versions
```

**Response:**
```json
{
  "1234-user-service": {
    "current": "1.2.3",
    "next": "1.2.4",
    "project_id": "1234",
    "app_name": "user-service",
    "last_updated": "2025-01-15T10:30:00Z"
  },
  "1234-payment-service": {
    "current": "2.0.1",
    "next": "2.0.2",
    "project_id": "1234",
    "app_name": "payment-service",
    "last_updated": "2025-01-15T09:15:00Z"
  }
}
```

### List Project Versions
List all versions for a specific project.

```http
GET /versions/{project-id}
```

**Parameters:**
- `project-id`: GitLab project ID

### Metrics
Prometheus metrics endpoint.

```http
GET /metrics
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | HTTP server port | 8080 | No |
| `REDIS_URL` | Redis connection URL | redis://localhost:6379 | No |
| `GIT_REPO_URL` | Git repository URL for version storage | - | Yes |
| `GIT_USERNAME` | Git username for authentication | version-service | No |
| `GIT_TOKEN` | Git access token | - | Yes |
| `GIT_BRANCH` | Git branch to use | main | No |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info | No |
| `GIN_MODE` | Gin framework mode (debug, release, test) | release | No |

## Docker Build

Build the Docker image:
```bash
docker build -t version-service:latest .
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Project Structure

```
├── main.go                 # Application entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── handlers/          # HTTP request handlers
│   ├── services/          # Business logic
│   ├── storage/           # Storage interfaces (Redis, Git)
│   ├── models/            # Data models
│   └── middleware/        # HTTP middleware
├── pkg/
│   └── semver/           # Semantic versioning package
├── .devcontainer/        # DevContainer configuration
├── .vscode/              # VS Code settings and launch config
├── Dockerfile            # Docker build file
├── Makefile             # Build commands
└── README.md            # This file
```

## CI/CD Integration

### GitLab CI Example

```yaml
variables:
  VERSION_SERVICE_URL: "http://version-service.example.com"

before_script:
  - APP_ID="${CI_PROJECT_ID}-${CI_PROJECT_NAME}"

get-version:
  script:
    - |
      if [ "$CI_COMMIT_BRANCH" == "main" ]; then
        VERSION=$(curl -X POST "${VERSION_SERVICE_URL}/version/${APP_ID}/increment" | jq -r .version)
      else
        VERSION=$(curl -X POST "${VERSION_SERVICE_URL}/version/${APP_ID}/dev" \
          -H "Content-Type: application/json" \
          -d "{\"sha\":\"${CI_COMMIT_SHA}\",\"branch\":\"${CI_COMMIT_BRANCH}\"}" | jq -r .version)
      fi
      echo "VERSION=${VERSION}" >> build.env
```

### GitHub Actions Example

```yaml
name: Build and Version
on: [push]

jobs:
  version:
    runs-on: ubuntu-latest
    steps:
      - name: Get Version
        run: |
          APP_ID="${{ github.repository_owner }}-${{ github.event.repository.name }}"
          if [ "${{ github.ref }}" == "refs/heads/main" ]; then
            VERSION=$(curl -X POST "http://version-service/version/${APP_ID}/increment" | jq -r .version)
          else
            VERSION=$(curl -X POST "http://version-service/version/${APP_ID}/dev" \
              -H "Content-Type: application/json" \
              -d "{\"sha\":\"${{ github.sha }}\",\"branch\":\"${{ github.ref_name }}\"}" | jq -r .version)
          fi
          echo "VERSION=${VERSION}" >> $GITHUB_ENV
```
