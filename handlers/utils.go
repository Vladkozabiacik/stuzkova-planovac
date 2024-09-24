package handlers

import (
	"encoding/json"
	"net/http"
)

func IPHandler(w http.ResponseWriter, r *http.Request) {
	userIP := r.RemoteAddr
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ip": userIP})
}
