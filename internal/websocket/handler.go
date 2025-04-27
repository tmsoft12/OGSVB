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
			log.Println("New client connected, total clients:", len(clients))
		case client := <-unregister:
			clientsMu.Lock()
			delete(clients, client)
			clientsMu.Unlock()
			log.Println("Client disconnected, total clients:", len(clients))
		case message := <-storage.BroadcastCh:
			clientsMu.Lock()
			for client := range clients {
				err := client.Conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					log.Println("WebSocket message send error:", err)
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
		_, _, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket message read error:", err)
			return
		}
	}
}
