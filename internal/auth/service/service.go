package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	jsoniter "github.com/json-iterator/go"

	"github.com/bxxf/znvo-backend/internal/auth/model"
	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/bxxf/znvo-backend/internal/redis"
	"github.com/bxxf/znvo-backend/internal/utils"
)

// AuthService provides methods for user authentication using WebAuthn.
type AuthService struct {
	logger           *logger.LoggerInstance
	webAuthnInstance *webauthn.WebAuthn
	redisService     *redis.RedisService
}

// NewAuthService creates a new AuthService instance with the provided logger and configuration.
func NewAuthService(logger *logger.LoggerInstance, cfg *envconfig.EnvConfig, redisService *redis.RedisService) *AuthService {
	webAuthn, err := NewWebAuthnClient(logger, cfg)
	if err != nil {
		logger.Error("Failed to create WebAuthn object", "error", err)
	}

	return &AuthService{
		logger:           logger,
		webAuthnInstance: webAuthn,
		redisService:     redisService,
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

	// marshal response body to string for use in FinishRegistration
	resBodyBytes, err := jsoniter.MarshalToString(resBody)
	if err != nil {
		return nil, utils.HandleError(err, "failed to marshal response body", *as.logger)
	}

	// fake request body
	req := &http.Request{
		Body: io.NopCloser(bytes.NewBufferString(resBodyBytes)),
	}

	// finish registration
	credential, err := as.webAuthnInstance.FinishRegistration(&user, *session, req)
	if err != nil {
		return nil, utils.HandleError(err, "failed to finish registration", *as.logger)
	}

	// marshal credential to JSON and store in Redis
	credJson, err := json.Marshal(credential)
	if err != nil {
		return nil, utils.HandleError(err, "failed to marshal credential", *as.logger)
	}

	go func() {
		redisClient := as.redisService.GetClient()
		_, err = redisClient.Set(redisClient.Context(), "cred:"+userID, string(credJson), 0).Result()
		if err != nil {
			as.logger.Error("Failed to store credential in Redis: ", err)
		}

	}()

	return credential, nil
}

// InitializeLogin starts the login process for an existing user identified by userId.
// It returns session data and credential assertion options for the client to complete the login.
func (as *AuthService) InitializeLogin(userID string) (*webauthn.SessionData, *protocol.CredentialAssertion, error) {

	redisClient := as.redisService.GetClient()
	credJson, err := redisClient.Get(redisClient.Context(), "cred:"+userID).Result()

	if err != nil {
		return nil, nil, utils.HandleError(err, "failed to retrieve credential from Redis", *as.logger)
	}

	var cred webauthn.Credential
	err = json.Unmarshal([]byte(credJson), &cred)
	if err != nil {
		return nil, nil, utils.HandleError(err, "failed to unmarshal credential", *as.logger)
	}
	//user := model.NewWebAuthnUserWithCredentials([]byte(userID), userID, []webauthn.Credential{*util.UserCredentials[userID]})
	options, sessionData, err := as.webAuthnInstance.BeginLogin(model.NewWebAuthnUserWithCredentials([]byte(userID), userID, []webauthn.Credential{cred}))
	if err != nil {
		return nil, nil, utils.HandleError(err, "failed to begin login", *as.logger)
	}

	return sessionData, options, nil
}

// FinishLogin completes the login process using the provided session data and response body.
// It validates the user's credentials and returns the user's credential on success.
func (as *AuthService) FinishLogin(sessionData *webauthn.SessionData, userID string, resBody map[string]interface{}) (*webauthn.Credential, error) {
	sessionData.Challenge = base64.RawStdEncoding.EncodeToString([]byte(sessionData.Challenge))

	redisClient := as.redisService.GetClient()
	credJson, err := redisClient.Get(redisClient.Context(), "cred:"+userID).Result()

	if err != nil {
		return nil, utils.HandleError(err, "failed to retrieve credential from Redis", *as.logger)
	}

	var cred webauthn.Credential
	err = json.Unmarshal([]byte(credJson), &cred)
	if err != nil {
		return nil, utils.HandleError(err, "failed to unmarshal credential", *as.logger)
	}
	// user := model.NewWebAuthnUserWithCredentials([]byte(userID), userID, []webauthn.Credential{*authUtil.UserCredentials[userID]})
	resBodyBytes, err := jsoniter.MarshalToString(resBody)
	if err != nil {
		return nil, utils.HandleError(err, "failed to marshal response body", *as.logger)
	}

	req := &http.Request{
		Body: io.NopCloser(bytes.NewBufferString(resBodyBytes)),
	}

	credential, err := as.webAuthnInstance.FinishLogin(model.NewWebAuthnUserWithCredentials([]byte(userID), userID, []webauthn.Credential{cred}), *sessionData, req)

	if err != nil {
		return nil, utils.HandleError(err, "failed to finish login", *as.logger)
	}

	return credential, nil
}
