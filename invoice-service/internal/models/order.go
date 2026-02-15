package models

import (
	"time"

	"gorm.io/gorm"
)

type DBEngine string
type OrderStatus string

const (
	DBMySQL    DBEngine = "mysql"
	DBPostgres DBEngine = "postgres"
	DBMongoDB  DBEngine = "mongodb"
	DBRedis    DBEngine = "redis"
)

const (
	StatusCreated      OrderStatus = "CREATED"
	StatusProvisioning OrderStatus = "PROVISIONING"
	StatusActive       OrderStatus = "ACTIVE"
	StatusSuspended    OrderStatus = "SUSPENDED"
	StatusTerminated   OrderStatus = "TERMINATED"
)

type Order struct {
	OrderID       uint64 `gorm:"column:order_id;primaryKey;autoIncrement" json:"order_id"`
	CustomerID    uint64 `gorm:"column:customer_id;not null" json:"customer_id"`
	CustomerEmail string `gorm:"column:customer_email;size:255;not null" json:"customer_email"`

	DBName    string   `gorm:"column:db_name;size:100;not null" json:"db_name"`
	DBEngine  DBEngine `gorm:"column:db_engine;type:enum('mysql','postgres','mongodb','redis');not null" json:"db_engine"`
	DBVersion string   `gorm:"column:db_version;size:20" json:"db_version,omitempty"`

	StorageGB int    `gorm:"column:storage_gb;not null" json:"storage_gb"`
	Region    string `gorm:"column:region;size:50;not null" json:"region"`

	OrderStatus  OrderStatus `gorm:"column:order_status;type:enum('CREATED','PROVISIONING','ACTIVE','SUSPENDED','TERMINATED');not null;default:'CREATED'" json:"order_status"`
	PriceMonthly float64     `gorm:"column:price_monthly;type:decimal(10,2);not null" json:"price_monthly"`

	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // optional (soft delete). Remove if you want hard delete only.
}

func (Order) TableName() string { return "orders" }
