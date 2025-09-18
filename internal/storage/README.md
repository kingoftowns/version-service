# Internal/Storage Package

## Overview
The storage package provides data persistence abstraction with dual storage implementations. Defines common interfaces and implements both Redis-based caching and Git-based persistent storage for version data.

## Components

### Storage Interface (interface.go)
Common interface for all storage implementations.

**Core Methods**:
- `GetVersion(ctx, appID)` - Retrieve single application version
- `SetVersion(ctx, appID, version)` - Store/update application version
- `ListVersions(ctx)` - Retrieve all versions across all projects
- `ListVersionsByProject(ctx, projectID)` - Retrieve versions filtered by project
- `DeleteVersion(ctx, appID)` - Remove specific application version
- `Health(ctx)` - Storage backend health check
- `RebuildCache(ctx, versions)` - Cache initialization/reconstruction

**GitPushable Interface**:
- `PushPendingCommits(ctx)` - Git-specific interface for background push operations
- Enables background retry of failed push operations

### RedisStorage (redis.go)
High-performance caching implementation using Redis.

**Purpose**:
- Fast read/write operations for frequently accessed data
- Cache layer for immediate response times
- Temporary storage with TTL-based expiration
- Transaction support for atomic operations

**Key Features**:
- **Key Structure**: Uses prefixed keys (`version:app-id`) for organized data
- **Set Tracking**: Maintains set of all app-ids (`versions:all`) for efficient listing
- **TTL Management**: 24-hour default TTL with automatic expiration refresh
- **Transaction Safety**: Pipeline operations for atomic multi-key updates
- **Bulk Operations**: Optimized batch retrieval using MGET for list operations

**Data Organization**:
- Individual versions stored as JSON-serialized AppVersion objects
- Set-based tracking for efficient enumeration
- Project filtering implemented via app-id prefix matching
- Cache rebuilding preserves TTL and set membership

**Error Handling**:
- Redis connection failures handled gracefully with detailed logging
- Missing key scenarios return nil (not found) rather than errors
- JSON marshaling/unmarshaling errors logged with context
- Transactional operations ensure data consistency

### GitStorage (git.go)
Persistent storage using Git repository with commit history.

**Purpose**:
- Durable version storage with complete audit trail
- Remote backup through Git repository hosting
- Version history preservation through commit log
- Multi-environment synchronization capability

**Key Features**:

#### Repository Management
- **Clone Handling**: Supports both existing and empty repository initialization
- **Branch Targeting**: Configurable branch for version storage
- **Authentication**: HTTP Basic Auth for private repository access
- **Temp Directory**: Uses system temp directory for local Git operations

#### File Structure
- **Single File Format**: All versions stored in `versions.json`
- **JSON Structure**: VersionsFile format with metadata and version map
- **Atomic Updates**: File-level commits ensure consistency

#### Concurrency Control
- **Mutex Protection**: Serializes all Git operations to prevent conflicts
- **Pull-Before-Write**: Always syncs latest changes before modifications
- **Conflict Resolution**: Reset and retry mechanism for merge conflicts

#### Resilient Operations
- **Local Commit First**: Ensures durability even if push fails
- **Push Failure Handling**: Graceful degradation with background retry
- **Empty Repository Support**: Automatic initialization of new repositories
- **Health Monitoring**: Git connectivity testing through pull operations

#### Background Push System
- **Unpushed Commit Detection**: Compares local and remote commit hashes
- **Periodic Retry**: Background goroutine for failed push operations
- **Network Resilience**: Handles temporary network issues with retry logic

**Error Handling**:
- Empty repository detection and automatic initialization
- Network failure differentiation (retryable vs permanent)
- Commit preservation even when push operations fail
- Comprehensive logging for debugging Git operations

**Integration Points**:
- Implements Storage interface for seamless service integration
- GitPushable interface enables background push retry from service layer
- Synchronizes with Redis cache through service orchestration

**Relationship to Application**:
The dual storage architecture provides both performance (Redis) and durability (Git), enabling fast API responses while maintaining complete version history and disaster recovery capabilities. The storage abstraction allows the service layer to treat both backends uniformly while leveraging their specific strengths.