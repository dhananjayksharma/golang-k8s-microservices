package main

import (
	"fmt"
	"order-service/router"

	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	viper.BindEnv("ORDER_PORT")

	port := viper.GetString("ORDER_PORT")
	if port == "" {
		port = "8081" // default
	}
	addr := fmt.Sprintf(":%s", port)
	r := router.SetupRouter()
	r.Run(addr)
}
