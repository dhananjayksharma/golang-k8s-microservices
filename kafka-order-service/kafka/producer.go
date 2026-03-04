package kafka

import (
	"encoding/json"
	"log"
	"time"

	"github.com/IBM/sarama"
)

type Producer struct {
	topic string
	sp    sarama.SyncProducer
}

func NewProducer(brokers []string, topic string) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_6_0_0
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Return.Successes = true

	// good default for ordering per key (key-based partitioning)
	cfg.Producer.Partitioner = sarama.NewHashPartitioner

	sp, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	return &Producer{topic: topic, sp: sp}, nil
}

func (p *Producer) Close() error { return p.sp.Close() }

type OrderCreatedEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

func (p *Producer) PublishOrderCreated(ev OrderCreatedEvent) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		// Key helps preserve order for the same order/user
		Key:   sarama.StringEncoder(ev.OrderID),
		Value: sarama.ByteEncoder(b),
	}

	partition, offset, err := p.sp.SendMessage(msg)
	if err != nil {
		return err
	}
	log.Printf("✅ produced event partition=%d offset=%d order_id=%s", partition, offset, ev.OrderID)
	return nil
}
