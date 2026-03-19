package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lpernett/godotenv"
	"labra-backend/internal/api/routes"
)

const PORT = "8080"

func main() {
	err := godotenv.Load("./../.env")
	if err != nil {
		fmt.Println(err)
	}

	mux := http.NewServeMux()

	routes.HealthRoute(mux)
	routes.Oauth(mux)
	// TODO: probably switch this to TLS

	listenOn := "localhost" + ":" + PORT

	fmt.Println("Server starting on :", listenOn)
	err = http.ListenAndServe(listenOn, mux)
	if err != nil {
		log.Fatalln(err)
	}
}
