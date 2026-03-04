package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	HClientID       = "X-Client-Id"
	HIdempotencyKey = "Idempotency-Key"
)

func RequireIdempotencyHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader(HClientID) == "" || c.GetHeader(HIdempotencyKey) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "missing X-Client-Id or Idempotency-Key",
			})
			return
		}
		c.Next()
	}
}
