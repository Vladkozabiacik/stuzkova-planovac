package main

import (
	"log"
	"net/http"
	"os"

	"stuzkova-planovac/handlers"
	"stuzkova-planovac/models"

	"github.com/joho/godotenv"
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

	db, err := models.ConnectToDB()
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/ws", handlers.HandleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/get-ip", handlers.IPHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/session-status", handlers.SessionStatusHandler)

	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
