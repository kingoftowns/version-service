package services

import (
	"context"
	"testing"
	"time"

	"github.com/company/version-service/internal/models"
	"github.com/company/version-service/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) GetVersion(ctx context.Context, appID string) (*models.AppVersion, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AppVersion), args.Error(1)
}

func (m *MockStorage) SetVersion(ctx context.Context, appID string, version *models.AppVersion) error {
	args := m.Called(ctx, appID, version)
	return args.Error(0)
}

func (m *MockStorage) ListVersions(ctx context.Context) (map[string]*models.AppVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.AppVersion), args.Error(1)
}

func (m *MockStorage) ListVersionsByProject(ctx context.Context, projectID string) (map[string]*models.AppVersion, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.AppVersion), args.Error(1)
}

func (m *MockStorage) DeleteVersion(ctx context.Context, appID string) error {
	args := m.Called(ctx, appID)
	return args.Error(0)
}

func (m *MockStorage) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestVersionService_GetVersion(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name       string
		appID      string
		setupMocks func(*MockStorage, *MockStorage)
		want       *models.AppVersion
		wantErr    bool
	}{
		{
			name:  "get existing version from redis",
			appID: "1234-user-service",
			setupMocks: func(redis *MockStorage, git *MockStorage) {
				existingVersion := &models.AppVersion{
					Current:     "1.2.3",
					ProjectID:   "1234",
					AppName:     "user-service",
					LastUpdated: time.Now(),
				}
				redis.On("GetVersion", mock.Anything, "1234-user-service").Return(existingVersion, nil)
			},
			want: &models.AppVersion{
				Current:   "1.2.3",
				Next:      "1.2.4",
				ProjectID: "1234",
				AppName:   "user-service",
			},
			wantErr: false,
		},
		{
			name:  "create new version when not exists",
			appID: "1234-new-service",
			setupMocks: func(redis *MockStorage, git *MockStorage) {
				redis.On("GetVersion", mock.Anything, "1234-new-service").Return(nil, nil)
				git.On("GetVersion", mock.Anything, "1234-new-service").Return(nil, nil)
				git.On("SetVersion", mock.Anything, "1234-new-service", mock.Anything).Return(nil)
			},
			want: &models.AppVersion{
				Current:   "1.0.0",
				Next:      "1.0.1",
				ProjectID: "1234",
				AppName:   "new-service",
			},
			wantErr: false,
		},
		{
			name:    "invalid app ID",
			appID:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := new(MockStorage)
			mockGit := new(MockStorage)

			if tt.setupMocks != nil {
				tt.setupMocks(mockRedis, mockGit)
			}

			service := &VersionService{
				redis:  mockRedis.(*storage.RedisStorage),
				git:    mockGit.(*storage.GitStorage),
				logger: logger,
			}

			got, err := service.GetVersion(context.Background(), tt.appID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.Current, got.Current)
				assert.Equal(t, tt.want.Next, got.Next)
				assert.Equal(t, tt.want.ProjectID, got.ProjectID)
				assert.Equal(t, tt.want.AppName, got.AppName)
			}

			mockRedis.AssertExpectations(t)
			mockGit.AssertExpectations(t)
		})
	}
}

func TestVersionService_CalculateNextVersion(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := &VersionService{
		logger: logger,
	}

	tests := []struct {
		name          string
		current       string
		incrementType models.IncrementType
		want          string
		wantErr       bool
	}{
		{
			name:          "increment patch",
			current:       "1.2.3",
			incrementType: models.IncrementTypePatch,
			want:          "1.2.4",
			wantErr:       false,
		},
		{
			name:          "increment minor",
			current:       "1.2.3",
			incrementType: models.IncrementTypeMinor,
			want:          "1.3.0",
			wantErr:       false,
		},
		{
			name:          "increment major",
			current:       "1.2.3",
			incrementType: models.IncrementTypeMajor,
			want:          "2.0.0",
			wantErr:       false,
		},
		{
			name:          "invalid version",
			current:       "invalid",
			incrementType: models.IncrementTypePatch,
			want:          "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.calculateNextVersion(tt.current, tt.incrementType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestVersionService_GetDevVersion(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRedis := new(MockStorage)
	mockGit := new(MockStorage)

	existingVersion := &models.AppVersion{
		Current:     "1.2.3",
		ProjectID:   "1234",
		AppName:     "user-service",
		LastUpdated: time.Now(),
	}
	mockRedis.On("GetVersion", mock.Anything, "1234-user-service").Return(existingVersion, nil)

	service := &VersionService{
		redis:  mockRedis.(*storage.RedisStorage),
		git:    mockGit.(*storage.GitStorage),
		logger: logger,
	}

	req := &models.DevVersionRequest{
		SHA:    "abc1234567890",
		Branch: "feature/new-feature",
	}

	got, err := service.GetDevVersion(context.Background(), "1234-user-service", req)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "1.2.4-dev-abc1234", got.Version)

	mockRedis.AssertExpectations(t)
}