package main

import (
	"log"
	"os"

	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/db"
	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/models"
	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required")
	}

	gdb, err := db.NewGormMySQL(dsn)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	if err := gdb.AutoMigrate(&models.Order{}); err != nil {
		log.Fatalf("auto-migrate error: %v", err)
	}

	r := gin.Default()
	routes.Register(r, gdb)

	if err := r.Run(":8111"); err != nil {
		log.Fatal(err)
	}
}
