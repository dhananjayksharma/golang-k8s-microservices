package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"order-service/database"
	"order-service/internal/cache"
	"order-service/internal/events"
	"order-service/router"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

const shutdownTimeout = 10 * time.Second

func main() {
	viper.AutomaticEnv()
	_ = viper.BindEnv("ORDER_PORT")

	port := viper.GetString("ORDER_PORT")
	if port == "" {
		port = "8081"
	}

	db, err := database.OpenPostgres()
	if err != nil {
		log.Fatalf("database initialization failed, please check all the configurations: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("database close failed: %v", err)
		}
	}()

	redisClient := cache.New()
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("redis close failed: %v", err)
		}
	}()

	// The context is cancelled when Ctrl+C or SIGTERM is received.
	appCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)

		if err := events.NewResultConsumer(db, redisClient.RDB).Run(appCtx); err != nil &&
			!errors.Is(err, context.Canceled) {
			log.Printf("inventory result consumer stopped with error: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%s", port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           router.SetupRouter(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("order-service listening on %s", addr)
		serverErr <- httpServer.ListenAndServe()
	}()

	select {
	case <-appCtx.Done():
		log.Printf("shutdown signal received: %v", appCtx.Err())

	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped unexpectedly: %v", err)
		}

		// Stop the consumer when the HTTP server exits unexpectedly.
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful HTTP shutdown failed: %v", err)

		if closeErr := httpServer.Close(); closeErr != nil {
			log.Printf("forced HTTP server close failed: %v", closeErr)
		}
	}

	select {
	case <-consumerDone:
		log.Println("inventory result consumer stopped")

	case <-shutdownCtx.Done():
		log.Printf("timed out waiting for inventory result consumer: %v", shutdownCtx.Err())
	}

	log.Println("order-service stopped gracefully")
}
