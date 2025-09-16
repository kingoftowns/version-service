# Version Service

A production-ready semantic versioning microservice in Go that manages version numbers for hundreds of applications across multiple GitLab repositories. This service provides a centralized, consistent way to generate orderable version numbers for CI/CD pipelines, replacing the need to use git SHAs.

## Features

- **Semantic Versioning**: Strict adherence to semantic versioning (major.minor.patch) format
- **Automatic Version Management**: Auto-increment patch versions on main branch, with support for minor and major increments
- **Dev Branch Support**: Generate development versions with SHA suffixes for non-production branches
- **Dual Storage**: Redis for fast caching and Git repository as the source of truth
- **RESTful API**: Simple HTTP API using the Gin framework
- **Production Ready**: Health checks, metrics (Prometheus), structured logging, graceful shutdown
- **High Availability**: Supports multiple replicas with concurrent request handling
- **Kubernetes Native**: Full Kubernetes deployment manifests with HPA and ingress

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│   Client    │────────▶│Version Service│────────▶│    Redis     │
│  (CI/CD)    │         │   (Go/Gin)   │         │   (Cache)    │
└─────────────┘         └──────────────┘         └──────────────┘
                                │
                                ▼
                        ┌──────────────┐
                        │     Git      │
                        │ (Source of   │
                        │   Truth)     │
                        └──────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Redis (for local development)
- Git repository for version storage
- GitLab/GitHub access token

### DevContainer Development (Recommended)

The easiest way to get started is using VS Code DevContainers:

1. Open the project in VS Code
2. Click "Reopen in Container" when prompted (or use Command Palette: "Dev Containers: Reopen in Container")
3. Wait for the container to build and Redis to start
4. Press `F5` to start debugging or use `Ctrl+Shift+P` and type "Debug: Start Debugging"

The devcontainer automatically:
- Sets up Go 1.21 development environment
- Starts Redis service
- Installs development tools (air, golangci-lint, delve)
- Configures VS Code with Go extensions and settings
- Maps ports 8080 (app) and 6379 (Redis)

### Local Development (Manual Setup)

1. Clone the repository:
```bash
git clone https://github.com/company/version-service.git
cd version-service
```

2. Copy environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Install dependencies:
```bash
go mod download
```

4. Start Redis:
```bash
make dev-redis
# or manually: docker run -d -p 6379:6379 redis:7-alpine
```

5. Run locally:
```bash
make run
# or with hot reload: make dev
```

### Docker Compose

1. Set required environment variables:
```bash
export GIT_REPO_URL="https://gitlab.com/company/versions.git"
export GIT_TOKEN="your-gitlab-token"
```

2. Start services:
```bash
docker-compose up -d
```

3. View logs:
```bash
docker-compose logs -f
```

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

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run ./...
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
├── tests/                # Integration tests
├── .devcontainer/        # DevContainer configuration
├── .vscode/              # VS Code settings and launch config
├── Dockerfile            # Docker build file
├── docker-compose.yml    # Docker Compose configuration
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

## Monitoring

### Metrics

The service exposes the following Prometheus metrics:

- `http_request_duration_seconds`: HTTP request duration histogram
- `http_requests_total`: Total HTTP requests counter
- `version_operations_total`: Version operation counter by type

### Grafana Dashboard

Import the provided Grafana dashboard for visualization:
1. Access Grafana UI
2. Import dashboard from `monitoring/grafana-dashboard.json`
3. Select your Prometheus data source

## Troubleshooting

### Common Issues

**Redis Connection Failed**
```bash
# Check Redis connectivity
redis-cli -h localhost -p 6379 ping

# Check Redis logs
docker-compose logs redis
```

**Git Authentication Failed**
```bash
# Verify Git credentials
git clone https://username:token@gitlab.com/company/versions.git

# Check Git token permissions (needs read/write access)
```

**Version Not Incrementing**
```bash
# Check Git repository for conflicts
kubectl exec -it -n version-service deployment/version-service -- sh
cd /tmp/version-service-*
git status
git log --oneline -5
```

### Debug Mode

Enable debug logging:
```bash
export LOG_LEVEL=debug
export GIN_MODE=debug
```

## License

Copyright (c) 2025 Company Name. All rights reserved.

## Support

For issues and questions:
- Create an issue in the GitLab repository
- Contact the platform team at platform@company.com
- Slack: #platform-support

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write tests for new functionality
- Update documentation for API changes
- Ensure all tests pass before submitting PR
- Follow semantic versioning for the service itself