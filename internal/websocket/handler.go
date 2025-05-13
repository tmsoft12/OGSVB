package websocketT

import (
	"ServerRoom/internal/storage"
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
)

type Client struct {
	Conn *websocket.Conn
}

var (
	clients    = make(map[*Client]bool)
	clientsMu  sync.Mutex
	register   = make(chan *Client)
	unregister = make(chan *Client)
)

func RunBroadcaster() {
	for {
		select {
		case client := <-register:
			clientsMu.Lock()
			clients[client] = true
			clientsMu.Unlock()
		case client := <-unregister:
			clientsMu.Lock()
			delete(clients, client)
			clientsMu.Unlock()
		case message := <-storage.BroadcastCh:
			clientsMu.Lock()
			for client := range clients {
				err := client.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					client.Conn.Close()
					delete(clients, client)
				}
			}
			clientsMu.Unlock()
		}
	}
}

func WebSocketHandler(c *websocket.Conn) {
	client := &Client{Conn: c}
	defer func() {
		unregister <- client
		c.Close()
	}()

	register <- client

	storage.Mutex.Lock()
	for _, data := range storage.SensorData {
		jsonData, err := json.Marshal(data)
		if err != nil {
			continue
		}
		err = c.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			storage.Mutex.Unlock()
			return
		}
	}
	storage.Mutex.Unlock()

	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket message read error:", err)
			return
		}
	}
}
