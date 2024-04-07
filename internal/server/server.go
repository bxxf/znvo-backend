package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rs/cors"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/bxxf/znvo-backend/internal/auth/router"
	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
)

type Server struct {
	authRouter *router.AuthRouter
	logger     *logger.LoggerInstance
	config     *envconfig.EnvConfig
}

func NewServer(authRouter *router.AuthRouter, logger *logger.LoggerInstance, config *envconfig.EnvConfig, lc fx.Lifecycle) *Server {
	server := &Server{
		authRouter: authRouter,
		logger:     logger,
		config:     config,
	}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			server.StartServer()
			return nil
		},
	})
	return server
}

// start server on an instance of server, therefore all routers is already provided
func (s *Server) StartServer() {
	mux := s.defineRoutes()
	// cors
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(mux)

	go func() {
		s.logger.Info(fmt.Sprintf("Starting server on port %s", s.config.Port))
		// Todo: Dynamically set port
		if err := http.ListenAndServe(":"+s.config.Port, h2c.NewHandler(handler, &http2.Server{})); err != nil {
			s.logger.Error(fmt.Sprintf("Failed to start server: %v", err))
		}
	}()
}
