// internal/middleware/recovery.go
package middleware

import (
	"net/http"

	"github.com/dhananjayksharma/golang-k8s-microservices/invoice-service/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		reqID, _ := c.Get(RequestIDKey)

		logger.Log.Error("panic recovered",
			zap.Any("panic", recovered),
			zap.String("request_id", reqID.(string)),
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":      "internal server error",
			"request_id": reqID,
		})
	})
}
