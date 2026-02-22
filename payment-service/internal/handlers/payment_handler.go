package handlers

import (
	"net/http"

	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/service"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) ListPayments(c *gin.Context) {
	payments, err := h.service.ListPayments(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, payments)
}
