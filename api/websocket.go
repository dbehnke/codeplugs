package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocketHub maintains the set of active clients and broadcasts messages
type WebSocketHub struct {
	// Registered clients.
	clients map[*websocket.Conn]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *websocket.Conn

	// Unregister requests from clients.
	unregister chan *websocket.Conn

	mu sync.Mutex
}

func newHub() *WebSocketHub {
	return &WebSocketHub{
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		clients:    make(map[*websocket.Conn]bool),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

var Hub = newHub()

// ImportProgress tracks the status of a running import
type ImportProgress struct {
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Status    string `json:"status"` // "running", "completed", "error"
	Message   string `json:"message"`
	mu        sync.Mutex
}

var CurrentProgress = &ImportProgress{Status: "idle"}

func BroadcastProgress() {
	CurrentProgress.mu.Lock()
	defer CurrentProgress.mu.Unlock()

	msg, _ := json.Marshal(map[string]interface{}{
		"type": "import_progress",
		"data": CurrentProgress,
	})
	Hub.broadcast <- msg
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	Hub.register <- conn

	defer func() {
		Hub.unregister <- conn
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
