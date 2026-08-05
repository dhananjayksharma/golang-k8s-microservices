package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang-k8s-microservices/inventory-service/internal/cache"
	"golang-k8s-microservices/inventory-service/internal/db"
	"golang-k8s-microservices/inventory-service/internal/events"
	"golang-k8s-microservices/inventory-service/internal/inventory"
	"golang-k8s-microservices/inventory-service/internal/logger"
	"golang-k8s-microservices/inventory-service/internal/middleware"
	"golang-k8s-microservices/inventory-service/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" || len(dsn) == 0 {
		dsn = "root:rootany@tcp(localhost:3306)/appdb?parseTime=true"
		log.Fatalf("dsn string not found error: %v", dsn)
	}

	gdb, err := db.NewGormMySQL(dsn)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	stockService := inventory.NewService(gdb)
	redisClient := cache.New()
	defer redisClient.Close()
	if err := stockService.Migrate(); err != nil {
		log.Fatalf("inventory migrate: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := events.NewConsumer(stockService, redisClient.RDB).Run(ctx); err != nil {
			log.Printf("rabbitmq consumer stopped: %v", err)
		}
	}()

	logger.Init("dev")
	defer logger.Log.Sync()

	r := gin.New()

	r.Use(
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Recovery(),
	)

	//r := gin.Default()
	routes.Register(r, gdb)

	log.Println("listening on :8914")
	if err := r.Run(":8914"); err != nil {
		log.Fatal(err)
	}
}
