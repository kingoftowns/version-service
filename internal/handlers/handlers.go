package handlers

import (
	"net/http"
	"strings"

	"github.com/company/version-service/internal/middleware"
	"github.com/company/version-service/internal/models"
	"github.com/company/version-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	service services.VersionServiceInterface
	logger  *logrus.Logger
}

func NewHandler(service services.VersionServiceInterface, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Health godoc
// @Summary Health check
// @Description Get health status of the service
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Failure 503 {object} models.HealthResponse
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	checks := h.service.Health(c.Request.Context())

	status := "healthy"
	for _, check := range checks {
		if strings.Contains(check, "unhealthy") {
			status = "unhealthy"
			break
		}
	}

	response := models.HealthResponse{
		Status: status,
		Checks: checks,
	}

	if status == "unhealthy" {
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetVersion godoc
// @Summary Get application version
// @Description Get the current version of an application
// @Tags version
// @Accept json
// @Produce json
// @Param app-id path string true "Application ID"
// @Success 200 {object} models.VersionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /version/{app-id} [get]
func (h *Handler) GetVersion(c *gin.Context) {
	appID := c.Param("app-id")
	if appID == "" {
		h.errorResponse(c, http.StatusBadRequest, "APP_ID_REQUIRED", "app ID is required", "")
		return
	}

	version, err := h.service.GetVersion(c.Request.Context(), appID)
	if err != nil {
		if strings.Contains(err.Error(), "invalid app ID") {
			h.errorResponse(c, http.StatusBadRequest, "INVALID_APP_ID", "Invalid app ID format", err.Error())
			return
		}
		h.logger.WithError(err).WithField("app_id", appID).Error("Failed to get version")
		h.errorResponse(c, http.StatusInternalServerError, "GET_VERSION_FAILED", "Failed to get version", err.Error())
		middleware.RecordVersionOperation("get", appID, "error")
		return
	}

	middleware.RecordVersionOperation("get", appID, "success")
	c.JSON(http.StatusOK, version)
}

// IncrementVersion godoc
// @Summary Increment application version
// @Description Increment the version of an application
// @Tags version
// @Accept json
// @Produce json
// @Param app-id path string true "Application ID"
// @Param type query string false "Increment type (major, minor, patch)" default(patch)
// @Success 200 {object} models.AppVersion
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /version/{app-id}/increment [post]
func (h *Handler) IncrementVersion(c *gin.Context) {
	appID := c.Param("app-id")
	if appID == "" {
		h.errorResponse(c, http.StatusBadRequest, "APP_ID_REQUIRED", "app ID is required", "")
		return
	}

	incrementType := models.IncrementTypePatch
	if typeParam := c.Query("type"); typeParam != "" {
		switch typeParam {
		case "major":
			incrementType = models.IncrementTypeMajor
		case "minor":
			incrementType = models.IncrementTypeMinor
		case "patch":
			incrementType = models.IncrementTypePatch
		default:
			h.errorResponse(c, http.StatusBadRequest, "INVALID_INCREMENT_TYPE", "Invalid increment type", "Valid types: major, minor, patch")
			return
		}
	}

	response, err := h.service.IncrementVersion(c.Request.Context(), appID, incrementType)
	if err != nil {
		if strings.Contains(err.Error(), "invalid app ID") {
			h.errorResponse(c, http.StatusBadRequest, "INVALID_APP_ID", "Invalid app ID format", err.Error())
			return
		}
		h.logger.WithError(err).WithField("app_id", appID).Error("Failed to increment version")
		h.errorResponse(c, http.StatusInternalServerError, "INCREMENT_FAILED", "Failed to increment version", err.Error())
		middleware.RecordVersionOperation("increment", appID, "error")
		return
	}

	middleware.RecordVersionOperation("increment", appID, "success")
	c.JSON(http.StatusOK, response)
}

// GetDevVersion godoc
// @Summary Get development version
// @Description Get a development version with branch and commit info
// @Tags version
// @Accept json
// @Produce json
// @Param app-id path string true "Application ID"
// @Param request body models.DevVersionRequest true "Development version request"
// @Success 200 {object} models.VersionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /version/{app-id}/dev [post]
func (h *Handler) GetDevVersion(c *gin.Context) {
	appID := c.Param("app-id")
	if appID == "" {
		h.errorResponse(c, http.StatusBadRequest, "APP_ID_REQUIRED", "app ID is required", "")
		return
	}

	var req models.DevVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error())
		return
	}

	response, err := h.service.GetDevVersion(c.Request.Context(), appID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid app ID") {
			h.errorResponse(c, http.StatusBadRequest, "INVALID_APP_ID", "Invalid app ID format", err.Error())
			return
		}
		h.logger.WithError(err).WithField("app_id", appID).Error("Failed to get dev version")
		h.errorResponse(c, http.StatusInternalServerError, "DEV_VERSION_FAILED", "Failed to get dev version", err.Error())
		middleware.RecordVersionOperation("dev", appID, "error")
		return
	}

	middleware.RecordVersionOperation("dev", appID, "success")
	c.JSON(http.StatusOK, response)
}

