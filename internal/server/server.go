package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bxxf/znvo-backend/internal/auth/router"
	"github.com/bxxf/znvo-backend/internal/config"
	"github.com/bxxf/znvo-backend/internal/logger"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	authRouter *router.AuthRouter
	logger     *logger.LoggerInstance
	config     *config.Config
}

func NewServer(authRouter *router.AuthRouter, logger *logger.LoggerInstance, config *config.Config, lc fx.Lifecycle) *Server {
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
	go func() {
		s.logger.Info(fmt.Sprintf("Starting server on port %s", s.config.Port))
		// Todo: Dynamically set port
		if err := http.ListenAndServe(":"+s.config.Port, h2c.NewHandler(mux, &http2.Server{})); err != nil {
			s.logger.Error(err.Error())
		}
	}()
}
