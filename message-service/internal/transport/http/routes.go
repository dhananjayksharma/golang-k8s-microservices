package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, h *Handler) {
	r.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })

	v1 := r.Group("/v1")
	{
		v1.POST("/orders", h.CreateOrder)
		v1.GET("/orders", h.ListOrders)
	}
}
