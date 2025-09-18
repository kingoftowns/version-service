package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/company/version-service/internal/models"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"
)

const (
	versionsFileName = "versions.json"
	commitMessage    = "Update versions"
	tempDirPrefix    = "version-service-"
)

type GitStorage struct {
	repoURL  string
	branch   string
	username string
	token    string
	localDir string
	repo     *git.Repository
	logger   *logrus.Logger
	mu       sync.Mutex
}

func NewGitStorage(repoURL, branch, username, token string, logger *logrus.Logger) (*GitStorage, error) {
	tempDir, err := os.MkdirTemp("", tempDirPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	gs := &GitStorage{
		repoURL:  repoURL,
		branch:   branch,
		username: username,
		token:    token,
		localDir: tempDir,
		logger:   logger,
	}

	if err := gs.clone(); err != nil {
		return nil, err
	}

	return gs, nil
}

func (g *GitStorage) clone() error {
	auth := &http.BasicAuth{
		Username: g.username,
		Password: g.token,
	}

	repo, err := git.PlainClone(g.localDir, false, &git.CloneOptions{
		URL:           g.repoURL,
		Auth:          auth,
		ReferenceName: plumbing.NewBranchReferenceName(g.branch),
		SingleBranch:  true,
		Progress:      nil,
	})

	if err != nil {
		// Handle empty repository case
		if err.Error() == "remote repository is empty" {
			g.logger.Info("Repository is empty, initializing new repository")

			// Initialize a new repository locally
			repo, err = git.PlainInit(g.localDir, false)
			if err != nil {
				return fmt.Errorf("failed to initialize repository: %w", err)
			}

			// Add remote
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{g.repoURL},
			})
			if err != nil {
				return fmt.Errorf("failed to add remote: %w", err)
			}

			// Create initial versions.json file
			vf := &models.VersionsFile{
				Versions:    make(map[string]*models.AppVersion),
				LastUpdated: time.Now(),
			}

			// Write the file directly since writeVersionsFile might depend on g.repo
			data, err := json.MarshalIndent(vf, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal versions file: %w", err)
			}
			filePath := filepath.Join(g.localDir, versionsFileName)
			if err := os.WriteFile(filePath, data, 0644); err != nil {
				return fmt.Errorf("failed to write initial versions file: %w", err)
			}

			// Create initial commit
			w, err := repo.Worktree()
			if err != nil {
				return fmt.Errorf("failed to get worktree: %w", err)
			}

			if _, err := w.Add(versionsFileName); err != nil {
				return fmt.Errorf("failed to add versions file: %w", err)
			}

			_, err = w.Commit("Initial commit", &git.CommitOptions{
				Author: &object.Signature{
					Name:  "Version Service",
					Email: "version-service@company.com",
					When:  time.Now(),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to create initial commit: %w", err)
			}

			// Push to remote to create the branch
			err = repo.Push(&git.PushOptions{
				Auth:       auth,
				RemoteName: "origin",
				RefSpecs: []config.RefSpec{
					config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", g.branch, g.branch)),
				},
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				g.logger.WithError(err).Warn("Failed to push initial commit, repository might stay empty")
			}

			g.repo = repo
			g.logger.Info("Empty repository initialized successfully")
			return nil
		}

		g.logger.WithError(err).Error("Failed to clone repository")
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	g.repo = repo
	g.logger.WithFields(logrus.Fields{
		"repo":   g.repoURL,
		"branch": g.branch,
	}).Info("Repository cloned successfully")

	return nil
}

func (g *GitStorage) pull() error {
	auth := &http.BasicAuth{
		Username: g.username,
		Password: g.token,
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Pull(&git.PullOptions{
		Auth:          auth,
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(g.branch),
		SingleBranch:  true,
		Force:         false,
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		if err.Error() == "remote repository is empty" {
			g.logger.Debug("Repository is empty, no changes to pull")
			return nil
		}
		g.logger.WithError(err).Warn("Pull failed, attempting reset and pull")

		ref, err := g.repo.Head()
		if err != nil {
			return fmt.Errorf("failed to get HEAD: %w", err)
		}

		if err := w.Reset(&git.ResetOptions{
			Commit: ref.Hash(),
			Mode:   git.HardReset,
		}); err != nil {
			return fmt.Errorf("failed to reset: %w", err)
		}

		err = w.Pull(&git.PullOptions{
			Auth:          auth,
			RemoteName:    "origin",
			ReferenceName: plumbing.NewBranchReferenceName(g.branch),
			SingleBranch:  true,
			Force:         true,
		})

		if err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("failed to pull: %w", err)
		}
	}

	return nil
}

func (g *GitStorage) push() error {
	auth := &http.BasicAuth{
		Username: g.username,
		Password: g.token,
	}

	err := g.repo.Push(&git.PushOptions{
		Auth:       auth,
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", g.branch, g.branch)),
		},
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

func (g *GitStorage) readVersionsFile() (*models.VersionsFile, error) {
	filePath := filepath.Join(g.localDir, versionsFileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.VersionsFile{
				Versions:    make(map[string]*models.AppVersion),
				LastUpdated: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to read versions file: %w", err)
	}

	var vf models.VersionsFile
	if err := json.Unmarshal(data, &vf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal versions file: %w", err)
	}

	if vf.Versions == nil {
		vf.Versions = make(map[string]*models.AppVersion)
	}

	return &vf, nil
}

func (g *GitStorage) writeVersionsFile(vf *models.VersionsFile) error {
	vf.LastUpdated = time.Now()

	data, err := json.MarshalIndent(vf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal versions file: %w", err)
	}

	filePath := filepath.Join(g.localDir, versionsFileName)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write versions file: %w", err)
	}

	return nil
}

func (g *GitStorage) commit(message string) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := w.Add(versionsFileName); err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	commit, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Version Service",
			Email: "version-service@company.com",
			When:  time.Now(),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	g.logger.WithField("commit", commit.String()).Debug("Changes committed")
	return nil
}

func (g *GitStorage) commitAndPush(message string) error {
	if err := g.commit(message); err != nil {
		return err
	}

	if err := g.push(); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

func (g *GitStorage) hasUnpushedCommits() (bool, error) {
	// Get local head
	localRef, err := g.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get local HEAD: %w", err)
	}

	// Get remote head
	remote, err := g.repo.Remote("origin")
	if err != nil {
		return false, fmt.Errorf("failed to get remote: %w", err)
	}

	auth := &http.BasicAuth{
		Username: g.username,
		Password: g.token,
	}

	refs, err := remote.List(&git.ListOptions{Auth: auth})
	if err != nil {
		// If we can't list remote refs, assume we have unpushed commits
		g.logger.WithError(err).Debug("Failed to list remote refs, assuming unpushed commits exist")
		return true, nil
	}

	// Find the remote branch reference
	remoteBranchRef := fmt.Sprintf("refs/heads/%s", g.branch)
	for _, ref := range refs {
		if ref.Name().String() == remoteBranchRef {
			// Compare local and remote commit hashes
			return localRef.Hash() != ref.Hash(), nil
		}
	}

	// Remote branch doesn't exist, so we have unpushed commits
	return true, nil
}

func (g *GitStorage) PushPendingCommits(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	hasUnpushed, err := g.hasUnpushedCommits()
	if err != nil {
		return fmt.Errorf("failed to check for unpushed commits: %w", err)
	}

	if !hasUnpushed {
		g.logger.Debug("No unpushed commits found")
		return nil
	}

	g.logger.Info("Pushing pending commits to remote")
	if err := g.push(); err != nil {
		return fmt.Errorf("failed to push pending commits: %w", err)
	}

	g.logger.Info("Successfully pushed pending commits")
	return nil
}

func (g *GitStorage) GetVersion(ctx context.Context, appID string) (*models.AppVersion, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pull(); err != nil {
		if err.Error() == "remote repository is empty" {
			g.logger.Debug("Repository is empty, no changes to pull")
		} else {
			g.logger.WithError(err).Warn("Failed to pull latest changes")
		}
	}

	vf, err := g.readVersionsFile()
	if err != nil {
		return nil, err
	}

	version, exists := vf.Versions[appID]
	if !exists {
		return nil, nil
	}

	return version, nil
}

func (g *GitStorage) SetVersion(ctx context.Context, appID string, version *models.AppVersion) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pull(); err != nil {
		if err.Error() == "remote repository is empty" {
			g.logger.Debug("Repository is empty, no changes to pull")
		} else {
			g.logger.WithError(err).Warn("Failed to pull latest changes")
		}
	}

	vf, err := g.readVersionsFile()
	if err != nil {
		return err
	}

	vf.Versions[appID] = version

	if err := g.writeVersionsFile(vf); err != nil {
		return err
	}

	commitMsg := fmt.Sprintf("%s: Update %s to %s", commitMessage, appID, version.Current)

	// Commit locally first
	if err := g.commit(commitMsg); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Try to push, but don't fail the entire operation if push fails
	if err := g.push(); err != nil {
		g.logger.WithError(err).WithFields(logrus.Fields{
			"app_id":  appID,
			"version": version.Current,
		}).Warn("Failed to push to remote, commit saved locally")
		return fmt.Errorf("push failed: %w", err)
	}

	g.logger.WithFields(logrus.Fields{
		"app_id":  appID,
		"version": version.Current,
	}).Info("Version persisted to Git")

	return nil
}

func (g *GitStorage) ListVersions(ctx context.Context) (map[string]*models.AppVersion, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pull(); err != nil {
		if err.Error() == "remote repository is empty" {
			g.logger.Debug("Repository is empty, no changes to pull")
		} else {
			g.logger.WithError(err).Warn("Failed to pull latest changes")
		}
	}

	vf, err := g.readVersionsFile()
	if err != nil {
		return nil, err
	}

	return vf.Versions, nil
}

func (g *GitStorage) ListVersionsByProject(ctx context.Context, projectID string) (map[string]*models.AppVersion, error) {
	allVersions, err := g.ListVersions(ctx)
	if err != nil {
		return nil, err
	}

	projectVersions := make(map[string]*models.AppVersion)
	for appID, version := range allVersions {
		if strings.HasPrefix(appID, projectID+"-") {
			projectVersions[appID] = version
		}
	}

	return projectVersions, nil
}

func (g *GitStorage) DeleteVersion(ctx context.Context, appID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.pull(); err != nil {
		if err.Error() == "remote repository is empty" {
			g.logger.Debug("Repository is empty, no changes to pull")
		} else {
			g.logger.WithError(err).Warn("Failed to pull latest changes")
		}
	}

	vf, err := g.readVersionsFile()
	if err != nil {
		return err
	}

	delete(vf.Versions, appID)

	if err := g.writeVersionsFile(vf); err != nil {
		return err
	}

	commitMsg := fmt.Sprintf("%s: Remove %s", commitMessage, appID)
	if err := g.commitAndPush(commitMsg); err != nil {
		return err
	}

	g.logger.WithField("app_id", appID).Info("Version deleted from Git")
	return nil
}

func (g *GitStorage) Health(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.pull()
}

func (g *GitStorage) RebuildCache(ctx context.Context, versions map[string]*models.AppVersion) error {
	// Git storage doesn't use cache, so this is a no-op
	return nil
}

func (g *GitStorage) Close() error {
	if g.localDir != "" {
		return os.RemoveAll(g.localDir)
	}
	return nil
}