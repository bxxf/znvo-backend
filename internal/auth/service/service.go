package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	jsoniter "github.com/json-iterator/go"

	"github.com/bxxf/znvo-backend/internal/auth/model"
	"github.com/bxxf/znvo-backend/internal/auth/util"
	authUtil "github.com/bxxf/znvo-backend/internal/auth/util"
	"github.com/bxxf/znvo-backend/internal/config"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/bxxf/znvo-backend/internal/utils"
)

// AuthService provides methods for user authentication using WebAuthn.
type AuthService struct {
	logger           *logger.LoggerInstance
	webAuthnInstance *webauthn.WebAuthn
}

// NewAuthService creates a new AuthService instance with the provided logger and configuration.
func NewAuthService(logger *logger.LoggerInstance, cfg *config.Config) *AuthService {
	webAuthn, err := NewWebAuthnClient(logger, cfg)
	if err != nil {
		logger.Error("Failed to create WebAuthn object", "error", err)
	}

	return &AuthService{
		logger:           logger,
		webAuthnInstance: webAuthn,
	}
}

// InitializeRegister begins the registration process for a new user identified by uuid.
// It returns session data and credential creation options necessary for completing registration.
func (as *AuthService) InitializeRegister(uuid string) (*webauthn.SessionData, *protocol.CredentialCreation, error) {
	user := model.NewWebAuthnUser([]byte(uuid), uuid)

	options, sessionData, err := as.webAuthnInstance.BeginRegistration(&user)
	if err != nil {
		return nil, nil, utils.HandleError(err, "failed to begin registration", *as.logger)
	}

	return sessionData, options, nil
}

// FinishRegister completes the user registration process using the provided session data and response body.
// It returns the newly created user credential on success.
func (as *AuthService) FinishRegister(session *webauthn.SessionData, userID string, resBody map[string]interface{}) (*webauthn.Credential, error) {
	session.Challenge = base64.RawStdEncoding.EncodeToString([]byte(session.Challenge))

	user := model.NewWebAuthnUser([]byte(userID), userID)
	resBodyBytes, err := jsoniter.MarshalToString(resBody)
	if err != nil {
		return nil, utils.HandleError(err, "failed to marshal response body", *as.logger)
	}

	req := &http.Request{
		Body: io.NopCloser(bytes.NewBufferString(resBodyBytes)),
	}

	credential, err := as.webAuthnInstance.FinishRegistration(&user, *session, req)
	if err != nil {
		return nil, utils.HandleError(err, "failed to finish registration", *as.logger)
	}

	authUtil.UserCredentials[userID] = credential
	return credential, nil
}

// InitializeLogin starts the login process for an existing user identified by userId.
// It returns session data and credential assertion options for the client to complete the login.
func (as *AuthService) InitializeLogin(userID string) (*webauthn.SessionData, *protocol.CredentialAssertion, error) {
	if _, ok := authUtil.UserCredentials[userID]; !ok {
		return nil, nil, fmt.Errorf("user %s not found", userID)
	}

	user := model.NewWebAuthnUserWithCredentials([]byte(userID), userID, []webauthn.Credential{*util.UserCredentials[userID]})
	options, sessionData, err := as.webAuthnInstance.BeginLogin(user)
	if err != nil {
		return nil, nil, utils.HandleError(err, "failed to begin login", *as.logger)
	}

	return sessionData, options, nil
}

// FinishLogin completes the login process using the provided session data and response body.
// It validates the user's credentials and returns the user's credential on success.
func (as *AuthService) FinishLogin(sessionData *webauthn.SessionData, userID string, resBody map[string]interface{}) (*webauthn.Credential, error) {
	sessionData.Challenge = base64.RawStdEncoding.EncodeToString([]byte(sessionData.Challenge))

	if _, ok := authUtil.UserCredentials[userID]; !ok {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	user := model.NewWebAuthnUserWithCredentials([]byte(userID), userID, []webauthn.Credential{*authUtil.UserCredentials[userID]})
	resBodyBytes, err := jsoniter.MarshalToString(resBody)
	if err != nil {
		return nil, utils.HandleError(err, "failed to marshal response body", *as.logger)
	}

	req := &http.Request{
		Body: io.NopCloser(bytes.NewBufferString(resBodyBytes)),
	}

	credential, err := as.webAuthnInstance.FinishLogin(user, *sessionData, req)
	if err != nil {
		return nil, utils.HandleError(err, "failed to finish login", *as.logger)
	}

	return credential, nil
}
