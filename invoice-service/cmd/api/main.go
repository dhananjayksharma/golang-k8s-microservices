package main

import (
	"log"
	"os"

	"golang-k8s-microservices/invoice-service/internal/db"
	"golang-k8s-microservices/invoice-service/internal/logger"
	"golang-k8s-microservices/invoice-service/internal/middleware"
	"golang-k8s-microservices/invoice-service/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" || len(dsn) == 0 {
		dsn = "root:root@tcp(localhost:3306)/appdb?parseTime=true"
		log.Fatalf("dsn string not found error: %v", dsn)
	}

	gdb, err := db.NewGormMySQL(dsn)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

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

	log.Println("listening on :8114")
	if err := r.Run(":8114"); err != nil {
		log.Fatal(err)
	}
}
