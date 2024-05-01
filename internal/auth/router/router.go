package router

// Routers - defines structure for gRPC requests and responses and format the data to the correct format

import (
	"context"

	"connectrpc.com/connect"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/bxxf/znvo-backend/gen/api/auth/v1"
	"github.com/bxxf/znvo-backend/gen/api/auth/v1/authconnect"
	"github.com/bxxf/znvo-backend/internal/auth/service"
	"github.com/bxxf/znvo-backend/internal/auth/session"
	"github.com/bxxf/znvo-backend/internal/auth/token"
	"github.com/bxxf/znvo-backend/internal/auth/util"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/bxxf/znvo-backend/internal/utils"
)

/* ------------------ AuthRouter Definition ------------------ */

type AuthRouter struct {
	logger            *logger.LoggerInstance
	authService       *service.AuthService
	tokenRepository   *token.TokenRepository
	sessionRepository *session.SessionRepository
}

type Definer interface {
	authconnect.AuthServiceHandler
}

func NewAuthRouter(logger *logger.LoggerInstance, authService *service.AuthService, tokenRepository *token.TokenRepository, sessionRepository *session.SessionRepository) *AuthRouter {
	return &AuthRouter{
		logger:            logger,
		authService:       authService,
		tokenRepository:   tokenRepository,
		sessionRepository: sessionRepository,
	}
}

/* ------------------ Global Variables ------------------ */

// Defining global variables - webauthn and jsoniter
var json = jsoniter.ConfigCompatibleWithStandardLibrary

/* ------------------ Authenticatiom Functions ------------------ */

func (ar *AuthRouter) InitializeRegister(ctx context.Context, req *connect.Request[authv1.InitializeRegisterRequest]) (*connect.Response[authv1.InitializeRegisterResponse], error) {

	// Generate random user ID
	userID := uuid.New().String()
	ar.logger.Debug("Initializing registration for user " + userID)

	// Initialize registration process thru webauthn
	sessionData, options, err := ar.authService.InitializeRegister(userID)
	if err != nil {
		return nil, utils.HandleError(err, "failed to initialize registration", *ar.logger)
	}
	// Encode options to JSON
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		ar.logger.Error(err.Error())
		return nil, err
	}

	// Create a new session (Placeholder - internal map -> will be merged to Redis later)
	sessionID, err := ar.sessionRepository.NewSession(sessionData)
	if err != nil {
		return nil, utils.HandleError(err, "failed to create session", *ar.logger)
	}

	response := &connect.Response[authv1.InitializeRegisterResponse]{
		Msg: &authv1.InitializeRegisterResponse{
			Sid:     sessionID,
			Options: string(optionsJSON),
		},
	}
	return response, nil
}

func (ar *AuthRouter) FinishRegister(ctx context.Context, req *connect.Request[authv1.FinishRegisterRequest]) (*connect.Response[authv1.FinishRegisterResponse], error) {
	// Usingchannels to get session data concurrently
	sessionDataChan := make(chan *webauthn.SessionData, 1)
	errChan := make(chan error, 1)

	go func() {
		sessionData, err := ar.sessionRepository.GetSession(req.Msg.GetSid())
		if err != nil {
			errChan <- err
			return
		}
		sessionDataChan <- sessionData
	}()

	// Transforming the request message to the body concurrently
	resBodyChan := make(chan *map[string]interface{}, 1)
	go func() {
		// Transform the request message to the body for webauthn - it needs to be http.Request so we need to fake it
		resBody := util.TransformRegisterMsgToBody(req.Msg)
		resBodyChan <- &resBody
	}()

	// Initialize variables for the results
	var sessionData *webauthn.SessionData
	var resBody *map[string]interface{}
	var err error

	// Wait for session data
	select {
	case sessionData = <-sessionDataChan:
	}

	// Wait for request body transformation
	select {
	case resBody = <-resBodyChan:
	case err = <-errChan:
		return nil, utils.HandleError(err, "failed to get session data", *ar.logger)
	}

	credential, err := ar.authService.FinishRegister(sessionData, req.Msg.GetUserid(), *resBody)
	if err != nil {
		return nil, utils.HandleError(err, "failed to finish registration", *ar.logger)
	}

	ar.logger.Debug("Registration completed for user " + string(credential.PublicKey))

	token, err := ar.tokenRepository.CreateAccessToken(req.Msg.GetUserid())
	if err != nil {
		return nil, utils.HandleError(err, "failed to create access token", *ar.logger)
	}

	response := &connect.Response[authv1.FinishRegisterResponse]{
		Msg: &authv1.FinishRegisterResponse{
			Token: token,
		},
	}

	return response, nil
}

