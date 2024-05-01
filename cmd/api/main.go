package main

import (
	"context"
	"log"

	"go.uber.org/fx"

	aiRouter "github.com/bxxf/znvo-backend/internal/ai/router"
	aiService "github.com/bxxf/znvo-backend/internal/ai/service"
	authRouter "github.com/bxxf/znvo-backend/internal/auth/router"
	"github.com/bxxf/znvo-backend/internal/auth/service"
	"github.com/bxxf/znvo-backend/internal/auth/session"
	"github.com/bxxf/znvo-backend/internal/auth/token"
	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
	"github.com/bxxf/znvo-backend/internal/redis"
	"github.com/bxxf/znvo-backend/internal/server"
)

func main() {
	app := fx.New(
		fx.Provide(
			logger.NewLogger,
			envconfig.NewEnvConfig,
			redis.NewRedisService,
			service.NewAuthService,
			session.NewSessionRepository,
			aiService.NewStreamStore,
			authRouter.NewAuthRouter,
			aiService.NewAiService,
			aiRouter.NewAiRouter,
			server.NewServer,
			token.NewTokenRepository,
		),
		fx.Invoke(
			func(s *server.Server) {
			},
			func(c *envconfig.EnvConfig) {
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
