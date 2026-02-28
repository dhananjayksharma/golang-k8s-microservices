package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	SKU            string `json:"sku" binding:"required"`
	VariantID      string `json:"variant_id"`
	Qty            int    `json:"qty" binding:"required,min=1,max=999"`
	ProductName    string `json:"product_name"`
	ImageURL       string `json:"image_url"`
	UnitPricePaise *int64 `json:"unit_price_paise"`
	MRPPaise       *int64 `json:"mrp_paise"`
	TaxRateBps     *int   `json:"tax_rate_bps"`
	ProductMeta    any    `json:"product_meta"`
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

	if req.OwnerType == "USER" && req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required for USER owner_type"})
		return
	}
	if req.OwnerType == "GUEST" && req.GuestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guest_id is required for GUEST owner_type"})
		return
	}

	clientID := c.GetHeader(HClientID)
	idemKey := c.GetHeader(HIdempotencyKey)
	reqHash, err := domain.HashRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}
	var userBin []byte
	if req.OwnerType == "USER" {
		userUUID, parseErr := uuid.Parse(req.UserID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}
		userBin = domain.UUIDToBin16(userUUID)
	}

	type createCartResp struct {
		CartID   string `json:"cart_id"`
		Status   string `json:"status"`
		Currency string `json:"currency"`
		Channel  string `json:"channel"`
	}

	statusCode := http.StatusOK
	var resp createCartResp

	err = withTx(h.db, func(tx *gorm.DB) error {
		var idemRow domain.CartIdempotency
		idemErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("client_id=? AND idempotency_key=?", clientID, idemKey).
			First(&idemRow).Error
		if idemErr == nil {
			if idemRow.RequestHash != reqHash {
				statusCode = http.StatusConflict
				resp = createCartResp{}
				return nil
			}
			if idemRow.State == "COMPLETED" && idemRow.ResponseBody != "" && idemRow.HTTPStatus != nil {
				statusCode = int(*idemRow.HTTPStatus)
				_ = json.Unmarshal([]byte(idemRow.ResponseBody), &resp)
				return nil
			}
		} else if !errors.Is(idemErr, gorm.ErrRecordNotFound) {
			return idemErr
		}

		if errors.Is(idemErr, gorm.ErrRecordNotFound) {
			newIdem := domain.CartIdempotency{
				ClientID:       clientID,
				IdempotencyKey: idemKey,
				Endpoint:       c.FullPath(),
				RequestHash:    reqHash,
				State:          "PROCESSING",
				ExpiresAt:      idemExpire(24 * time.Hour),
			}
			if createErr := tx.Create(&newIdem).Error; createErr != nil {
				refetchErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("client_id=? AND idempotency_key=?", clientID, idemKey).
					First(&idemRow).Error
				if refetchErr != nil {
					return createErr
				}
				if idemRow.RequestHash != reqHash {
					statusCode = http.StatusConflict
					return nil
				}
				if idemRow.State == "COMPLETED" && idemRow.ResponseBody != "" && idemRow.HTTPStatus != nil {
					statusCode = int(*idemRow.HTTPStatus)
					_ = json.Unmarshal([]byte(idemRow.ResponseBody), &resp)
					return nil
				}
			}
		}

		query := tx.Where("owner_type=? AND channel=? AND status='ACTIVE'", req.OwnerType, req.Channel)
		if req.OwnerType == "USER" {
			query = query.Where("user_id=?", userBin)
		} else {
			query = query.Where("guest_id=?", req.GuestID)
		}

		var cart domain.Cart
		findErr := query.First(&cart).Error
		if findErr != nil {
			if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return findErr
			}
			cartUUID := uuid.New()
			cart = domain.Cart{
				CartID:    domain.UUIDToBin16(cartUUID),
				OwnerType: req.OwnerType,
				UserID:    userBin,
				GuestID:   req.GuestID,
				Channel:   req.Channel,
				Status:    "ACTIVE",
				Currency:  req.Currency,
				Version:   1,
			}
			if createErr := tx.Create(&cart).Error; createErr != nil {
				refetchErr := query.First(&cart).Error
				if refetchErr != nil {
					return createErr
				}
			}
		}

		if err := tx.Where("cart_id=?", cart.CartID).
			Attrs(domain.CartTotals{
				SubtotalPaise:   0,
				TaxPaise:        0,
				ShippingPaise:   0,
				DiscountPaise:   0,
				GrandTotalPaise: 0,
				PricingVersion:  1,
			}).
			FirstOrCreate(&domain.CartTotals{}).Error; err != nil {
			return err
		}

		cartUUID, convErr := domain.Bin16ToUUID(cart.CartID)
		if convErr != nil {
			return convErr
		}
		resp = createCartResp{
			CartID:   cartUUID.String(),
			Status:   cart.Status,
			Currency: cart.Currency,
			Channel:  cart.Channel,
		}

		if statusCode != http.StatusOK {
			return tx.Model(&domain.CartIdempotency{}).
				Where("client_id=? AND idempotency_key=?", clientID, idemKey).
				Updates(map[string]any{
					"state":         "COMPLETED",
					"http_status":   int16(statusCode),
					"response_body": mustJSON(gin.H{"error": "idempotency key reused with different request payload"}),
				}).Error
		}

		return tx.Model(&domain.CartIdempotency{}).
			Where("client_id=? AND idempotency_key=?", clientID, idemKey).
			Updates(map[string]any{
				"resource_id":   cart.CartID,
				"state":         "COMPLETED",
				"http_status":   int16(http.StatusOK),
				"response_body": mustJSON(resp),
			}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode == http.StatusConflict {
		c.JSON(http.StatusConflict, gin.H{"error": "idempotency key reused with different request payload"})
		return
	}
	c.JSON(statusCode, resp)
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
