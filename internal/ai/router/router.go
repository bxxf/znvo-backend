package router

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	aiv1 "github.com/bxxf/znvo-backend/gen/api/ai/v1"
	"github.com/bxxf/znvo-backend/internal/ai/service"
	"github.com/bxxf/znvo-backend/internal/auth/token"
	"github.com/bxxf/znvo-backend/internal/logger"
)

// Routers - defines structure for gRPC requests and responses and format the data to the correct format

/* ------------------ AIRouter Definition ------------------ */

type AiRouter struct {
	logger          *logger.LoggerInstance
	tokenRepository *token.TokenRepository

	aiService   *service.AiService
	streamStore *service.StreamStore
}

func NewAiRouter(logger *logger.LoggerInstance, tokenRepository *token.TokenRepository, aiService *service.AiService, streamStore *service.StreamStore) *AiRouter {
	return &AiRouter{
		logger:          logger,
		tokenRepository: tokenRepository,
		aiService:       aiService,
		streamStore:     streamStore,
	}
}

var contx = context.Background()

/* ------------------ AI Functions ------------------ */

func (ar *AiRouter) StartSession(
	ctx context.Context,
	req *connect.Request[aiv1.StartSessionRequest],
	stream *connect.ServerStream[aiv1.StartSessionResponse],
) error {
	userToken := req.Msg.UserToken

	if userToken == "" {
		return status.Error(codes.InvalidArgument, "User token is required")
	}

	parsedToken, err := ar.tokenRepository.ParseAccessToken(userToken)

	if err != nil {
		return status.Error(codes.Unauthenticated, "Invalid user token")
	}

	ar.logger.Debug("Starting session for user " + parsedToken.UserID)

	resp, err := ar.aiService.StartConversation(contx)
	if err != nil {
		return status.Error(codes.Internal, "Failed to start conversation")
	}

	ar.streamStore.SaveStream(resp.SessionID, stream, parsedToken.UserID)

	stream.Send(&aiv1.StartSessionResponse{
		SessionId:   resp.SessionID,
		Message:     resp.Message,
		MessageType: aiv1.MessageType_CHAT,
	})

	for {
		select {
		case <-ctx.Done(): // Check if the context is done/cancelled
			fmt.Println("Stream context cancelled, closing stream")
			ar.aiService.CloseSession(resp.SessionID)
			ar.streamStore.CloseSession(resp.SessionID)
			return ctx.Err()
		default:
			time.Sleep(time.Minute * 1)
		}
	}

	return nil
}

func (ar *AiRouter) SendMsg(ctx context.Context, req *connect.Request[aiv1.SendMsgRequest]) (*connect.Response[aiv1.SendMsgResponse], error) {
	message := req.Msg.Message
	sessionID := req.Msg.SessionId
	userToken := req.Msg.UserToken

	if userToken == "" {
		return nil, status.Error(codes.InvalidArgument, "User token is required")
	}

	parsedToken, err := ar.tokenRepository.ParseAccessToken(userToken)

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid user token")
	}

	if !ar.streamStore.CheckSessionOwner(sessionID, parsedToken.UserID) {
		return nil, status.Error(codes.PermissionDenied, "You do not have permission to send messages to this session")
	}

	if message == "" {
		return nil, status.Error(codes.InvalidArgument, "Message is required")
	}

	go func() {

		resp, err := ar.aiService.SendMessage(contx, sessionID, message, service.MessageTypeUser)

		if err != nil {
			ar.logger.Error("Failed to send message: ", err)
			return
		}

		ar.streamStore.SendMessage(sessionID, &aiv1.StartSessionResponse{
			Message:     resp.Message,
			SessionId:   resp.SessionID,
			MessageType: aiv1.MessageType_CHAT,
		})

	}()
	return &connect.Response[aiv1.SendMsgResponse]{
		Msg: &aiv1.SendMsgResponse{
			Message: "Successfully sent message",
		},
	}, nil
}
