package main

import (
	"context"
	"log"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/config"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/db"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/kafka"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/outbox"
)

func main() {
	cfg := config.Load()

	gdb, err := db.NewMySQL(cfg.MySQL.DSN())
	if err != nil {
		log.Fatal(err)
	}

	prod := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer prod.Close()

	pub := outbox.NewPublisher(gdb, prod)
	log.Printf("cart-worker publishing outbox to topic=%s\n", cfg.Kafka.Topic)
	log.Fatal(pub.Run(context.Background()))
}
