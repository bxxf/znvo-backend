package service

import (
	"log"

	authUtils "github.com/bxxf/znvo-backend/internal/auth/utils"
	"github.com/bxxf/znvo-backend/internal/config"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/bxxf/znvo-backend/internal/utils"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

/* ------------------ AuthRouter Definition ------------------ */

type AuthService struct {
	logger           *logger.LoggerInstance
	webAuthnInstance *webauthn.WebAuthn
}

func NewAuthService(logger *logger.LoggerInstance, config *config.Config) *AuthService {
	// Fetch origin from utils based on if app is hosted on fly
	rpId, origin := utils.GetOriginAndRpId(config)

	webAuthnConfig := &webauthn.Config{
		RPDisplayName: "Security Key for Znvo",
		RPID:          rpId,
		RPOrigin:      origin,
	}

	// Create WebAuthn object
	webAuthn, err := webauthn.New(webAuthnConfig)
	if err != nil {
		log.Fatalf("Failed to create webauthn object: %v", err)
	}

	return &AuthService{
		logger:           logger,
		webAuthnInstance: webAuthn,
	}
}

/* ------------------ Global Variables ------------------ */

// Settings for webauthn registration
var authSelection = protocol.AuthenticatorSelection{
	RequireResidentKey: protocol.ResidentKeyRequired(),
	UserVerification:   protocol.VerificationPreferred,
}

/* ------------------ Authenticatiom Functions ------------------ */

func (as *AuthService) InitializeRegister(uuid string) (*webauthn.SessionData, *protocol.CredentialCreation, error) {

	// Create new user
	user := authUtils.NewWebAuthnUser(
		[]byte(uuid), uuid,
	)

	// Begin registration process
	options, sessionData, err := as.webAuthnInstance.BeginRegistration(&user, webauthn.WithAuthenticatorSelection(authSelection))
	if err != nil {
		as.logger.Error(err.Error())
		return nil, nil, err
	}

	return sessionData, options, nil
}
