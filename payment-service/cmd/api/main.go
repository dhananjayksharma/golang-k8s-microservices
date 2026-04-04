package main

import (
	"log"
	"os"
	"strconv"

	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/db"
	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/middleware"
	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/routes"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" || len(dsn) == 0 {
		log.Fatalf("dsn string not found error: %v", dsn)
	}
	isLocal, _ := strconv.ParseBool(os.Getenv("LOCAL_DB"))
	var gdb *gorm.DB
	var err error
	// var err error
	if isLocal {
		gdb, err = db.ConnectMySQLNoTLS(dsn)
		if err != nil {
			log.Fatalf("db connect error: %v", err)
		}
	} else {
		capempath := os.Getenv("MYSQL_DBPEM")
		if capempath == "" || len(capempath) == 0 {
			log.Fatalf("capempath string not found error: %v", capempath)
		}
		gdb, err = db.ConnectMySQLTLS(dsn, capempath)
		if err != nil {
			log.Fatalf("db connect error: %v", err)
		}
	}
	gin.SetMode(gin.DebugMode)
	r := gin.Default()
	var activeRateLimiter = "v2"
	if activeRateLimiter == "v2" {
		r.Use(middleware.RateLimiterMiddleware())
	} else {
		rl := middleware.NewIPRateLimiter(rate.Limit(5), 10)
		r.Use(rl.Middleware())
	}

	routes.Register(r, gdb)

	log.Println("listening on :8112")
	if err := r.Run(":8112"); err != nil {
		log.Fatal(err)
	}
}
