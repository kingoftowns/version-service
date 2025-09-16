package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		entry := logger.WithFields(logrus.Fields{
			"latency":     latency,
			"client_ip":   clientIP,
			"method":      method,
			"path":        path,
			"status_code": statusCode,
		})

		msg := "Request processed"

		if statusCode >= 500 {
			entry.Error(msg)
		} else if statusCode >= 400 {
			entry.Warn(msg)
		} else {
			entry.Info(msg)
		}
	}
}