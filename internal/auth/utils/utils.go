package utils

import (
	"errors"
	"net/http"
	"strings"
)

// Base64 utils for webauthn - webauthn usually sends base64 encoded binary data

func Base64ToUrlSafe(base64 string) string {
	base64 = strings.ReplaceAll(base64, "+", "-")
	base64 = strings.ReplaceAll(base64, "/", "_")
	return strings.TrimRight(base64, "=")
}

func GetUserIDFromRequest(r *http.Request) (string, error) {
	// Extract the user ID from query parameters
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		return "", errors.New("user ID not provided")
	}
	return userID, nil
}