// ListVersions godoc
// @Summary List all versions
// @Description Get a list of all application versions
// @Tags version
// @Accept json
// @Produce json
// @Success 200 {object} map[string]models.AppVersion
// @Failure 500 {object} models.ErrorResponse
// @Router /versions [get]
func (h *Handler) ListVersions(c *gin.Context) {
	versions, err := h.service.ListVersions(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list versions")
		h.errorResponse(c, http.StatusInternalServerError, "LIST_FAILED", "Failed to list versions", err.Error())
		return
	}

	c.JSON(http.StatusOK, versions)
}

// ListVersionsByProject godoc
// @Summary List versions by project
// @Description Get a list of versions for a specific project
// @Tags version
// @Accept json
// @Produce json
// @Param project-id path string true "Project ID"
// @Success 200 {object} map[string]models.AppVersion
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /versions/{project-id} [get]
func (h *Handler) ListVersionsByProject(c *gin.Context) {
	projectID := c.Param("project-id")
	if projectID == "" {
		h.errorResponse(c, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "project ID is required", "")
		return
	}

	versions, err := h.service.ListVersionsByProject(c.Request.Context(), projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to list versions by project")
		h.errorResponse(c, http.StatusInternalServerError, "LIST_FAILED", "Failed to list versions", err.Error())
		return
	}

	c.JSON(http.StatusOK, versions)
}

// DeleteVersion godoc
// @Summary Delete application version
// @Description Delete a specific application version or entire project
// @Tags version
// @Accept json
// @Produce json
// @Param id path string true "Application ID (project-id-app-name) or Project ID (project-id)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /delete/{id} [delete]
func (h *Handler) DeleteVersion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.errorResponse(c, http.StatusBadRequest, "ID_REQUIRED", "ID is required", "")
		return
	}

	// Check if this is a project ID (no dash-separated app name) or app ID
	if strings.Contains(id, "-") && len(strings.Split(id, "-")) >= 2 {
		// This looks like an app ID (project-id-app-name)
		_, err := h.service.GetVersion(c.Request.Context(), id)
		if err != nil {
			if strings.Contains(err.Error(), "invalid app ID") {
				h.errorResponse(c, http.StatusBadRequest, "INVALID_APP_ID", "Invalid app ID format", err.Error())
				return
			}
			// If GetVersion fails, it might not exist, but we'll try to delete anyway
		}

		if err := h.service.DeleteVersion(c.Request.Context(), id); err != nil {
			h.logger.WithError(err).WithField("app_id", id).Error("Failed to delete version")
			h.errorResponse(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete version", err.Error())
			middleware.RecordVersionOperation("delete", id, "error")
			return
		}

		h.logger.WithField("app_id", id).Info("Version deleted successfully")
		middleware.RecordVersionOperation("delete", id, "success")
		c.JSON(http.StatusOK, map[string]string{
			"message": "Version deleted successfully",
			"app_id":  id,
		})
	} else {
		// This looks like a project ID only
		projectID := id

		if err := h.service.DeleteProject(c.Request.Context(), projectID); err != nil {
			h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to delete project")
			h.errorResponse(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete project", err.Error())
			return
		}

		h.logger.WithField("project_id", projectID).Info("Project deleted successfully")
		c.JSON(http.StatusOK, map[string]string{
			"message":    "Project deleted successfully",
			"project_id": projectID,
		})
	}
}

func (h *Handler) errorResponse(c *gin.Context, statusCode int, code, message, details string) {
	response := models.ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}
	c.JSON(statusCode, response)
}