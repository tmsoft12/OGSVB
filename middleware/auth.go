package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte("supersecret")

// Protected middleware to check if the token is valid
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Token'ı cookie'den al
		tokenStr := c.Cookies("token")
		if tokenStr == "" {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}

		// Token'ı parse et
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// Algoritma kontrolü
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Claimleri kontrol et (exp gibi)
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid token claims"})
		}

		// Expiration (exp) claim'ini kontrol et
		if exp, ok := claims["exp"].(float64); ok {
			// Eğer token süresi geçmişse
			if int64(exp) < time.Now().Unix() {
				return c.Status(401).JSON(fiber.Map{"error": "Token expired"})
			}
		} else {
			// Exp claim'i yoksa hata döndür
			return c.Status(401).JSON(fiber.Map{"error": "Token does not contain an expiration time"})
		}

		// Her şey yolunda, route'a devam et
		return c.Next()
	}
}
