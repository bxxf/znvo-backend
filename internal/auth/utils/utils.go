package utils

import (
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Base64 utils for webauthn - webauthn usually sends base64 encoded binary data

func Base64ToUrlSafe(base64 string) string {
	base64 = strings.ReplaceAll(base64, "+", "-")
	base64 = strings.ReplaceAll(base64, "/", "_")
	return strings.TrimRight(base64, "=")
}

var (
	UserCredentials = make(map[string]*webauthn.Credential)
	DemoUser        = WebAuthnUser{id: []byte("user_id_12345"), displayName: "testUser"}
)
