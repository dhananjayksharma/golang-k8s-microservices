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
	CartPromoID   []byte    `gorm:"column:cart_promo_id;type:binary(16);primaryKey"`
	CartID        []byte    `gorm:"column:cart_id;type:binary(16);index;not null"`
	PromoCode     string    `gorm:"column:promo_code;not null"`
	PromoType     string    `gorm:"column:promo_type;not null"`
	DiscountPaise int64     `gorm:"column:discount_paise;not null"`
	PromoMeta     string    `gorm:"column:promo_meta;type:json"`
	Status        string    `gorm:"column:status;not null"`
	AppliedAt     time.Time `gorm:"column:applied_at;autoCreateTime"`
}

func (CartPromotion) TableName() string { return "cart_promotions" }

type CartTotals struct {
	CartID          []byte    `gorm:"column:cart_id;type:binary(16);primaryKey"`
	SubtotalPaise   int64     `gorm:"column:subtotal_paise;not null"`
	TaxPaise        int64     `gorm:"column:tax_paise;not null"`
	ShippingPaise   int64     `gorm:"column:shipping_paise;not null"`
	DiscountPaise   int64     `gorm:"column:discount_paise;not null"`
	GrandTotalPaise int64     `gorm:"column:grand_total_paise;not null"`
	PricingVersion  int       `gorm:"column:pricing_version;not null"`
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
	EventID       []byte    `gorm:"column:event_id;type:binary(16);primaryKey"`
	Consumer      string    `gorm:"column:consumer;not null"`
	EventType     string    `gorm:"column:event_type;not null"`
	CorrelationID []byte    `gorm:"column:correlation_id;type:binary(16)"`
	ProcessedAt   time.Time `gorm:"column:processed_at;autoCreateTime"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }

type EventEnvelope struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Producer       string    `json:"producer"`
	OccurredAt     time.Time `json:"occurred_at"`
	CorrelationID  string    `json:"correlation_id"` // cart_id or order_id
	TraceID        string    `json:"trace_id,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	Data           any       `json:"data"`
}
