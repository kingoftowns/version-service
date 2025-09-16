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
	service *services.VersionService
	logger  *logrus.Logger
}

func NewHandler(service *services.VersionService, logger *logrus.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

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

func (h *Handler) ListVersions(c *gin.Context) {
	versions, err := h.service.ListVersions(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list versions")
		h.errorResponse(c, http.StatusInternalServerError, "LIST_FAILED", "Failed to list versions", err.Error())
		return
	}

	c.JSON(http.StatusOK, versions)
}

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

func (h *Handler) errorResponse(c *gin.Context, statusCode int, code, message, details string) {
	response := models.ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}
	c.JSON(statusCode, response)
}