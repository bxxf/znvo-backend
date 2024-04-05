package service

import (
	"log"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/bxxf/znvo-backend/internal/config"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/bxxf/znvo-backend/internal/utils"
)

func NewWebAuthnClient(logger *logger.LoggerInstance, config *config.Config) (*webauthn.WebAuthn, error) {
	// Fetch origin from utils based on if app is hosted on fly
	rpId, origin := utils.GetOriginAndRpId(config)

	webAuthnConfig := &webauthn.Config{
		RPDisplayName: "Security Key for Znvo",
		RPID:          rpId,
		RPOrigin:      origin,
	}

	logger.Info("Webauthn initiated with RP ID: " + rpId + " and origin: " + origin)

	// Create WebAuthn object
	webAuthn, err := webauthn.New(webAuthnConfig)
	if err != nil {
		log.Fatalf("Failed to create webauthn object: %v", err)
	}

	return webAuthn, err
}
