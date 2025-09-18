# Internal/Middleware Package

## Overview
The middleware package provides HTTP middleware components for request processing, logging, and metrics collection. Implements cross-cutting concerns that apply to all HTTP requests.

## Components

### LoggingMiddleware (logging.go)
Structured HTTP request logging middleware using logrus.

**Purpose**:
- Logs all HTTP requests with comprehensive metadata
- Provides request timing and performance insights
- Enables audit trails and debugging information

**Key Functionality**:
- Captures request start time and calculates latency
- Logs client IP, HTTP method, full path with query parameters
- Records response status codes for monitoring
- Uses appropriate log levels based on HTTP status (Error 5xx, Warn 4xx, Info 2xx/3xx)

**Log Fields**:
- `latency` - Request processing duration
- `client_ip` - Client IP address for request tracking
- `method` - HTTP method (GET, POST, etc.)
- `path` - Full request path including query parameters
- `status_code` - HTTP response status code

**Integration Points**:
- Applied globally in `main.go` router setup
- Uses logrus logger instance passed from application initialization
- Executes after request processing to capture complete request lifecycle

### MetricsMiddleware (metrics.go)
Prometheus metrics collection middleware for observability.

**Purpose**:
- Collects HTTP request metrics for monitoring and alerting
- Provides performance insights and SLA tracking
- Enables operational visibility into service behavior

**Metrics Collected**:
- `http_request_duration_seconds` - Histogram of request latencies by method, path, status
- `http_requests_total` - Counter of total requests by method, path, status
- `version_operations_total` - Counter of version-specific operations by type, app-id, status

**Key Functionality**:
- `MetricsMiddleware()` - Collects general HTTP metrics
- `RecordVersionOperation(operation, appID, status)` - Records domain-specific version operation metrics
- Uses Prometheus client library with automatic registration
- Measures request duration with high precision timing

**Metric Labels**:
- HTTP metrics: method, path (route template), status
- Version operation metrics: operation type, app-id, success/error status
- Enables detailed filtering and aggregation in monitoring systems

**Integration Points**:
- Applied globally for all HTTP endpoints
- Used by handlers to record specific version operations
- Metrics exposed via `/metrics` endpoint for Prometheus scraping
- Follows Prometheus naming conventions and best practices

**Relationship to Application**:
These middleware components provide essential observability and debugging capabilities, enabling operational visibility into request patterns, performance characteristics, and system health without impacting core business logic.