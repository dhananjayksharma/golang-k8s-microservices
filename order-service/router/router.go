package router

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"order-service/controller"
	"order-service/internal/api"
	"order-service/repository"
	"order-service/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.New()

	// Middleware order matters:
	// 1. logger
	// 2. recovery
	// 3. request ID
	// 4. CORS
	r.Use(
		gin.Logger(),
		gin.Recovery(),
		requestID(),
		configureCORS(),
	)

	r.GET("/health/live", func(c *gin.Context) {
		api.Success(c, http.StatusOK, gin.H{
			"status": "alive",
		})
	})

	r.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			api.Failure(
				c,
				http.StatusServiceUnavailable,
				"DATABASE_NOT_READY",
				"database is not ready",
				nil,
			)
			return
		}

		api.Success(c, http.StatusOK, gin.H{
			"status": "ready",
		})
	})

	orderRepository := repository.NewPostgresOrderRepository(db)
	orderService := service.NewOrderService(orderRepository)
	orderController := controller.NewOrderController(orderService)

	orders := r.Group("/orders")
	{
		orders.POST("", orderController.CreateOrder)
		orders.GET("", orderController.GetAllOrders)
		orders.GET("/:id", orderController.GetOrderByID)
		orders.PUT("/:id", orderController.UpdateOrder)
		orders.DELETE("/:id", orderController.DeleteOrder)
	}

	return r
}

func configureCORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:8088",
			"http://127.0.0.1:8088",
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"Idempotency-Key",
			"X-Request-ID",
		},
		ExposeHeaders: []string{
			"X-Request-ID",
		},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
