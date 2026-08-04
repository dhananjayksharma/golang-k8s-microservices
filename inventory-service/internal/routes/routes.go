package routes

import (
	"net/http"

	"golang-k8s-microservices/inventory-service/internal/handlers"
	"golang-k8s-microservices/inventory-service/internal/inventory"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Register(r *gin.Engine, gdb *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	h := handlers.NewInvoiceHandler(gdb)
	stockService := inventory.NewService(gdb)

	v1 := r.Group("/v1")
	{
		v1.POST("/invoices", h.Create)
		v1.GET("/invoices", h.List)
		v1.GET("/invoices/:id", h.GetByID)
		v1.PATCH("/invoices/:id", h.Update)
		v1.DELETE("/invoices/:id", h.Delete)
		v1.GET("/invoices/:id/:actions", h.InvoiceActions)
		v1.GET("/invoices/inventory/:id", h.GetInventoryByID)

	}
	r.GET("/v1/inventory", func(c *gin.Context) {
		items, err := stockService.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, items)
	})

	v2 := r.Group("/v2")
	{
		v2.GET("/invoices", h.Listv2)
	}
}
