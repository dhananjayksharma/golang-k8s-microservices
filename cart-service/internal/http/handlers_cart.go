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
				ResponseBody:   "{}",
				State:          "IN_PROGRESS",
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

		cartTotals := domain.CartTotals{
			CartID:          cart.CartID,
			SubtotalPaise:   0,
			TaxPaise:        0,
			ShippingPaise:   0,
			DiscountPaise:   0,
			GrandTotalPaise: 0,
			PricingVersion:  1,
		}
		if err := tx.FirstOrCreate(&cartTotals, domain.CartTotals{CartID: cart.CartID}).Error; err != nil {
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
	cartID, ok := parseBin16FromParam(c, "cartId")
	if !ok {
		return
	}

	var cart domain.Cart
	if err := h.db.Where("cart_id = ?", cartID).First(&cart).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cart not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var items []domain.CartItem
	if err := h.db.Where("cart_id = ?", cartID).Order("added_at asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var promos []domain.CartPromotion
	if err := h.db.Where("cart_id = ?", cartID).Order("applied_at asc").Find(&promos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totals := domain.CartTotals{CartID: cartID}
	if err := h.db.Where("cart_id = ?", cartID).First(&totals).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cartUUID, err := domain.Bin16ToUUID(cart.CartID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID := ""
	if len(cart.UserID) == 16 {
		if u, convErr := domain.Bin16ToUUID(cart.UserID); convErr == nil {
			userID = u.String()
		}
	}

	itemResp := make([]gin.H, 0, len(items))
	for _, it := range items {
		itemID := ""
		if len(it.CartItemID) == 16 {
			if u, convErr := domain.Bin16ToUUID(it.CartItemID); convErr == nil {
				itemID = u.String()
			}
		}
		itemResp = append(itemResp, gin.H{
			"cart_item_id":     itemID,
			"sku":              it.SKU,
			"variant_id":       it.VariantID,
			"qty":              it.Qty,
			"product_name":     it.ProductName,
			"image_url":        it.ImageURL,
			"currency":         it.Currency,
			"unit_price_paise": it.UnitPricePaise,
			"mrp_paise":        it.MRPPaise,
			"tax_rate_bps":     it.TaxRateBps,
			"product_meta":     it.ProductMeta,
			"availability":     it.Availability,
			"added_at":         it.AddedAt,
			"updated_at":       it.UpdatedAt,
		})
	}

	promoResp := make([]gin.H, 0, len(promos))
	for _, p := range promos {
		promoID := ""
		if len(p.CartPromoID) == 16 {
			if u, convErr := domain.Bin16ToUUID(p.CartPromoID); convErr == nil {
				promoID = u.String()
			}
		}
		promoResp = append(promoResp, gin.H{
			"cart_promo_id":  promoID,
			"promo_code":     p.PromoCode,
			"promo_type":     p.PromoType,
			"discount_paise": p.DiscountPaise,
			"promo_meta":     p.PromoMeta,
			"status":         p.Status,
			"applied_at":     p.AppliedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"cart": gin.H{
			"cart_id":    cartUUID.String(),
			"owner_type": cart.OwnerType,
			"user_id":    userID,
			"guest_id":   cart.GuestID,
			"channel":    cart.Channel,
			"status":     cart.Status,
			"currency":   cart.Currency,
			"locale":     cart.Locale,
			"version":    cart.Version,
			"created_at": cart.CreatedAt,
			"updated_at": cart.UpdatedAt,
			"expires_at": cart.ExpiresAt,
		},
		"items": itemResp,
		"totals": gin.H{
			"subtotal_paise":    totals.SubtotalPaise,
			"tax_paise":         totals.TaxPaise,
			"shipping_paise":    totals.ShippingPaise,
			"discount_paise":    totals.DiscountPaise,
			"grand_total_paise": totals.GrandTotalPaise,
			"pricing_version":   totals.PricingVersion,
			"computed_at":       totals.ComputedAt,
		},
		"promotions": promoResp,
	})
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
	cartID := c.Param("cartId")
	cartIDBin, ok := parseBin16FromParam(c, "cartId")
	if !ok {
		return
	}
	clientID := c.GetHeader("X-Client-Id")
	idemKey := c.GetHeader("Idempotency-Key")

	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 1) load cart + items + totals + promos (FOR UPDATE is ideal)
		// 2) mark cart status = CHECKED_OUT
		// 3) insert outbox event

		ev := domain.EventEnvelope{
			EventID:        uuid.NewString(),
			EventType:      "CartCheckedOut.v1",
			Producer:       "cart-service",
			OccurredAt:     time.Now().UTC(),
			CorrelationID:  cartID,
			IdempotencyKey: idemKey,
			Data: map[string]any{
				"cart_id":   cartID,
				"client_id": clientID,
				// include items + totals snapshot here
			},
		}
		payloadBytes, _ := json.Marshal(ev)

		ob := domain.CartOutbox{
			OutboxID:      domain.UUIDToBin16(uuid.New()),
			AggregateType: "CART",
			AggregateID:   cartIDBin, // BINARY(16) cart_id
			EventType:     ev.EventType,
			Payload:       string(payloadBytes),
			Status:        "NEW",
		}
		if err := tx.Create(&ob).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "CHECKED_OUT"})
}

func (h *Handlers) NOCheckout(c *gin.Context) {
	cartID, ok := parseBin16FromParam(c, "cartId")
	if !ok {
		return
	}

	clientID := c.GetHeader(HClientID)
	idemKey := c.GetHeader(HIdempotencyKey)
	reqHash, err := domain.HashRequest(struct {
		CartID string `json:"cart_id"`
	}{
		CartID: c.Param("cartId"),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	statusCode := http.StatusOK
	respBody := any(gin.H{"cart_id": c.Param("cartId"), "status": "CHECKED_OUT"})

	err = withTx(h.db, func(tx *gorm.DB) error {
		var idemRow domain.CartIdempotency
		idemErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("client_id=? AND idempotency_key=?", clientID, idemKey).
			First(&idemRow).Error
		if idemErr == nil {
			if idemRow.RequestHash != reqHash {
				statusCode = http.StatusConflict
				respBody = gin.H{"error": "idempotency key reused with different request payload"}
				return nil
			}
			if idemRow.State == "COMPLETED" && idemRow.ResponseBody != "" && idemRow.HTTPStatus != nil {
				statusCode = int(*idemRow.HTTPStatus)
				var replay any
				if unmarshalErr := json.Unmarshal([]byte(idemRow.ResponseBody), &replay); unmarshalErr == nil {
					respBody = replay
				}
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
				ResponseBody:   "{}",
				State:          "IN_PROGRESS",
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
					respBody = gin.H{"error": "idempotency key reused with different request payload"}
					return nil
				}
				if idemRow.State == "COMPLETED" && idemRow.ResponseBody != "" && idemRow.HTTPStatus != nil {
					statusCode = int(*idemRow.HTTPStatus)
					var replay any
					if unmarshalErr := json.Unmarshal([]byte(idemRow.ResponseBody), &replay); unmarshalErr == nil {
						respBody = replay
					}
					return nil
				}
			}
		}

		var cart domain.Cart
		cartErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("cart_id = ?", cartID).
			First(&cart).Error
		if errors.Is(cartErr, gorm.ErrRecordNotFound) {
			statusCode = http.StatusNotFound
			respBody = gin.H{"error": "cart not found"}
		} else if cartErr != nil {
			return cartErr
		} else {
			switch cart.Status {
			case "ACTIVE":
				updateErr := tx.Model(&domain.Cart{}).
					Where("cart_id=? AND version=?", cartID, cart.Version).
					Updates(map[string]any{
						"status":  "CHECKED_OUT",
						"version": cart.Version + 1,
					}).Error
				if updateErr != nil {
					return updateErr
				}

				outbox := domain.CartOutbox{
					OutboxID:      domain.UUIDToBin16(uuid.New()),
					AggregateType: "Cart",
					AggregateID:   cartID,
					EventType:     "CartCheckedOut",
					Payload: mustJSON(gin.H{
						"cart_id":        c.Param("cartId"),
						"status":         "CHECKED_OUT",
						"checked_out_at": time.Now().UTC().Format(time.RFC3339Nano),
					}),
					Status: "NEW",
				}
				if createErr := tx.Create(&outbox).Error; createErr != nil {
					return createErr
				}
			case "CHECKED_OUT":
				// Idempotent success for already checked out cart: no duplicate outbox event.
			default:
				statusCode = http.StatusConflict
				respBody = gin.H{"error": "cart is not in ACTIVE state"}
			}
		}

		return tx.Model(&domain.CartIdempotency{}).
			Where("client_id=? AND idempotency_key=?", clientID, idemKey).
			Updates(map[string]any{
				"resource_id":   cartID,
				"state":         "COMPLETED",
				"http_status":   int16(statusCode),
				"response_body": mustJSON(respBody),
			}).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(statusCode, respBody)
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
