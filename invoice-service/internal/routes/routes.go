package routes

import (
	"net/http"

	"github.com/dhananjayksharma/golang-k8s-microservices/invoice-service/internal/handlers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Register(r *gin.Engine, gdb *gorm.DB) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"newok": true}) })
	const invoices = "/invoices"
	h := handlers.NewInvoiceHandler(gdb)
	//r1 := middleware.NewIPRateLimiter(rate.Limit(10), 20)
	v1 := r.Group("/v1")
	{
		v1.POST(invoices, h.Create)
		//v1.GET(invoices, r1.middleware, h.List)
		v1.GET(invoices, h.List)
		v1.GET(invoices+"/:id", h.GetByID)
		v1.PATCH(invoices+"/:id", h.Update)
		v1.DELETE(invoices+"/:id", h.Delete)
	}
	v2 := r.Group("/v2")
	{
		v2.GET(invoices, h.Listv2)
	}
	v3 := r.Group("/v3")
	{
		v3.GET(invoices, h.Listv3)
	}
}
