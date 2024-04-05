package router

// Routers - defines structure for gRPC requests and responses and format the data to the correct format

import (
	"context"
	"encoding/base64"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"

	authv1 "github.com/bxxf/znvo-backend/gen/api/auth/v1"
	"github.com/bxxf/znvo-backend/internal/auth/service"
	sessionUtils "github.com/bxxf/znvo-backend/internal/auth/session/utils"
	"github.com/bxxf/znvo-backend/internal/logger"
)

/* ------------------ AuthRouter Definition ------------------ */
type AuthRouter struct {
	logger      *logger.LoggerInstance
	authService *service.AuthService
}

func NewAuthRouter(logger *logger.LoggerInstance, authService *service.AuthService) *AuthRouter {
	return &AuthRouter{
		logger:      logger,
		authService: authService,
	}
}

/* ------------------ Global Variables ------------------ */

// Defining global variables - webauthn and jsoniter
var json = jsoniter.ConfigCompatibleWithStandardLibrary

/* ------------------ Authenticatiom Functions ------------------ */

func (ar *AuthRouter) InitializeRegister(ctx context.Context, req *connect.Request[authv1.InitializeRegisterRequest]) (*connect.Response[authv1.InitializeRegisterResponse], error) {

	// Generate random user ID
	userId := uuid.New().String()
	ar.logger.Debug("Initializing registration for user " + userId)

	// Initialize registration process thru webauthn
	sessionData, options, err := ar.authService.InitializeRegister(userId)
	if err != nil {
		ar.logger.Error(err.Error())
		return nil, err
	}
	// Encode options to JSON
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		ar.logger.Error(err.Error())
		return nil, err
	}

	// Create a new session (Placeholder - internal map -> will be merged to Redis later)
	sessionID := sessionUtils.NewSession(sessionData)

	response := &connect.Response[authv1.InitializeRegisterResponse]{
		Msg: &authv1.InitializeRegisterResponse{
			Sid:     sessionID,
			Options: string(optionsJSON),
		},
	}
	return response, nil
}

func (ar *AuthRouter) FinishRegister(ctx context.Context, req *connect.Request[authv1.FinishRegisterRequest]) (*connect.Response[authv1.FinishRegisterResponse], error) {

	// Get session data from session ID
	sessionData, ok := sessionUtils.GetSession(req.Msg.GetSid())
	if !ok {
		ar.logger.Error("Session not found")
		return nil, MissingSession
	}

	// fake response body - we do not have http.Request in grpc connect request - it is a parameter in webauthn library
	resBody := make(map[string]interface{})

	resBody["type"] = "public-key"
	resBody["id"] = req.Msg.GetCredid()
	resBody["rawId"] = req.Msg.GetCredid()

	resBody["response"] = map[string]interface{}{
		"clientDataJSON":    req.Msg.GetClientdata(),
		"attestationObject": req.Msg.GetAttestation(),
	}

	// Finish registration process thru webauthn
	credential, err := ar.authService.FinishRegister(sessionData, req.Msg.GetUserid(), resBody)

	if err != nil {
		ar.logger.Error(err.Error())
		return nil, err
	}

	response := &connect.Response[authv1.FinishRegisterResponse]{
		Msg: &authv1.FinishRegisterResponse{
			Token: base64.StdEncoding.EncodeToString(credential.PublicKey),
		},
	}

	return response, nil
}
