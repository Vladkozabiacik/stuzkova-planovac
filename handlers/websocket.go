package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var connections = make([]*websocket.Conn, 0)
var mu sync.Mutex

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	userIP := r.RemoteAddr

	mu.Lock()
	connections = append(connections, conn)
	mu.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		var messageData map[string]interface{}
		if err := json.Unmarshal(msg, &messageData); err == nil {
			msg, _ = json.Marshal(messageData)
			BroadcastMessage(msg)
		}
	}

	disconnect(conn, userIP)
}

func disconnect(conn *websocket.Conn, userIP string) {
	mu.Lock()
	for i, c := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
			break
		}
	}
	mu.Unlock()

	disconnectionMessage := map[string]interface{}{
		"type": "disconnected",
		"ip":   userIP,
	}
	msg, _ := json.Marshal(disconnectionMessage)
	BroadcastMessage(msg)
}

func BroadcastMessage(msg []byte) {
	mu.Lock()
	defer mu.Unlock()
	for _, conn := range connections {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Println("Error writing message:", err)
			conn.Close()
		}
	}
}
