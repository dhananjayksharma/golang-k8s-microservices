package routes

import (
	"net/http"

	"github.com/dhananjayksharma/golang-k8s-microservices/invoice-service/internal/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Register(r *gin.Engine, gdb *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	h := handlers.NewInvoiceHandler(gdb)

	v1 := r.Group("/v1")
	{
		v1.POST("/invoices", h.Create)
		v1.GET("/invoices", h.List)
		v1.GET("/invoices/:id", h.GetByID)
		v1.PATCH("/invoices/:id", h.Update)
		v1.DELETE("/invoices/:id", h.Delete)
		//v1.GET("/downloadinvoice/:id", h.GeneratePDF)
		//v1.GET("/invoices/:id/download", h.GeneratePDF)
		v1.GET("/invoices/:id/:actions", h.InvoiceActions)

	}
	v2 := r.Group("/v2")
	{
		v2.GET("/invoices", h.Listv2)
	}
}
