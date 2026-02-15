package handlers

import "golang-k8s-microservices/invoice-service/internal/models"

type CreateInvoiceRequest struct {
	CustomerID    uint64          `json:"customer_id" binding:"required"`
	CustomerEmail string          `json:"customer_email" binding:"required,email"`
	DBName        string          `json:"db_name" binding:"required"`
	DBEngine      models.DBEngine `json:"db_engine" binding:"required,oneof=mysql postgres mongodb redis"`
	DBVersion     string          `json:"db_version"`
	StorageGB     int             `json:"storage_gb" binding:"required,gt=0"`
	Region        string          `json:"region" binding:"required"`
	PriceMonthly  float64         `json:"price_monthly" binding:"required,gt=0"`
}

type UpdateInvoiceRequest struct {
	CustomerEmail *string             `json:"customer_email" binding:"omitempty,email"`
	DBName        *string             `json:"db_name"`
	DBEngine      *models.DBEngine    `json:"db_engine" binding:"omitempty,oneof=mysql postgres mongodb redis"`
	DBVersion     *string             `json:"db_version"`
	StorageGB     *int                `json:"storage_gb" binding:"omitempty,gt=0"`
	Region        *string             `json:"region"`
	OrderStatus   *models.OrderStatus `json:"order_status" binding:"omitempty,oneof=CREATED PROVISIONING ACTIVE SUSPENDED TERMINATED"`
	PriceMonthly  *float64            `json:"price_monthly" binding:"omitempty,gt=0"`
}
