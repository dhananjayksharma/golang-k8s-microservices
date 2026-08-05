package controller

import (
	"errors"
	"fmt"
	"time"

	"order-service/internal/events"

	"net/http"
	"order-service/service"
	"order-service/utility"
	"strings"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

type OrderController struct {
	orderService service.OrderService
}

func NewOrderController(orderService service.OrderService) OrderController {
	return OrderController{orderService: orderService}
}

func (c *OrderController) CreateOrder(ctx *gin.Context) {
	var order service.Order
	if err := ctx.ShouldBindJSON(&order); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := c.orderService.Create(ctx.Request.Context(), order)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}

	event := events.OrderCreated{
		EventID: uuid.NewString(), EventType: events.RoutingOrderCreated, OccurredAt: time.Now().UTC(),
		OrderID: created.ID, SKU: created.SKU, CustomerID: created.CustomerID, IdempotencyKey: created.IdempotencyKey,
		Quantity: created.Quantity, UnitPrice: created.UnitPrice, Currency: created.Currency, TotalAmount: created.TotalAmount,
	}
	if err := utility.PublishJSON(ctx.Request.Context(), events.ExchangeOrders, events.RoutingOrderCreated, event); err != nil {
		fmt.Printf("publish order.created event: %v\n", err)
	}

	ctx.JSON(http.StatusCreated, created)
}

func (c *OrderController) GetAllOrders(ctx *gin.Context) {
	orders, err := c.orderService.GetAll(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, orders)
}

func (c *OrderController) GetOrderByID(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	order, err := c.orderService.GetByID(ctx.Request.Context(), id)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, order)
}

func (c *OrderController) UpdateOrder(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var order service.Order
	if err := ctx.ShouldBindJSON(&order); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := c.orderService.Update(ctx.Request.Context(), id, order)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, updated)
}

func (c *OrderController) DeleteOrder(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	if err := c.orderService.Delete(ctx.Request.Context(), id); err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

func parseID(ctx *gin.Context) (string, bool) {
	id := strings.TrimSpace(ctx.Param("id"))
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return "", false
	}
	return id, true
}

func writeServiceError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrOrderNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
	case errors.Is(err, service.ErrInvalidOrder):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
