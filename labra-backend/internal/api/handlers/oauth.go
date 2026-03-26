package handlers

import (
	"fmt"
	"net/http"

	"labra-backend/internal/api/services"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	services.Authenticate(w, r)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	body, err := services.Callback(w, r)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("%s", err)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(body)
}
