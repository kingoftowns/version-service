package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/company/version-service/internal/clients"
	"github.com/company/version-service/internal/models"
	"github.com/company/version-service/internal/storage"
	"github.com/company/version-service/pkg/semver"
	"github.com/sirupsen/logrus"
)

type VersionService struct {
	redis         storage.Storage
	git           storage.Storage
	gitLabClient  *clients.GitLabClient
	logger        *logrus.Logger
	mu            sync.RWMutex
	gitHealth     gitHealthStatus
	gitHealthMu   sync.RWMutex
	gitMetrics    gitMetrics
	gitMetricsMu  sync.RWMutex
	pushNeeded    bool
}

type gitHealthStatus struct {
	lastSuccess    time.Time
	lastFailure    time.Time
	recentFailures int
}

type gitMetrics struct {
	operationsTotal     int64
	operationsSucceeded int64
	operationsFailed    int64
	retriesTotal        int64
	lastOperationTime   time.Time
	avgLatencyMs        float64
}


func NewVersionService(redis storage.Storage, git storage.Storage, gitLabClient *clients.GitLabClient, logger *logrus.Logger) *VersionService {
	return &VersionService{
		redis:        redis,
		git:          git,
		gitLabClient: gitLabClient,
		logger:       logger,
		gitHealth: gitHealthStatus{
			lastSuccess: time.Now(),
		},
	}
}

func (s *VersionService) Initialize(ctx context.Context) error {
	versions, err := s.git.ListVersions(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to load versions from Git")
		return fmt.Errorf("failed to load versions from Git: %w", err)
	}

	if err := s.redis.RebuildCache(ctx, versions); err != nil {
		s.logger.WithError(err).Warn("Failed to rebuild Redis cache")
	}

	s.logger.WithField("count", len(versions)).Info("Version service initialized")

	// Start background goroutines
	go s.logMetricsPeriodically()
	go s.periodicPushRetry()

	return nil
}

func (s *VersionService) GetVersion(ctx context.Context, appID string) (*models.AppVersion, error) {
	projectID, appName, err := models.ParseAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	version, err := s.redis.GetVersion(ctx, appID)
	if err != nil {
		s.logger.WithError(err).WithField("app_id", appID).Warn("Failed to get version from Redis")
	}

	if version == nil {
		version, err = s.git.GetVersion(ctx, appID)
		if err != nil {
			return nil, fmt.Errorf("failed to get version from Git: %w", err)
		}

		if version == nil {
			// Try to find existing tags from GitLab
			var initialVersion string
			if s.gitLabClient != nil {
				gitLabTag, err := s.gitLabClient.GetLatestTag(ctx, projectID)
				if err != nil {
					s.logger.WithError(err).WithFields(logrus.Fields{
						"app_id":     appID,
						"project_id": projectID,
					}).Warn("Failed to fetch tags from GitLab, using default version")
				} else if gitLabTag != "" {
					initialVersion = gitLabTag
					s.logger.WithFields(logrus.Fields{
						"app_id":     appID,
						"project_id": projectID,
						"version":    gitLabTag,
					}).Info("Using latest tag from GitLab as initial version")
				}
			}

			// Use GitLab tag if found, otherwise default to 1.0.0
			if initialVersion == "" {
				initialVersion = "1.0.0"
			}

			version = &models.AppVersion{
				Current:     initialVersion,
				ProjectID:   projectID,
				AppName:     appName,
				LastUpdated: time.Now(),
			}

			if err := s.saveVersion(ctx, appID, version); err != nil {
				return nil, err
			}
		} else {
			// Cache in Redis synchronously when fetched from Git
			if err := s.redis.SetVersion(ctx, appID, version); err != nil {
				s.logger.WithError(err).WithField("app_id", appID).Warn("Failed to cache version in Redis")
				// Non-fatal: continue even if caching fails
			} else {
				s.logger.WithFields(logrus.Fields{
					"app_id":  appID,
					"version": version.Current,
				}).Debug("Version cached in Redis from Git")
			}
		}
	}

	return version, nil
}

