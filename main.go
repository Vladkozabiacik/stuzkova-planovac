package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var connections = make([]*websocket.Conn, 0) // Store active connections
var mu sync.Mutex                            // Mutex to protect access to connections

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	// Add the new connection to the list
	mu.Lock()
	connections = append(connections, conn)
	mu.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		log.Printf("Received message: %s", msg)

		// Broadcast the message to all connected clients
		broadcastMessage(msg)
	}

	// Remove the connection when done
	mu.Lock()
	for i, c := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...) // Remove the connection
			break
		}
	}
	mu.Unlock()
}

func broadcastMessage(msg []byte) {
	mu.Lock()
	defer mu.Unlock()
	for _, conn := range connections {
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error writing message:", err)
			conn.Close() // Close the connection on error
		}
	}
}
