package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"stuzkova-planovac/models"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte(os.Getenv("COOKIE_KEY")))

// HTTP handler for user registration

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	_, err = models.DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", user.Username, hashedPassword)
	if err != nil {
		log.Printf("Error saving user to database: %v", err)
		http.Error(w, "Error saving user to database", http.StatusInternalServerError)
		return
	}

	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Error retrieving session: %v", err)
		http.Error(w, "Could not retrieve session", http.StatusInternalServerError)
		return
	}

	session.Values["username"] = user.Username
	if err := session.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		http.Error(w, "Could not save session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

// HTTP handler for user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
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
	err := models.DB.QueryRow("SELECT password FROM users WHERE username = $1", user.Username).Scan(&hashedPassword)
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

	// Return a success message
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful", "username": user.Username})
}

// HTTP handler for user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Invalidate the session
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1 // Set session age to -1 to delete it
	session.Save(r, w)

	// Return success message
	json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"})
}

// HTTP handler to retrieve session information
func SessionHandler(w http.ResponseWriter, r *http.Request) {
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
