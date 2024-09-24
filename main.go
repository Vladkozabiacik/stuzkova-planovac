package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Global variables and configuration
var (
	upgrader    = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }} // Upgrader for WebSocket connections
	connections = make([]*websocket.Conn, 0)                                                  // Stores active WebSocket connections
	mu          sync.Mutex                                                                    // Mutex for handling concurrent WebSocket connections
	db          *sql.DB                                                                       // Database connection
	store       = sessions.NewCookieStore([]byte(os.Getenv("cookie_store_key")))              // Cookie store for session management
)

// Main function to load environment variables, connect to the database, and set up HTTP routes
func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	// Get port number from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to port 8080 if not specified
	}

	// Connect to the database
	db, err = connectToDB()
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}
	defer db.Close()

	// Setup HTTP routes
	http.HandleFunc("/ws", handleWebSocket)                 // WebSocket endpoint
	http.Handle("/", http.FileServer(http.Dir("./static"))) // Serve static files
	http.HandleFunc("/get-ip", ipHandler)                   // Endpoint to get client's IP address
	http.HandleFunc("/register", registerHandler)           // User registration endpoint
	http.HandleFunc("/login", loginHandler)                 // User login endpoint
	http.HandleFunc("/logout", logoutHandler)               // User logout endpoint
	http.HandleFunc("/session", sessionHandler)             // Session info endpoint

	// Start the server
	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Connect to PostgreSQL database
func connectToDB() (*sql.DB, error) {
	// Get database credentials from environment variables
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// Connection string to PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open the connection
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	// Ping to ensure the connection is valid
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully connected to PostgreSQL database!")
	return db, nil
}

// WebSocket handler for handling new connections and messages
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	userIP := r.RemoteAddr // Get the client's IP address

	// Add the new connection to the global list
	mu.Lock()
	connections = append(connections, conn)
	mu.Unlock()

	// Read messages from the WebSocket connection
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// Parse incoming message data
		var messageData map[string]interface{}
		if err := json.Unmarshal(msg, &messageData); err == nil {
			if messageData["type"] == "message" { // Regular chat message
				msg, _ = json.Marshal(messageData)
			} else if messageData["type"] == "cursor" { // Cursor data
				msg, _ = json.Marshal(messageData)
			}

			broadcastMessage(msg) // Broadcast the message to all connected clients
		}
	}

	// Handle disconnection and broadcast the event
	disconnectionMessage := map[string]interface{}{
		"type": "disconnected",
		"ip":   userIP,
	}
	msg, _ := json.Marshal(disconnectionMessage)
	broadcastMessage(msg)

	// Remove the connection from the list of active connections
	mu.Lock()
	for i, c := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
			break
		}
	}
	mu.Unlock()
}

// Broadcasts a message to all connected WebSocket clients
func broadcastMessage(msg []byte) {
	mu.Lock()
	defer mu.Unlock()
	for _, conn := range connections {
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error writing message:", err)
			conn.Close()
		}
	}
}

// HTTP handler for retrieving the client's IP address
func ipHandler(w http.ResponseWriter, r *http.Request) {
	userIP := r.RemoteAddr // Get client's IP address
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ip": userIP})
}

// HTTP handler for user registration
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body to get username and password
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Insert user data into the database
	_, err = db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", user.Username, hashedPassword)
	if err != nil {
		http.Error(w, "Error saving user to database", http.StatusInternalServerError)
		return
	}

	// Create a session for the registered user
	session, _ := store.Get(r, "session")
	session.Values["username"] = user.Username
	session.Save(r, w)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

// HTTP handler for user login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode request body to get username and password
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Retrieve the hashed password from the database
	var hashedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = $1", user.Username).Scan(&hashedPassword)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Compare the hashed password with the provided password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create a session for the logged-in user
	session, _ := store.Get(r, "session")
	session.Values["username"] = user.Username
	session.Save(r, w)

	// Log the successful login
	log.Printf("User %s logged in successfully.", user.Username)

	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful", "username": user.Username})
}

// HTTP handler for user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Invalidate the session
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1 // Set session age to -1 to delete it
	session.Save(r, w)

	// Return success message
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"})
}

// HTTP handler to retrieve session information
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve the session
	session, _ := store.Get(r, "session")

	// Check if the session has a valid username
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "No active session"})
		return
	}

	// Return the username from the session
	json.NewEncoder(w).Encode(map[string]string{"username": username})
}
