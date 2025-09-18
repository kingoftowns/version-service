# Internal/Handlers Package

## Overview
The handlers package implements HTTP request handlers for the version service REST API. Provides the HTTP interface layer that processes incoming requests and coordinates with the service layer.

## Components

### Handler Struct (handlers.go)
Main HTTP handler that implements all API endpoints for version management.

**Dependencies**:
- `services.VersionServiceInterface` - Core business logic service
- `*logrus.Logger` - Structured logging instance

**Key Endpoints**:

#### GET /health
Health check endpoint that reports service and dependency status.
- Returns aggregated health from Redis and Git storage
- Provides detailed check results for monitoring systems
- Uses HTTP 503 for unhealthy status, 200 for healthy

#### GET /version/{app-id}
Retrieves current version for a specific application.
- Parses app-id parameter (format: project-id-app-name)
- Returns version from cache or storage, creates default if none exists
- Integrates with GitLab client to bootstrap from existing tags
- Tracks metrics for monitoring

#### POST /version/{app-id}/increment
Increments application version using semantic versioning.
- Supports increment types: major, minor, patch (default: patch)
- Uses query parameter `type` to specify increment level
- Thread-safe with mutex protection for concurrent requests
- Returns new version after successful increment

#### POST /version/{app-id}/dev
Generates development version with commit SHA.
- Requires JSON body with `sha` and `branch` fields
- Creates pre-release version with dev suffix (e.g., 1.2.3-dev-abc1234)
- Used for development builds and feature branch deployments

#### GET /versions
Lists all application versions across all projects.
- Returns complete map of app-id to version data
- Includes metadata like last updated timestamp

#### GET /versions/{project-id}
Lists all versions for applications within a specific project.
- Filters versions by project ID prefix
- Useful for project-level version management

#### DELETE /delete/{id}
Deletes version data for applications or entire projects.
- Smart routing: detects if ID is app-id or project-id
- App-id format (project-id-app-name) deletes single application
- Project-id format deletes all applications in project
- Removes from both cache and persistent storage

**Error Handling**:
- Standardized error responses with error codes and details
- Proper HTTP status codes for different error types
- Structured logging for debugging and monitoring
- Graceful degradation when dependencies fail

**Input Validation**:
- App-id format validation (project-id-app-name pattern)
- JSON body validation for dev version requests
- Query parameter validation for increment types

**Integration Points**:
- Delegates all business logic to `internal/services.VersionService`
- Uses `internal/models` for request/response structures
- Integrates with `internal/middleware` for logging and metrics
- Follows Swagger/OpenAPI documentation standards

**Relationship to Application**:
This package serves as the HTTP interface layer, translating REST API calls into service operations while maintaining clean separation of concerns between HTTP handling and business logic.