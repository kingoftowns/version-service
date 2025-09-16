package services

import (
	"context"
	"github.com/company/version-service/internal/models"
)

type VersionServiceInterface interface {
	Health(ctx context.Context) map[string]string
	GetVersion(ctx context.Context, appID string) (*models.AppVersion, error)
	IncrementVersion(ctx context.Context, appID string, incrementType models.IncrementType) (*models.VersionResponse, error)
	GetDevVersion(ctx context.Context, appID string, req *models.DevVersionRequest) (*models.VersionResponse, error)
	ListVersions(ctx context.Context) (map[string]*models.AppVersion, error)
	ListVersionsByProject(ctx context.Context, projectID string) (map[string]*models.AppVersion, error)
}