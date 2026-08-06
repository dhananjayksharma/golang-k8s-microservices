package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"order-service/internal/api"
	"order-service/internal/events"
	"order-service/service"
	"order-service/utility"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		api.Failure(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", map[string]any{"cause": err.Error()})
		return
	}

	created, err := c.orderService.Create(ctx.Request.Context(), order)
	if err != nil {
		api.ServiceError(ctx, err)
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

	api.Success(ctx, http.StatusCreated, created)
}

func (c *OrderController) GetAllOrders(ctx *gin.Context) {
	orders, err := c.orderService.GetAll(ctx.Request.Context())
	if err != nil {
		api.ServiceError(ctx, err)
		return
	}
	api.Success(ctx, http.StatusOK, orders)
}

func (c *OrderController) GetOrderByID(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	order, err := c.orderService.GetByID(ctx.Request.Context(), id)
	if err != nil {
		api.ServiceError(ctx, err)
		return
	}
	api.Success(ctx, http.StatusOK, order)
}

func (c *OrderController) UpdateOrder(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var order service.Order
	if err := ctx.ShouldBindJSON(&order); err != nil {
		api.Failure(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", map[string]any{"cause": err.Error()})
		return
	}

	updated, err := c.orderService.Update(ctx.Request.Context(), id, order)
	if err != nil {
		api.ServiceError(ctx, err)
		return
	}
	api.Success(ctx, http.StatusOK, updated)
}

func (c *OrderController) DeleteOrder(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}

	if err := c.orderService.Delete(ctx.Request.Context(), id); err != nil {
		api.ServiceError(ctx, err)
		return
	}
	api.NoContent(ctx)
}

func parseID(ctx *gin.Context) (string, bool) {
	id := strings.TrimSpace(ctx.Param("id"))
	if id == "" {
		api.Failure(ctx, http.StatusBadRequest, "MISSING_ID", "id is required", nil)
		return "", false
	}
	if _, err := uuid.Parse(id); err != nil {
		api.Failure(ctx, http.StatusBadRequest, "INVALID_ID", "id must be a UUID", nil)
		return "", false
	}
	return id, true
}
