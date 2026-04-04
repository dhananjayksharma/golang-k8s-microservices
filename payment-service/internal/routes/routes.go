package routes

import (
	"net/http"

	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Register(r *gin.Engine, gdb *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"newok": true}) })

	h := handlers.NewOrderHandler(gdb)
	//r1 := middleware.NewIPRateLimiter(rate.Limit(10), 20)
	v1 := r.Group("/v1")
	{
		v1.POST("/payments", h.Create)
		//v1.GET("/payments", r1.middleware, h.List)
		v1.GET("/payments", h.List)
		v1.GET("/payments/:id", h.GetByID)
		v1.PATCH("/payments/:id", h.Update)
		v1.DELETE("/payments/:id", h.Delete)
	}
	v2 := r.Group("/v2")
	{
		v2.GET("/payments", h.Listv2)
	}
	v3 := r.Group("/v3")
	{
		v3.GET("/payments", h.Listv3)
	}
}
