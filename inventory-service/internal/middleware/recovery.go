// internal/middleware/recovery.go
package middleware

import (
	"net/http"

	"golang-k8s-microservices/inventory-service/internal/api"
	"golang-k8s-microservices/inventory-service/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestID, _ := c.Get(RequestIDKey)
		logger.Log.Error("panic recovered", zap.Any("panic", recovered), zap.Any("request_id", requestID))
		api.Failure(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	})
}
