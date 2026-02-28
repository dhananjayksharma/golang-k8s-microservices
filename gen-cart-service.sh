#!/usr/bin/env bash
set -euo pipefail

ROOT="cart-service"

echo "Creating $ROOT structure..."

# ---------- folders ----------
mkdir -p "$ROOT/cmd/server" \
         "$ROOT/cmd/worker" \
         "$ROOT/internal/config" \
         "$ROOT/internal/db" \
         "$ROOT/internal/domain" \
         "$ROOT/internal/http" \
         "$ROOT/internal/outbox" \
         "$ROOT/internal/kafka" \
         "$ROOT/internal/repo"

# ---------- go.mod ----------
cat > "$ROOT/go.mod" <<'EOF'
module cart-service

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/segmentio/kafka-go v0.4.47
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
)
EOF

# ---------- cmd/server/main.go ----------
cat > "$ROOT/cmd/server/main.go" <<'EOF'
package main

import (
	"log"
	"os"
	"strconv"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/config"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/db"
	httpx "github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/http"
)

func main() {
	cfg := config.Load()

	gdb, err := db.NewMySQL(cfg.MySQL.DSN())
	if err != nil {
		log.Fatal(err)
	}

	r := httpx.NewRouter(gdb)
	log.Printf("cart-service listening on :%d\n", cfg.HTTPPort)
	log.Fatal(r.Run(":" + strconv.Itoa(cfg.HTTPPort)))
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
EOF

# ---------- cmd/worker/main.go ----------
cat > "$ROOT/cmd/worker/main.go" <<'EOF'
package main

import (
	"context"
	"log"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/config"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/db"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/kafka"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/outbox"
)

func main() {
	cfg := config.Load()

	gdb, err := db.NewMySQL(cfg.MySQL.DSN())
	if err != nil {
		log.Fatal(err)
	}

	prod := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer prod.Close()

	pub := outbox.NewPublisher(gdb, prod)
	log.Printf("cart-worker publishing outbox to topic=%s\n", cfg.Kafka.Topic)
	log.Fatal(pub.Run(context.Background()))
}
EOF

# ---------- internal/config/config.go ----------
cat > "$ROOT/internal/config/config.go" <<'EOF'
package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort int
	MySQL    MySQL
	Kafka    Kafka
}

type MySQL struct {
	User string
	Pass string
	Host string
	Port int
	DB   string
}

type Kafka struct {
	Brokers []string
	Topic   string
	GroupID string
}

func Load() Config {
	return Config{
		HTTPPort: mustInt(getenv("HTTP_PORT", "8085")),
		MySQL: MySQL{
			User: getenv("DB_USER", "root"),
			Pass: getenv("DB_PASS", "root"),
			Host: getenv("DB_HOST", "127.0.0.1"),
			Port: mustInt(getenv("DB_PORT", "3306")),
			DB:   getenv("DB_NAME", "cartdb"),
		},
		Kafka: Kafka{
			Brokers: strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ","),
			Topic:   getenv("KAFKA_TOPIC", "cart.events"),
			GroupID: getenv("KAFKA_GROUP_ID", "cart-service"),
		},
	}
}

