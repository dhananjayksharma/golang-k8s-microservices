package http

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	h := NewHandlers(db)

	v1 := r.Group("/v1")
	{
		v1.POST("/carts", RequireIdempotencyHeaders(), h.CreateOrGetActiveCart)
		v1.GET("/carts/:cartId", h.GetCart)

		v1.POST("/carts/:cartId/items", RequireIdempotencyHeaders(), h.AddItem)
		v1.PATCH("/carts/:cartId/items/:sku", RequireIdempotencyHeaders(), h.UpdateQty)
		v1.DELETE("/carts/:cartId/items/:sku", RequireIdempotencyHeaders(), h.RemoveItem)

		v1.POST("/carts/:cartId/promotions", RequireIdempotencyHeaders(), h.ApplyPromotion)
		v1.POST("/carts/:cartId/checkout", RequireIdempotencyHeaders(), h.Checkout)
	}

	return r
}