func (s *VersionService) IncrementVersion(ctx context.Context, appID string, incrementType models.IncrementType) (*models.VersionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectID, appName, err := models.ParseAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	currentVersion, err := s.GetVersion(ctx, appID)
	if err != nil {
		return nil, err
	}

	newVersion, err := s.calculateNextVersion(currentVersion.Current, incrementType)
	if err != nil {
		return nil, err
	}

	updatedVersion := &models.AppVersion{
		Current:     newVersion,
		ProjectID:   projectID,
		AppName:     appName,
		LastUpdated: time.Now(),
	}

	if err := s.saveVersion(ctx, appID, updatedVersion); err != nil {
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"app_id":      appID,
		"old_version": currentVersion.Current,
		"new_version": newVersion,
		"type":        incrementType,
	}).Info("Version incremented")

	return &models.VersionResponse{Version: newVersion}, nil
}

func (s *VersionService) GetDevVersion(ctx context.Context, appID string, req *models.DevVersionRequest) (*models.VersionResponse, error) {
	currentVersion, err := s.GetVersion(ctx, appID)
	if err != nil {
		return nil, err
	}

	v, err := semver.Parse(currentVersion.Current)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	devVersion := v.WithDevSuffix(req.SHA)

	s.logger.WithFields(logrus.Fields{
		"app_id":  appID,
		"sha":     req.SHA,
		"branch":  req.Branch,
		"version": devVersion.String(),
	}).Debug("Dev version generated")

	return &models.VersionResponse{Version: devVersion.String()}, nil
}

func (s *VersionService) ListVersions(ctx context.Context) (map[string]*models.AppVersion, error) {
	versions, err := s.redis.ListVersions(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to list versions from Redis, falling back to Git")
		versions, err = s.git.ListVersions(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list versions: %w", err)
		}
	}

	// No need to calculate next versions anymore - simplified API

	return versions, nil
}

func (s *VersionService) ListVersionsByProject(ctx context.Context, projectID string) (map[string]*models.AppVersion, error) {
	versions, err := s.redis.ListVersionsByProject(ctx, projectID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to list versions from Redis, falling back to Git")
		versions, err = s.git.ListVersionsByProject(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list versions by project: %w", err)
		}
	}

	// No need to calculate next versions anymore - simplified API

	return versions, nil
}

func (s *VersionService) calculateNextVersion(current string, incrementType models.IncrementType) (string, error) {
	v, err := semver.Parse(current)
	if err != nil {
		return "", fmt.Errorf("invalid semantic version: %w", err)
	}

	var next *semver.Version
	switch incrementType {
	case models.IncrementTypeMajor:
		next = v.IncrementMajor()
	case models.IncrementTypeMinor:
		next = v.IncrementMinor()
	case models.IncrementTypePatch:
		next = v.IncrementPatch()
	default:
		next = v.IncrementPatch()
	}

	return next.String(), nil
}

func (s *VersionService) saveVersion(ctx context.Context, appID string, version *models.AppVersion) error {
	// Save to Redis first (synchronous - fast, critical path)
	if err := s.redis.SetVersion(ctx, appID, version); err != nil {
		return fmt.Errorf("failed to save version to Redis: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"app_id":  appID,
		"version": version.Current,
	}).Debug("Version cached in Redis")

	// Save to Git asynchronously (slow, network I/O)
	go func() {
		s.saveVersionToGitWithRetry(appID, version)
	}()

	return nil
}

