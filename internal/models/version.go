package models

import (
	"fmt"
	"strings"
	"time"
)

type AppVersion struct {
	Current     string    `json:"current"`
	ProjectID   string    `json:"project_id"`
	AppName     string    `json:"app_name"`
	RepoName    string    `json:"repo_name,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

type DevVersionRequest struct {
	SHA    string `json:"sha" binding:"required"`
	Branch string `json:"branch" binding:"required"`
}

type IncrementType string

const (
	IncrementTypePatch IncrementType = "patch"
	IncrementTypeMinor IncrementType = "minor"
	IncrementTypeMajor IncrementType = "major"
)

type VersionResponse struct {
	Version string `json:"version"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type VersionsFile struct {
	Versions    map[string]*AppVersion `json:"versions"`
	LastUpdated time.Time              `json:"last_updated"`
}

func ParseAppID(appID string) (projectID, appName string, err error) {
	parts := strings.Split(appID, "-")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid app ID format: %s", appID)
	}
	projectID = parts[0]
	appName = strings.Join(parts[1:], "-")
	return projectID, appName, nil
}

func FormatAppID(projectID, appName string) string {
	return fmt.Sprintf("%s-%s", projectID, appName)
}