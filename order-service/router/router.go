package router

import (
	"context"
	"database/sql"
	"net/http"
	"order-service/controller"
	"order-service/repository"
	"order-service/service"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})

	r.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	orderRepository := repository.NewPostgresOrderRepository(db)
	orderService := service.NewOrderService(orderRepository)
	orderController := controller.NewOrderController(orderService)

	orders := r.Group("/orders")
	{
		orders.POST("/", orderController.CreateOrder)
		orders.GET("/", orderController.GetAllOrders)
		orders.GET("/:id", orderController.GetOrderByID)
		orders.PUT("/:id", orderController.UpdateOrder)
		orders.DELETE("/:id", orderController.DeleteOrder)
	}

	return r
}
