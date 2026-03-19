package handlers

import (
	"encoding/json"
	"net/http"
)

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Status string
		Text   string
	}{
		Status: "success",
		Text:   "healthy",
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}
