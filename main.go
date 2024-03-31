package main

import (
	"fmt"

	config "github.com/kerosiinikone/go-docker-grpc/config"
)

func main() {
	cfg := config.New()
	fmt.Printf("%v", cfg)
}