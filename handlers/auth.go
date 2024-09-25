package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"stuzkova-planovac/models"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var (
	store = sessions.NewCookieStore([]byte("your-secret-key")) // TODO: key encrypted from db or smth
)

func init() {
	store.Options = &sessions.Options{
		Path:     "/", // Path where the cookie is valid
		MaxAge:   3600,
		HttpOnly: true,  // Prevents JavaScript access to the cookie
		Secure:   false, // Set to true when using HTTPS
		SameSite: http.SameSiteLaxMode,
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil || user.Username == "" || user.Password == "" {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Bad request: invalid input", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var exists bool
	err = models.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)", user.Username).Scan(&exists)
	if err != nil || exists {
		log.Printf("Username already exists or error checking: %v", err)
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}

	_, err = models.DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", user.Username, hashedPassword)
	if err != nil {
		log.Printf("Error saving user to database: %v", err)
		http.Error(w, "Error saving user to database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully", "username": user.Username})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var hashedPassword string
	err := models.DB.QueryRow("SELECT password FROM users WHERE username = $1", user.Username).Scan(&hashedPassword)
	if err != nil {
		log.Printf("Error retrieving user: %v", err)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	session, err := store.Get(r, "session-name")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		http.Error(w, "Error starting session", http.StatusInternalServerError)
		return
	}

	session.Values["username"] = user.Username
	if err := session.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		http.Error(w, "Error saving session", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s logged in successfully.", user.Username)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful", "username": user.Username})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Error retrieving session", http.StatusInternalServerError)
		return
	}

	delete(session.Values, "username")
	session.Save(r, w)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"})
}

func SessionStatusHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, "Error retrieving session", http.StatusInternalServerError)
		return
	}

	loggedIn := false
	username := ""
	if usernameVal, ok := session.Values["username"].(string); ok {
		loggedIn = true
		username = usernameVal
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"loggedIn": loggedIn,
		"username": username,
	})
}
