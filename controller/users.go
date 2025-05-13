package controller

import (
	"ServerRoom/internal/storage"
	"ServerRoom/models"
	"ServerRoom/utils"

	"github.com/gofiber/fiber/v2"
)

func Register(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}
	user.Password = hashedPassword

	query := `INSERT INTO users (username, password) VALUES ($1, $2)`
	_, err = storage.DbPool.Exec(c.Context(), query, user.Username, user.Password)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"message":  "User registered successfully",
		"username": user.Username,
	})
}

func Login(c *fiber.Ctx) error {
	var input models.User
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	var storedPassword string
	query := `SELECT password FROM users WHERE username=$1`
	err := storage.DbPool.QueryRow(c.Context(), query, input.Username).Scan(&storedPassword)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	if !utils.CheckPasswordHash(input.Password, storedPassword) {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"message":  "Login successful",
		"username": input.Username,
	})
}
