package storage

import (
	"context"
	"github.com/company/version-service/internal/models"
)

type Storage interface {
	GetVersion(ctx context.Context, appID string) (*models.AppVersion, error)
	SetVersion(ctx context.Context, appID string, version *models.AppVersion) error
	ListVersions(ctx context.Context) (map[string]*models.AppVersion, error)
	ListVersionsByProject(ctx context.Context, projectID string) (map[string]*models.AppVersion, error)
	DeleteVersion(ctx context.Context, appID string) error
	Health(ctx context.Context) error
	RebuildCache(ctx context.Context, versions map[string]*models.AppVersion) error
}

// GitStorage specific interface for push operations
type GitPushable interface {
	PushPendingCommits(ctx context.Context) error
}