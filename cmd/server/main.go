package main

import (
	"ServerRoom/internal/api"
	"ServerRoom/internal/config"
	mqttclient "ServerRoom/internal/mqtt"
	"ServerRoom/internal/storage"
	websocketT "ServerRoom/internal/websocket"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	cfg := config.Load()
	go websocketT.RunBroadcaster()
	err := storage.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}

	go mqttclient.Start(cfg.MQTTBroker)

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://192.168.100.7:5173",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowCredentials: true,
	}))

	api.SetupRoutes(app)
	log.Fatal(app.Listen(cfg.ServerPort))
}
