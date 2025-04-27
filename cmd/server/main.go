package main

import (
	"ServerRoom/internal/api"
	"ServerRoom/internal/config"
	mqttclient "ServerRoom/internal/mqtt"
	"ServerRoom/internal/storage"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.Load()

	err := storage.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}

	go mqttclient.Start(cfg.MQTTBroker)

	app := fiber.New()

	api.SetupRoutes(app)
	log.Fatal(app.Listen(cfg.ServerPort))
}
