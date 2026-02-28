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
