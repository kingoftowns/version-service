package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	versionOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "version_operations_total",
		Help: "Total number of version operations",
	}, []string{"operation", "app_id", "status"})
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		duration := time.Since(start).Seconds()

		httpDuration.WithLabelValues(method, path, status).Observe(duration)
		httpRequests.WithLabelValues(method, path, status).Inc()
	}
}

func RecordVersionOperation(operation, appID, status string) {
	versionOperations.WithLabelValues(operation, appID, status).Inc()
}