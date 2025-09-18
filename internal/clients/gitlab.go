package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/company/version-service/pkg/semver"
	"github.com/sirupsen/logrus"
)

type GitLabClient struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
	logger      *logrus.Logger
}

type GitLabTag struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Target  string `json:"target"`
	Commit  struct {
		ID             string    `json:"id"`
		ShortID        string    `json:"short_id"`
		Title          string    `json:"title"`
		CreatedAt      time.Time `json:"created_at"`
		AuthorName     string    `json:"author_name"`
		AuthorEmail    string    `json:"author_email"`
		CommittedDate  time.Time `json:"committed_date"`
	} `json:"commit"`
	Release *struct {
		TagName     string `json:"tag_name"`
		Description string `json:"description"`
	} `json:"release"`
}

func NewGitLabClient(baseURL, accessToken string, logger *logrus.Logger) *GitLabClient {
	return &GitLabClient{
		baseURL:     baseURL,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (c *GitLabClient) GetLatestTag(ctx context.Context, projectID string) (string, error) {
	if c.accessToken == "" {
		c.logger.Debug("GitLab access token not configured, skipping tag lookup")
		return "", nil
	}

	url := fmt.Sprintf("%s/projects/%s/repository/tags", c.baseURL, projectID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags from GitLab: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.WithField("project_id", projectID).Debug("GitLab project not found")
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"status":     resp.StatusCode,
		}).Warn("GitLab API returned non-OK status")
		return "", fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	var tags []GitLabTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", fmt.Errorf("failed to decode GitLab response: %w", err)
	}

	if len(tags) == 0 {
		c.logger.WithField("project_id", projectID).Debug("No tags found in GitLab project")
		return "", nil
	}

	// Find the latest semantic version tag
	latestVersion := c.findLatestSemanticVersion(tags)
	if latestVersion != "" {
		c.logger.WithFields(logrus.Fields{
			"project_id": projectID,
			"version":    latestVersion,
		}).Info("Found latest tag from GitLab")
	}

	return latestVersion, nil
}

func (c *GitLabClient) findLatestSemanticVersion(tags []GitLabTag) string {
	var validVersions []struct {
		tag     string
		version *semver.Version
	}

	for _, tag := range tags {
		// Try to parse the tag as a semantic version
		// Handle tags with or without 'v' prefix
		tagName := tag.Name
		if tagName != "" && tagName[0] == 'v' {
			tagName = tagName[1:]
		}

		version, err := semver.Parse(tagName)
		if err == nil {
			validVersions = append(validVersions, struct {
				tag     string
				version *semver.Version
			}{
				tag:     tag.Name,
				version: version,
			})
		}
	}

	if len(validVersions) == 0 {
		return ""
	}

	// Sort versions in descending order (latest first)
	sort.Slice(validVersions, func(i, j int) bool {
		vi := validVersions[i].version
		vj := validVersions[j].version

		if vi.Major != vj.Major {
			return vi.Major > vj.Major
		}
		if vi.Minor != vj.Minor {
			return vi.Minor > vj.Minor
		}
		if vi.Patch != vj.Patch {
			return vi.Patch > vj.Patch
		}
		// Consider pre-release versions as lower
		if vi.Prerelease != "" && vj.Prerelease == "" {
			return false
		}
		if vi.Prerelease == "" && vj.Prerelease != "" {
			return true
		}
		return vi.Prerelease > vj.Prerelease
	})

	// Return the version string without 'v' prefix for consistency
	latestTag := validVersions[0].tag
	if latestTag != "" && latestTag[0] == 'v' {
		return latestTag[1:]
	}
	return latestTag
}