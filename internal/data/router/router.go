package router

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	datav1 "github.com/bxxf/znvo-backend/gen/api/data/v1"
	"github.com/bxxf/znvo-backend/internal/auth/token"
	"github.com/bxxf/znvo-backend/internal/data/service"
	"github.com/bxxf/znvo-backend/internal/logger"
)

// Routers - defines structure for gRPC requests and responses and format the data to the correct format

/* ------------------ DataRouter Definition ------------------ */

type DataRouter struct {
	logger          *logger.LoggerInstance
	dataService     *service.DataService
	tokenRepository *token.TokenRepository
}

func NewDataRouter(logger *logger.LoggerInstance, service *service.DataService, tokenRepository *token.TokenRepository) *DataRouter {
	return &DataRouter{
		logger:          logger,
		dataService:     service,
		tokenRepository: tokenRepository,
	}
}

var contx = context.Background()

/* ------------------ Data Functions ------------------ */

func (dr *DataRouter) ShareUserData(ctx context.Context, req *connect.Request[datav1.ShareDataRequest]) (*connect.Response[datav1.ShareDataResponse], error) {
	data := req.Msg.Data
	receiver := req.Msg.Recipient
	userToken := req.Msg.UserToken

	if userToken == "" {
		return nil, status.Error(codes.InvalidArgument, "User token is required")
	}

	parsedToken, err := dr.tokenRepository.ParseAccessToken(userToken)

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid user token")
	}

	if data == "" {
		return nil, status.Error(codes.InvalidArgument, "Data is required")
	}

	err = dr.dataService.ShareData(data, parsedToken.UserID, receiver)

	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to share data")
	}

	return &connect.Response[datav1.ShareDataResponse]{
		Msg: &datav1.ShareDataResponse{
			Success: true,
		},
	}, nil
}

func (dr *DataRouter) GetSharedData(ctx context.Context, req *connect.Request[datav1.GetSharedDataRequest]) (*connect.Response[datav1.GetSharedDataResponse], error) {
	userToken := req.Msg.UserToken

	if userToken == "" {
		return nil, status.Error(codes.InvalidArgument, "User token is required")
	}

	parsedToken, err := dr.tokenRepository.ParseAccessToken(userToken)

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid user token")
	}

	data := dr.dataService.GetSharedData(parsedToken.UserID)

	if data == "" {
		return nil, status.Error(codes.NotFound, "No shared data found")
	}

	return &connect.Response[datav1.GetSharedDataResponse]{
		Msg: &datav1.GetSharedDataResponse{
			Data: data,
		},
	}, nil
}
