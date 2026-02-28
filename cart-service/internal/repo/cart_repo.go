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
