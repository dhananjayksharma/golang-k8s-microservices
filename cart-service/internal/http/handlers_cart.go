package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handlers struct {
	db *gorm.DB
}

func NewHandlers(db *gorm.DB) *Handlers { return &Handlers{db: db} }

// ---- Requests ----

type CreateCartReq struct {
	OwnerType string `json:"owner_type" binding:"required,oneof=USER GUEST"`
	UserID    string `json:"user_id"`
	GuestID   string `json:"guest_id"`
	Channel   string `json:"channel"`
	Currency  string `json:"currency"`
}

type AddItemReq struct {
	SKU           string  `json:"sku" binding:"required"`
	VariantID     string  `json:"variant_id"`
	Qty           int     `json:"qty" binding:"required,min=1,max=999"`
	ProductName   string  `json:"product_name"`
	ImageURL      string  `json:"image_url"`
	UnitPricePaise *int64 `json:"unit_price_paise"`
	MRPPaise      *int64  `json:"mrp_paise"`
	TaxRateBps    *int    `json:"tax_rate_bps"`
	ProductMeta   any     `json:"product_meta"`
}

type UpdateQtyReq struct {
	Qty int `json:"qty" binding:"required,min=1,max=999"`
}

type ApplyPromotionReq struct {
	PromoCode string `json:"promo_code" binding:"required"`
	PromoType string `json:"promo_type"` // COUPON/GIFT_CARD/WALLET
}

// ---- Handlers (minimal stubs) ----

func (h *Handlers) CreateOrGetActiveCart(c *gin.Context) {
	var req CreateCartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: implement idempotency + find/create active cart + totals row
	c.JSON(http.StatusOK, gin.H{"todo": "CreateOrGetActiveCart"})
}

func (h *Handlers) GetCart(c *gin.Context) {
	// TODO: load cart + items + totals + promos
	c.JSON(http.StatusOK, gin.H{"todo": "GetCart"})
}

func (h *Handlers) AddItem(c *gin.Context) {
	var req AddItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: implement idempotent add + totals recompute + optional CartUpdated outbox
	c.JSON(http.StatusOK, gin.H{"todo": "AddItem"})
}

func (h *Handlers) UpdateQty(c *gin.Context) {
	var req UpdateQtyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: implement idempotent qty update + totals recompute
	c.JSON(http.StatusOK, gin.H{"todo": "UpdateQty"})
}

func (h *Handlers) RemoveItem(c *gin.Context) {
	// TODO: implement idempotent remove + totals recompute
	c.JSON(http.StatusOK, gin.H{"todo": "RemoveItem"})
}

func (h *Handlers) ApplyPromotion(c *gin.Context) {
	var req ApplyPromotionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: apply promo + totals recompute
	c.JSON(http.StatusOK, gin.H{"todo": "ApplyPromotion"})
}

func (h *Handlers) Checkout(c *gin.Context) {
	// TODO: mark cart CHECKED_OUT + create CartCheckedOut outbox event + idempotency
	c.JSON(http.StatusOK, gin.H{"todo": "Checkout"})
}

// ---- helpers ----

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func parseBin16FromParam(c *gin.Context, param string) ([]byte, bool) {
	idStr := c.Param(param)
	u, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + param})
		return nil, false
	}
	return domain.UUIDToBin16(u), true
}

// Example for idempotency expire
func idemExpire(d time.Duration) time.Time { return time.Now().Add(d) }

// Example: use tx.Transaction(...) in your real methods
func withTx(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
