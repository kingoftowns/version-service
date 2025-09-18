# Pkg/Semver Package

## Overview
The semver package provides semantic versioning parsing, manipulation, and comparison functionality. Implements core semantic versioning operations required by the version service for version increment calculations and validation.

## Components

### Version Struct (semver.go)
Primary data structure representing a semantic version.

**Fields**:
- `Major` - Major version number (breaking changes)
- `Minor` - Minor version number (new features, backward compatible)
- `Patch` - Patch version number (bug fixes, backward compatible)
- `Prerelease` - Optional pre-release identifier (e.g., "dev-abc1234", "beta.1")

**Semantic Versioning Compliance**:
- Follows SemVer 2.0.0 specification
- Supports standard three-part versioning (major.minor.patch)
- Handles optional pre-release identifiers with dash separator
- Validates version format using regex pattern matching

### Core Functions

#### Parse(version) → (*Version, error)
Parses string representation into Version struct.

**Input Format**: `major.minor.patch[-prerelease]`
**Examples**: "1.2.3", "2.0.0-beta.1", "1.0.0-dev-abc1234"
**Validation**: Uses regex pattern to ensure strict SemVer compliance
**Error Handling**: Returns descriptive error for invalid format

#### String() → string
Converts Version struct back to canonical string representation.

**Output Format**: Always produces valid SemVer string
**Pre-release Handling**: Includes dash separator when pre-release exists
**Consistency**: Round-trip parsing (Parse → String → Parse) preserves equivalence

### Version Increment Methods

#### IncrementPatch() → *Version
Increments patch version, resets nothing.
- 1.2.3 → 1.2.4
- Used for bug fixes and backward-compatible changes
- Clears pre-release identifier

#### IncrementMinor() → *Version
Increments minor version, resets patch to 0.
- 1.2.3 → 1.3.0
- Used for new features that maintain backward compatibility
- Clears pre-release identifier

#### IncrementMajor() → *Version
Increments major version, resets minor and patch to 0.
- 1.2.3 → 2.0.0
- Used for breaking changes that affect backward compatibility
- Clears pre-release identifier

### Development Version Support

#### WithDevSuffix(sha) → *Version
Creates development version with commit SHA identifier.

**Purpose**: Generates unique pre-release versions for development builds
**Format**: Appends "dev-{short-sha}" as pre-release identifier
**SHA Handling**: Truncates SHA to 7 characters for brevity
**Example**: "1.2.3" + "abc1234567" → "1.2.3-dev-abc1234"

### Utility Functions

#### IsValid(version) → bool
Validates version string format without parsing.
- Convenience function for format checking
- Returns true for valid SemVer strings, false otherwise
- Used for input validation in API layers

#### Compare(v1, v2) → (int, error)
Compares two version strings lexicographically.

**Return Values**:
- Negative: v1 < v2
- Zero: v1 == v2
- Positive: v1 > v2

**Comparison Rules**:
1. Major version takes precedence
2. Minor version compared if major versions equal
3. Patch version compared if major and minor equal
4. Pre-release versions are considered lower than release versions
5. Pre-release identifiers compared lexicographically

**Error Handling**: Returns error if either version string is invalid

**Integration Points**:
- Used by `internal/services.VersionService` for increment operations
- Used by `internal/clients.GitLabClient` for version comparison and sorting
- Provides foundation for all version manipulation throughout the application

**Relationship to Application**:
This package provides the fundamental semantic versioning operations that enable the version service to correctly increment versions, compare version precedence, and generate development builds while maintaining strict compliance with semantic versioning standards.