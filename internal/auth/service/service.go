package service

// Services - contains all the logic used by the router, but no work with the data
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
	authUtils "github.com/bxxf/znvo-backend/internal/auth/util"
	"github.com/bxxf/znvo-backend/internal/config"
	"github.com/bxxf/znvo-backend/internal/logger"
)

/* ------------------ AuthRouter Definition ------------------ */
type AuthService struct {
	logger           *logger.LoggerInstance
	webAuthnInstance *webauthn.WebAuthn
}

func NewAuthService(logger *logger.LoggerInstance, config *config.Config) *AuthService {

	webAuthn, err := NewWebAuthnClient(logger, config)
	if err != nil {
		logger.Error("Failed to create webauthn object: %v", err)
	}

	return &AuthService{
		logger:           logger,
		webAuthnInstance: webAuthn,
	}
}

/* ------------------ Authenticatiom Functions ------------------ */

func (as *AuthService) InitializeRegister(uuid string) (*webauthn.SessionData, *protocol.CredentialCreation, error) {

	// Create new user
	user := model.NewWebAuthnUser(
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

func (as *AuthService) FinishRegister(session *webauthn.SessionData, userId string, resBody map[string]interface{}) (*webauthn.Credential, error) {
	session.Challenge = base64.RawStdEncoding.EncodeToString([]byte(session.Challenge))

	wUser := model.NewWebAuthnUser(
		[]byte(userId), userId,
	)

	resBodyBytes, err := jsoniter.MarshalToString(resBody)

	as.logger.Debug("Response body: " + resBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response body: %v", err)
	}

	res := &http.Request{
		Body: io.NopCloser(bytes.NewReader([]byte(resBodyBytes))),
	}

	//fmt.Printf("Session UserID: %v\n", base64.URLEncoding.EncodeToString(session.UserID))
	//fmt.Printf("User ID: %v\n", base64.URLEncoding.EncodeToString(wUser.WebAuthnID()))
	user, err := as.webAuthnInstance.FinishRegistration(&wUser, *session, res)

	if err != nil {
		return nil, fmt.Errorf("failed to finish registration: %v", err)
	}

	authUtils.UserCredentials[userId] = user

	return user, nil
}

func (as *AuthService) InitializeLogin(userId string) (*webauthn.SessionData, *protocol.CredentialAssertion, error) {
	if _, ok := authUtils.UserCredentials[userId]; !ok {
		as.logger.Error("User %s not found", userId)
		return nil, nil, fmt.Errorf("user not found")
	}

	user := model.NewWebAuthnUserWithCredentials(
		[]byte(userId), userId,
		[]webauthn.Credential{
			*authUtils.UserCredentials[userId],
		},
	)

	options, sessionData, err := as.webAuthnInstance.BeginLogin(user)
	if err != nil {
		as.logger.Error(err.Error())
		return nil, nil, err
	}

	return sessionData, options, nil
}

func (as *AuthService) FinishLogin(sessionData *webauthn.SessionData, userID string, resBody map[string]interface{}) (*webauthn.Credential, error) {
	sessionData.Challenge = base64.RawStdEncoding.EncodeToString([]byte(sessionData.Challenge))

	if _, ok := authUtils.UserCredentials[userID]; !ok {
		return nil, fmt.Errorf("user not found")
	}

	reqUser := model.NewWebAuthnUserWithCredentials(
		[]byte(userID), userID, []webauthn.Credential{
			*authUtils.UserCredentials[userID],
		},
	)

	resBodyBytes, err := jsoniter.MarshalToString(resBody)

	as.logger.Debug("Response body: " + resBodyBytes)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal response body: %v", err)
	}

	res := &http.Request{
		Body: io.NopCloser(bytes.NewReader([]byte(resBodyBytes))),
	}

	user, err := as.webAuthnInstance.FinishLogin(reqUser, *sessionData, res)
	if err != nil {
		return nil, fmt.Errorf("failed to finish login: %v", err)
	}

	return user, nil
}
