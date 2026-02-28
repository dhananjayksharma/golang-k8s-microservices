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
