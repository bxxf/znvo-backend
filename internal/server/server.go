package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rs/cors"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	aiRouter "github.com/bxxf/znvo-backend/internal/ai/router"
	authRouter "github.com/bxxf/znvo-backend/internal/auth/router"
	dataRouter "github.com/bxxf/znvo-backend/internal/data/router"
	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
)

type Server struct {
	authRouter *authRouter.AuthRouter
	aiRouter   *aiRouter.AiRouter
	dataRouter *dataRouter.DataRouter

	logger *logger.LoggerInstance
	config *envconfig.EnvConfig
}

func NewServer(authRouter *authRouter.AuthRouter, aiRouter *aiRouter.AiRouter, logger *logger.LoggerInstance, config *envconfig.EnvConfig, dataRouter *dataRouter.DataRouter, lc fx.Lifecycle) *Server {
	server := &Server{
		authRouter: authRouter,
		aiRouter:   aiRouter,
		dataRouter: dataRouter,
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
		AllowedOrigins: []string{"https://znvo.co.uk", "http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(mux)

	go func() {
		s.logger.Info(fmt.Sprintf("Starting server on port %s", s.config.Port))
		if err := http.ListenAndServe(":"+s.config.Port, h2c.NewHandler(handler, &http2.Server{})); err != nil {
			s.logger.Error(fmt.Sprintf("Failed to start server: %v", err))
		}
	}()
}
