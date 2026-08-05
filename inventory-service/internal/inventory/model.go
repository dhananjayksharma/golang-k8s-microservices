package inventory

import "time"

type StockItem struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	SKU       string    `gorm:"size:100;uniqueIndex;not null" json:"sku"`
	Available int64     `gorm:"not null;default:0;check:available >= 0" json:"available"`
	Reserved  int64     `gorm:"not null;default:0;check:reserved >= 0" json:"reserved"`
	Version   int64     `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Reservation struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	OrderID   string    `gorm:"type:char(36);uniqueIndex;not null" json:"order_id"`
	SKU       string    `gorm:"size:100;index;not null" json:"sku"`
	Quantity  int64     `gorm:"not null" json:"quantity"`
	Status    string    `gorm:"size:30;not null" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