func (s *VersionService) saveVersionToGitWithRetry(appID string, version *models.AppVersion) {
	const maxRetries = 3
	const baseDelay = time.Second
	startTime := time.Now()

	s.updateGitMetrics(true, 0, 0) // Start operation

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Create a new context with timeout for each attempt
		gitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		attemptStart := time.Now()

		err := s.git.SetVersion(gitCtx, appID, version)
		attemptLatency := time.Since(attemptStart)
		cancel()

		if err == nil {
			// Success - update health status and metrics
			totalLatency := time.Since(startTime)
			s.updateGitHealth(true)
			s.updateGitMetrics(false, attempt, totalLatency.Milliseconds())
			s.logger.WithFields(logrus.Fields{
				"app_id":     appID,
				"version":    version.Current,
				"attempt":    attempt + 1,
				"latency_ms": totalLatency.Milliseconds(),
			}).Info("Version persisted to Git")
			return
		}

		// Track retry
		if attempt > 0 {
			s.trackRetry()
		}

		// Check if this is a push failure (commit succeeded but push failed)
		if s.isPushFailure(err) {
			// For push failures, mark that we need a push retry
			s.markPushNeeded()
			totalLatency := time.Since(startTime)
			s.updateGitHealth(false)
			s.updateGitMetrics(false, attempt, totalLatency.Milliseconds())
			s.logger.WithError(err).WithFields(logrus.Fields{
				"app_id":     appID,
				"version":    version.Current,
				"attempt":    attempt + 1,
				"latency_ms": totalLatency.Milliseconds(),
			}).Warn("Version committed locally but push failed - will retry push in background")
			return
		}

		// Check if this is a retryable error
		if !s.isRetryableError(err) {
			totalLatency := time.Since(startTime)
			s.updateGitHealth(false)
			s.updateGitMetrics(false, attempt, totalLatency.Milliseconds())
			s.logger.WithError(err).WithFields(logrus.Fields{
				"app_id":     appID,
				"version":    version.Current,
				"attempt":    attempt + 1,
				"latency_ms": totalLatency.Milliseconds(),
			}).Error("Non-retryable error persisting version to Git")
			return
		}

		// Log the attempt
		s.logger.WithError(err).WithFields(logrus.Fields{
			"app_id":        appID,
			"version":       version.Current,
			"attempt":       attempt + 1,
			"max_retries":   maxRetries,
			"attempt_latency_ms": attemptLatency.Milliseconds(),
		}).Warn("Failed to persist version to Git, will retry")

		// Wait before retry (exponential backoff)
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<attempt) // 1s, 2s, 4s
			time.Sleep(delay)
		}
	}

	// All retries exhausted - mark push needed for background process
	totalLatency := time.Since(startTime)
	s.updateGitHealth(false)
	s.updateGitMetrics(false, maxRetries-1, totalLatency.Milliseconds())

	// Mark that push is needed for background push process
	s.markPushNeeded()

	s.logger.WithFields(logrus.Fields{
		"app_id":     appID,
		"version":    version.Current,
		"attempts":   maxRetries,
		"latency_ms": totalLatency.Milliseconds(),
	}).Error("Failed to persist version to Git after all retries - will retry push in background")
}

func (s *VersionService) isRetryableError(err error) bool {
	// Consider network, timeout, and temporary Git service errors as retryable
	// Authentication and permission errors are not retryable
	errStr := err.Error()

	// Non-retryable errors
	if containsAny(errStr, []string{
		"authentication failed",
		"permission denied",
		"access denied",
		"unauthorized",
		"forbidden",
		"invalid credentials",
	}) {
		return false
	}

	// Retryable errors (network, timeout, temporary issues)
	return containsAny(errStr, []string{
		"timeout",
		"connection",
		"network",
		"temporary",
		"unavailable",
		"refused",
		"reset",
		"push failed",
	}) || true // Default to retryable for unknown errors
}

func (s *VersionService) isPushFailure(err error) bool {
	errStr := err.Error()
	return containsAny(errStr, []string{
		"push failed",
		"failed to push",
	})
}

func containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

func (s *VersionService) updateGitHealth(success bool) {
	s.gitHealthMu.Lock()
	defer s.gitHealthMu.Unlock()

	if success {
		s.gitHealth.lastSuccess = time.Now()
		s.gitHealth.recentFailures = 0
	} else {
		s.gitHealth.lastFailure = time.Now()
		s.gitHealth.recentFailures++
	}
}

