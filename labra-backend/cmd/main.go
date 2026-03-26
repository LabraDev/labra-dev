package main

import (
	"fmt"
	"os"

	"labra-backend/internal/api/routes"
	"labra-backend/internal/api/services"

	"github.com/go-fuego/fuego"
	"github.com/lpernett/godotenv"
)

const (
	PORT = "8080"
	HOST = "localhost"
)

func init() {
	err := godotenv.Load("./../.env")
	if err != nil {
		fmt.Println(err)
	}

	gh_client := os.Getenv("GH_CLIENT_ID")
	gh_secret := os.Getenv("GH_CLIENT_SECRET")

	services.InitOauth(gh_client, gh_secret)
}

func main() {
	listenOn := HOST + ":" + PORT

	s := fuego.NewServer(
		fuego.WithAddr(listenOn),
	)

	routes.HealthRoute(s)
	routes.Oauth(s)

	// TODO: probably switch this to TLS

	fmt.Println("Server starting on :", listenOn)
	s.Run()
}
