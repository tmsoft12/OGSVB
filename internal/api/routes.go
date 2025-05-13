package api

import (
	"ServerRoom/controller"
	websocketT "ServerRoom/internal/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func SetupRoutes(app *fiber.App) {
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	app.Get("/ws", websocket.New(websocketT.WebSocketHandler))
	app.Post("/login", controller.Login)
	app.Post("/register", controller.Register)
}
