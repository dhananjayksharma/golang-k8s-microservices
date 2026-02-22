package main

import (
	"github.com/gin-gonic/gin"

	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/handlers"
	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/repository"
	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/service"
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
