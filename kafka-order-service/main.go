package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kafka-order-service/kafka"

	"github.com/gin-gonic/gin"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type CreateOrderReq struct {
	OrderID  string `json:"order_id" binding:"required"`
	UserID   string `json:"user_id" binding:"required"`
	Amount   int64  `json:"amount" binding:"required"`
	Currency string `json:"currency" binding:"required"`
}

func parseBrokers(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))

	for _, p := range parts {
		b := strings.TrimSpace(p)
		if b != "" {
			brokers = append(brokers, b)
		}
	}
	if len(brokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS is empty")
	}
	return brokers, nil
}

func main() {
	brokers, err := parseBrokers(getenv("KAFKA_BROKERS", "localhost:9092"))

	// brokers := strings.Split(getenv("KAFKA_BROKERS", "localhost/127.0.0.1:9092"), ",") //  []string{"127.0.0.1", "9092"} //

	log.Printf("Kafka brokers = %v", brokers)

	// if err != nil {
	// 	log.Fatal(err)
	// }
	// os.Exit(0)
	topic := getenv("KAFKA_TOPIC", "order.events")
	groupID := getenv("KAFKA_GROUP_ID", "order-service")
	if strings.TrimSpace(topic) == "" {
		log.Fatal("KAFKA_TOPIC is empty")
	}
	if strings.TrimSpace(groupID) == "" {
		log.Fatal("KAFKA_GROUP_ID is empty")
	}

	producer, err := kafka.NewProducer(brokers, topic)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	consumer, err := kafka.NewConsumerGroup(brokers, groupID, topic)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	// consumer run in background
	ctx, cancel := context.WithCancel(context.Background())
	go consumer.Run(ctx)

	r := gin.Default()

	r.POST("/orders", func(c *gin.Context) {
		var req CreateOrderReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ev := kafka.OrderCreatedEvent{
			EventID:   req.OrderID + "-" + time.Now().Format("20060102150405"),
			EventType: "order.created",
			OrderID:   req.OrderID,
			UserID:    req.UserID,
			Amount:    req.Amount,
			Currency:  req.Currency,
			CreatedAt: time.Now().UTC(),
		}

		if err := producer.PublishOrderCreated(ev); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish event"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"status": "created", "order_id": req.OrderID})
	})

	srv := &http.Server{
		Addr:    ":" + getenv("PORT", "8088"),
		Handler: r,
	}

	go func() {
		log.Printf("🚀 order-service listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("🛑 shutting down...")
	cancel()

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()
	_ = srv.Shutdown(ctxTimeout)
}
