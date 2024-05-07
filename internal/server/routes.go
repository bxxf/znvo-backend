package server

import (
	"net/http"

	"connectrpc.com/grpcreflect"

	"github.com/bxxf/znvo-backend/gen/api/ai/v1/aiconnect"
	"github.com/bxxf/znvo-backend/gen/api/auth/v1/authconnect"
	"github.com/bxxf/znvo-backend/gen/api/data/v1/dataconnect"
)

func (s *Server) defineRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle(authconnect.NewAuthServiceHandler(s.authRouter))
	mux.Handle(aiconnect.NewAiServiceHandler(s.aiRouter))
	mux.Handle(dataconnect.NewDataServiceHandler(s.dataRouter))

	// Add reflection for development
	if s.config.Env == "development" {

		reflector := grpcreflect.NewStaticReflector(
			"auth.v1.AuthService",
			"ai.v1.AiService",
			"data.v1.DataService",
		)

		mux.Handle(grpcreflect.NewHandlerV1(reflector))
		mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	}

	return mux
}
