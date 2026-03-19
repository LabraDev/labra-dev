package routes

import (
	"net/http"

	"labra-backend/internal/api/handlers"
)

func HealthRoute(mux *http.ServeMux) {
	mux.HandleFunc("/health", handlers.HandleHealth)
}
