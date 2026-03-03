package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

type Consumer struct {
	group sarama.ConsumerGroup
	topic string
}

func NewConsumerGroup(brokers []string, groupID, topic string) (*Consumer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_6_0_0

	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRange()
	cfg.Consumer.Return.Errors = true

	cg, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}
	return &Consumer{group: cg, topic: topic}, nil
}

func (c *Consumer) Close() error { return c.group.Close() }

type handler struct{}

func (h handler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h handler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h handler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var ev OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			log.Printf("❌ bad message: %v value=%s", err, string(msg.Value))
			// mark consumed to avoid poison-loop (or send to DLQ in real systems)
			sess.MarkMessage(msg, "bad_json")
			continue
		}

		log.Printf("📥 consumed topic=%s partition=%d offset=%d event=%s order_id=%s amount=%d %s",
			msg.Topic, msg.Partition, msg.Offset, ev.EventType, ev.OrderID, ev.Amount, ev.Currency)

		// ✅ your business logic here:
		// - update read model
		// - trigger shipping workflow
		// - create invoice
		// - etc.

		sess.MarkMessage(msg, "")
	}
	return nil
}

func (c *Consumer) Run(ctx context.Context) {
	h := handler{}
	errs := c.group.Errors()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-errs:
				if !ok {
					return
				}
				if err != nil {
					log.Printf("❌ consumer group error: %v", err)
				}
			}
		}
	}()

	for {
		if err := c.group.Consume(ctx, []string{c.topic}, h); err != nil {
			log.Printf("❌ consumer error: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}
