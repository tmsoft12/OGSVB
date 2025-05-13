package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte("supersecret") // Aynı key olmalı

// Protected middleware - Token doğrulama
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Cookie'den token al
		tokenStr := c.Cookies("token")
		if tokenStr == "" {
			// Token yoksa 401 dön
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}

		// Token'ı çözümle ve doğrula
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			// Geçersiz token durumunda 401 dön
			return c.Status(401).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Token geçerliyse, işlemi devam ettir
		return c.Next()
	}
}
