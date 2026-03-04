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
