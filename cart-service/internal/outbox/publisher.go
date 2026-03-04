package outbox

import (
	"context"
	"time"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/domain"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/kafka"
	"gorm.io/gorm"
)

type Publisher struct {
	db       *gorm.DB
	producer *kafka.Producer
}

func NewPublisher(db *gorm.DB, producer *kafka.Producer) *Publisher {
	return &Publisher{db: db, producer: producer}
}

func (p *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = p.publishBatch(ctx, 50)
		}
	}
}

func (p *Publisher) publishBatch(ctx context.Context, limit int) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		var rows []domain.CartOutbox
		if err := tx.Raw(`
			SELECT * FROM cart_outbox
			WHERE status='NEW'
			ORDER BY created_at ASC
			LIMIT ?
			FOR UPDATE
		`, limit).Scan(&rows).Error; err != nil {
			return err
		}

		for _, r := range rows {
			key := string(r.AggregateID)
			if err := p.producer.Publish(ctx, key, []byte(r.Payload)); err != nil {
				_ = tx.Model(&domain.CartOutbox{}).
					Where("outbox_id = ?", r.OutboxID).
					Update("status", "FAILED").Error
				continue
			}
			now := time.Now()
			if err := tx.Model(&domain.CartOutbox{}).
				Where("outbox_id = ?", r.OutboxID).
				Updates(map[string]any{"status": "PUBLISHED", "published_at": &now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
