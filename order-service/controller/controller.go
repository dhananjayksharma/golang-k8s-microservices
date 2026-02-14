package controller

import (
	"fmt"
	"net/http"
	"order-service/service"
	"order-service/utility"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderController struct {
	orderService service.OrderService
}

func NewOrderController() OrderController {
	return OrderController{
		orderService: service.NewOrderService(),
	}
}

func (c *OrderController) CreateOrder(ctx *gin.Context) {
	var order service.Order
	if err := ctx.ShouldBindJSON(&order); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created := c.orderService.Create(order)
	orderDetail := fmt.Sprintf("New order created with ID: %s, Price:%v", strconv.Itoa(created.ID), order.Price)
	utility.PublishOrderEvent(orderDetail)
	ctx.JSON(http.StatusOK, created)
}

func (c *OrderController) GetAllOrders(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.orderService.GetAll())
}

func (c *OrderController) GetOrderByID(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	order, found := c.orderService.GetByID(id)
	if !found {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
		return
	}
	ctx.JSON(http.StatusOK, order)
}

func (c *OrderController) UpdateOrder(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	var order service.Order
	if err := ctx.ShouldBindJSON(&order); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, found := c.orderService.Update(id, order)
	if !found {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
		return
	}
	ctx.JSON(http.StatusOK, updated)
}

func (c *OrderController) DeleteOrder(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	success := c.orderService.Delete(id)
	if !success {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}
