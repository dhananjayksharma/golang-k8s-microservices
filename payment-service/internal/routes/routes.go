package routes

import (
	"net/http"

	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Register(r *gin.Engine, gdb *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	h := handlers.NewOrderHandler(gdb)

	v1 := r.Group("/v1")
	{
		v1.POST("/orders", h.Create)
		v1.GET("/orders", h.List)
		v1.GET("/orders/:id", h.GetByID)
		v1.PATCH("/orders/:id", h.Update)
		v1.DELETE("/orders/:id", h.Delete)
	}
	v2 := r.Group("/v2")
	{
		v2.GET("/orders", h.Listv2)
	}
	v3 := r.Group("/v3")
	{
		v3.GET("/orders", h.Listv3)
	}
}
