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

var (
	upgrader    = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	connections = make([]*websocket.Conn, 0)
	mu          sync.Mutex
	db          *sql.DB
	store       = sessions.NewCookieStore([]byte("super-secret-key")) // key to change + add to .env
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err = connectToDB()
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}
	defer db.Close()

	http.HandleFunc("/ws", handleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/get-ip", ipHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler) // New route for logout

	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
func connectToDB() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully connected to PostgreSQL database!")
	return db, nil
}
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
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

		var cursorData map[string]interface{}
		if err := json.Unmarshal(msg, &cursorData); err == nil && cursorData["type"] == "cursor" {
			cursorData["ip"] = userIP
			msg, _ = json.Marshal(cursorData)
			broadcastMessage(msg)
		} else {
			log.Printf("Received plain message: %s", msg)
			broadcastMessage(msg)
		}
	}

	disconnectionMessage := map[string]interface{}{
		"type": "disconnected",
		"ip":   userIP,
	}
	msg, _ := json.Marshal(disconnectionMessage)
	broadcastMessage(msg)

	mu.Lock()
	for i, c := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
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
			conn.Close()
		}
	}
}

func ipHandler(w http.ResponseWriter, r *http.Request) {
	userIP := r.RemoteAddr
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ip": userIP})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", user.Username, hashedPassword)
	if err != nil {
		http.Error(w, "Error saving user to database", http.StatusInternalServerError)
		return
	}

	// Create session for registered user
	session, _ := store.Get(r, "session")
	session.Values["username"] = user.Username
	session.Save(r, w)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var hashedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = $1", user.Username).Scan(&hashedPassword)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create session for logged in user
	session, _ := store.Get(r, "session")
	session.Values["username"] = user.Username
	session.Save(r, w)

	// Log the login event
	log.Printf("User %s logged in successfully.", user.Username)

	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful", "username": user.Username})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1 // Invalidate the session
	session.Save(r, w)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"})
}
