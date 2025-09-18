package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/company/version-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockVersionService struct {
	mock.Mock
}

func (m *MockVersionService) Health(ctx context.Context) map[string]string {
	args := m.Called(ctx)
	return args.Get(0).(map[string]string)
}

func (m *MockVersionService) GetVersion(ctx context.Context, appID string) (*models.AppVersion, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AppVersion), args.Error(1)
}

func (m *MockVersionService) IncrementVersion(ctx context.Context, appID string, incrementType models.IncrementType) (*models.VersionResponse, error) {
	args := m.Called(ctx, appID, incrementType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VersionResponse), args.Error(1)
}

func (m *MockVersionService) GetDevVersion(ctx context.Context, appID string, req *models.DevVersionRequest) (*models.VersionResponse, error) {
	args := m.Called(ctx, appID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VersionResponse), args.Error(1)
}

func (m *MockVersionService) ListVersions(ctx context.Context) (map[string]*models.AppVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.AppVersion), args.Error(1)
}

func (m *MockVersionService) ListVersionsByProject(ctx context.Context, projectID string) (map[string]*models.AppVersion, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.AppVersion), args.Error(1)
}

func (m *MockVersionService) DeleteVersion(ctx context.Context, appID string) error {
	args := m.Called(ctx, appID)
	return args.Error(0)
}

func (m *MockVersionService) DeleteProject(ctx context.Context, projectID string) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func TestHealth_Healthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockVersionService)
	handler := NewHandler(mockService, logrus.New())

	mockService.On("Health", mock.Anything).Return(map[string]string{
		"redis": "healthy",
		"git":   "healthy",
	})

	router := gin.New()
	router.GET("/health", handler.Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "healthy", response.Checks["redis"])
	assert.Equal(t, "healthy", response.Checks["git"])

	mockService.AssertExpectations(t)
}

func TestHealth_Unhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockVersionService)
	handler := NewHandler(mockService, logrus.New())

	mockService.On("Health", mock.Anything).Return(map[string]string{
		"redis": "healthy",
		"git":   "unhealthy: connection failed",
	})

	router := gin.New()
	router.GET("/health", handler.Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", response.Status)

	mockService.AssertExpectations(t)
}

func TestGetVersion_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockVersionService)
	handler := NewHandler(mockService, logrus.New())

	expectedVersion := &models.AppVersion{
		Current:   "1.0.0",
		ProjectID: "1234",
		AppName:   "user-service",
	}

	mockService.On("GetVersion", mock.Anything, "1234-user-service").Return(expectedVersion, nil)

	router := gin.New()
	router.GET("/version/:app-id", handler.GetVersion)

	req, _ := http.NewRequest("GET", "/version/1234-user-service", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AppVersion
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedVersion.Current, response.Current)
	assert.Equal(t, expectedVersion.ProjectID, response.ProjectID)

	mockService.AssertExpectations(t)
}

func TestGetVersion_InvalidAppID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockVersionService)
	handler := NewHandler(mockService, logrus.New())

	router := gin.New()
	router.GET("/version/:app-id", handler.GetVersion)

	req, _ := http.NewRequest("GET", "/version/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteVersion_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockVersionService)
	handler := NewHandler(mockService, logrus.New())

	// Mock both GetVersion and DeleteVersion calls
	mockService.On("GetVersion", mock.Anything, "1234-user-service").Return(&models.AppVersion{
		Current:   "1.0.0",
		ProjectID: "1234",
		AppName:   "user-service",
	}, nil)
	mockService.On("DeleteVersion", mock.Anything, "1234-user-service").Return(nil)

	router := gin.New()
	router.DELETE("/delete/:id", handler.DeleteVersion)

	req, _ := http.NewRequest("DELETE", "/delete/1234-user-service", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Version deleted successfully", response["message"])
	assert.Equal(t, "1234-user-service", response["app_id"])

	mockService.AssertExpectations(t)
}

func TestDeleteProject_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockVersionService)
	handler := NewHandler(mockService, logrus.New())

	mockService.On("DeleteProject", mock.Anything, "1234").Return(nil)

	router := gin.New()
	router.DELETE("/delete/:id", handler.DeleteVersion)

	req, _ := http.NewRequest("DELETE", "/delete/1234", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Project deleted successfully", response["message"])
	assert.Equal(t, "1234", response["project_id"])

	mockService.AssertExpectations(t)
}