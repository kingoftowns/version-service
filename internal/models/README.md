# Internal/Models Package

## Overview
The models package defines data structures and types used throughout the version service. Provides the core domain models for version management, API requests/responses, and system configuration.

## Components

### Core Data Structures (version.go)

#### AppVersion
Primary domain model representing a versioned application.

**Fields**:
- `Current` - Current semantic version string (e.g., "1.2.3")
- `ProjectID` - Project identifier extracted from app-id
- `AppName` - Application name extracted from app-id
- `RepoName` - Optional repository name for metadata
- `LastUpdated` - Timestamp of last version change

**Purpose**:
- Represents the complete state of an application's version
- Used for storage persistence and API responses
- Contains metadata for tracking and auditing

#### DevVersionRequest
Request structure for development version generation.

**Fields**:
- `SHA` - Git commit SHA (required for dev version suffix)
- `Branch` - Git branch name (required for context)

**Purpose**:
- Input validation for POST /version/{app-id}/dev endpoint
- Ensures required fields are present for dev version creation

#### IncrementType
Enumeration for semantic version increment operations.

**Values**:
- `IncrementTypePatch` - Patch level increment (1.2.3 → 1.2.4)
- `IncrementTypeMinor` - Minor level increment (1.2.3 → 1.3.0)
- `IncrementTypeMajor` - Major level increment (1.2.3 → 2.0.0)

**Purpose**:
- Type-safe specification of version increment behavior
- Used by increment endpoint and service logic

### API Response Models

#### VersionResponse
Simplified version response for API endpoints.

**Fields**:
- `Version` - Version string only

**Purpose**:
- Lightweight response for increment and dev version operations
- Focused on version value without metadata

#### ErrorResponse
Standardized error response structure.

**Fields**:
- `Error` - Human-readable error message
- `Code` - Error code for client handling (optional)
- `Details` - Additional error details for debugging (optional)

**Purpose**:
- Consistent error response format across all endpoints
- Enables structured error handling in clients

#### HealthResponse
Health check response structure.

**Fields**:
- `Status` - Overall health status ("healthy"/"unhealthy")
- `Checks` - Map of individual component health results

**Purpose**:
- Provides detailed health information for monitoring
- Enables granular health check visibility

### Storage Models

#### VersionsFile
Structure for Git-based persistence format.

**Fields**:
- `Versions` - Map of app-id to AppVersion objects
- `LastUpdated` - File-level timestamp

**Purpose**:
- JSON serialization format for Git storage
- Maintains file-level metadata for versioning

### Utility Functions

#### ParseAppID(appID) → (projectID, appName, error)
Parses composite app-id into constituent parts.

**Format**: `project-id-app-name` (dash-separated)
**Purpose**: Enables project-level operations and validation

#### FormatAppID(projectID, appName) → appID
Constructs app-id from project and application components.

**Purpose**: Consistent app-id formatting across the system

**Relationship to Application**:
These models define the contract between all service layers, ensuring consistent data representation from HTTP handlers through business logic to storage persistence. The app-id parsing functions enable hierarchical organization where projects contain multiple applications.