func (s *VersionService) updateGitMetrics(isStart bool, retries int, latencyMs int64) {
	s.gitMetricsMu.Lock()
	defer s.gitMetricsMu.Unlock()

	if isStart {
		s.gitMetrics.operationsTotal++
		s.gitMetrics.lastOperationTime = time.Now()
	} else {
		// Operation completed
		if latencyMs > 0 {
			// Update success or failure
			if retries == 0 || latencyMs < 30000 { // If no retries or completed within timeout
				s.gitMetrics.operationsSucceeded++
			} else {
				s.gitMetrics.operationsFailed++
			}

			// Update average latency (simple moving average approximation)
			if s.gitMetrics.avgLatencyMs == 0 {
				s.gitMetrics.avgLatencyMs = float64(latencyMs)
			} else {
				s.gitMetrics.avgLatencyMs = (s.gitMetrics.avgLatencyMs*0.9 + float64(latencyMs)*0.1)
			}
		} else {
			// Failed operation
			s.gitMetrics.operationsFailed++
		}
	}
}

func (s *VersionService) trackRetry() {
	s.gitMetricsMu.Lock()
	defer s.gitMetricsMu.Unlock()
	s.gitMetrics.retriesTotal++
}

func (s *VersionService) logMetricsPeriodically() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.gitMetricsMu.RLock()
		metrics := s.gitMetrics
		s.gitMetricsMu.RUnlock()

		if metrics.operationsTotal > 0 {
			successRate := float64(metrics.operationsSucceeded) / float64(metrics.operationsTotal) * 100
			s.logger.WithFields(logrus.Fields{
				"git_operations_total":     metrics.operationsTotal,
				"git_operations_succeeded": metrics.operationsSucceeded,
				"git_operations_failed":    metrics.operationsFailed,
				"git_success_rate_pct":     fmt.Sprintf("%.1f", successRate),
				"git_retries_total":        metrics.retriesTotal,
				"git_avg_latency_ms":       fmt.Sprintf("%.1f", metrics.avgLatencyMs),
				"git_last_operation":       metrics.lastOperationTime.Format(time.RFC3339),
			}).Info("Git operation metrics")
		}
	}
}

func (s *VersionService) markPushNeeded() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pushNeeded = true
}

func (s *VersionService) periodicPushRetry() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		pushNeeded := s.pushNeeded
		s.mu.RUnlock()

		if pushNeeded {
			s.logger.Info("Starting periodic Git push retry")
			if err := s.retryPendingPushes(); err != nil {
				s.logger.WithError(err).Error("Failed to push pending commits")
			} else {
				s.mu.Lock()
				s.pushNeeded = false
				s.mu.Unlock()
				s.logger.Info("Successfully pushed pending commits")
			}
		}
	}
}

func (s *VersionService) retryPendingPushes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Check if the Git storage supports push operations
	gitPushable, ok := s.git.(storage.GitPushable)
	if !ok {
		return fmt.Errorf("Git storage does not support push operations")
	}

	// Try to push pending commits
	if err := gitPushable.PushPendingCommits(ctx); err != nil {
		return fmt.Errorf("failed to push pending commits: %w", err)
	}

	s.updateGitHealth(true)
	return nil
}

func (s *VersionService) Health(ctx context.Context) map[string]string {
	checks := make(map[string]string)

	if err := s.redis.Health(ctx); err != nil {
		checks["redis"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		checks["redis"] = "healthy"
	}

	// Enhanced Git health check that considers recent operation status
	s.gitHealthMu.RLock()
	gitHealth := s.gitHealth
	s.gitHealthMu.RUnlock()

	if err := s.git.Health(ctx); err != nil {
		checks["git"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		// Check recent Git operation status (last 5 minutes)
		now := time.Now()
		recentWindow := 5 * time.Minute

		if gitHealth.recentFailures > 0 && now.Sub(gitHealth.lastFailure) < recentWindow {
			// Recent failures within the window
			if gitHealth.recentFailures >= 3 {
				checks["git"] = fmt.Sprintf("degraded: %d recent failures, last failure: %s",
					gitHealth.recentFailures, gitHealth.lastFailure.Format(time.RFC3339))
			} else {
				checks["git"] = fmt.Sprintf("degraded: %d recent failures",
					gitHealth.recentFailures)
			}
		} else if !gitHealth.lastSuccess.IsZero() && now.Sub(gitHealth.lastSuccess) > recentWindow {
			// No recent successes
			checks["git"] = fmt.Sprintf("degraded: no successful operations in last %v",
				recentWindow)
		} else {
			checks["git"] = "healthy"
		}
	}

	return checks
}