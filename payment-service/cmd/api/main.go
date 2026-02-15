package main

import (
	"log"
	"os"

	"go-gin-mysql-k8s/internal/db"
	"go-gin-mysql-k8s/internal/routes"

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

	r := gin.Default()
	routes.Register(r, gdb)

	log.Println("listening on :8112")
	if err := r.Run(":8112"); err != nil {
		log.Fatal(err)
	}
}
