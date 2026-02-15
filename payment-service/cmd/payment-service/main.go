package main

import (
	"github.com/gin-gonic/gin"

	"go-gin-mysql-k8s/internal/handlers"
	"go-gin-mysql-k8s/internal/repository"
	"go-gin-mysql-k8s/internal/service"
)

func main() {
	r := gin.Default()

	// Dummy repository (no DB)
	paymentRepo := repository.NewDummyPaymentRepository()

	paymentService := service.NewPaymentService(paymentRepo)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	r.GET("/payments", paymentHandler.ListPayments)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	r.Run(":8111")
}
