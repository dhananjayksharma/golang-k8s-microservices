package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficientStock = errors.New("insufficient stock")

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }
func (s *Service) Migrate() error     { return s.db.AutoMigrate(&StockItem{}, &Reservation{}) }
func (s *Service) Reserve(ctx context.Context, orderID, sku string, qty int64) (Reservation, error) {
	var result Reservation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing Reservation
		if err := tx.Where("order_id = ?", orderID).First(&existing).Error; err == nil {
			result = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var item StockItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("sku = ?", sku).First(&item).Error; err != nil {
			return err
		}
		if item.Available < qty {
			return ErrInsufficientStock
		}
		item.Available -= qty
		item.Reserved += qty
		item.Version++
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		result = Reservation{ID: uuid.NewString(), OrderID: orderID, SKU: sku, Quantity: qty, Status: "RESERVED"}
		return tx.Create(&result).Error
	})
	if err != nil {
		return Reservation{}, fmt.Errorf("reserve stock: %w", err)
	}
	return result, nil
}
func (s *Service) List(ctx context.Context) ([]StockItem, error) {
	var out []StockItem
	return out, s.db.WithContext(ctx).Order("sku").Find(&out).Error
}
