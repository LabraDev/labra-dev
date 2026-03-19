package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// TODO, CHECK IF THIS IS A GIANT SECURITY RISK
var (
	ouathConfig = &oauth2.Config{}
	verifier    string
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate gh here

	gh_client := os.Getenv("GH_CLIENT_ID")
	gh_secret := os.Getenv("GH_CLIENT_SECRET")

	fmt.Println(gh_client, "\n---\n", gh_secret)
	// Todo: replace redirect host
	ouathConfig = &oauth2.Config{
		ClientID:     gh_client,
		ClientSecret: gh_secret,
		Scopes:       []string{"repo", "user"},
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/callback",
	}

	// This protects against CSRF attacks
	verifier = oauth2.GenerateVerifier()

	url := ouathConfig.AuthCodeURL("state", oauth2.S256ChallengeOption(verifier))
	fmt.Printf("Please visit %v for the auth\n", url)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Todo: Validate state, this is a secuirty issue!!!
	code := r.URL.Query().Get("code")
	// state := r.URL.Query().Get("state")

	fmt.Println(code)
	if code == "" {
		log.Println("code is empty")
	}

	tok, err := ouathConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		log.Println(err)
	}

	client := ouathConfig.Client(ctx, tok)
	w.Write([]byte("Login succesful"))
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	w.Write(body)
	client.Get("http://localhost:8080/health")
}
