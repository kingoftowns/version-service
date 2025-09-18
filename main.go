package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/company/version-service/internal/clients"
	"github.com/company/version-service/internal/config"
	"github.com/company/version-service/internal/handlers"
	"github.com/company/version-service/internal/middleware"
	"github.com/company/version-service/internal/services"
	"github.com/company/version-service/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/company/version-service/docs"
)

// @title           Version Service API
// @version         1.0
// @description     A service for managing application versions with Git-based persistence
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://github.com/company/version-service
// @contact.email  support@company.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

func main() {
	logger := setupLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	redisStorage, err := storage.NewRedisStorage(cfg.RedisURL, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Redis storage")
	}
	defer redisStorage.Close()

	gitStorage, err := storage.NewGitStorage(cfg.GitRepoURL, cfg.GitBranch, cfg.GitUsername, cfg.GitToken, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Git storage")
	}
	defer gitStorage.Close()

	gitLabClient := clients.NewGitLabClient(cfg.GitLabBaseURL, cfg.GitLabAccessToken, logger)

	versionService := services.NewVersionService(redisStorage, gitStorage, gitLabClient, logger)

	ctx := context.Background()
	if err := versionService.Initialize(ctx); err != nil {
		logger.WithError(err).Error("Failed to initialize version service")
	}

	router := setupRouter(versionService, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.WithField("port", cfg.Port).Info("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server exited")
}

func setupLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	return logger
}

func setupRouter(service *services.VersionService, logger *logrus.Logger) *gin.Engine {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.MetricsMiddleware())

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Version-Service", "1.0.0")
		c.Next()
	})

	handler := handlers.NewHandler(service, logger)

	router.GET("/health", handler.Health)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/")
	{
		v1.GET("/version/:app-id", handler.GetVersion)
		v1.POST("/version/:app-id/increment", handler.IncrementVersion)
		v1.POST("/version/:app-id/dev", handler.GetDevVersion)
		v1.GET("/versions", handler.ListVersions)
		v1.GET("/versions/:project-id", handler.ListVersionsByProject)
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Endpoint not found",
			"path":  c.Request.URL.Path,
		})
	})

	return router
}
