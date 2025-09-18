# Internal/Config Package

## Overview
The config package handles application configuration loading from environment variables with sensible defaults. Provides centralized configuration management for all service dependencies.

## Components

### Config Struct (config.go)
Centralized configuration structure that defines all application settings.

**Configuration Fields**:
- `Port` - HTTP server port (default: 8080)
- `RedisURL` - Redis connection string for caching layer
- `GitRepoURL` - Git repository URL for persistent version storage (required)
- `GitUsername` - Git commit author username (default: "version-service")
- `GitToken` - Git authentication token (required)
- `GitBranch` - Target Git branch for commits (default: "main")
- `GitLabBaseURL` - GitLab API base URL (default: GitLab.com API)
- `GitLabAccessToken` - GitLab API token for tag fetching (optional)
- `LogLevel` - Logging verbosity level (default: "info")

**Key Functionality**:
- `Load()` - Loads configuration from environment variables with validation
- `getEnv(key, defaultValue)` - Helper for environment variable retrieval with fallbacks
- Validates required configuration fields (GIT_REPO_URL, GIT_TOKEN)
- Returns descriptive errors for missing critical configuration

**Environment Variable Mapping**:
- PORT → Port
- REDIS_URL → RedisURL
- GIT_REPO_URL → GitRepoURL (required)
- GIT_USERNAME → GitUsername
- GIT_TOKEN → GitToken (required)
- GIT_BRANCH → GitBranch
- GITLAB_BASE_URL → GitLabBaseURL
- GITLAB_ACCESS_TOKEN → GitLabAccessToken
- LOG_LEVEL → LogLevel

**Integration Points**:
- Used by `main.go` during application initialization
- Configuration passed to storage layers (Redis, Git) and external clients (GitLab)
- Required for proper service bootstrap and dependency injection

**Design Principles**:
- Fail fast on missing required configuration
- Provide sensible defaults for optional settings
- Environment-first configuration approach
- Clear separation between required and optional settings

**Relationship to Application**:
This package serves as the single source of truth for application configuration, ensuring all components receive consistent settings and enabling easy environment-specific deployments.