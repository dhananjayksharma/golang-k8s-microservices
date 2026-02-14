package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"message-service/internal/config"
	"message-service/internal/repository"
	"message-service/internal/service"
	thttp "message-service/internal/transport/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// CLI flags
	debug := flag.Bool("debug", false, "enable debug mode")
	flag.Parse()
	fmt.Println("debug:::", debug)
	if *debug {
		gin.SetMode(gin.DebugMode)
		log.Println("🚀 running in DEBUG mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}

	col := client.Database(cfg.MongoDB).Collection(cfg.MongoCol)

	repo := repository.NewMongoOrderRepo(col)
	svc := service.NewOrderService(repo)
	h := thttp.NewHandler(svc)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	thttp.RegisterRoutes(r, h)

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
