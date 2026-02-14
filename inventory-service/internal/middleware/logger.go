// internal/middleware/logger.go
package middleware

import (
	"time"

	"golang-k8s-microservices/inventory-service/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		reqID, _ := c.Get(RequestIDKey)

		fields := []zap.Field{
			zap.String("request_id", reqID.(string)),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		}

		if len(c.Errors) > 0 {
			logger.Log.Error("http request failed",
				append(fields, zap.String("errors", c.Errors.String()))...,
			)
		} else {
			logger.Log.Info("http request",
				fields...,
			)
		}
	}
}
