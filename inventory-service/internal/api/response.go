package api

import (
	"errors"
	"net/http"

	"golang-k8s-microservices/inventory-service/internal/inventory"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Success(c *gin.Context, status int, data any) {
	c.JSON(status, Response{Kind: ResponseKindStandard, Data: data, Meta: requestMeta(c)})
}

func SuccessWithMeta(c *gin.Context, status int, data any, meta Meta) {
	if meta.RequestID == "" {
		meta.RequestID = requestID(c)
	}
	c.JSON(status, Response{Kind: ResponseKindStandard, Data: data, Meta: &meta})
}

func VerifiedSource(c *gin.Context, status int, source string, verified bool, reason string) {
	c.JSON(status, Response{
		Kind:         ResponseKindVerifySource,
		VerifySource: &VerifySource{Source: source, Verified: verified, Reason: reason},
		Meta:         requestMeta(c),
	})
}

func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

func Failure(c *gin.Context, status int, code, message string, details map[string]any) {
	c.AbortWithStatusJSON(status, Response{
		Kind:  ResponseKindError,
		Error: &Error{Code: code, Message: message, Details: details},
		Meta:  requestMeta(c),
	})
}

func DomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		Failure(c, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	case errors.Is(err, inventory.ErrInsufficientStock):
		Failure(c, http.StatusConflict, "INSUFFICIENT_STOCK", err.Error(), nil)
	default:
		Failure(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requestID(c *gin.Context) string {
	value, ok := c.Get("request_id")
	if !ok {
		return ""
	}
	id, _ := value.(string)
	return id
}

func requestMeta(c *gin.Context) *Meta {
	id := requestID(c)
	if id == "" {
		return nil
	}
	return &Meta{RequestID: id}
}
