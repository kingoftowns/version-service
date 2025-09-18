# Internal/Services Package

## Overview
The services package implements the core business logic for version management. Acts as the orchestration layer between HTTP handlers and storage backends, providing the primary interface for all version-related operations.

## Components

### VersionServiceInterface (interface.go)
Defines the contract for version service operations.

**Methods**:
- `Health(ctx)` - Health check aggregation from dependencies
- `GetVersion(ctx, appID)` - Retrieve application version with smart fallbacks
- `IncrementVersion(ctx, appID, incrementType)` - Semantic version increment operations
- `GetDevVersion(ctx, appID, request)` - Development version generation
- `ListVersions(ctx)` - List all application versions
- `ListVersionsByProject(ctx, projectID)` - List versions filtered by project
- `DeleteVersion(ctx, appID)` - Remove specific application version
- `DeleteProject(ctx, projectID)` - Remove all versions in a project

### VersionService (version.go)
Primary implementation of version service business logic with multi-storage architecture.

**Dependencies**:
- `storage.Storage` (redis) - Fast caching layer for frequently accessed data
- `storage.Storage` (git) - Persistent storage with commit history
- `clients.GitLabClient` - External version discovery from existing repositories
- `*logrus.Logger` - Structured logging for operations and debugging

**Key Features**:

#### Dual Storage Architecture
- **Redis**: High-speed cache for active lookups and list operations
- **Git**: Persistent storage with commit history and remote backup
- **Async Persistence**: Git operations run asynchronously to maintain response speed
- **Cache Rebuilding**: Redis cache automatically rebuilt from Git on startup

#### Smart Version Discovery
- Attempts version lookup in order: Redis → Git → GitLab → Default (1.0.0)
- GitLab integration fetches existing semantic version tags for project bootstrapping
- Automatic version initialization for new applications
- Graceful fallback chain when dependencies are unavailable

#### Thread-Safe Operations
- Mutex protection for concurrent increment operations
- Atomic cache updates with Redis transactions
- Git operations serialized to prevent conflicts

#### Resilient Git Operations
- Async Git persistence with retry logic and exponential backoff
- Local commit success even when remote push fails
- Background push retry mechanism for failed operations
- Comprehensive error classification (retryable vs permanent failures)
- Health tracking with recent operation status monitoring

#### Error Handling and Monitoring
- Structured error responses with context
- Operation metrics tracking (success/failure rates, latencies, retry counts)
- Periodic health status logging
- Graceful degradation when storage backends fail

**Core Workflows**:

#### Version Retrieval (`GetVersion`)
1. Check Redis cache for immediate response
2. Fall back to Git storage if cache miss
3. If no version exists, query GitLab for existing tags
4. Create default version (1.0.0) if no existing version found
5. Cache newly discovered/created versions in Redis

#### Version Increment (`IncrementVersion`)
1. Thread-safe lock acquisition for consistency
2. Retrieve current version using smart discovery
3. Calculate next version using semantic versioning rules
4. Save to Redis immediately for fast response
5. Persist to Git asynchronously with retry logic

#### Development Versions (`GetDevVersion`)
1. Retrieve base version from current state
2. Generate pre-release version with commit SHA suffix
3. Return without persisting (ephemeral development builds)

**Background Processes**:
- **Metrics Logging**: Periodic Git operation statistics and health reporting
- **Push Retry**: Background retry of failed Git push operations
- **Health Monitoring**: Tracks recent operation success/failure patterns

**Relationship to Application**:
This service serves as the central orchestrator of version management, implementing business rules while coordinating between multiple storage backends and external services. It provides the reliability and performance characteristics needed for production version management through caching, async operations, and comprehensive error handling.