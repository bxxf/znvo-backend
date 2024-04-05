package router

// Routers - defines structure for gRPC requests and responses and format the data to the correct format

import (
	"context"

	"connectrpc.com/connect"
	authv1 "github.com/bxxf/znvo-backend/gen/api/auth/v1"
	"github.com/bxxf/znvo-backend/internal/auth/service"
	sessionUtils "github.com/bxxf/znvo-backend/internal/auth/session/utils"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
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
