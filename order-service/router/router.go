package router

import (
	"order-service/controller"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	orderController := controller.NewOrderController()

	v1 := r.Group("/orders")
	{
		v1.POST("/", orderController.CreateOrder)
		v1.GET("/", orderController.GetAllOrders)
		v1.GET("/:id", orderController.GetOrderByID)
		v1.PUT("/:id", orderController.UpdateOrder)
		v1.DELETE("/:id", orderController.DeleteOrder)
	}

	return r
}
