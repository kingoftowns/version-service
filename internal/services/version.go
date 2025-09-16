package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/company/version-service/internal/models"
	"github.com/company/version-service/internal/storage"
	"github.com/company/version-service/pkg/semver"
	"github.com/sirupsen/logrus"
)

type VersionService struct {
	redis  storage.Storage
	git    storage.Storage
	logger *logrus.Logger
	mu     sync.RWMutex
}

func NewVersionService(redis storage.Storage, git storage.Storage, logger *logrus.Logger) *VersionService {
	return &VersionService{
		redis:  redis,
		git:    git,
		logger: logger,
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
			version = &models.AppVersion{
				Current:     "1.0.0",
				Next:        "1.0.1",
				ProjectID:   projectID,
				AppName:     appName,
				LastUpdated: time.Now(),
			}

			if err := s.saveVersion(ctx, appID, version); err != nil {
				return nil, err
			}
		} else {
			go func() {
				if err := s.redis.SetVersion(context.Background(), appID, version); err != nil {
					s.logger.WithError(err).WithField("app_id", appID).Warn("Failed to cache version in Redis")
				}
			}()
		}
	}

	nextVersion, err := s.calculateNextVersion(version.Current, models.IncrementTypePatch)
	if err != nil {
		return nil, err
	}
	version.Next = nextVersion

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

	v, err := semver.Parse(currentVersion.Next)
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

	for appID, version := range versions {
		nextVersion, err := s.calculateNextVersion(version.Current, models.IncrementTypePatch)
		if err != nil {
			s.logger.WithError(err).WithField("app_id", appID).Warn("Failed to calculate next version")
			continue
		}
		version.Next = nextVersion
	}

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

	for appID, version := range versions {
		nextVersion, err := s.calculateNextVersion(version.Current, models.IncrementTypePatch)
		if err != nil {
			s.logger.WithError(err).WithField("app_id", appID).Warn("Failed to calculate next version")
			continue
		}
		version.Next = nextVersion
	}

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
	if err := s.git.SetVersion(ctx, appID, version); err != nil {
		return fmt.Errorf("failed to save version to Git: %w", err)
	}

	go func() {
		if err := s.redis.SetVersion(context.Background(), appID, version); err != nil {
			s.logger.WithError(err).WithField("app_id", appID).Warn("Failed to cache version in Redis")
		}
	}()

	return nil
}

func (s *VersionService) Health(ctx context.Context) map[string]string {
	checks := make(map[string]string)

	if err := s.redis.Health(ctx); err != nil {
		checks["redis"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		checks["redis"] = "healthy"
	}

	if err := s.git.Health(ctx); err != nil {
		checks["git"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		checks["git"] = "healthy"
	}

	return checks
}