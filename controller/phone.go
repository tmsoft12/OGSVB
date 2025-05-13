package controller

import (
	"ServerRoom/internal/storage"
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

type PhoneRequest struct {
	PhoneNumber string `json:"phoneNumber"`
}

func GetPhone(c *fiber.Ctx) error {
	var phoneNumber string
	err := storage.DbPool.QueryRow(context.Background(), "SELECT phoneNumber FROM phones LIMIT 1").Scan(&phoneNumber)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Telefon belgisi alnanok",
		})
	}
	return c.JSON(fiber.Map{
		"phoneNumber": phoneNumber,
	})
}

func UpdatePhoneNumber(c *fiber.Ctx) error {
	var req PhoneRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ýalňyş sorag gurluşy",
		})
	}

	if req.PhoneNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Telefon belgisi boş bolup bilmez",
		})
	}

	_, err := storage.DbPool.Exec(context.Background(), "UPDATE phones SET phoneNumber=$1 WHERE id=$2", req.PhoneNumber, 1)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Telefon belgisi täzelenmedi",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Telefon belgisi üstünlikli täzelendi",
	})
}

func GetEventsByTopic(c *fiber.Ctx) error {
	topic := c.Query("topic")
	if topic == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "topic parametri gerekli",
		})
	}

	limit := c.QueryInt("limit", 10)
	page := c.QueryInt("page", 1)

	offset := (page - 1) * limit

	rows, err := storage.DbPool.Query(context.Background(), `
		SELECT id, topic, "value", timestamp 
		FROM events 
		WHERE topic=$1 
		ORDER BY timestamp DESC 
		LIMIT $2 OFFSET $3`, topic, limit, offset)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Maglumat çekilende säwlik",
		})
	}
	defer rows.Close()

	type Event struct {
		ID        int       `json:"id"`
		Topic     string    `json:"topic"`
		Value     string    `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}

	var events []Event

	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Topic, &e.Value, &e.Timestamp); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Satyr çekilende säwlik",
			})
		}
		events = append(events, e)
	}

	// Toplam veri sayısını da almak isteyebilirsiniz
	var totalCount int
	err = storage.DbPool.QueryRow(context.Background(), `
		SELECT COUNT(*) 
		FROM events 
		WHERE topic=$1`, topic).Scan(&totalCount)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Toplam veri sayısı alınırken hata oluştu",
		})
	}

	// Pagination bilgisi ile birlikte verileri döndür
	hasPrev := page > 1
	hasNext := (page * limit) < totalCount

	return c.JSON(fiber.Map{
		"data":       events,
		"totalCount": totalCount,
		"page":       page,
		"limit":      limit,
		"hasPrev":    hasPrev,
		"hasNext":    hasNext,
	})
}
