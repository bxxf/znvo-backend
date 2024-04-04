package main

import (
	"context"
	"log"

	"go.uber.org/fx"

	"github.com/bxxf/znvo-backend/internal/auth/router"
	"github.com/bxxf/znvo-backend/internal/auth/service"
	"github.com/bxxf/znvo-backend/internal/auth/token"
	"github.com/bxxf/znvo-backend/internal/config"
	"github.com/bxxf/znvo-backend/internal/key"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/bxxf/znvo-backend/internal/server"
)

func main() {
	app := fx.New(
		fx.Provide(
			logger.NewLogger,
			config.NewConfig,
			service.NewAuthService,
			router.NewAuthRouter,
			server.NewServer,
			key.NewKeyRepository,
			token.NewTokenRepository,
		),
		fx.Invoke(
			func(s *server.Server) {
			},
			func(c *config.Config) {
			}),
	)

	ctx := context.Background()
	// start the application
	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// wait for the application to stop
	<-app.Done()

	// stop the application
	if err := app.Stop(ctx); err != nil {
		log.Fatal(err)
	}

}
