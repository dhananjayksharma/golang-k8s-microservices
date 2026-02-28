package main

import (
	"log"
	"os"
	"strconv"

	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/config"
	"github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/db"
	httpx "github.com/dhananjayksharma/golang-k8s-microservices/cart-service/internal/http"
)

func main() {
	cfg := config.Load()

	gdb, err := db.NewMySQL(cfg.MySQL.DSN())
	if err != nil {
		log.Fatal(err)
	}

	r := httpx.NewRouter(gdb)
	log.Printf("cart-service listening on :%d\n", cfg.HTTPPort)
	log.Fatal(r.Run(":" + strconv.Itoa(cfg.HTTPPort)))
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
