package server

import (
	"net/http"

	"connectrpc.com/grpcreflect"
	"github.com/bxxf/znvo-backend/gen/api/auth/v1/authconnect"
)

func (s *Server) defineRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle(authconnect.NewAuthServiceHandler(s.authRouter))

	reflector := grpcreflect.NewStaticReflector(
		"auth.v1.AuthService",
	)

	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
