package utils

import (
	"encoding/base64"

	"github.com/go-webauthn/webauthn/webauthn"
)

// This file contains utility functions for webAuthn to make the code cleaner

type WebAuthnUser struct {
	id          []byte
	displayName string
	credentials []webauthn.Credential
}

// Overriding webauthn.User functions
func NewWebAuthnUser(id []byte, displayName string) WebAuthnUser {
	return WebAuthnUser{
		id:          id,
		displayName: displayName,
	}
}

func NewWebAuthnUserWithCredentials(id []byte, displayName string, credentials []webauthn.Credential) *WebAuthnUser {
	return &WebAuthnUser{
		id:          id,
		displayName: displayName,
		credentials: credentials,
	}
}

func (wau *WebAuthnUser) WebAuthnID() []byte {
	encoded1 := base64.URLEncoding.EncodeToString(wau.id)
	return []byte(base64.URLEncoding.EncodeToString([]byte(encoded1)))
}

func (wau *WebAuthnUser) CleanID() string {
	return base64.URLEncoding.EncodeToString(wau.id)
}

func (wau *WebAuthnUser) WebAuthnName() string {
	return wau.displayName
}

func (wau *WebAuthnUser) WebAuthnDisplayName() string {
	return wau.displayName
}

func (wau *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

func (wau *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return wau.credentials
}

func (wau *WebAuthnUser) SetWebAuthnCredentials(credentials []webauthn.Credential) {
	wau.credentials = credentials
}

type ClientCredential struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	RawID    string `json:"rawId"`
	Response struct {
		ClientDataJSON    string `json:"clientDataJSON"`
		AttestationObject string `json:"attestationObject"`
	} `json:"response"`
	Transports []string `json:"transports,omitempty"`
}
