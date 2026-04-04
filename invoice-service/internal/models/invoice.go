package models

import (
	"time"

	"gorm.io/gorm"
)

type DBEngine string
type InvoiceStatus string

const (
	DBMySQL    DBEngine = "mysql"
	DBPostgres DBEngine = "postgres"
	DBMongoDB  DBEngine = "mongodb"
	DBRedis    DBEngine = "redis"
)

const (
	StatusNew       InvoiceStatus = "NEW"
	StatusActive    InvoiceStatus = "ACTIVE"
	StatusCancelled InvoiceStatus = "CANCELLED"
)

type Invoice struct {
	ID            uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderId       int64          `gorm:"order_id" json:"order_id"`
	InvoiceNumber string         `gorm:"invoice_number" json:"invoice_number"`
	TotalAmount   float64        `gorm:"total_amount" json:"total_amount"`
	Status        string         `gorm:"status" json:"status"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"` // optional (soft delete). Remove if you want hard delete only.
}

func (Invoice) TableName() string { return "invoices" }
