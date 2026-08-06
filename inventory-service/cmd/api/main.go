package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang-k8s-microservices/inventory-service/internal/cache"
	"golang-k8s-microservices/inventory-service/internal/db"
	"golang-k8s-microservices/inventory-service/internal/events"
	"golang-k8s-microservices/inventory-service/internal/inventory"
	"golang-k8s-microservices/inventory-service/internal/logger"
	"golang-k8s-microservices/inventory-service/internal/middleware"
	"golang-k8s-microservices/inventory-service/internal/routes"

	"github.com/gin-gonic/gin"
)

const (
	defaultAddress  = ":8914"
	shutdownTimeout = 10 * time.Second
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:rootany@tcp(localhost:3306)/appdb?parseTime=true"
		log.Printf("MYSQL_DSN is not set; using local default DSN")
	}

	gdb, err := db.NewGormMySQL(dsn)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		log.Fatalf("get underlying mysql connection: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("close mysql connection: %v", err)
		}
	}()

	stockService := inventory.NewService(gdb)
	if err := stockService.Migrate(); err != nil {
		log.Fatalf("inventory migrate: %v", err)
	}

	redisClient := cache.New()
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("close redis client: %v", err)
		}
	}()

	logger.Init("dev")
	defer func() {
		_ = logger.Log.Sync()
	}()

	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	var backgroundWG sync.WaitGroup
	backgroundWG.Add(1)
	go func() {
		defer backgroundWG.Done()

		err := events.NewConsumer(stockService, redisClient.RDB).Run(rootCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("rabbitmq consumer stopped with error: %v", err)
			return
		}

		log.Println("rabbitmq consumer stopped")
	}()

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Recovery(),
	)
	routes.Register(r, gdb)

	httpServer := &http.Server{
		Addr:              defaultAddress,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("inventory-service listening on %s", httpServer.Addr)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	select {
	case <-rootCtx.Done():
		log.Printf("shutdown signal received: %v", rootCtx.Err())
	case err := <-serverErr:
		if err != nil {
			log.Printf("http server stopped unexpectedly: %v", err)
		}
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful HTTP shutdown failed: %v", err)

		if closeErr := httpServer.Close(); closeErr != nil {
			log.Printf("force-close HTTP server: %v", closeErr)
		}
	}

	consumerStopped := make(chan struct{})
	go func() {
		backgroundWG.Wait()
		close(consumerStopped)
	}()

	select {
	case <-consumerStopped:
		log.Println("background workers stopped")
	case <-shutdownCtx.Done():
		log.Printf("background worker shutdown timed out: %v", shutdownCtx.Err())
	}

	log.Println("inventory-service stopped gracefully")
}
