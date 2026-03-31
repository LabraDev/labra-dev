package utils

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
)

func GenerateState() (string, error) {
	state := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, state); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(state), nil
}

func GetClientIP(r *http.Request) string {
	val := r.Header.Get("X-Forwarded-For")
	if len(val) == 0 {
		return r.RemoteAddr
	}
	return val
}
