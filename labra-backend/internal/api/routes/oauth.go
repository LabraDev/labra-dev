package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func Oauth(s *fuego.Server) {
	fuego.GetStd(s, "/v1/login", handlers.LoginHandler)
	fuego.GetStd(s, "/v1/callback", handlers.CallbackHandler)
}