func (m MySQL) DSN() string {
	// parseTime=true is required for time.Time mapping
	return m.User + ":" + m.Pass + "@tcp(" + m.Host + ":" + strconv.Itoa(m.Port) + ")/" + m.DB +
		"?parseTime=true&loc=UTC&charset=utf8mb4,utf8&collation=utf8mb4_unicode_ci"
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func mustInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
EOF

# ---------- internal/db/mysql.go ----------
cat > "$ROOT/internal/db/mysql.go" <<'EOF'
package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQL(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(30)
	sqlDB.SetMaxIdleConns(10)
	return gdb, nil
}
EOF

# ---------- internal/domain/model.go ----------
cat > "$ROOT/internal/domain/model.go" <<'EOF'
package domain

import "time"

// BINARY(16) columns map well to []byte in Go.

type Cart struct {
	CartID    []byte `gorm:"column:cart_id;type:binary(16);primaryKey"`
	OwnerType string `gorm:"column:owner_type;not null"`
	UserID    []byte `gorm:"column:user_id;type:binary(16)"`
	GuestID   string `gorm:"column:guest_id"`
	Channel   string `gorm:"column:channel;not null"`
	Status    string `gorm:"column:status;not null"`
	Currency  string `gorm:"column:currency;not null"`
	Locale    string `gorm:"column:locale"`
	Version   int    `gorm:"column:version;not null"`

	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
}

func (Cart) TableName() string { return "carts" }

type CartItem struct {
	CartItemID []byte `gorm:"column:cart_item_id;type:binary(16);primaryKey"`
	CartID     []byte `gorm:"column:cart_id;type:binary(16);index;not null"`

	SKU       string `gorm:"column:sku;not null"`
	VariantID string `gorm:"column:variant_id"`
	Qty       int    `gorm:"column:qty;not null"`

	ProductName string `gorm:"column:product_name"`
	ImageURL    string `gorm:"column:image_url"`

	Currency       string `gorm:"column:currency;not null"`
	UnitPricePaise *int64 `gorm:"column:unit_price_paise"`
	MRPPaise       *int64 `gorm:"column:mrp_paise"`
	TaxRateBps     *int   `gorm:"column:tax_rate_bps"`
	ProductMeta    string `gorm:"column:product_meta;type:json"`

	Availability string    `gorm:"column:availability;not null"`
	AddedAt      time.Time `gorm:"column:added_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (CartItem) TableName() string { return "cart_items" }

type CartPromotion struct {
	CartPromoID   []byte `gorm:"column:cart_promo_id;type:binary(16);primaryKey"`
	CartID        []byte `gorm:"column:cart_id;type:binary(16);index;not null"`
	PromoCode     string `gorm:"column:promo_code;not null"`
	PromoType     string `gorm:"column:promo_type;not null"`
	DiscountPaise int64  `gorm:"column:discount_paise;not null"`
	PromoMeta     string `gorm:"column:promo_meta;type:json"`
	Status        string `gorm:"column:status;not null"`
	AppliedAt     time.Time `gorm:"column:applied_at;autoCreateTime"`
}

func (CartPromotion) TableName() string { return "cart_promotions" }

type CartTotals struct {
	CartID          []byte `gorm:"column:cart_id;type:binary(16);primaryKey"`
	SubtotalPaise   int64  `gorm:"column:subtotal_paise;not null"`
	TaxPaise        int64  `gorm:"column:tax_paise;not null"`
	ShippingPaise   int64  `gorm:"column:shipping_paise;not null"`
	DiscountPaise   int64  `gorm:"column:discount_paise;not null"`
	GrandTotalPaise int64  `gorm:"column:grand_total_paise;not null"`
	PricingVersion  int    `gorm:"column:pricing_version;not null"`
	ComputedAt      time.Time `gorm:"column:computed_at;autoCreateTime"`
}

func (CartTotals) TableName() string { return "cart_totals" }

type CartIdempotency struct {
	ClientID       string `gorm:"column:client_id;primaryKey"`
	IdempotencyKey string `gorm:"column:idempotency_key;primaryKey"`
	Endpoint       string `gorm:"column:endpoint;not null"`
	RequestHash    string `gorm:"column:request_hash;not null"`

	ResourceID   []byte `gorm:"column:resource_id;type:binary(16)"`
	HTTPStatus   *int16 `gorm:"column:http_status"`
	ResponseBody string `gorm:"column:response_body;type:json"`
	State        string `gorm:"column:state;not null"`

	LockedAt  *time.Time `gorm:"column:locked_at"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (CartIdempotency) TableName() string { return "cart_idempotency" }

type CartOutbox struct {
	OutboxID      []byte `gorm:"column:outbox_id;type:binary(16);primaryKey"`
	AggregateType string `gorm:"column:aggregate_type;not null"`
	AggregateID   []byte `gorm:"column:aggregate_id;type:binary(16);index;not null"`
	EventType     string `gorm:"column:event_type;not null"`
	Payload       string `gorm:"column:payload;type:json;not null"`
	Status        string `gorm:"column:status;not null"`

	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	PublishedAt *time.Time `gorm:"column:published_at"`
}

func (CartOutbox) TableName() string { return "cart_outbox" }

type ProcessedEvent struct {
	EventID        []byte `gorm:"column:event_id;type:binary(16);primaryKey"`
	Consumer       string `gorm:"column:consumer;not null"`
	EventType      string `gorm:"column:event_type;not null"`
	CorrelationID  []byte `gorm:"column:correlation_id;type:binary(16)"`
	ProcessedAt    time.Time `gorm:"column:processed_at;autoCreateTime"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }
EOF

# ---------- internal/domain/dto.go ----------
cat > "$ROOT/internal/domain/dto.go" <<'EOF'
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
)

func UUIDToBin16(u uuid.UUID) []byte {
	b := make([]byte, 16)
	copy(b, u[:])
	return b
}

func Bin16ToUUID(b []byte) (uuid.UUID, error) {
	return uuid.FromBytes(b)
}

func HashRequest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}
EOF

# ---------- internal/domain/pricing.go ----------
cat > "$ROOT/internal/domain/pricing.go" <<'EOF'
package domain

// Keep pricing logic centralized.
// For MVP: recompute totals from items + applied promotions.
// Later: integrate pricing engine / tax rules / shipping rules.

type PricingSummary struct {
	SubtotalPaise   int64
	TaxPaise        int64
	ShippingPaise   int64
	DiscountPaise   int64
	GrandTotalPaise int64
}
EOF

# ---------- internal/http/middleware_idempotency.go ----------
cat > "$ROOT/internal/http/middleware_idempotency.go" <<'EOF'
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	HClientID       = "X-Client-Id"
	HIdempotencyKey = "Idempotency-Key"
)

func RequireIdempotencyHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader(HClientID) == "" || c.GetHeader(HIdempotencyKey) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "missing X-Client-Id or Idempotency-Key",
			})
			return
		}
		c.Next()
	}
}
EOF

# ---------- internal/http/router.go ----------
cat > "$ROOT/internal/http/router.go" <<'EOF'
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
EOF

# ---------- internal/http/handlers_cart.go ----------
cat > "$ROOT/internal/http/handlers_cart.go" <<'EOF'
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
EOF

# ---------- internal/outbox/publisher.go ----------
cat > "$ROOT/internal/outbox/publisher.go" <<'EOF'
package outbox

import (
	"context"
	"time"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/kafka"
	"gorm.io/gorm"
)

type Publisher struct {
	db       *gorm.DB
	producer *kafka.Producer
}

func NewPublisher(db *gorm.DB, producer *kafka.Producer) *Publisher {
	return &Publisher{db: db, producer: producer}
}

func (p *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = p.publishBatch(ctx, 50)
		}
	}
}

func (p *Publisher) publishBatch(ctx context.Context, limit int) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var rows []domain.CartOutbox
		if err := tx.Raw(`
			SELECT * FROM cart_outbox
			WHERE status='NEW'
			ORDER BY created_at ASC
			LIMIT ?
			FOR UPDATE
		`, limit).Scan(&rows).Error; err != nil {
			return err
		}

		for _, r := range rows {
			key := string(r.AggregateID)
			if err := p.producer.Publish(ctx, key, []byte(r.Payload)); err != nil {
				_ = tx.Model(&domain.CartOutbox{}).
					Where("outbox_id = ?", r.OutboxID).
					Update("status", "FAILED").Error
				continue
			}
			now := time.Now()
			if err := tx.Model(&domain.CartOutbox{}).
				Where("outbox_id = ?", r.OutboxID).
				Updates(map[string]any{"status": "PUBLISHED", "published_at": &now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
EOF

# ---------- internal/kafka/producer.go ----------
cat > "$ROOT/internal/kafka/producer.go" <<'EOF'
package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	w *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		w: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			Async:        false,
			BatchTimeout: 50 * time.Millisecond,
		},
	}
}

func (p *Producer) Close() error { return p.w.Close() }

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	return p.w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})
}
EOF

# ---------- internal/kafka/consumer.go ----------
cat > "$ROOT/internal/kafka/consumer.go" <<'EOF'
package kafka

// Placeholder for consuming events (optional for cart-service).
// If you consume external events (catalog price update, inventory signals),
// implement processed_events idempotency.

EOF

# ---------- internal/repo/cart_repo.go ----------
cat > "$ROOT/internal/repo/cart_repo.go" <<'EOF'
package repo

import (
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"
	"gorm.io/gorm"
)

type CartRepo struct{ db *gorm.DB }

func NewCartRepo(db *gorm.DB) *CartRepo { return &CartRepo{db: db} }

func (r *CartRepo) GetActiveCartByUser(tx *gorm.DB, userID []byte, channel string) (*domain.Cart, error) {
	var c domain.Cart
	err := tx.Where("owner_type='USER' AND user_id=? AND channel=? AND status='ACTIVE'", userID, channel).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}
EOF

# ---------- internal/repo/idem_repo.go ----------
cat > "$ROOT/internal/repo/idem_repo.go" <<'EOF'
package repo

import (
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"
	"gorm.io/gorm"
)

type IdemRepo struct{ db *gorm.DB }

func NewIdemRepo(db *gorm.DB) *IdemRepo { return &IdemRepo{db: db} }

func (r *IdemRepo) Create(tx *gorm.DB, row *domain.CartIdempotency) error {
	return tx.Create(row).Error
}

func (r *IdemRepo) Get(tx *gorm.DB, clientID, idemKey string) (*domain.CartIdempotency, error) {
	var rrow domain.CartIdempotency
	if err := tx.First(&rrow, "client_id=? AND idempotency_key=?", clientID, idemKey).Error; err != nil {
		return nil, err
	}
	return &rrow, nil
}
EOF

# ---------- internal/repo/outbox_repo.go ----------
cat > "$ROOT/internal/repo/outbox_repo.go" <<'EOF'
package repo

import (
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"
	"gorm.io/gorm"
)

type OutboxRepo struct{ db *gorm.DB }

func NewOutboxRepo(db *gorm.DB) *OutboxRepo { return &OutboxRepo{db: db} }

func (r *OutboxRepo) Add(tx *gorm.DB, row *domain.CartOutbox) error {
	return tx.Create(row).Error
}
EOF

# ---------- internal/repo/processed_repo.go ----------
cat > "$ROOT/internal/repo/processed_repo.go" <<'EOF'
package repo

import (
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"
	"gorm.io/gorm"
)

type ProcessedRepo struct{ db *gorm.DB }

func NewProcessedRepo(db *gorm.DB) *ProcessedRepo { return &ProcessedRepo{db: db} }

// InsertProcessed returns true if inserted, false if already exists (duplicate).
func (r *ProcessedRepo) InsertProcessed(tx *gorm.DB, row *domain.ProcessedEvent) (bool, error) {
	err := tx.Create(row).Error
	if err != nil {
		// In real code: check mysql duplicate key and return (false, nil)
		return false, err
	}
	return true, nil
}
EOF

echo "✅ Done."
echo ""
echo "Next steps:"
echo "  cd $ROOT"
echo "  go mod tidy"
echo "  go test ./... (optional)"