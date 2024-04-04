package model

import (
	"encoding/base64"

	"github.com/go-webauthn/webauthn/webauthn"
)

// This file contains utility functions for webAuthn to make the code cleaner
// this was necessary to make to override the webauthn.User struct to change encoding logic

type WebAuthnUser struct {
	Id          []byte
	DisplayName string
	Credentials []webauthn.Credential
}

// Overriding webauthn.User functions
func NewWebAuthnUser(id []byte, displayName string) WebAuthnUser {
	return WebAuthnUser{
		Id:          id,
		DisplayName: displayName,
	}
}

func NewWebAuthnUserWithCredentials(id []byte, displayName string, credentials []webauthn.Credential) *WebAuthnUser {
	return &WebAuthnUser{
		Id:          id,
		DisplayName: displayName,
		Credentials: credentials,
	}
}

func (wau *WebAuthnUser) WebAuthnID() []byte {
	encoded1 := base64.URLEncoding.EncodeToString(wau.Id)
	return []byte(base64.URLEncoding.EncodeToString([]byte(encoded1)))
}

func (wau *WebAuthnUser) CleanID() string {
	return base64.URLEncoding.EncodeToString(wau.Id)
}

func (wau *WebAuthnUser) WebAuthnName() string {
	return wau.DisplayName
}

func (wau *WebAuthnUser) WebAuthnDisplayName() string {
	return wau.DisplayName
}

func (wau *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

func (wau *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return wau.Credentials
}

func (wau *WebAuthnUser) SetWebAuthnCredentials(credentials []webauthn.Credential) {
	wau.Credentials = credentials
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
