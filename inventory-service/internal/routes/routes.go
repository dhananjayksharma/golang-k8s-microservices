package routes

import (
	"net/http"

	"golang-k8s-microservices/inventory-service/internal/api"
	"golang-k8s-microservices/inventory-service/internal/handlers"
	"golang-k8s-microservices/inventory-service/internal/inventory"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Register(r *gin.Engine, gdb *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) { api.Success(c, http.StatusOK, gin.H{"status": "ready"}) })

	h := handlers.NewInvoiceHandler(gdb)
	stockService := inventory.NewService(gdb)

	v1 := r.Group("/v1")
	{
		invoices := v1.Group("/invoices")
		{
			invoices.POST("", h.Create)
			invoices.GET("", h.List)
			invoices.GET("/:id", h.GetByID)
			invoices.PATCH("/:id", h.Update)
			invoices.DELETE("/:id", h.Delete)
			invoices.GET("/:id/preview", h.Preview)
			invoices.GET("/:id/download", h.Download)
			invoices.GET("/:id/document", h.Generate)
			invoices.POST("/:id/send-email", h.SendEmail)
			invoices.POST("/:id/upload", h.Upload)
		}

		v1.GET("/inventory", func(c *gin.Context) {
			items, err := stockService.List(c.Request.Context())
			if err != nil {
				api.DomainError(c, err)
				return
			}
			api.Success(c, http.StatusOK, items)
		})
	}
	v2 := r.Group("/v2")
	v2.GET("/invoices", h.Listv2)
}
