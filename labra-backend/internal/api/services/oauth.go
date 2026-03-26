package services

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var oauthConfig = &oauth2.Config{}

// Note having verifier as a global is extremly dangerous, will move to db when I get to it
var verifier string

func InitOauth(gh_client, gh_secret string) {
	oauthConfig = &oauth2.Config{
		ClientID:     gh_client,
		ClientSecret: gh_secret,
		Scopes:       []string{"repo", "user"},
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/v1/callback",
	}
}

func Authenticate(w http.ResponseWriter, r *http.Request) {
	verifier = oauth2.GenerateVerifier()

	// TODO: generate a state, probably with the std hash lib
	url := oauthConfig.AuthCodeURL("state", oauth2.S256ChallengeOption(verifier))

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func Callback(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	ctx := context.Background()

	code := r.URL.Query().Get("code")
	_ = r.URL.Query().Get("state")

	if code == "" {
		return nil, fmt.Errorf("error: code is unavaliable")
	}

	tok, err := oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, err
	}

	client := oauthConfig.Client(ctx, tok)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
