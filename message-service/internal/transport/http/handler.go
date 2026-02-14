package http

import (
	"net/http"
	"strconv"

	"message-service/internal/domain"
	"message-service/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.OrderService
}

func NewHandler(svc *service.OrderService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req domain.Order
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	out, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) ListOrders(c *gin.Context) {
	userID := c.Query("user_id")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.ParseInt(limitStr, 10, 64)

	out, err := h.svc.ListByUser(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": out})
}
