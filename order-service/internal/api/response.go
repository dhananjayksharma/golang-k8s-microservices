package api

import (
	"errors"
	"net/http"

	"order-service/service"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, status int, data any) {
	c.JSON(status, Response{Kind: ResponseKindStandard, Data: data, Meta: requestMeta(c)})
}

func VerifiedSource(c *gin.Context, status int, source string, verified bool, reason string) {
	c.JSON(status, Response{
		Kind:         ResponseKindVerifySource,
		VerifySource: &VerifySource{Source: source, Verified: verified, Reason: reason},
		Meta:         requestMeta(c),
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Failure(c *gin.Context, status int, code, message string, details map[string]any) {
	c.AbortWithStatusJSON(status, Response{
		Kind:  ResponseKindError,
		Error: &Error{Code: code, Message: message, Details: details},
		Meta:  requestMeta(c),
	})
}

func ServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrOrderNotFound):
		Failure(c, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found", nil)
	case errors.Is(err, service.ErrInvalidOrder):
		Failure(c, http.StatusBadRequest, "INVALID_ORDER", err.Error(), nil)
	default:
		Failure(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requestMeta(c *gin.Context) *Meta {
	requestID := c.GetString("request_id")
	if requestID == "" {
		return nil
	}
	return &Meta{RequestID: requestID}
}
