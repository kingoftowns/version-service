# Internal/Clients Package

## Overview
The clients package provides external service integration clients for the version service. Currently contains the GitLab API client for fetching repository tags and version information.

## Components

### GitLabClient (gitlab.go)
Primary client for GitLab API integration that fetches repository tags to determine existing version information.

**Purpose**:
- Retrieves latest semantic version tags from GitLab repositories
- Used as fallback when initializing new applications without existing version data
- Provides version discovery for projects that already have tagged releases

**Key Functionality**:
- `GetLatestTag(ctx, projectID)` - Fetches and parses repository tags from GitLab API
- `findLatestSemanticVersion(tags)` - Filters and sorts tags to find the highest semantic version
- Handles both 'v' prefixed and non-prefixed version tags
- Implements proper error handling for missing projects and API failures

**Integration Points**:
- Used by `internal/services.VersionService.GetVersion()` when no version exists in storage
- Depends on `pkg/semver` for version parsing and comparison
- Configured via `internal/config` for base URL and access token

**Data Structures**:
- `GitLabTag` - Represents GitLab API tag response with commit metadata
- Includes release information and commit details for comprehensive tag data

**Error Handling**:
- Gracefully handles missing access tokens (logs debug, returns empty)
- Returns nil for non-existent projects (404 responses)
- Logs warnings for API errors while allowing service to continue

**Relationship to Application**:
This client enables the version service to bootstrap new applications with existing GitLab tag versions rather than defaulting to 1.0.0, providing continuity for projects migrating to the version service.