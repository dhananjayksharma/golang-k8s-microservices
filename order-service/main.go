package main

import (
	"context"
	"fmt"
	"log"
	"order-service/database"
	"order-service/internal/cache"
	"order-service/internal/events"
	"order-service/router"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	_ = viper.BindEnv("ORDER_PORT")

	port := viper.GetString("ORDER_PORT")
	if port == "" {
		port = "8081"
	}

	db, err := database.OpenPostgres()
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}
	defer db.Close()

	redisClient := cache.New()
	defer redisClient.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := events.NewResultConsumer(db, redisClient.RDB).Run(ctx); err != nil {
			log.Printf("inventory result consumer stopped: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%s", port)
	r := router.SetupRouter(db)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
