package main

import (
	"log"

	"example.com/go-service-template/internal/di"
)

func main() {
	app, cleanup, err := di.New()
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	defer cleanup()

	if err := app.Run(":8080"); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
