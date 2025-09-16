# Build stage
FROM golang:1.24-alpine AS builder

# Install git and ca-certificates for HTTPS
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o version-service \
    main.go

# Final stage
FROM alpine:3.19

# Install ca-certificates and git (needed for git operations)
RUN apk add --no-cache ca-certificates git tzdata && \
    adduser -D -g '' appuser

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary from builder
COPY --from=builder /build/version-service /usr/local/bin/version-service

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/usr/local/bin/version-service"]