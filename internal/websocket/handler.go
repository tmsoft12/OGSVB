package websocketT

import (
	"ServerRoom/internal/storage"
	"encoding/json"
	"log"

	"github.com/gofiber/websocket/v2"
)

func WebSocketHandler(c *websocket.Conn) {
	defer c.Close()

	storage.Mutex.Lock()
	for _, data := range storage.SensorData {
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Println("Cache JSON marshaling error:", err)
			continue
		}

		err = c.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			log.Println("WebSocket initial data send error:", err)
			storage.Mutex.Unlock()
			return
		}
	}
	storage.Mutex.Unlock()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket message read error:", err)
			return
		}

		// Mesajı işler ve gönderebilirsiniz
		err = c.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("WebSocket message send error:", err)
			return
		}
	}
}