func (ar *AuthRouter) GetUser(ctx context.Context, req *connect.Request[authv1.GetUserRequest]) (*connect.Response[authv1.GetUserResponse], error) {
	// Get user details
	user, err := ar.tokenRepository.ParseAccessToken(req.Msg.GetToken())

	if err != nil {
		return nil, status.New(codes.Unauthenticated, err.Error()).Err()
	}

	response := &connect.Response[authv1.GetUserResponse]{
		Msg: &authv1.GetUserResponse{
			Id: user.UserID,
		},
	}

	return response, nil
}

func (ar *AuthRouter) InitializeLogin(ctx context.Context, req *connect.Request[authv1.InitializeLoginRequest]) (*connect.Response[authv1.InitializeLoginResponse], error) {
	userID := req.Msg.GetUserid()

	if userID == "" {
		return nil, status.New(codes.InvalidArgument, "user id is required").Err()
	}

	ar.logger.Debug("Initializing login for user " + userID)

	// Initialize login process thru webauthn
	sessionData, options, err := ar.authService.InitializeLogin(userID)
	if err != nil {
		return nil, utils.HandleError(err, "failed to initialize login", *ar.logger)
	}
	// Encode options to JSON
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, utils.HandleError(err, "failed to marshal options", *ar.logger)
	}

	// Create a new session (Placeholder - internal map -> will be merged to Redis later)
	sessionID, err := ar.sessionRepository.NewSession(sessionData)

	if err != nil {
		return nil, utils.HandleError(err, "failed to create session", *ar.logger)
	}

	response := &connect.Response[authv1.InitializeLoginResponse]{
		Msg: &authv1.InitializeLoginResponse{
			Sid:     sessionID,
			Options: string(optionsJSON),
		},
	}
	return response, nil
}

func (ar *AuthRouter) FinishLogin(ctx context.Context, req *connect.Request[authv1.FinishLoginRequest]) (*connect.Response[authv1.FinishLoginResponse], error) {
	// Usingchannels to get session data concurrently
	sessionDataChan := make(chan *webauthn.SessionData, 1)
	errChan := make(chan error, 1)

	go func() {
		sessionData, err := ar.sessionRepository.GetSession(req.Msg.GetSid())
		if err != nil {
			errChan <- err
			return
		}
		sessionDataChan <- sessionData
	}()

	// Transforming the request message to the body concurrently
	resBodyChan := make(chan *map[string]interface{}, 1)
	go func() {
		// Transform the request message to the body for webauthn - it needs to be http.Request so we need to fake it
		resBody := util.TransformLoginMsgToBody(req.Msg)
		resBodyChan <- &resBody
	}()

	// Initialize variables for the results
	var sessionData *webauthn.SessionData
	var resBody *map[string]interface{}
	var err error

	// Wait for session data
	select {
	case sessionData = <-sessionDataChan:
	}

	// Wait for request body transformation
	select {
	case resBody = <-resBodyChan:
	case err = <-errChan:
		return nil, utils.HandleError(err, "failed to get session data", *ar.logger)
	}

	credential, err := ar.authService.FinishLogin(sessionData, req.Msg.GetUserid(), *resBody)
	if err != nil {
		return nil, utils.HandleError(err, "failed to finish login", *ar.logger)
	}

	ar.logger.Debug("Login completed for user " + string(credential.PublicKey))

	token, err := ar.tokenRepository.CreateAccessToken(req.Msg.GetUserid())
	if err != nil {
		return nil, utils.HandleError(err, "failed to create access token", *ar.logger)
	}

	response := &connect.Response[authv1.FinishLoginResponse]{
		Msg: &authv1.FinishLoginResponse{
			Token: token,
		},
	}

	return response, nil
}
